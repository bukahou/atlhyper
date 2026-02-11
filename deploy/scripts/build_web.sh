#!/bin/bash
set -e

# ============================================
# 版本标签（在此修改）
# - test:   测试环境
# - latest: 最新稳定版
# - v1.x.x: 正式发布版本
# ============================================
# TAG="latest"
TAG="v0.2.2"
# TAG="test"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR/../.."
source "$SCRIPT_DIR/_common.sh"
build_and_push web
