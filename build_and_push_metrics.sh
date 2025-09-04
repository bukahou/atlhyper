#!/bin/bash
set -e

IMAGE_NAME="bukahou/atlhyper-metrics"
TAG="v1.1.0"

echo "🔧 [Step 1] Checking Buildx builder"
docker buildx create --name mybuilder --use || true
docker buildx inspect --bootstrap

echo "🚀 [Step 2] Building and pushing: ${IMAGE_NAME}:${TAG}"
docker buildx build \
  -f Dockerfile.metrics \
  --platform linux/amd64,linux/arm64 \
  -t ${IMAGE_NAME}:${TAG} \
  --no-cache \
  --push .
