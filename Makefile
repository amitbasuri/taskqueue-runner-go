NAME=task-service
WORKER_NAME=task-worker
BUILD_DIR ?= bin
SERVER_SRC=./cmd/server
WORKER_SRC=./cmd/worker

NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

# Ensure Go bin is in PATH for kind
export PATH := $(shell go env GOPATH)/bin:$(PATH)

.PHONY: setup deps test build build-server build-worker clean all help
all: deps test build

# Show help
help:
	@echo "$(OK_COLOR)╔════════════════════════════════════════════════════════════╗$(NO_COLOR)"
	@echo "$(OK_COLOR)║  Background Task Management System - Makefile Help         ║$(NO_COLOR)"
	@echo "$(OK_COLOR)╚════════════════════════════════════════════════════════════╝$(NO_COLOR)"
	@echo ""
	@echo "$(OK_COLOR)🚀 Quick Start:$(NO_COLOR)"
	@echo "  make run-k8s              - Deploy and run on Kubernetes (one command!)"
	@echo "  make docker-up            - Run with Docker Compose"
	@echo "  make test-integration     - Run integration tests"
	@echo ""
	@echo "$(OK_COLOR)📦 Build & Development:$(NO_COLOR)"
	@echo "  make build                - Build server and worker binaries"
	@echo "  make build-server         - Build API server only"
	@echo "  make build-worker         - Build worker only"
	@echo ""
	@echo "$(OK_COLOR)🐳 Docker Compose:$(NO_COLOR)"
	@echo "  make docker-build         - Build Docker images"
	@echo "  make docker-up            - Start all services"
	@echo "  make docker-down          - Stop all services"
	@echo ""
	@echo "$(OK_COLOR)☸️  Kubernetes (kind):$(NO_COLOR)"
	@echo "  make run-k8s              - Deploy to kind cluster with Bitnami PostgreSQL"
	@echo "  make k8s-down             - Delete kind cluster"
	@echo "  make k8s-stop             - Stop kind cluster (preserves state)"
	@echo "  make k8s-start            - Start a stopped kind cluster"
	@echo ""
	@echo "$(OK_COLOR)🧪 Testing:$(NO_COLOR)"
	@echo "  make test-integration            - Run integration tests (Docker Compose)"
	@echo "  make test-integration-docker     - Run tests against Docker Compose"
	@echo "  make test-integration-k8s        - Run tests against Kubernetes"
	@echo "  make check-tools                 - Check test prerequisites"
	@echo ""
	@echo "$(OK_COLOR)🗄️  Database:$(NO_COLOR)"
	@echo "  make migrate-up           - Run database migrations"
	@echo "  make migrate-down         - Rollback last migration"
	@echo ""
	@echo "$(OK_COLOR)🔧 Setup:$(NO_COLOR)"
	@echo "  make setup                - Initial development setup"
	@echo "  make deps                 - Download Go dependencies"
	@echo ""
	@echo "For more details: https://github.com/amitbasuri/taskqueue-go"
	@echo ""

# Setup development environment
setup:
	@echo "$(OK_COLOR)==> Setting up development environment$(NO_COLOR)"
	@echo "$(OK_COLOR)==> Installing Kubernetes tools$(NO_COLOR)"
	@command -v kind >/dev/null 2>&1 || brew install kind
	@command -v kubectl >/dev/null 2>&1 || brew install kubectl
	@command -v helm >/dev/null 2>&1 || brew install helm
	@echo "$(OK_COLOR)==> Installing golang-migrate$(NO_COLOR)"
	@command -v migrate >/dev/null 2>&1 || brew install golang-migrate
	@echo "$(OK_COLOR)==> Installing Go dependencies$(NO_COLOR)"
	@go mod download
	@echo "$(OK_COLOR)✅ Setup complete!$(NO_COLOR)"
	@echo ""
	@echo "Installed tools:"
	@echo "  ✓ kind: $$(kind version 2>/dev/null || echo 'not found')"
	@echo "  ✓ kubectl: $$(kubectl version --client --short 2>/dev/null || echo 'not found')"
	@echo "  ✓ helm: $$(helm version --short 2>/dev/null || echo 'not found')"
	@echo "  ✓ migrate: $$(migrate -version 2>/dev/null || echo 'not found')"

