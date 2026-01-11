package main

import (
	"fmt"
	"log"
	"os"
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
		if event.Op&fsnotify.Write == fsnotify.Write {

			service := event.Name

			version, namespace := readFile(service)

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

	fmt.Printf("[DEPLOY] Processing: %s\n", job.service)

	depFile := job.service
	lastfile := strings.Replace(depFile, ".dep", ".last", 1)

	newVersion, newNamespace := readFile(job.service)
	lastVersion, lastNamespace := readFile(lastfile)

	fmt.Printf("[COMPARE] New: %s/%s | Last: %s/%s\n",
		newNamespace, newVersion, lastNamespace, lastVersion)

	if newVersion != lastVersion || newNamespace != lastNamespace {
		fmt.Printf("[DEPLOYING] %s to %s\n", newVersion, newNamespace)

		d.DeployTok8s(depFile, newVersion, newNamespace)
		content := newVersion + " " + newNamespace
		os.WriteFile(lastfile, []byte(content), 0644)
		fmt.Println("[SAVED] Updated .last file")
	} else {
		fmt.Println("[SKIP] No changes detected")
	}

	//re check for new file
	currentVersion, currNamespace := readFile(depFile)
	if newVersion != currentVersion || newNamespace != currNamespace {
		d.jobs <- DeployService{service: job.service, version: newVersion, namespace: job.namespace}
	}

}

func readFile(filepath string) (string, string) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", ""
	}

	parts := strings.Split(strings.TrimSpace(string(content)), " ")

	if len(parts) != 2 {
		return "", ""
	}

	return parts[0], parts[1]
}
