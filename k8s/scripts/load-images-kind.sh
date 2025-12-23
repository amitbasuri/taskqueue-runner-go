#!/bin/bash
set -e

CLUSTER_NAME="task-queue"

echo "🚀 Loading Docker images"

kind load docker-image task-queue-server:latest --name "${CLUSTER_NAME}"
kind load docker-image task-queue-worker:latest --name "${CLUSTER_NAME}"

echo "✅ Images loaded!"