deps:
	go mod download

build: build-server build-worker
	@echo "$(OK_COLOR)==> Built both binaries$(NO_COLOR)"

build-server:
	@echo "$(OK_COLOR)==> Building the API server (producer)...$(NO_COLOR)"
	@CGO_ENABLED=0 go build -v -ldflags="-s -w" -o "$(BUILD_DIR)/$(NAME)" "$(SERVER_SRC)"

build-worker:
	@echo "$(OK_COLOR)==> Building the worker (consumer)...$(NO_COLOR)"
	@CGO_ENABLED=0 go build -v -ldflags="-s -w" -o "$(BUILD_DIR)/$(WORKER_NAME)" "$(WORKER_SRC)"

clean:
	@echo "$(WARN_COLOR)==> Cleaning build artifacts$(NO_COLOR)"
	@rm -rf $(BUILD_DIR)

# Docker commands
docker-build:
	@echo "$(OK_COLOR)==> Building Docker images$(NO_COLOR)"
	@DOCKER_BUILDKIT=0 docker-compose build
	@echo "$(OK_COLOR)==> Tagging images for Kubernetes$(NO_COLOR)"
	@docker tag amit-basuri-server:latest task-queue-server:latest
	@docker tag amit-basuri-worker:latest task-queue-worker:latest
	@echo "$(OK_COLOR)==> Docker images ready$(NO_COLOR)"

docker-up:
	@echo "$(OK_COLOR)==> Starting services in Docker$(NO_COLOR)"
	@docker-compose up -d

docker-down:
	@echo "$(WARN_COLOR)==> Stopping Docker services$(NO_COLOR)"
	@docker-compose down

# Integration tests
test-integration: test-integration-docker

# Run integration tests against Docker Compose
test-integration-docker: check-tools
	@echo "$(OK_COLOR)╔════════════════════════════════════════════════════════════╗$(NO_COLOR)"
	@echo "$(OK_COLOR)║  Running Integration Tests - Docker Compose                ║$(NO_COLOR)"
	@echo "$(OK_COLOR)╚════════════════════════════════════════════════════════════╝$(NO_COLOR)"
	@echo ""
	@echo "$(OK_COLOR)==> Stopping kind cluster if running (to free port 8080)...$(NO_COLOR)"
	@$(MAKE) k8s-stop 2>/dev/null || true
	@echo "$(OK_COLOR)==> Cleaning up Docker Compose...$(NO_COLOR)"
	@docker-compose down --remove-orphans -v 2>/dev/null || true
	@echo "$(OK_COLOR)==> Rebuilding Docker images with latest code...$(NO_COLOR)"
	@DOCKER_BUILDKIT=0 docker-compose build
	@echo "$(OK_COLOR)==> Starting Docker Compose services...$(NO_COLOR)"
	@docker-compose up -d
	@echo "$(OK_COLOR)==> Waiting 15 seconds for services to be ready...$(NO_COLOR)"
	@sleep 15
	@echo ""
	@echo "$(OK_COLOR)==> Running Node.js integration tests against Docker Compose...$(NO_COLOR)"
	@cd tests && API_URL=http://localhost:8080 node integration.test.js || { \
		echo ""; \
		echo "$(ERROR_COLOR)==> Tests failed! Check the output above.$(NO_COLOR)"; \
		exit 1; \
	}
	@echo ""
	@echo "$(OK_COLOR)╔════════════════════════════════════════════════════════════╗$(NO_COLOR)"
	@echo "$(OK_COLOR)║  ✅ Docker Compose Integration Tests Passed!               ║$(NO_COLOR)"
	@echo "$(OK_COLOR)╚════════════════════════════════════════════════════════════╝$(NO_COLOR)"
	@echo ""
	@echo "$(OK_COLOR)💡 Tip: Restart kind cluster with: docker start task-queue-control-plane$(NO_COLOR)"

