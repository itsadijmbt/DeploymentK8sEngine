package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"k8s.io/client-go/kubernetes"
)

type Daemon struct {

	//map {name}:{*ptr} for mutex of each process
	serviceLocks map[string]*sync.Mutex
	// protects service map from races
	locksMutex sync.Mutex
	jobs       chan DeployService
	k8sClient  *kubernetes.Clientset
}

type DeployService struct {
	service   string
	version   string
	namespace string
}

// initlaise a new daemon

func NewDaemon(worker int) *Daemon {

	// create a new k8s client
	k8sClient, err := Newk8sclient()

	if err != nil {
		fmt.Printf("Error creating k8s client: %v\n", err)
		os.Exit(1)
	}

	d := &Daemon{
		serviceLocks: make(map[string]*sync.Mutex),
		jobs:         make(chan DeployService, 500),
		k8sClient:    k8sClient,
	}

	for i := 0; i < worker; i++ {
		go d.Worker()
	}
	return d
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
		os.Exit(1)
	}
	err = ConfigureSystem()

	if err != nil {
		log.Fatalf("Warning: .env file not found, using system environment variables")

	}

	daemon := NewDaemon(100)
	// this main go routine watches the file fills the channel
	daemon.watchFiles()

}

func (d *Daemon) Worker() {
	for job := range d.jobs {
		d.DeployService(job)
	}
}

// service map + dependency locker
// protects the service map data structure itself
// protects the value associated with each key,
// each key gets its own independent lock
func (d *Daemon) getServiceLocker(service string) *sync.Mutex {
	d.locksMutex.Lock()
	defer d.locksMutex.Unlock()

	if d.serviceLocks[service] == nil {
		d.serviceLocks[service] = &sync.Mutex{}
	}
	return d.serviceLocks[service]
}
func getPATH() (string, error) {
	path := os.Getenv("DEPS")

	if path == "" {
		log.Fatal("DEPS environment variable not set")
		return " ", fmt.Errorf("DEPS environment variable not set")
	}
	return path, nil
}

// the main engine
func (d *Daemon) watchFiles() {
	// 1. Create the watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Failed to create watcher:", err)
	}
	defer watcher.Close()

	path, err := getPATH()

	if err != nil {
		return
	}

	err = watcher.Add(path)
	if err != nil {
		log.Fatal("Failed to watch folder:", err)
	}
	log.Printf(" Watching path: %s", path)

	lastEventTime := make(map[string]time.Time)
	var maplock sync.Mutex

	for {
		select {

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			isDep := strings.HasSuffix(event.Name, ".dep")
			isWrite := event.Op&fsnotify.Write == fsnotify.Write
			isCreate := event.Op&fsnotify.Create == fsnotify.Create

			if isDep && (isWrite || isCreate) {

				maplock.Lock()
				// prevents Duplicate Jobs, stops 1 Save from becoming 5 Jobs.
				//  this is the main fix for the "Loop".
				lastTime, exists := lastEventTime[event.Name]
				now := time.Now()

				if exists && now.Sub(lastTime) < 500*time.Millisecond {
					// Debounce: If less than 500ms, ignore.
					// (Reduced from 1s to 500ms to feel more responsive to manual edits)
					maplock.Unlock()
					continue
				}
				lastEventTime[event.Name] = now
				// manual unlock to prevent deadlock inside a non returning for loop
				// not uisng defer
				maplock.Unlock()

				service := event.Name
				_, namespace, err := extractServiceName(service)

				if err != nil {
					log.Printf("⚠️ Check filename format: %v", err)
					continue
				}

				version := readFile(service)
				if version == "" {
					continue
				}

				// Send job
				d.jobs <- DeployService{service: service, version: version, namespace: namespace}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}

			log.Printf("⚠️ Watcher error: %v", err)
		}
	}
}

func (d *Daemon) DeployService(job DeployService) {

	//get the lock for the service
	lock := d.getServiceLocker(job.service)

	//lock the service
	lock.Lock()
	defer lock.Unlock()

	// fmt.Printf("[DEPLOY] Processing: %s\n", job.service)

	depFile := job.service
	lastfile := strings.Replace(depFile, ".dep", ".last", 1)

	newVersion := readFile(job.service)
	lastVersion := readFile(lastfile)

	// the reason we ingore the ns because for a new ns the process is different

	fmt.Printf("file name is is %s\n", depFile)
	fmt.Printf(" the new version = %s old is = %s\n", newVersion, lastVersion)

	// fmt.Printf("[COMPARE] New: %s/%s | Last: %s/%s\n",
	// 	newNamespace, newVersion, lastNamespace, lastVersion)

	if newVersion != lastVersion {

		serviceName, namespace, err := extractServiceName(depFile)
		if err != nil {
			log.Printf("❌ Invalid filename: %v", err)
			return
		}
		versionAtStart := newVersion
		err1 := d.DeployTok8s(serviceName, newVersion, namespace)
		slackengine(err1, serviceName, newVersion, namespace)

		if err1 != nil {
			fmt.Printf("[ERROR] Deployment failed: %v\n", err1)

			log.Printf("⏸️  Waiting 60 seconds before next attempt...")
			time.Sleep(3 * time.Second)
			return

		} else {

			content := newVersion
			os.WriteFile(lastfile, []byte(content), 0644)
			log.Printf("✅ Updated .last file to %s", newVersion)
		}
		currentVersion := readFile(depFile)
		if currentVersion != versionAtStart {
			log.Printf("🔄 File changed during deployment (%s → %s), re-enqueueing",
				versionAtStart, currentVersion)

			d.jobs <- DeployService{
				service:   job.service,
				version:   currentVersion, //suing current version
				namespace: namespace,
			}
		} else {
			log.Printf("📝 File unchanged, no re-enqueue")
		}
	} else {
		log.Printf("⏭️  Skipped: versions already match")
	}

}

func readFile(filepath string) string {
	content, err := os.ReadFile(filepath)
	if err != nil {

		return ""
	}

	text := strings.TrimSpace(string(content))

	return text
}

func slackengine(err error, serviceName string, newVersion string, newNamespace string) {

	if err != nil {

		log.Printf("❌ Deployment failed: %v", err)

		SlackNotifier(SlackMessage{
			Message: "Deployment Failed",
			Details: fmt.Sprintf(
				"service:%s\nversion:%s\nnamespace:%s\nerror:%s",
				serviceName,
				newVersion,
				newNamespace,
				err.Error(),
			),
			MessageType: MsgDeploymentFailure,
		})

	} else {

		log.Printf("✅ Deployment succeeded")

		SlackNotifier(SlackMessage{
			Message: "Deployment Successful",
			Details: fmt.Sprintf(
				"service:%s\nversion:%s\nnamespace:%s",
				serviceName,
				newVersion,
				newNamespace,
			),
			MessageType: MsgDeploymentSuccess,
		})

	}

}

func extractServiceName(filePath string) (string, string, error) {

	filename := filepath.Base(filePath)
	namewithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	parts := strings.Split(namewithoutExt, "_")

	if len(parts) != 2 {
		log.Printf("invalid filename: '%s', expected: {service}_{namespace}.dep", filename)
		return "", "", fmt.Errorf("invalid filename: '%s', expected: {service}_{namespace}.dep", filename)
	}

	service := parts[0]
	namespace := parts[1]

	if service == "" || namespace == "" {
		return "", "", fmt.Errorf(
			"invalid filename: service and namespace cannot be empty",
		)
	}

	return service, namespace, nil
}
