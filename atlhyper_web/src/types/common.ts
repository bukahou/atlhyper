/**
 * 通用类型定义
 */

// API 响应结构
export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

// 分页请求
export interface PaginationRequest {
  page: number;
  pageSize: number;
}

// 分页响应
export interface PaginationResponse<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}

// 集群请求基础类型
export interface ClusterRequest {
  ClusterID: string;
}

// 命名空间请求
export interface NamespaceRequest extends ClusterRequest {
  Namespace: string;
}

// 状态类型
export type ResourceStatus = "Running" | "Pending" | "Failed" | "Succeeded" | "Unknown";

// Node 状态
export type NodeStatus = "Ready" | "NotReady" | "Unknown";

// 语言类型
export type Language = "zh" | "ja";

// 主题类型
export type Theme = "light" | "dark" | "system";
