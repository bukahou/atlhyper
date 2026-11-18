/**
 * Cluster 数据源代理 —— 存储（PV / PVC）
 *
 * 约定同 cluster.ts：读函数返回解包后的数据，最后一个参数是可选的 AbortSignal。
 * 页面与查询工厂从 `@/datasource/cluster` 导入，不直接导入本文件。
 */

import { getDataSourceMode } from "@/config/data-source";
import { mockGetPVList, mockGetPVDetail, mockGetPVCList, mockGetPVCDetail } from "@/mock/cluster";
import * as pvApi from "@/api/pv";
import * as pvcApi from "@/api/pvc";

export async function getPVList(params: { cluster_id: string }, signal?: AbortSignal) {
  if (getDataSourceMode("pv") === "mock") return mockGetPVList().data.data;
  return (await pvApi.getPVList(params, signal)).data.data ?? [];
}

export async function getPVDetail(params: { ClusterID: string; Name: string }, signal?: AbortSignal) {
  if (getDataSourceMode("pv") === "mock") return mockGetPVDetail(params.Name).data.data;
  return (await pvApi.getPVDetail(params, signal)).data.data;
}

export async function getPVCList(params: { cluster_id: string; namespace?: string }, signal?: AbortSignal) {
  if (getDataSourceMode("pvc") === "mock") return mockGetPVCList({ namespace: params.namespace }).data.data;
  return (await pvcApi.getPVCList(params, signal)).data.data ?? [];
}

export async function getPVCDetail(
  params: { ClusterID: string; Namespace: string; Name: string },
  signal?: AbortSignal,
) {
  if (getDataSourceMode("pvc") === "mock") return mockGetPVCDetail(params.Name, params.Namespace).data.data;
  return (await pvcApi.getPVCDetail(params, signal)).data.data;
}
