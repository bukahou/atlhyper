#!/bin/bash
# =====================================================
# 🧠 AtlHyper AI Service 构建与推送脚本
# -----------------------------------------------------
# 使用前请确保已登录 Docker Hub：docker login
# =====================================================

set -e

IMAGE_NAME="bukahou/atlhyper-aiservice"
# TAG="v1.0.0"
# TAG="latest"
TAG="test"

echo "🔧 [Step 1] Checking Buildx builder"
docker buildx create --name mybuilder --use || true
docker buildx inspect --bootstrap

echo "🚀 [Step 2] Building and pushing image: ${IMAGE_NAME}:${TAG}"
docker buildx build \
  -f Dockerfile.aiservice \
  --platform linux/amd64,linux/arm64 \
  -t ${IMAGE_NAME}:${TAG} \
  --push .

echo "✅ Build and push completed: ${IMAGE_NAME}:${TAG}"
