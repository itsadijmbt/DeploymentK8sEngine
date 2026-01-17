<div align="center">

# 🚀 DeploymentK8sEngine
### Event-Driven GitOps Controller built from First Principles in Go

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)](https://kubernetes.io/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=for-the-badge)](http://makeapullrequest.com)

_A lightweight, concurrency-safe deployment daemon that watches your filesystem and updates Kubernetes in real-time._

[**Explore the Docs**](#-architecture) · [**View Demo**](#-screenshots) · [**Report Bug**](issues)

</div>

---

## ⚡ Overview

**DeploymentK8sEngine** eliminates the need for manual `kubectl` commands by implementing a **file-based GitOps workflow**. It watches a dedicated dependency directory for changes and automatically reconciles your Kubernetes cluster state to match.

Unlike generic tools, this engine was **built from scratch** to solve specific distributed system challenges: race conditions, atomic file saves, and concurrent deployment locking.

> **Why build this?**
> To master Kubernetes internals, Go concurrency patterns (Channels, Mutexes, Goroutines), and system-level file watching—not by copying templates, but by solving the hard problems myself.

---

## ✨ Key Features

| Feature | Description |
| :--- | :--- |
| **🔄 Event-Driven** | Zero-latency deployments triggered instantly by `fsnotify` file system events. |
| **🔒 Thread-Safe** | **Per-service Mutex Locking** ensures no two workers ever fight over the same deployment. |
| **🚀 High Concurrency** | **Worker Pool Pattern** with 100 concurrent workers and a buffered job queue. |
| **🛡️ Atomic Save Safe** | Custom **Debounce Logic** handles OS-level "Atomic Save" events (VS Code/Vim) to prevent infinite loops. |
| **🎯 Smart Selectors** | Dynamic discovery of Pods using `deployment.Spec.Selector` (no hardcoded label guessing). |
| **📊 Slack Ops** | Real-time, color-coded notifications for Success, Failure, and Timeouts. |
| **🐳 ECR Native** | Seamless integration with AWS ECR for private image pulls using K8s Secrets. |

---

## 🏗️ Architecture

The system follows a producer-consumer model using Go channels to decouple file events from deployment logic.

```mermaid
graph TD
    CI[CI/CD Pipeline] -->|SCP/Write| Deps[/Dependency Files/]
    
    subgraph "Deployment Daemon (Go)"
        Deps -->|fsnotify| Watcher[File Watcher]
        Watcher -->|Debounce Logic| Queue[Job Queue Channel]
        Queue --> Pool[Worker Pool (100)]
        
        Pool -->|Lock| Mutex{Per-Service Mutex}
        Mutex -->|Reconcile| K8sClient[K8s Client]
    end
    
    K8sClient -->|Update Image| Cluster((Kubernetes Cluster))
    K8sClient -->|Status Webhook| Slack[Slack Notifications]
    Cluster -->|Pod Status| K8sClient