package main

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

//@@@@@@@@@@@@@@@@@@@@@@@@@@@@ FLOW @@@@@@@@@@@@@@@@@@@@@@@@
/*
 waitForRollout waits for a Kubernetes deployment to complete its rollout
  polls the deployment status every 2 seconds until:
 all replicas are updated with new version
 all replicas are ready and healthy
 no unavailable replicas
 rturns error if timeout or deployment fails

 api -> deployement cluster (cretes new replica set)-> Replica controller creates Pods ->
 -> scheduler assign pods to nodes -> kubelet on node pull image -> starts containers -> run health probes
 -> pod is ready -> old are killed

*/

func (d *Daemon) WaitForRollout(deplyementClients v1.DeploymentInterface, serviceName string, namespace string, timeout time.Duration) error {

	//1 context
	// withtimeout/with cancel always need a parent so wee pass the root context i.e background
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	tickCount := 0
	//polling

	for {
		// check for any two cases either ticker has done or it has ticked
		select {

		case <-ctx.Done():
			log.Printf("timeout waiting for deployment rollout")
			return fmt.Errorf("timeout waiting for deployment rollout", timeout)

		case <-ticker.C:
			// inside channel time to check if a new tick is delivered
			tickCount++
			deployment, err := deplyementClients.Get(ctx, serviceName, metav1.GetOptions{})
			if err != nil {
				log.Printf("failed to get deployement")
				return fmt.Errorf("failed to get deployment: %w", err)
			}

			status := deployment.Status
			desired := *deployment.Spec.Replicas
			// Log status every 10 seconds
			if tickCount%5 == 0 {
				log.Printf(" [%s] Waiting... Updated: %d/%d | Ready: %d/%d | Unavail: %d",
					serviceName, status.UpdatedReplicas, desired, status.ReadyReplicas, desired, status.UnavailableReplicas)
			}
			if status.UpdatedReplicas == desired && status.ReadyReplicas == desired && status.UnavailableReplicas == 0 {
				// all are upated, all are ready and unavailvble =0
				return nil
			}

			if status.UnavailableReplicas > 0 {
				// i.e some replica are failing
				podErr := d.checkPodErrors(namespace, deployment.Spec.Selector)
				if podErr != nil {
					return fmt.Errorf("rollout failed: %w", podErr)
				}
			}

		}
	}
}

// pods -> pod-> pod.status.ContainerStatus (containerStatus)-> containerStaus.Status has many states (terminated, waiting )
// func (d *Daemon) checkPodErrors(serviceName string, namespace string) error {

// 	ctx := context.Background()

// 	pods, err := d.k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
// 		LabelSelector: fmt.Sprintf("app=%s", serviceName),
// 	})

// 	if err != nil {
// 		// we dont exit might be temporary
// 		log.Printf(" Could not list pods: %v", err)
// 		return nil
// 	}

// 	for _, pod := range pods.Items {

// 		for _, containerStatus := range pod.Status.ContainerStatuses {

// 			if containerStatus.State.Waiting != nil {
// 				reason := containerStatus.State.Waiting.Reason
// 				message := containerStatus.State.Waiting.Message

// 				// image doesn't exist or auth failed
// 				if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
// 					log.Printf("timeout waiting for deployment rollout")
// 					return fmt.Errorf("image pull failed: %s - %s", reason, message)
// 				}

// 				// Invalid image format
// 				if reason == "InvalidImageName" {
// 					return fmt.Errorf("invalid image name: %s", message)
// 				}

// 				// Container keeps crashing on startup
// 				if reason == "CrashLoopBackOff" {
// 					return fmt.Errorf("container crashing repeatedly: %s", message)
// 				}

// 				// Container waiting for other reasons (resources, etc)
// 			}

// 			//!nil ie crashed
// 			if containerStatus.State.Terminated != nil {
// 				exitCode := containerStatus.State.Terminated.ExitCode

// 				if exitCode != 0 {
// 					return fmt.Errorf("Container Terminated with Code %d: %s,", exitCode, containerStatus.State.Terminated.Message)
// 				}

// 			}

// 		}

// 	}

// 	return nil

// }
// checkPodErrors now uses the Official Selector from the deployment
func (d *Daemon) checkPodErrors(namespace string, labelSelector *metav1.LabelSelector) error {
	ctx := context.Background()

	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return fmt.Errorf("invalid label selector: %v", err)
	}
	targetString := selector.String()

	allpods, err := d.k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("❌ CRITICAL: Could not list pods in namespace '%s': %v", namespace, err)
		return nil
	}

	if len(allpods.Items) == 0 {
		log.Printf(" Namespace appears empty to GoClient %s", namespace)
		return nil
	}

	// we find 0 pods, something is very wrong (Deployment exists but no pods?)
	var matchedPods []corev1.Pod

	for _, pod := range allpods.Items {
		podLabel := labels.Set(pod.Labels)
		if selector.Matches(podLabel) {
			matchedPods = append(matchedPods, pod)
		} else {
			log.Printf("   [MISMATCH] Pod: %s | Has: %v | Wants: %s", pod.Name, pod.Labels, targetString)
		}
	}

	for _, pod := range matchedPods {

		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Waiting != nil {
				reason := containerStatus.State.Waiting.Reason
				message := containerStatus.State.Waiting.Message

				if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
					return fmt.Errorf("image pull failed: %s - %s", reason, message)
				}
				if reason == "CrashLoopBackOff" {
					return fmt.Errorf("crash loop: %s", message)
				}
				if reason == "InvalidImageName" {
					return fmt.Errorf("invalid image name: %s", message)
				}
				if containerStatus.State.Terminated != nil {
					exitCode := containerStatus.State.Terminated.ExitCode

					if exitCode != 0 {
						return fmt.Errorf("Container Terminated with Code %d: %s,", exitCode, containerStatus.State.Terminated.Message)
					}

				}

			}
		}
	}
	return nil
}
