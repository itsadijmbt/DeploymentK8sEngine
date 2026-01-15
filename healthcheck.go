package main

import (
	"context"
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	//polling

	for {
		// check for any two cases either ticker has done or it has ticked
		select {

		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for deployment rollout", timeout)

		case <-ticker.C:
			// inside channel time to check if a new tick is delivered

			deployment, err := deplyementClients.Get(ctx, serviceName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get deployment: %w", err)
			}

			status := deployment.Status
			desired := *deployment.Spec.Replicas

			if status.UnavailableReplicas > 0 {
				// i.e some replica are failing
				podErr := d.checkPodErrors(serviceName, namespace)
				if podErr != nil {
					return fmt.Errorf("rollout failed: %w", podErr)
				}
			}
			if status.UpdatedReplicas == desired && status.ReadyReplicas == desired && status.UnavailableReplicas == 0 {
				// all are upated, all are ready and unavailvble =0
				return nil
			}

		}
	}
}

// pods -> pod-> pod.status.ContainerStatus (containerStatus)-> containerStaus.Status has many states (terminated, waiting )
func (d *Daemon) checkPodErrors(serviceName string, namespace string) error {

	ctx := context.Background()

	pods, err := d.k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", serviceName),
	})

	if err != nil {
		// we dont exit might be temporary
		log.Printf(" Could not list pods: %v", err)
		return nil
	}

	for _, pod := range pods.Items {

		for _, containerStatus := range pod.Status.ContainerStatuses {

			if containerStatus.State.Waiting != nil {
				reason := containerStatus.State.Waiting.Reason
				message := containerStatus.State.Waiting.Message

				// image doesn't exist or auth failed
				if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
					return fmt.Errorf("image pull failed: %s - %s", reason, message)
				}

				// Invalid image format
				if reason == "InvalidImageName" {
					return fmt.Errorf("invalid image name: %s", message)
				}

				// Container keeps crashing on startup
				if reason == "CrashLoopBackOff" {
					return fmt.Errorf("container crashing repeatedly: %s", message)
				}

				// Container waiting for other reasons (resources, etc)
			}

			//!nil ie crashed
			if containerStatus.State.Terminated != nil {
				exitCode := containerStatus.State.Terminated.ExitCode

				if exitCode != 0 {
					return fmt.Errorf("Container Terminated with Code %d: %s,", exitCode, containerStatus.State.Terminated.Message)
				}

			}

		}

	}

	return nil

}
