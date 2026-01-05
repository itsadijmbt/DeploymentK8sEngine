package main

import (
	"fmt"
	"time"
)

// complete file path | docker version | namespace
func DeployTok8s(filepath string, dockerImageVersion string, namespace string) {

	//core k8s api's
	fmt.Printf(">>> WOULD DEPLOY: %s in namespace %s\n", dockerImageVersion, namespace)
	time.Sleep(10 * time.Second)
}
