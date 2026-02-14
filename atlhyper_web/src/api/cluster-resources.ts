/**
 * Cluster Resources API (Job, CronJob, PV, PVC, NetworkPolicy, ResourceQuota, LimitRange, ServiceAccount)
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

export interface NetworkPolicyItem {
  name: string;
  namespace: string;
  podSelector: string;
  policyTypes: string[];
  ingressRuleCount: number;
  egressRuleCount: number;
  createdAt: string;
  age: string;
}

export interface ResourceQuotaItem {
  name: string;
  namespace: string;
  scopes?: string[];
  hard: Record<string, string>;
  used: Record<string, string>;
  createdAt: string;
  age: string;
}

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

export interface ServiceAccountItem {
  name: string;
  namespace: string;
  secretsCount: number;
  imagePullSecretsCount: number;
  automountServiceAccountToken?: boolean;
  createdAt: string;
  age: string;
}

// ============================================================
// 响应类型
// ============================================================

interface ListResponse<T> {
  message: string;
  data: T[];
  total: number;
}

// ============================================================
// API 查询参数
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

export function getCronJobList(params: ClusterResourceParams) {
  return get<ListResponse<CronJobItem>>("/api/v2/cronjobs", params);
}

export function getPVList(params: { cluster_id: string }) {
  return get<ListResponse<PVItem>>("/api/v2/pvs", params);
}

export function getPVCList(params: ClusterResourceParams) {
  return get<ListResponse<PVCItem>>("/api/v2/pvcs", params);
}

export function getNetworkPolicyList(params: ClusterResourceParams) {
  return get<ListResponse<NetworkPolicyItem>>("/api/v2/network-policies", params);
}

export function getResourceQuotaList(params: ClusterResourceParams) {
  return get<ListResponse<ResourceQuotaItem>>("/api/v2/resource-quotas", params);
}

export function getLimitRangeList(params: ClusterResourceParams) {
  return get<ListResponse<LimitRangeItem>>("/api/v2/limit-ranges", params);
}

export function getServiceAccountList(params: ClusterResourceParams) {
  return get<ListResponse<ServiceAccountItem>>("/api/v2/service-accounts", params);
}
