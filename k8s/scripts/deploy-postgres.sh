#!/bin/bash
set -e

NAMESPACE="task-queue"

echo "🚀 Deploying PostgreSQL via Bitnami Helm"

helm upgrade --install postgresql \
    oci://registry-1.docker.io/bitnamicharts/postgresql \
    --namespace "${NAMESPACE}" \
    --values k8s/manifests/postgres-values.yaml \
    --wait

kubectl wait --for=condition=ready pod \
    -l app.kubernetes.io/name=postgresql \
    -n "${NAMESPACE}" \
    --timeout=300s

echo "✅ PostgreSQL ready!"
