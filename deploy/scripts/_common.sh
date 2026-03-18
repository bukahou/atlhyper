#!/bin/bash
# ============================================
# AtlHyper 构建脚本公共模块
# ============================================

# 公共构建函数
# 每次构建同时推送 latest + VERSION 两个 tag（只构建一次）
build_and_push() {
  local SERVICE=$1
  local IMAGE="bukahou/atlhyper-${SERVICE}"
  local DOCKERFILE="deploy/docker/Dockerfile.${SERVICE}"

  # 构建 tag 列表：始终包含 latest，VERSION 非空时追加版本 tag
  local TAGS=("-t" "${IMAGE}:latest")
  if [[ -n "${VERSION}" && "${VERSION}" != "latest" ]]; then
    TAGS+=("-t" "${IMAGE}:${VERSION}")
  fi

  echo "============================================"
  echo "🚀 Building: ${IMAGE}"
  echo "📌 Tags: latest${VERSION:+, ${VERSION}}"
  echo "📦 Dockerfile: ${DOCKERFILE}"
  echo "============================================"

  # 创建/使用 Buildx builder
  docker buildx create --name mybuilder --use 2>/dev/null || true
  docker buildx inspect --bootstrap

  # 构建并推送（amd64 + arm64），一次构建多个 tag
  docker buildx build \
    -f "$PROJECT_ROOT/$DOCKERFILE" \
    --platform linux/amd64,linux/arm64 \
    "${TAGS[@]}" \
    --push "$PROJECT_ROOT"

  echo "✅ Done: ${IMAGE} → latest${VERSION:+, ${VERSION}}"
}
