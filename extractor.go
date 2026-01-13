package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func getECR() string {
	ecr := os.Getenv("ECR_REPO")

	if ecr == "" {
		fmt.Println("ECR_URL not set in env variables")
	}
	return ecr
}

// complete file path | docker version | namespace
func (d *Daemon) DeployTok8s(serviceName string, dockerImageVersion string, namespace string) error {

	// get image name -> get deploys for ns + create a context -> get current dep -> update dep in spec ->  apply

	//core k8s api's
	// fmt.Printf(">>> WOULD DEPLOY: %s in namespace %s\n", dockerImageVersion, namespace)

	ecrRepo := getECR()

	// fmt.Printf(">>> ECR REPO LINK IS %s", ecrRepo)

	//!!!!!!!! note now since filename is changed we have to change image extraction

	var fullImage string

	if ecrRepo != "" {
		fullImage = fmt.Sprintf("%s:%s", ecrRepo, dockerImageVersion)
	} else {
		fullImage = dockerImageVersion // Use as-is if no ECR
	}

	log.Printf(" Full image : %s", fullImage)

	//ensure the ns exists and if not create a new

	// err := d.ensureNs(namespace)
	// if err != nil {
	// 	log.Printf(" namespace error  in extractor.go \n ")
	// 	return err
	// }

	// get deps for this ns

	deploymentsClient := d.k8sClient.AppsV1().Deployments(namespace)
	ctx := context.TODO()

	// curr dep from this ns

	deployment, err := deploymentsClient.Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		log.Printf(" failed to get deployements error  in extractor.go \n ")
		return err
	}

	// update the dep in spec

	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("no containers found in deployment %s", serviceName)
	}

	//update
	oldImage := deployment.Spec.Template.Spec.Containers[0].Image
	deployment.Spec.Template.Spec.Containers[0].Image = fullImage

	log.Printf("🔄 Updating image: %s → %s", oldImage, fullImage)

	// now apply

	_, err = deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})

	if err != nil {
		log.Printf(" deployment error  in extractor.go \n ")
		return err
	}
	return nil
}

//k8s will not check for health we have to create a service for hat

// k8s has to create a namepsace if it is not present
func (d *Daemon) ensureNs(namespace string) error {

	ctx := context.TODO()

	_, err := d.k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})

	if err == nil {
		// Namespace exists
		return nil
	}

	if !strings.Contains(err.Error(), "not found") {
		log.Printf(" system error  in extractor.go \n ")
		return fmt.Errorf("System Error\n")
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	_, err = d.k8sClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})

	if err != nil {
		log.Printf(" namespace creation error  in extractor.go \n ")
		return fmt.Errorf("Failed to create namespace %s: %v", namespace, err)
	}

	return nil

}
