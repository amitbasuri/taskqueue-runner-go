#!/bin/bash
set -e

NAMESPACE="task-queue"

echo "🚀 Deploying application"

kubectl apply -f k8s/manifests/configmap.yaml
kubectl apply -f k8s/manifests/secret.yaml
kubectl apply -f k8s/manifests/server-deployment.yaml
kubectl apply -f k8s/manifests/server-service.yaml
kubectl apply -f k8s/manifests/worker-deployment.yaml

kubectl wait --for=condition=available \
    deployment/task-queue-server \
    -n "${NAMESPACE}" \
    --timeout=300s

kubectl wait --for=condition=available \
    deployment/task-queue-worker \
    -n "${NAMESPACE}" \
    --timeout=300s

echo "✅ Application ready!"
echo ""
echo "Access: http://localhost:8080"
