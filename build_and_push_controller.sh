#!/bin/bash

# ⚠️ 注意：このスクリプトを使用する前に、ローカルで Docker Hub にログインしてください（docker login）
# ⚠️ Note: Please make sure you are logged in to Docker Hub locally (docker login) before running this script.
#
# 🔧 このスクリプトは個人用です。使用する場合は、IMAGE_NAME を自分のリポジトリに変更してください。
# 🔧 This script is for personal use. If you want to use it, please change IMAGE_NAME to your own repository.


set -e

IMAGE_NAME="bukahou/neurocontroller"
TAG="v3.0.1"

echo "🔧 [Step 1] Checking Buildx builder"
docker buildx create --name mybuilder --use || true
docker buildx inspect --bootstrap

echo "🚀[Step 2] Building and pushing: ${IMAGE_NAME}:${TAG}"
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f Dockerfile.controller \
  -t ${IMAGE_NAME}:${TAG} \
  --no-cache \
  --push .
