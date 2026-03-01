/**
 * CronJob API
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get } from "./request";
import type { JobPodTemplate } from "./job";

// ============================================================
// 类型定义（匹配后端 model 响应）
// ============================================================

export interface CronJobItem {
  name: string;
  namespace: string;
  schedule: string;
  suspend: boolean;
  activeJobs: number;
  lastScheduleTime: string;
  lastSuccessfulTime: string;
  createdAt: string;
  age: string;
}

export interface CronJobDetail {
  name: string;
  namespace: string;
  uid: string;
  ownerKind?: string;
  ownerName?: string;
  createdAt: string;
  age: string;
  schedule: string;
  suspend: boolean;
  concurrencyPolicy?: string;
  activeJobs: number;
  successfulJobsHistoryLimit?: number;
  failedJobsHistoryLimit?: number;
  lastScheduleTime: string;
  lastSuccessfulTime: string;
  lastScheduleAgo: string;
  lastSuccessAgo: string;
  template?: JobPodTemplate;
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

export function getCronJobList(params: ClusterResourceParams) {
  return get<ListResponse<CronJobItem>>("/api/v2/cronjobs", params);
}

export function getCronJobDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  return get<DetailResponse<CronJobDetail>>(
    `/api/v2/cronjobs/${encodeURIComponent(params.Name)}`,
    { cluster_id: params.ClusterID, namespace: params.Namespace }
  );
}
