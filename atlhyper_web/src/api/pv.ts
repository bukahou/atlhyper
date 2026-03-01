/**
 * PersistentVolume API
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get } from "./request";

// ============================================================
// 类型定义（匹配后端 model 响应）
// ============================================================

export interface PVItem {
  name: string;
  capacity: string;
  phase: string;
  storageClass: string;
  accessModes: string[];
  reclaimPolicy: string;
  createdAt: string;
  age: string;
}

export interface PVDetail {
  name: string;
  uid: string;
  capacity: string;
  phase: string;
  storageClass: string;
  accessModes: string[];
  reclaimPolicy: string;
  volumeSourceType?: string;
  claimRefName?: string;
  claimRefNamespace?: string;
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
// API Functions
// ============================================================

export function getPVList(params: { cluster_id: string }) {
  return get<ListResponse<PVItem>>("/api/v2/pvs", params);
}

export function getPVDetail(params: { ClusterID: string; Name: string }) {
  return get<DetailResponse<PVDetail>>(
    `/api/v2/pvs/${encodeURIComponent(params.Name)}`,
    { cluster_id: params.ClusterID }
  );
}