# Run integration tests against Kubernetes
test-integration-k8s: check-tools
	@echo "$(OK_COLOR)╔════════════════════════════════════════════════════════════╗$(NO_COLOR)"
	@echo "$(OK_COLOR)║  Running Integration Tests - Kubernetes (kind)              ║$(NO_COLOR)"
	@echo "$(OK_COLOR)╚════════════════════════════════════════════════════════════╝$(NO_COLOR)"
	@echo ""
	@echo "$(OK_COLOR)==> Stopping Docker Compose if running (to free port 8080)...$(NO_COLOR)"
	@docker-compose down --remove-orphans 2>/dev/null || true
	@echo "$(OK_COLOR)==> Checking if kind cluster is running...$(NO_COLOR)"
	@if kind get clusters | grep -q "^task-queue$$" && docker ps | grep -q task-queue-control-plane; then \
		echo "$(OK_COLOR)==> kind cluster is running$(NO_COLOR)"; \
		if kubectl get deployment task-queue-server -n task-queue >/dev/null 2>&1; then \
			echo "$(OK_COLOR)==> Application already deployed$(NO_COLOR)"; \
		else \
			echo "$(WARN_COLOR)==> Application not deployed, doing full deployment...$(NO_COLOR)"; \
			$(MAKE) k8s-down; \
			$(MAKE) run-k8s; \
		fi; \
	else \
		if kind get clusters | grep -q "^task-queue$$"; then \
			echo "$(WARN_COLOR)==> kind cluster exists but is stopped, cleaning up...$(NO_COLOR)"; \
			$(MAKE) k8s-down; \
		fi; \
		echo "$(WARN_COLOR)==> Deploying fresh kind cluster...$(NO_COLOR)"; \
		$(MAKE) run-k8s; \
	fi
	@echo ""
	@echo "$(OK_COLOR)==> Checking pod status...$(NO_COLOR)"
	@kubectl get pods -n task-queue
	@echo ""
	@echo "$(OK_COLOR)==> Waiting for all pods to be ready...$(NO_COLOR)"
	@kubectl wait --for=condition=Ready pods --all -n task-queue --timeout=120s || { \
		echo "$(ERROR_COLOR)==> Pods not ready!$(NO_COLOR)"; \
		exit 1; \
	}
	@echo ""
	@echo "$(OK_COLOR)==> Running Node.js integration tests against Kubernetes...$(NO_COLOR)"
	@cd tests && API_URL=http://localhost:8080 node integration.test.js || { \
		echo ""; \
		echo "$(ERROR_COLOR)==> Tests failed!$(NO_COLOR)"; \
		echo "$(ERROR_COLOR)==> View logs with: kubectl logs deployment/task-queue-server -n task-queue$(NO_COLOR)"; \
		exit 1; \
	}
	@echo ""
	@echo "$(OK_COLOR)╔════════════════════════════════════════════════════════════╗$(NO_COLOR)"
	@echo "$(OK_COLOR)║  ✅ Kubernetes Integration Tests Passed!                   ║$(NO_COLOR)"
	@echo "$(OK_COLOR)╚════════════════════════════════════════════════════════════╝$(NO_COLOR)"
	@echo ""
	@echo "$(OK_COLOR)💡 Tip: View pod logs with: kubectl logs -f deployment/task-queue-worker -n task-queue$(NO_COLOR)"

