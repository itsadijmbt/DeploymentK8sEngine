package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

// the main engine
func (d *Daemon) watchFiles() {

	watcher, _ := fsnotify.NewWatcher()
	//add the file engine
	watcher.Add("/Users/win 10/Desktop/GO/K8sEngine/deps")

	//creating a channel for concurrent deps
	for event := range watcher.Events {
		if event.Op&fsnotify.Write == fsnotify.Write && strings.HasSuffix(event.Name, ".dep") {

			//service == filename
			service := event.Name
			_, namespace := extractServiceName(service)

			version := readFile(service)

			d.jobs <- DeployService{service: service, version: version, namespace: namespace}
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
		// fmt.Printf("[DEPLOYING] %s to %s\n", newVersion, newNamespace)

		// extract {service-name}_{namsepace}.dep

		serviceName, namespace := extractServiceName(depFile)
		err1 := d.DeployTok8s(serviceName, newVersion, namespace)
		slackengine(err1, serviceName, newVersion, namespace)

		if err1 != nil {
			fmt.Printf("[ERROR] Deployment failed: %v\n", err1)
		} else {
			content := newVersion
			os.WriteFile(lastfile, []byte(content), 0644)
		}

		// fmt.Println("[SAVED] Updated .last file")
	} else {
		// fmt.Println("[SKIP] No changes detected")
	}

	//re check for new file

	currentVersion := readFile(depFile)
	if newVersion != currentVersion {
		d.jobs <- DeployService{service: job.service, version: newVersion, namespace: job.namespace}
	}

}

func readFile(filepath string) string {
	content, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.WriteFile(filepath, []byte(""), 0644)
		}
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

func extractServiceName(filePath string) (string, string) {

	filename := filepath.Base(filePath)
	namewithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	parts := strings.Split(namewithoutExt, "_")

	if len(parts) != 2 {
		log.Printf("invalid filename: '%s', expected: {service}_{namespace}.dep", filename)
		return "", ""
	}

	service := parts[0]
	namespace := parts[1]

	return service, namespace
}
