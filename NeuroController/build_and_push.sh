#!/bin/bash
set -e

IMAGE_NAME="bukahou/zgmf-x10a"
TAG="neurocontroller"

echo "🔧 [Step 1] 确保 Buildx 可用"
docker buildx create --name mybuilder --use || true
docker buildx inspect --bootstrap

echo "🚀 [Step 2] 开始构建并推送: ${IMAGE_NAME}:${TAG}"
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ${IMAGE_NAME}:${TAG} \
  --no-cache \
  --push .
