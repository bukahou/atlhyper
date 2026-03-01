/**
 * Job API
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get } from "./request";

// ============================================================
// 类型定义（匹配后端 model 响应）
// ============================================================

export interface JobItem {
  name: string;
  namespace: string;
  active: number;
  succeeded: number;
  failed: number;
  complete: boolean;
  startTime: string;
  finishTime: string;
  createdAt: string;
  age: string;
}

// 容器详情（Job/CronJob 的 PodTemplate 使用）
export interface ContainerSpec {
  name: string;
  image: string;
  imagePullPolicy?: string;
  ports?: { name?: string; containerPort: number; protocol?: string }[];
  requests?: Record<string, string>;
  limits?: Record<string, string>;
  livenessProbe?: { type: string; path?: string; port?: number; command?: string };
  readinessProbe?: { type: string; path?: string; port?: number; command?: string };
  startupProbe?: { type: string; path?: string; port?: number; command?: string };
  command?: string[];
  args?: string[];
}

export interface JobPodTemplate {
  containers: ContainerSpec[];
  volumes?: { name: string; type: string; source?: string }[];
  serviceAccountName?: string;
  nodeSelector?: Record<string, string>;
}

export interface WorkloadCondition {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
}

export interface JobDetail {
  name: string;
  namespace: string;
  uid: string;
  ownerKind?: string;
  ownerName?: string;
  createdAt: string;
  age: string;
  status: string;
  active: number;
  succeeded: number;
  failed: number;
  completions?: number;
  parallelism?: number;
  backoffLimit?: number;
  startTime: string;
  finishTime: string;
  duration: string;
  template?: JobPodTemplate;
  conditions?: WorkloadCondition[];
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

export function getJobList(params: ClusterResourceParams) {
  return get<ListResponse<JobItem>>("/api/v2/jobs", params);
}

export function getJobDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  return get<DetailResponse<JobDetail>>(
    `/api/v2/jobs/${encodeURIComponent(params.Name)}`,
    { cluster_id: params.ClusterID, namespace: params.Namespace }
  );
}
