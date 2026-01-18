/**
 * 通用类型定义
 *
 * 适配 Master V2 API
 * - 参数命名使用 snake_case (cluster_id, namespace)
 * - 响应直接返回数据，不再包装 code/message
 */

// ============================================================
// API 响应结构（保留用于兼容）
// ============================================================

// 旧版 API 响应结构（已废弃，仅保留兼容）
export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

// ============================================================
// 请求参数类型（Master V2）
// ============================================================

// 分页请求
export interface PaginationRequest {
  limit?: number;
  offset?: number;
}

// 分页响应
export interface PaginationResponse<T> {
  list: T[];
  total: number;
  limit: number;
  offset: number;
}

// 集群查询参数（GET 请求用）
export interface ClusterQueryParams {
  cluster_id: string;
}

// 命名空间查询参数
export interface NamespaceQueryParams extends ClusterQueryParams {
  namespace?: string;
}

// 资源查询参数
export interface ResourceQueryParams extends NamespaceQueryParams {
  limit?: number;
  offset?: number;
}

// ============================================================
// 旧版请求类型（保留兼容，后续移除）
// ============================================================

/** @deprecated 使用 ClusterQueryParams 替代 */
export interface ClusterRequest {
  ClusterID: string;
}

/** @deprecated 使用 NamespaceQueryParams 替代 */
export interface NamespaceRequest extends ClusterRequest {
  Namespace: string;
}

// ============================================================
// 通用类型
// ============================================================

// 状态类型
export type ResourceStatus = "Running" | "Pending" | "Failed" | "Succeeded" | "Unknown";

// Node 状态
export type NodeStatus = "Ready" | "NotReady" | "Unknown";

// 语言类型
export type Language = "zh" | "ja";

// 主题类型
export type Theme = "light" | "dark" | "system";
