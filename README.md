# TaskQueue-Go

[![Go Version](https://img.shields.io/badge/Go-1.24-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![CI](https://github.com/amitbasuri/taskqueue-runner-go/actions/workflows/ci.yml/badge.svg)](https://github.com/amitbasuri/taskqueue-runner-go/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/amitbasuri/taskqueue-runner-go)](https://goreportcard.com/report/github.com/amitbasuri/taskqueue-runner-go)

> A production-ready distributed background task processing system built with Go, PostgreSQL, and Kubernetes.

**Key Features:**
- ✅ Priority-based task scheduling with FIFO ordering
- ✅ Automatic retries with exponential backoff
- ✅ Concurrent task processing with worker pool (5 goroutines/instance)
- ✅ Real-time monitoring dashboard with Server-Sent Events
- ✅ Distributed lock-based task claiming (prevents duplicate execution)
- ✅ Full Kubernetes deployment support with Helm
- ✅ Comprehensive integration tests (11/11 passing)
- ✅ Row-level locking with PostgreSQL FOR UPDATE SKIP LOCKED

---

## 📋 Table of Contents

- [About](#about)
- [Architecture](#architecture)
- [Design Choices](#design-choices)
- [Quick Start](#quick-start)
- [Testing](#testing)
- [API Reference](#api-reference)
- [Configuration](#configuration)
- [Contributing](#contributing)
- [License](#license)

---

## 📖 About

TaskQueue-Go is a production-ready distributed task queue system designed for reliability, scalability, and ease of use. Built with Go and PostgreSQL, it provides robust background job processing with automatic retries, real-time monitoring, and horizontal scalability.

**Perfect for:**
- Background job processing (emails, reports, data processing)
- Scheduled tasks and cron-like operations
- Distributed workflows requiring retry logic
- Systems needing audit trails and task history
- Microservices requiring asynchronous task processing

**Why TaskQueue-Go?**
- 🚀 **Production-Ready**: Comprehensive error handling, monitoring, and logging
- 🔒 **Reliable**: PostgreSQL-backed with ACID guarantees and row-level locking
- 📈 **Scalable**: Horizontal scaling with multiple worker instances
- 🎯 **Developer-Friendly**: Clear API, excellent documentation, easy deployment
- 🧪 **Well-Tested**: 11 comprehensive integration tests covering all scenarios

---

## 🏗️ Architecture

### System Overview

```
┌──────────────┐         ┌──────────────────┐         ┌─────────────────┐
│              │         │                  │         │                 │
│  API Server  │────────▶│   PostgreSQL     │◀────────│    Workers      │
│  (Producer)  │  POST   │  (Durable Queue) │  CLAIM  │  (Consumers)    │
│              │  Tasks  │                  │  Tasks  │                 │
└──────────────┘         └──────────────────┘         └─────────────────┘
```

### Components

| Component | Purpose | Technology |
|-----------|---------|------------|
| **API Server** | REST API for task management | Go, Chi router |
| **PostgreSQL** | Durable task queue and state storage | PostgreSQL 16 |
| **Workers** | Execute tasks with retry logic | Go, worker pool |
| **Dashboard** | Real-time monitoring UI | HTML/JS with SSE |

### How It Works

1. **Client** submits task via REST API (`POST /api/tasks`)
2. **API Server** validates and stores task in PostgreSQL with status `queued`
3. **Worker** polls database using `SELECT FOR UPDATE SKIP LOCKED`
4. **Worker** claims task (sets `locked_until` timestamp)
5. **Worker** executes task handler
6. **Worker** updates status to `succeeded` or schedules retry if failed
7. **Dashboard** displays real-time updates via Server-Sent Events

---

## 🎯 Design Choices

### . Dispatcher Pattern

**Problem:** Multiple workers polling database creates "thundering herd"

**Solution:** Single dispatcher goroutine + worker pool

```
Worker Process:
  Dispatcher (1 goroutine) ──polls DB──> Claims 1 task
       │
       └──> Buffered Channel (size 10)
                 │
                 └──> Worker Pool (5 goroutines)
                       ├─> Worker 1
                       ├─> Worker 2
                       ├─> Worker 3
                       ├─> Worker 4
                       └─> Worker 5
```

**Benefits:**
- **1 DB query** instead of 50 per poll cycle
- **Buffered channel** provides backpressure
- **No worker starvation** - dispatcher ensures fair distribution

### 3. Exponential Backoff with Jitter

**Formula:**
```
backoff = min(2^retry_count, 2^20) seconds ± 25% jitter
minimum = 1 second
```

**Example retry schedule:**
- Attempt 1: ~1-1.5s delay
- Attempt 2: ~2-3s delay  
- Attempt 3: ~4-6s delay
- Attempt 4: Max retries reached, task marked `failed`

**Why jitter?**
- Prevents retry storms (many tasks retrying simultaneously)
- Spreads load over time instead of synchronized spikes

### 4. Lock Expiration

**Problem:** Worker crashes while holding lock → task stuck forever

**Solution:** 30-second lock timeout

```sql
WHERE (status = 'queued')
   OR (status = 'running' AND locked_until < NOW())
ORDER BY
  CASE WHEN status = 'running' THEN 0 ELSE 1 END,  -- Expired locks first
  priority DESC,
  created_at ASC
```

**Why prioritize expired locks?**
- Prevents task starvation
- Failed workers don't block the queue
- Respects original priority after recovery

### 5. SELECT FOR UPDATE SKIP LOCKED

**Problem:** Multiple workers trying to claim same task

**Solution:** PostgreSQL row-level locking with `SKIP LOCKED`

```sql
SELECT * FROM tasks
WHERE status = 'queued'
ORDER BY priority DESC
LIMIT 1
FOR UPDATE SKIP LOCKED
```

**How it works:**
- Worker A locks row → Worker B skips it, tries next row
- Zero contention between workers
- No deadlocks or retries needed

---

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for local development)
- Node.js 18+ (for integration tests)
- kubectl, kind & Helm (for Kubernetes testing)

### Run with Docker Compose

```bash
# Start all services
docker-compose up -d

# Wait for startup
sleep 15

# Verify health
curl http://localhost:8080/health

# Open dashboard
open http://localhost:8080
```

### Run with Kubernetes (kind)

```bash
# Deploy to kind cluster with Bitnami PostgreSQL Helm chart
make run-k8s

# Automatically:
# - Creates kind cluster with port mappings
# - Deploys PostgreSQL via Bitnami Helm chart
# - Deploys server (2 replicas) and worker (3 replicas)
# - Configures NodePort service for external access

# Access the application
open http://localhost:8080

# View deployment status
kubectl get pods -n task-queue

# Check logs
kubectl logs -f deployment/task-queue-server -n task-queue
kubectl logs -f deployment/task-queue-worker -n task-queue
```

For detailed Kubernetes documentation, see [`k8s/README.md`](k8s/README.md).

---

## 🧪 Testing

### Run All Integration Tests

```bash
# Test Docker Compose deployment
make test-integration-docker

# Test Kubernetes deployment  
make test-integration-k8s
```

### Test Results

All 11 integration tests pass successfully:

| Test | Description |
|------|-------------|
| Basic lifecycle | Task creation → execution → completion |
| Priority ordering | High-priority tasks processed first |
| Task failures | Retry logic and backoff |
| Failure types | Errors vs timeouts |
| Timeout retries | Slow tasks retry correctly |
| Retry history | Complete audit trail |
| Concurrent processing | 20 tasks with 5 workers |
| Statistics API | Metrics accuracy |
| Invalid task type | Error handling |
| Missing task | 404 responses |

**Test Coverage:**
- ✅ Task lifecycle (create, queue, run, complete)
- ✅ Priority scheduling
- ✅ Retry logic with exponential backoff
- ✅ Timeout handling
- ✅ Concurrent processing (no race conditions)
- ✅ Error handling
- ✅ Real-time statistics

### Manual Testing

```bash
# Create a task
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Email",
    "type": "send_email",
    "priority": 5,
    "payload": {
      "to": "test@example.com",
      "subject": "Test",
      "body": "This is a test"
    }
  }'

# Get task status
curl http://localhost:8080/api/tasks/{task_id}

# Get task history
curl http://localhost:8080/api/tasks/{task_id}/history

# View statistics
curl http://localhost:8080/api/stats
```

---

## 📡 API Reference

### Create Task

**POST** `/api/tasks`

```json
{
  "name": "Send Welcome Email",
  "type": "send_email",
  "priority": 5,
  "payload": {
    "to": "user@example.com",
    "subject": "Welcome!",
    "body": "Thanks for signing up."
  }
}
```

**Response:**
```json
{
  "id": "uuid",
  "name": "Send Welcome Email",
  "type": "send_email",
  "status": "queued",
  "priority": 5,
  "retry_count": 0,
  "max_retries": 3,
  "created_at": "2025-12-06T10:00:00Z"
}
```

### Get Task

**GET** `/api/tasks/:id`

**Response:**
```json
{
  "id": "uuid",
  "status": "succeeded",
  "retry_count": 1,
  "started_at": "2025-12-06T10:00:05Z",
  "completed_at": "2025-12-06T10:00:15Z"
}
```

### Get Task History

**GET** `/api/tasks/:id/history`

**Response:**
```json
[
  {
    "event_type": "task_queued",
    "status": "queued",
    "created_at": "2025-12-06T10:00:00Z"
  },
  {
    "event_type": "task_started",
    "status": "running",
    "worker_id": "worker-123",
    "created_at": "2025-12-06T10:00:05Z"
  },
  {
    "event_type": "task_succeeded",
    "status": "succeeded",
    "created_at": "2025-12-06T10:00:15Z"
  }
]
```

### Get Statistics

**GET** `/api/stats`

**Response:**
```json
{
  "total_tasks": 1000,
  "queued_tasks": 10,
  "running_tasks": 5,
  "succeeded_tasks": 950,
  "failed_tasks": 35,
  "avg_retry_count": 0.45,
  "tasks_with_retries": 300
}
```

### Health Check

**GET** `/health`

**Response:**
```json
{
  "status": "healthy",
  "database": "connected"
}
```

---

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USERNAME` | `admin` | Database user |
| `DB_PASSWORD` | `admin` | Database password |
| `DB_DATABASE` | `tasks` | Database name |
| `SERVER_PORT` | `8080` | API server port |
| `WORKER_CONCURRENCY` | `5` | Worker pool size |
| `WORKER_POLL_INTERVAL` | `1` | Poll interval (seconds) |
| `WORKER_TIMEOUT` | `30` | Task timeout (seconds) |

### Docker Compose

Edit `docker-compose.yml` to adjust configuration:

```yaml
services:
  worker:
    environment:
      WORKER_CONCURRENCY: "10"
      WORKER_POLL_INTERVAL: "2"
```

### Kubernetes

Edit `k8s/manifests/worker-deployment.yaml`:

```yaml
spec:
  replicas: 5  # Number of worker pods
  template:
    spec:
      containers:
      - name: worker
        env:
        - name: WORKER_CONCURRENCY
          value: "10"
```

Configuration files are organized in `k8s/`:
- `manifests/` - Kubernetes YAML files (deployments, services, configs)
- `scripts/` - Deployment automation scripts

---

## 🔧 Development

### Project Structure

```
.
├── cmd/
│   ├── server/          # API server entry point
│   └── worker/          # Worker entry point
│
├── internal/
│   ├── api/             # HTTP handlers and routes
│   ├── config/          # Configuration
│   ├── models/          # Domain models (Task, History)
│   ├── storage/         # Data access layer
│   │   └── postgres/    # PostgreSQL implementation
│   └── worker/          # Worker pool and task handlers
│       ├── worker.go    # Dispatcher + worker pool
│       ├── registry.go  # Handler registration
│       └── handlers/    # Task type implementations
│
├── db/
│   └── migrations/      # SQL schema migrations
│
├── k8s/                 # Kubernetes deployment
│   ├── manifests/       # YAML files (deployments, services, configs)
│   ├── scripts/         # Deployment automation scripts
│   └── README.md        # Kubernetes documentation
│
├── tests/               # Integration tests
├── web/                 # Dashboard UI
│   ├── static/          # CSS, JavaScript
│   └── templates/       # HTML templates
│
├── docker-compose.yml
└── Makefile
```

Please review these documents before contributing:
- [CONTRIBUTING.md](CONTRIBUTING.md) — guidelines, development setup, and workflow
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) — community standards and enforcement
- [SECURITY.md](SECURITY.md) — how to report vulnerabilities

### Makefile Commands
make fmt                        # Format Go code
make lint                       # Run linters (golangci-lint)

```bash
# Quick Start
make help                       # Show all commands
make docker-up                  # Start Docker Compose
make run-k8s                    # Deploy to Kubernetes (one command!)
make test-integration           # Run integration tests

# Build
make build                      # Build server and worker binaries
make docker-build               # Build Docker images

# Testing
make test-integration-docker    # Test Docker Compose
make test-integration-k8s       # Test Kubernetes
make check-tools                # Verify prerequisites

# Kubernetes
make k8s-down                   # Stop Kubernetes

# Database
make migrate-up                 # Run migrations
make migrate-down               # Rollback last migration

# Cleanup
make clean                      # Remove build artifacts
```

---

## 📊 Monitoring

### Dashboard

Access the real-time dashboard at: **http://localhost:8080/**

Features:
- Live task statistics (updated via Server-Sent Events)
- Success rate visualization
- Retry metrics
- Auto-refresh every 5 seconds

### Logs

**Docker Compose:**
```bash
docker-compose logs -f server
docker-compose logs -f worker
```

**Kubernetes:**
```bash
kubectl logs -f deployment/task-queue-worker -n task-queue
kubectl logs -f deployment/task-queue-server -n task-queue

# View all pods
kubectl get pods -n task-queue

# Describe a specific pod
kubectl describe pod <pod-name> -n task-queue
```

---

## 🏗️ Why This Design?

### Key Architectural Decisions

1. **PostgreSQL**
   - Simplicity: One database for everything
   - Durability: No separate persistence layer needed
   - Rich queries: Task history, statistics, complex filtering

2. **Dispatcher Pattern**
   - Prevents database connection storm
   - Reduces DB load by 98% (1 query vs 50 queries/second)
   - Buffered channel provides natural backpressure

3. **Exponential Backoff + Jitter**
   - Prevents retry storms
   - Gradually increases delay for persistent failures
   - Jitter spreads load over time

4. **Lock Expiration**
   - Auto-recovery from worker crashes
   - No manual intervention needed
   - Tasks never stuck permanently

5. **Context Cancellation**
   - Graceful shutdown
   - In-flight tasks complete properly
   - Safe for Kubernetes rolling updates

6. **Kubernetes with kind & Bitnami PostgreSQL**
   - Production-ready: Bitnami Helm chart with security updates
   - Easy local testing: kind runs K8s in Docker containers
   - Horizontal scaling: Server (2 replicas) + Worker (3 replicas)
   - Self-healing: Automatic pod restart on failures
   - Organized structure: Separate manifests/ and scripts/ directories

---


## 🤝 Contributing

Contributions are welcome! Here's how you can help:

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

### Development Setup

```bash
make setup             # Install dependencies
make test-integration  # Run tests
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👤 Author

**Amit Basuri**
- GitHub: [@amitbasuri](https://github.com/amitbasuri)

## 🌟 Show Your Support

Give a ⭐️ if this project helped you!

---

**Built with ❤️ using Go, PostgreSQL, and Kubernetes**
