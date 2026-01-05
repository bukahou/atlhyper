#!/bin/bash
# ============================================
# AtlHyper 构建脚本公共模块
# ============================================

# 公共构建函数
build_and_push() {
  local SERVICE=$1
  local IMAGE="bukahou/atlhyper-${SERVICE}"
  local DOCKERFILE="deploy/docker/Dockerfile.${SERVICE}"

  echo "============================================"
  echo "🚀 Building: ${IMAGE}:${TAG}"
  echo "📦 Dockerfile: ${DOCKERFILE}"
  echo "============================================"

  # 创建/使用 Buildx builder
  docker buildx create --name mybuilder --use 2>/dev/null || true
  docker buildx inspect --bootstrap

  # 构建并推送（amd64 + arm64）
  docker buildx build \
    -f "$PROJECT_ROOT/$DOCKERFILE" \
    --platform linux/amd64,linux/arm64 \
    -t "${IMAGE}:${TAG}" \
    --push "$PROJECT_ROOT"

  echo "✅ Done: ${IMAGE}:${TAG}"
}
