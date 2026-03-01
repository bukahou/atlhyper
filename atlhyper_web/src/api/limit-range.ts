/**
 * LimitRange API
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get } from "./request";

// ============================================================
// 类型定义（匹配后端 model 响应）
// ============================================================

export interface LimitRangeItemEntry {
  type: string;
  max?: Record<string, string>;
  min?: Record<string, string>;
  default?: Record<string, string>;
  defaultRequest?: Record<string, string>;
  maxLimitRequestRatio?: Record<string, string>;
}

export interface LimitRangeItem {
  name: string;
  namespace: string;
  items: LimitRangeItemEntry[];
  createdAt: string;
  age: string;
}

export interface LimitRangeDetail {
  name: string;
  namespace: string;
  items: LimitRangeItemEntry[];
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

export function getLimitRangeList(params: ClusterResourceParams) {
  return get<ListResponse<LimitRangeItem>>("/api/v2/limit-ranges", params);
}

export function getLimitRangeDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  return get<DetailResponse<LimitRangeDetail>>(
    `/api/v2/limit-ranges/${encodeURIComponent(params.Name)}`,
    { cluster_id: params.ClusterID, namespace: params.Namespace }
  );
}
