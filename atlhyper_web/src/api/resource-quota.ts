/**
 * ResourceQuota API
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get } from "./request";

// ============================================================
// 类型定义（匹配后端 model 响应）
// ============================================================

export interface ResourceQuotaItem {
  name: string;
  namespace: string;
  scopes?: string[];
  hard: Record<string, string>;
  used: Record<string, string>;
  createdAt: string;
  age: string;
}

export interface ResourceQuotaDetail {
  name: string;
  namespace: string;
  scopes?: string[];
  hard: Record<string, string>;
  used: Record<string, string>;
  createdAt: string;
  age: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

// ============================================================
// 响应类型（内部使用）
// ============================================================

interface ListResponse<T> {
  message: string;
  data: T[];
  total: number;
}

interface DetailResponse<T> {
  message: string;
  data: T;
}

// ============================================================
// API 查询参数（内部使用）
// ============================================================

interface ClusterResourceParams {
  cluster_id: string;
  namespace?: string;
}

// ============================================================
// API Functions
// ============================================================

export function getResourceQuotaList(params: ClusterResourceParams) {
  return get<ListResponse<ResourceQuotaItem>>("/api/v2/resource-quotas", params);
}

export function getResourceQuotaDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  return get<DetailResponse<ResourceQuotaDetail>>(
    `/api/v2/resource-quotas/${encodeURIComponent(params.Name)}`,
    { cluster_id: params.ClusterID, namespace: params.Namespace }
  );
}
