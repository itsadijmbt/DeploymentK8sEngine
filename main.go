package main

import (
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Daemon struct {

	//map {name}:{*ptr} for mutex of each process
	serviceLocks map[string]*sync.Mutex
	// protects service map from races
	locksMutex sync.Mutex
	jobs       chan DeployService
}

type DeployService struct {
	service string
	version string
}

// initlaise a new daemon

func NewDaemon(worker int) *Daemon {

	d := &Daemon{
		serviceLocks: make(map[string]*sync.Mutex),
		jobs:         make(chan DeployService, 500),
	}

	for i := 0; i < worker; i++ {
		go d.Worker()
	}
	return d
}

func main() {

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
			version := readFile(service)

			d.jobs <- DeployService{service: service, version: version}
		}
	}
}

func (d *Daemon) DeployService(job DeployService) {

	//get the lock for the service
	lock := d.getServiceLocker(job.service)

	//lock the service
	lock.Lock()
	defer lock.Unlock()

	//getting cwv
	currentVersion := readFile(job.service)

	DeployTok8s(job.service, currentVersion)

	// for concurrnet redeployement if file changed in between
	newVersion := readFile(job.service)

	if newVersion != currentVersion {
		d.jobs <- DeployService{service: job.service, version: newVersion}
	}

}

func readFile(filepath string) string {

	content, err := os.ReadFile(filepath)

	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))

}
