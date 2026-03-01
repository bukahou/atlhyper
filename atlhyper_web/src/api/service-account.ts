/**
 * ServiceAccount API
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get } from "./request";

// ============================================================
// 类型定义（匹配后端 model 响应）
// ============================================================

export interface ServiceAccountItem {
  name: string;
  namespace: string;
  secretsCount: number;
  imagePullSecretsCount: number;
  automountServiceAccountToken?: boolean;
  createdAt: string;
  age: string;
}

export interface ServiceAccountDetail {
  name: string;
  namespace: string;
  secretsCount: number;
  imagePullSecretsCount: number;
  automountServiceAccountToken?: boolean;
  secretNames?: string[];
  imagePullSecretNames?: string[];
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

export function getServiceAccountList(params: ClusterResourceParams) {
  return get<ListResponse<ServiceAccountItem>>("/api/v2/service-accounts", params);
}

export function getServiceAccountDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  return get<DetailResponse<ServiceAccountDetail>>(
    `/api/v2/service-accounts/${encodeURIComponent(params.Name)}`,
    { cluster_id: params.ClusterID, namespace: params.Namespace }
  );
}
