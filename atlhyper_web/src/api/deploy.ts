/**
 * 部署管理 API
 *
 * 功能四：部署配置
 * 功能五：同步状态
 * 功能六：部署历史
 */

import { get, put, post } from "./request";

// ============================================================
// 响应类型
// ============================================================

interface DeployConfigResponse {
  data: {
    repoUrl: string;
    paths: string[];
    intervalSec: number;
    autoDeploy: boolean;
    clusterId: string;
  } | null;
}

interface SaveConfigResponse {
  message: string;
}

interface KustomizePathsResponse {
  data: string[];
}

interface TestConnectionResponse {
  data: {
    success: boolean;
  };
}

interface DeployHistoryResponse {
  data: {
    id: number;
    clusterId: string;
    path: string;
    namespace: string;
    commitSha: string;
    commitMessage: string;
    deployedAt: string;
    trigger: string;
    status: string;
    durationMs: number;
    resourceTotal: number;
    resourceChanged: number;
    errorMessage?: string;
  }[];
  total: number;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取部署配置
 * GET /api/deploy/config
 */
export function getConfig(clusterId: string) {
  return get<DeployConfigResponse>("/api/deploy/config", { clusterId });
}

/**
 * 保存部署配置
 * PUT /api/deploy/config
 */
export function saveConfig(data: {
  clusterId: string;
  repoUrl: string;
  paths: string[];
  intervalSec: number;
  autoDeploy: boolean;
}) {
  return put<SaveConfigResponse>("/api/deploy/config", data);
}

/**
 * 扫描 kustomize 路径
 * GET /api/deploy/kustomize-paths
 */
export function getKustomizePaths(repo: string) {
  return get<KustomizePathsResponse>("/api/deploy/kustomize-paths", { repo });
}

/**
 * 测试连接
 * POST /api/deploy/test-connection
 */
export function testConnection() {
  return post<TestConnectionResponse>("/api/deploy/test-connection");
}

/**
 * 获取部署历史
 * GET /api/deploy/history
 */
export function getHistory(params: { clusterId: string; path?: string }) {
  return get<DeployHistoryResponse>("/api/deploy/history", params);
}

// ============================================================
// 同步状态 & 回滚
// ============================================================

interface PathStatusResponse {
  data: {
    path: string;
    namespace: string;
    inSync: boolean;
    resourceCount: number;
    lastSyncAt: string;
  }[];
}

interface SyncResponse {
  message: string;
}

interface RollbackResponse {
  message: string;
}

/**
 * 获取同步状态
 * GET /api/deploy/status
 */
export function getStatus() {
  return get<PathStatusResponse>("/api/deploy/status");
}

/**
 * 手动触发同步
 * POST /api/deploy/sync
 */
export function syncNow(path: string) {
  return post<SyncResponse>("/api/deploy/sync", { path });
}

/**
 * 回滚部署
 * POST /api/deploy/rollback
 */
export function rollback(path: string, targetCommitSha: string) {
  return post<RollbackResponse>("/api/deploy/rollback", { path, targetCommitSha });
}
