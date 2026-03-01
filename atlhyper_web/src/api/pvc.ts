/**
 * PersistentVolumeClaim API
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get } from "./request";

// ============================================================
// 类型定义（匹配后端 model 响应）
// ============================================================

export interface PVCItem {
  name: string;
  namespace: string;
  phase: string;
  volumeName: string;
  storageClass: string;
  accessModes: string[];
  requestedCapacity: string;
  actualCapacity: string;
  createdAt: string;
  age: string;
}

export interface PVCDetail {
  name: string;
  namespace: string;
  uid: string;
  phase: string;
  volumeName: string;
  storageClass: string;
  accessModes: string[];
  requestedCapacity: string;
  actualCapacity: string;
  volumeMode?: string;
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

export function getPVCList(params: ClusterResourceParams) {
  return get<ListResponse<PVCItem>>("/api/v2/pvcs", params);
}

export function getPVCDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  return get<DetailResponse<PVCDetail>>(
    `/api/v2/pvcs/${encodeURIComponent(params.Name)}`,
    { cluster_id: params.ClusterID, namespace: params.Namespace }
  );
}
