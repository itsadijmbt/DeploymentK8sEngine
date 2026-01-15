# DeploymentK8sEngine

GitOps-Style Kubernetes Deployment Daemon
A lightweight, event-driven Kubernetes deployment automation system built from first principles in Go. Automatically deploys applications to Kubernetes clusters based on filesystem changes, implementing GitOps principles with built-in concurrency control and Slack notifications.
Show Image
Show Image
Show Image
🎯 Overview
This daemon watches dependency files in a directory and automatically updates Kubernetes Deployments when versions change. It eliminates manual kubectl commands while providing safety through concurrency control, per-service locking, and real-time Slack notifications.
Built to learn: Created from first principles to deeply understand Kubernetes internals, Go concurrency patterns, and GitOps workflows - not by copying templates.
✨ Features

🔄 Event-Driven Deployments - Automatically deploys on file changes using fsnotify
🔒 Concurrency Control - Per-service mutex locking prevents deployment conflicts
🚀 Worker Pool Pattern - 100 concurrent workers with buffered job queue
🎯 Multi-Namespace Support - Deploy same service to multiple namespaces independently
📊 Slack Integration - Color-coded notifications with deployment status
🔁 Reconciliation Loop - Handles rapid successive changes gracefully
🐳 ECR Integration - Seamless AWS ECR image URL construction
📝 File-Based State - Simple, reliable state management without databases

🏗️ Architecture
┌─────────────┐
│   Jenkins   │ Builds image, pushes to ECR
│   CI/CD     │ Writes dependency file via SCP
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│  Dependency Files (/deps/)          │
│  ┌─────────────────────────────┐   │
│  │ nginx-app_dev.dep           │   │
│  │ nginx-app_production.dep    │   │
│  │ backend_staging.dep         │   │
│  └─────────────────────────────┘   │
└──────┬──────────────────────────────┘
       │ fsnotify detects changes
       ▼
┌─────────────────────────────────────┐
│  Deployment Daemon                  │
│  ┌─────────────────────────────┐   │
│  │ File Watcher (fsnotify)     │   │
│  │         ↓                    │   │
│  │ Job Queue (buffered channel)│   │
│  │         ↓                    │   │
│  │ Worker Pool (100 goroutines)│   │
│  │         ↓                    │   │
│  │ Per-Service Locking         │   │
│  │         ↓                    │   │
│  │ Kubernetes Client           │   │
│  └─────────────────────────────┘   │
└──────┬──────────────────────────────┘
       │
       ├──────────────┬─────────────────┐
       ▼              ▼                 ▼
┌──────────┐   ┌──────────┐   ┌──────────────┐
│ K8s      │   │ Slack    │   │ .last Files  │
│ API      │   │ Webhooks │   │ (state)      │
└──────────┘   └──────────┘   └──────────────┘
🚀 Quick Start
Prerequisites

Go 1.25+
Kubernetes cluster (local or remote)
kubectl configured with access to cluster
Slack webhook URL (optional, for notifications)