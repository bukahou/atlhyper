#!/bin/bash

# ⚠️ 注意：使用本脚本需本地已登录 Docker Hub（docker login）

set -e

IMAGE_NAME="bukahou/neurocontroller"
TAG="v1.1.0"

echo "🔧 [Step 1] Checking Buildx builder"
docker buildx create --name mybuilder --use || true
docker buildx inspect --bootstrap

echo "🚀[Step 2] Building and pushing: ${IMAGE_NAME}:${TAG}"
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ${IMAGE_NAME}:${TAG} \
  --no-cache \
  --push .
