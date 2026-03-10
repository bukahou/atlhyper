/**
 * GitHub 集成 API
 *
 * 功能一：连接管理
 * 功能二：仓库管理
 */

import { get, post, del, put } from "./request";

// ============================================================
// 响应类型
// ============================================================

interface ConnectionResponse {
  message: string;
  data: {
    connected: boolean;
    accountLogin: string;
    avatarUrl: string;
    installationId: number;
  };
}

interface ConnectResponse {
  data: {
    authUrl: string;
  };
}

interface CallbackResponse {
  data: {
    connected: boolean;
    accountLogin: string;
    avatarUrl: string;
    installationId: number;
  };
}

interface ReposResponse {
  data: {
    fullName: string;
    defaultBranch: string;
    private: boolean;
    mappingEnabled: boolean;
  }[];
}

interface MappingToggleResponse {
  message: string;
  data?: {
    repoDirs: string[];
  };
}

interface RepoDirsResponse {
  data: string[];
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 GitHub 连接状态
 * GET /api/github/connection
 */
export function getConnection() {
  return get<ConnectionResponse>("/api/github/connection");
}

/**
 * 发起 OAuth 连接
 * POST /api/github/connect
 */
export function connect() {
  return post<ConnectResponse>("/api/github/connect");
}

/**
 * OAuth 回调
 * POST /api/github/callback
 */
export function callback(code: string) {
  return post<CallbackResponse>("/api/github/callback", { code });
}

/**
 * 断开连接
 * DELETE /api/github/connection
 */
export function disconnect() {
  return del<{ message: string }>("/api/github/connection");
}

/**
 * 获取已授权仓库列表
 * GET /api/github/repos
 */
export function getRepos() {
  return get<ReposResponse>("/api/github/repos");
}

/**
 * 切换仓库映射开关
 * PUT /api/github/repos/:repo/mapping
 */
export function toggleMapping(repo: string, enabled: boolean) {
  return put<MappingToggleResponse>(
    `/api/github/repos/${encodeURIComponent(repo)}/mapping`,
    { enabled }
  );
}

/**
 * 获取仓库顶层目录
 * GET /api/github/repos/:repo/dirs
 */
export function getRepoDirs(repo: string) {
  return get<RepoDirsResponse>(
    `/api/github/repos/${encodeURIComponent(repo)}/dirs`
  );
}

// ============================================================
// Namespace 管理
// ============================================================

interface NamespaceResponse {
  data: string[];
}

/**
 * 获取仓库已配置的 Namespace 列表
 * GET /api/github/repos/:repo/namespaces
 */
export function getRepoNamespaces(repo: string) {
  return get<NamespaceResponse>(
    `/api/github/repos/${encodeURIComponent(repo)}/namespaces`
  );
}

/**
 * 为仓库添加 Namespace
 * POST /api/github/repos/:repo/namespaces
 */
export function addRepoNamespace(repo: string, namespace: string) {
  return post<NamespaceResponse>(
    `/api/github/repos/${encodeURIComponent(repo)}/namespaces`,
    { namespace }
  );
}

/**
 * 移除仓库的 Namespace
 * DELETE /api/github/repos/:repo/namespaces/:ns
 */
export function removeRepoNamespace(repo: string, namespace: string) {
  return del<{ message: string }>(
    `/api/github/repos/${encodeURIComponent(repo)}/namespaces/${encodeURIComponent(namespace)}`
  );
}

// ============================================================
// 映射管理
// ============================================================

interface MappingResponse {
  data: {
    id: number;
    clusterId: string;
    repo: string;
    namespace: string;
    deployment: string;
    container: string;
    imagePrefix: string;
    sourcePath: string;
    confirmed: boolean;
  };
}

interface MappingListResponse {
  data: {
    id: number;
    clusterId: string;
    repo: string;
    namespace: string;
    deployment: string;
    container: string;
    imagePrefix: string;
    sourcePath: string;
    confirmed: boolean;
  }[];
}

/**
 * 获取所有映射
 * GET /api/github/mappings
 */
export function getMappings() {
  return get<MappingListResponse>("/api/github/mappings");
}

/**
 * 创建映射
 * POST /api/github/mappings
 */
export function createMapping(data: {
  clusterId: string;
  repo: string;
  namespace: string;
  deployment: string;
  container?: string;
  imagePrefix?: string;
  sourcePath?: string;
}) {
  return post<MappingResponse>("/api/github/mappings", data);
}

/**
 * 更新映射
 * PUT /api/github/mappings/:id
 */
export function updateMapping(id: number, data: {
  namespace?: string;
  deployment?: string;
  container?: string;
  imagePrefix?: string;
  sourcePath?: string;
}) {
  return put<MappingResponse>(`/api/github/mappings/${id}`, data);
}

/**
 * 确认映射
 * PUT /api/github/mappings/:id/confirm
 */
export function confirmMapping(id: number) {
  return put<{ message: string }>(`/api/github/mappings/${id}/confirm`, {});
}

/**
 * 删除映射
 * DELETE /api/github/mappings/:id
 */
export function deleteMapping(id: number) {
  return del<{ message: string }>(`/api/github/mappings/${id}`);
}
