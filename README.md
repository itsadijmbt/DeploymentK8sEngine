<img width="580" height="240" alt="image" src="https://github.com/user-attachments/assets/3b0130fd-9203-4adc-b6fe-1f9ff8d2a5fb" /><div align="center">

#  DeploymentK8sEngine
### Event-Driven GitOps Controller built from First Principles in Go

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)](https://kubernetes.io/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge)](LICENSE)


_A lightweight, concurrency-safe deployment daemon that watches your filesystem and updates Kubernetes in real-time._

[**Explore the Docs**](#-Architecture) · [**View Demo**](#-screenshots) · [**Guide Bug**](#-QuickStart)

</div>

---

##  Overview

**DeploymentK8sEngine** eliminates the need for manual `kubectl` commands by implementing a **file-based GitOps workflow**. It watches a dedicated dependency directory for changes and automatically reconciles your Kubernetes cluster state to match.

Unlike generic tools, this engine was **built from scratch** to solve specific distributed system challenges: race conditions, atomic file saves, and concurrent deployment locking.

> **Why build this?**
> To master Kubernetes internals, Go concurrency patterns (Channels, Mutexes, Goroutines), and system-level file watching—not by copying templates, but by solving the hard problems myself.

---

##  Key Features

| Feature | Description |
| :--- | :--- |
|  Event-Driven | Zero-latency deployments triggered instantly by `fsnotify` file system events. |
|  Thread-Safe | **Per-service Mutex Locking** ensures no two workers ever fight over the same deployment. |
|  High Concurrency | **Worker Pool Pattern** with 100 concurrent workers and a buffered job queue. |
|  Atomic Save Safe | Custom **Debounce Logic** handles OS-level "Atomic Save" events (VS Code/Vim) to prevent infinite loops. |
|  Smart Selectors | Dynamic discovery of Pods using `deployment.Spec.Selector` (no hardcoded label guessing). |
|  Slack Ops | Real-time, color-coded notifications for Success, Failure, and Timeouts. |
|  ECR Native | Seamless integration with AWS ECR for private image pulls using K8s Secrets. |

---

##  Architecture

The system follows a producer-consumer model using Go channels to decouple file events from deployment logic.

```mermaid
graph TD
    CI[CI/CD Pipeline] -->|SCP/Write| Deps[/Dependency Files/]
    
    subgraph "Deployment Daemon (Go)"
        Deps -->|fsnotify| Watcher[File Watcher]
        Watcher -->|Debounce Logic| Queue[Job Queue Channel]
        Queue --> Pool["Worker Pool (100)"]
        
        Pool -->|Lock| Mutex{Per-Service Mutex}
        Mutex -->|Reconcile| K8sClient[K8s Client]
    end
    
    K8sClient -->|Update Image| Cluster((Kubernetes Cluster))
    K8sClient -->|Status Webhook| Slack[Slack Notifications]
    Cluster -->|Pod Status| K8sClient
```

# 🚀 QuickStart

## Prerequisites
* **Go 1.21+**
* **Kubernetes Cluster** (Minikube, Docker Desktop, or EKS)
* `kubectl` configured locally

## Installation

### 1. Clone the repository
```bash
git clone [https://github.com/yourusername/DeploymentK8sEngine.git](https://github.com/yourusername/DeploymentK8sEngine.git)
cd DeploymentK8sEngine
```

2. Configure Environment
Create a .env file in the root directory:
```bash
WEBHOOK_FOR_SLACK=[https://hooks.slack.com/services/YOUR/WEBHOOK](https://hooks.slack.com/services/YOUR/WEBHOOK)
ECR_REPO=your-account.dkr.ecr.region.amazonaws.com
DEPS="file-path-to-monitor"
```
4. Run the Daemon
Start the engine to begin watching for file changes:
```bash
go run .
```


Trigger a Deployment
Simply create or edit a file in your ./deps folder. The filename determines the service and namespace.
```bash
# Format: {service}_{namespace}.dep
echo "nginx:1.24.3" > deps/nginx-app_default.dep
```

🆕 Deploying to New Namespaces (Dynamic Creation)
1.Create the Dependency File: Define your service and the new namespace you want (e.g., qa-env).
```bash
echo "nginx:1.25.0" > deps/nginx-app_qa-env.dep
```
2.Auto-Creation: The engine will detect that qa-env is missing and automatically run kubectl create ns qa-env.
```bash
⚠️Important: The engine creates the Namespace, but it cannot create the Pods yet because the Deployment manifest does not exist in the new namespace.
```
3. Apply Structural YAML: You must provide the base Kubernetes structure (Deployment/Service YAML) for the new namespace.
```bash
# Manually apply the structural yaml to the new namespace
kubectl apply -f nginx.yaml -n qa-env
```
## screenshots

1: CORRECT DEPLOYMENT

<img width="468" height="54" alt="image" src="https://github.com/user-attachments/assets/2f142478-fbb7-4918-a79d-cf04c14fec96" />
<img width="441" height="46" alt="image" src="https://github.com/user-attachments/assets/f184496c-041a-4b05-b043-216a748309e6" />

2: FAILED DEPLOYMENT


<img width="424" height="44" alt="image" src="https://github.com/user-attachments/assets/898215a7-3532-4f0e-8335-97931a38db76" />
<img width="797" height="126" alt="image" src="https://github.com/user-attachments/assets/f371d571-db01-4794-8efc-c12b8a987b94" />

3: NAMESPACE FAILURE
Creates ns with user asking to create file for deployment
<img width="520" height="51" alt="image" src="https://github.com/user-attachments/assets/d689cb8c-1df3-491b-9ae5-37c6155aaeb5" />
<img width="723" height="115" alt="image" src="https://github.com/user-attachments/assets/727ca93c-965f-4df3-a901-7e5513b44cb8" />
<img width="367" height="49" alt="image" src="https://github.com/user-attachments/assets/721288fa-5dba-4ea3-b990-42858cdd1b61" />





<div align="center"> <sub>Built with ❤️ and ☕ by Aditya Bhatt</sub> </div>



























