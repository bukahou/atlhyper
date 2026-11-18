/**
 * Cluster 数据源代理 —— 工作负载（StatefulSet / DaemonSet / Job / CronJob）
 *
 * 约定同 cluster.ts：读函数返回解包后的数据，最后一个参数是可选的 AbortSignal。
 * 页面与查询工厂从 `@/datasource/cluster` 导入，不直接导入本文件。
 */

import { getDataSourceMode } from "@/config/data-source";
import {
  mockGetStatefulSetList,
  mockGetStatefulSetDetail,
  mockGetDaemonSetList,
  mockGetDaemonSetDetail,
  mockGetJobList,
  mockGetJobDetail,
  mockGetCronJobList,
  mockGetCronJobDetail,
} from "@/mock/cluster";
import * as workloadApi from "@/api/workload";
import * as jobApi from "@/api/job";
import * as cronJobApi from "@/api/cronjob";
import type { NamespacedListParams, NamespacedResourceParams } from "./cluster-params";

// ============================================================
// StatefulSet / DaemonSet
// ============================================================

export async function getStatefulSetList(params: NamespacedListParams, signal?: AbortSignal) {
  if (getDataSourceMode("statefulset") === "mock") {
    return mockGetStatefulSetList({ namespace: params.namespace }).data.data;
  }
  return (await workloadApi.getStatefulSetList(params, signal)).data.data ?? [];
}

export async function getStatefulSetDetail(params: NamespacedResourceParams, signal?: AbortSignal) {
  if (getDataSourceMode("statefulset") === "mock") {
    return mockGetStatefulSetDetail(params.Name, params.Namespace).data.data;
  }
  return (await workloadApi.getStatefulSetDetail(params, signal)).data.data;
}

export async function getDaemonSetList(params: NamespacedListParams, signal?: AbortSignal) {
  if (getDataSourceMode("daemonset") === "mock") {
    return mockGetDaemonSetList({ namespace: params.namespace }).data.data;
  }
  return (await workloadApi.getDaemonSetList(params, signal)).data.data ?? [];
}

export async function getDaemonSetDetail(params: NamespacedResourceParams, signal?: AbortSignal) {
  if (getDataSourceMode("daemonset") === "mock") {
    return mockGetDaemonSetDetail(params.Name, params.Namespace).data.data;
  }
  return (await workloadApi.getDaemonSetDetail(params, signal)).data.data;
}

// ============================================================
// Job / CronJob
// ============================================================

export async function getJobList(params: NamespacedListParams, signal?: AbortSignal) {
  if (getDataSourceMode("job") === "mock") return mockGetJobList({ namespace: params.namespace }).data.data;
  return (await jobApi.getJobList(params, signal)).data.data ?? [];
}

export async function getJobDetail(params: NamespacedResourceParams, signal?: AbortSignal) {
  if (getDataSourceMode("job") === "mock") return mockGetJobDetail(params.Name, params.Namespace).data.data;
  return (await jobApi.getJobDetail(params, signal)).data.data;
}

export async function getCronJobList(params: NamespacedListParams, signal?: AbortSignal) {
  if (getDataSourceMode("cronjob") === "mock") return mockGetCronJobList({ namespace: params.namespace }).data.data;
  return (await cronJobApi.getCronJobList(params, signal)).data.data ?? [];
}

export async function getCronJobDetail(params: NamespacedResourceParams, signal?: AbortSignal) {
  if (getDataSourceMode("cronjob") === "mock") return mockGetCronJobDetail(params.Name, params.Namespace).data.data;
  return (await cronJobApi.getCronJobDetail(params, signal)).data.data;
}