# Check required tools are installed
check-tools:
	@echo "$(OK_COLOR)==> Checking required tools...$(NO_COLOR)"
	@command -v go >/dev/null 2>&1 || { echo "$(ERROR_COLOR)==> Error: Go is not installed. Please install Go from https://golang.org/$(NO_COLOR)"; exit 1; }
	@echo "$(OK_COLOR)  ✓ Go installed: $$(go version)$(NO_COLOR)"
	@command -v docker >/dev/null 2>&1 || { echo "$(ERROR_COLOR)==> Error: Docker is not installed. Please install Docker from https://www.docker.com/$(NO_COLOR)"; exit 1; }
	@echo "$(OK_COLOR)  ✓ Docker installed: $$(docker --version)$(NO_COLOR)"
	@command -v docker-compose >/dev/null 2>&1 || { echo "$(ERROR_COLOR)==> Error: Docker Compose is not installed. Please install Docker Compose$(NO_COLOR)"; exit 1; }
	@echo "$(OK_COLOR)  ✓ Docker Compose installed: $$(docker-compose --version)$(NO_COLOR)"
	@command -v node >/dev/null 2>&1 || { echo "$(ERROR_COLOR)==> Error: Node.js is not installed. Please install Node.js 18+ from https://nodejs.org/$(NO_COLOR)"; exit 1; }
	@echo "$(OK_COLOR)  ✓ Node.js installed: $$(node --version)$(NO_COLOR)"
	@node_version=$$(node --version | cut -d'v' -f2 | cut -d'.' -f1); \
	if [ $$node_version -lt 18 ]; then \
		echo "$(ERROR_COLOR)==> Error: Node.js version 18 or higher is required (found: $$(node --version))$(NO_COLOR)"; \
		exit 1; \
	fi
	@echo "$(OK_COLOR)==> All required tools are installed!$(NO_COLOR)"

# Migration commands (requires golang-migrate CLI)
migrate-up:
	@echo "$(OK_COLOR)==> Running migrations up$(NO_COLOR)"
	@migrate -path db/migrations -database "postgres://admin:admin@localhost:8848/tasks?sslmode=disable" up

migrate-down:
	@echo "$(WARN_COLOR)==> Rolling back last migration$(NO_COLOR)"
	@migrate -path db/migrations -database "postgres://admin:admin@localhost:8848/tasks?sslmode=disable" down 1

# Kubernetes deployment with kind

# Simple one-command to deploy to Kubernetes using kind
run-k8s: k8s-check-tools
	@echo "$(OK_COLOR)==> Setting up Kubernetes cluster with kind$(NO_COLOR)"
	@./k8s/scripts/setup-cluster-kind.sh
	@echo "$(OK_COLOR)==> Building and loading Docker images$(NO_COLOR)"
	@$(MAKE) docker-build
	@./k8s/scripts/load-images-kind.sh
	@echo "$(OK_COLOR)==> Deploying PostgreSQL with Bitnami Helm chart$(NO_COLOR)"
	@./k8s/scripts/deploy-postgres.sh
	@echo "$(OK_COLOR)==> Deploying application to Kubernetes$(NO_COLOR)"
	@./k8s/scripts/deploy-app.sh
	@echo "$(OK_COLOR)✅ Kubernetes deployment complete!$(NO_COLOR)"
	@kubectl get pods -n task-queue

# Simple one-command to stop Kubernetes
k8s-down:
	@echo "$(WARN_COLOR)==> Deleting kind cluster...$(NO_COLOR)"
	@kind delete cluster --name task-queue || true
	@echo "$(OK_COLOR)✅ Kubernetes cluster deleted$(NO_COLOR)"

# Stop kind cluster temporarily (keeps state for quick restart)
k8s-stop:
	@echo "$(WARN_COLOR)==> Stopping kind cluster container (preserving state)...$(NO_COLOR)"
	@docker stop $$(docker ps -q --filter "name=task-queue-control-plane") 2>/dev/null || true
	@echo "$(OK_COLOR)✅ kind cluster stopped$(NO_COLOR)"

