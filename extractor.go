package main

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// needs the to crete a clientset which needs a kubeconfig
func Newk8sclient() (*kubernetes.Clientset, error) {

	home, err := os.UserHomeDir()

	if err != nil {
		return nil, fmt.Errorf("Failed to get the home Dir")

	}

	// now get the k8s config file
	kubeconfigPath := filepath.Join(home, ".kube", "config")

	//loading the k8s config file

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)

	if err != nil {
		return nil, fmt.Errorf("Failed to load the k8s config file")
	}
	//a collection of clients for k8s API groups ->CoreV1(),APPSV1(),BATCHV1()
	//create a clientset-> a wrapper for k8sapi call + brings in
	// http client + load certs + rate limits
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to crete a new clientset")
	}
	return clientset, nil

}

// complete file path | docker version | namespace
func (d *Daemon) DeployTok8s(filepath string, dockerImageVersion string, namespace string) {

	//core k8s api's
	fmt.Printf(">>> WOULD DEPLOY: %s in namespace %s\n", dockerImageVersion, namespace)

	// deployments, err := d.k8sClient.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	fmt.Printf(" [ERROR] Failed to list deployments: %v\n", err)
	// 	return
	// }

	// fmt.Printf("[K8S] Found %d deployments in namespace '%s'\n", len(deployments.Items), namespace)
}
