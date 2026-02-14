/**
 * Cluster Resources API (Job, CronJob, PV, PVC, NetworkPolicy, ResourceQuota, LimitRange, ServiceAccount)
 *
 * 当前阶段: Mock 数据直接返回
 * 未来: 替换为 get<...>("/api/v2/...", params)
 */

import {
  mockJobs,
  mockCronJobs,
  mockPVs,
  mockPVCs,
  mockNetworkPolicies,
  mockResourceQuotas,
  mockLimitRanges,
  mockServiceAccounts,
} from "./mock/cluster-resources";

// Re-export types
export type { JobItem } from "./mock/cluster-resources";
export type { CronJobItem } from "./mock/cluster-resources";
export type { PVItem } from "./mock/cluster-resources";
export type { PVCItem } from "./mock/cluster-resources";
export type { NetworkPolicyItem } from "./mock/cluster-resources";
export type { ResourceQuotaItem } from "./mock/cluster-resources";
export type { LimitRangeItem } from "./mock/cluster-resources";
export type { ServiceAccountItem } from "./mock/cluster-resources";

// ============================================================
// API Functions (currently returning mock data)
// ============================================================

export function getJobList() {
  return mockJobs();
}

export function getCronJobList() {
  return mockCronJobs();
}

export function getPVList() {
  return mockPVs();
}

export function getPVCList() {
  return mockPVCs();
}

export function getNetworkPolicyList() {
  return mockNetworkPolicies();
}

export function getResourceQuotaList() {
  return mockResourceQuotas();
}

export function getLimitRangeList() {
  return mockLimitRanges();
}

export function getServiceAccountList() {
  return mockServiceAccounts();
}