# Start a stopped kind cluster
k8s-start:
	@echo "$(OK_COLOR)==> Starting kind cluster container...$(NO_COLOR)"
	@docker start $$(docker ps -aq --filter "name=task-queue-control-plane") 2>/dev/null || { \
		echo "$(WARN_COLOR)==> Cluster not found, use 'make run-k8s' to create it$(NO_COLOR)"; \
		exit 1; \
	}
	@echo "$(OK_COLOR)✅ kind cluster started$(NO_COLOR)"

# Check if Kubernetes tools are installed
k8s-check-tools:
	@echo "$(OK_COLOR)==> Checking Kubernetes tools...$(NO_COLOR)"
	@command -v kubectl >/dev/null 2>&1 || { \
		echo "$(ERROR_COLOR)==> Error: kubectl is not installed$(NO_COLOR)"; \
		echo ""; \
		echo "Install kubectl:"; \
		echo "  macOS:    brew install kubectl"; \
		echo "  Linux:    curl -LO https://dl.k8s.io/release/\$$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"; \
		echo "  Windows:  choco install kubernetes-cli"; \
		echo ""; \
		exit 1; \
	}
	@echo "$(OK_COLOR)  ✓ kubectl installed: $$(kubectl version --client --short 2>/dev/null || kubectl version --client)$(NO_COLOR)"
	@command -v kind >/dev/null 2>&1 || { \
		echo "$(WARN_COLOR)==> kind is not installed, installing now...$(NO_COLOR)"; \
		echo "$(OK_COLOR)==> Installing kind via Homebrew...$(NO_COLOR)"; \
		brew install kind || { \
			echo "$(ERROR_COLOR)==> Failed to install kind via brew$(NO_COLOR)"; \
			echo ""; \
			echo "Manual installation:"; \
			echo "  macOS/Linux:  go install sigs.k8s.io/kind@v0.30.0"; \
			echo "  Or download from: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"; \
			echo ""; \
			exit 1; \
		}; \
		echo "$(OK_COLOR)  ✓ kind installed successfully$(NO_COLOR)"; \
	}
	@echo "$(OK_COLOR)  ✓ kind installed: $$(kind version)$(NO_COLOR)"
	@command -v helm >/dev/null 2>&1 || { \
		echo "$(WARN_COLOR)==> helm is not installed, installing now...$(NO_COLOR)"; \
		echo "$(OK_COLOR)==> Installing helm via Homebrew...$(NO_COLOR)"; \
		brew install helm || { \
			echo "$(ERROR_COLOR)==> Failed to install helm$(NO_COLOR)"; \
			echo ""; \
			echo "Manual installation:"; \
			echo "  macOS:    brew install helm"; \
			echo "  Linux:    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash"; \
			echo "  Windows:  choco install kubernetes-helm"; \
			echo ""; \
			exit 1; \
		}; \
		echo "$(OK_COLOR)  ✓ helm installed successfully$(NO_COLOR)"; \
	}
	@echo "$(OK_COLOR)  ✓ helm installed: $$(helm version --short)$(NO_COLOR)"
	@command -v docker >/dev/null 2>&1 || { \
		echo "$(ERROR_COLOR)==> Error: Docker is not installed$(NO_COLOR)"; \
		echo ""; \
		echo "Install Docker:"; \
		echo "  macOS:    brew install --cask docker"; \
		echo "  Linux:    https://docs.docker.com/engine/install/"; \
		echo "  Windows:  https://docs.docker.com/desktop/install/windows-install/"; \
		echo ""; \
		exit 1; \
	}
	@echo "$(OK_COLOR)  ✓ Docker installed: $$(docker --version)$(NO_COLOR)"
	@docker info >/dev/null 2>&1 || { \
		echo "$(ERROR_COLOR)==> Error: Docker daemon is not running$(NO_COLOR)"; \
		echo "Please start Docker Desktop or the Docker daemon"; \
		exit 1; \
	}
	@echo "$(OK_COLOR)  ✓ Docker daemon is running$(NO_COLOR)"
	@echo "$(OK_COLOR)==> All Kubernetes tools ready!$(NO_COLOR)"

