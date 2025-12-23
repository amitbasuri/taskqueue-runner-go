#!/bin/bash
set -e

CLUSTER_NAME="task-queue"

echo "🚀 Setting up kind cluster"

# Create cluster if it doesn't exist
if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "Creating kind cluster..."
    kind create cluster --name "${CLUSTER_NAME}" --config k8s/manifests/kind-config.yaml
fi

# Create namespace
kubectl apply -f k8s/manifests/namespace.yaml

echo "✅ Cluster ready!"
