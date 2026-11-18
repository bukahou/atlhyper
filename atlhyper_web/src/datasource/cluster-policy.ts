/**
 * Cluster 数据源代理 —— 策略（NetworkPolicy / ResourceQuota / LimitRange / ServiceAccount）
 *
 * 约定同 cluster.ts：读函数返回解包后的数据，最后一个参数是可选的 AbortSignal。
 * 页面与查询工厂从 `@/datasource/cluster` 导入，不直接导入本文件。
 */

import { getDataSourceMode } from "@/config/data-source";
import {
  mockGetNetworkPolicyList,
  mockGetNetworkPolicyDetail,
  mockGetResourceQuotaList,
  mockGetResourceQuotaDetail,
  mockGetLimitRangeList,
  mockGetLimitRangeDetail,
  mockGetServiceAccountList,
  mockGetServiceAccountDetail,
} from "@/mock/cluster";
import * as networkPolicyApi from "@/api/network-policy";
import * as resourceQuotaApi from "@/api/resource-quota";
import * as limitRangeApi from "@/api/limit-range";
import * as serviceAccountApi from "@/api/service-account";
import type { NamespacedListParams, NamespacedResourceParams } from "./cluster-params";

export async function getNetworkPolicyList(params: NamespacedListParams, signal?: AbortSignal) {
  if (getDataSourceMode("netpol") === "mock") {
    return mockGetNetworkPolicyList({ namespace: params.namespace }).data.data;
  }
  return (await networkPolicyApi.getNetworkPolicyList(params, signal)).data.data ?? [];
}

export async function getNetworkPolicyDetail(params: NamespacedResourceParams, signal?: AbortSignal) {
  if (getDataSourceMode("netpol") === "mock") {
    return mockGetNetworkPolicyDetail(params.Name, params.Namespace).data.data;
  }
  return (await networkPolicyApi.getNetworkPolicyDetail(params, signal)).data.data;
}

export async function getResourceQuotaList(params: NamespacedListParams, signal?: AbortSignal) {
  if (getDataSourceMode("quota") === "mock") {
    return mockGetResourceQuotaList({ namespace: params.namespace }).data.data;
  }
  return (await resourceQuotaApi.getResourceQuotaList(params, signal)).data.data ?? [];
}

export async function getResourceQuotaDetail(params: NamespacedResourceParams, signal?: AbortSignal) {
  if (getDataSourceMode("quota") === "mock") {
    return mockGetResourceQuotaDetail(params.Name, params.Namespace).data.data;
  }
  return (await resourceQuotaApi.getResourceQuotaDetail(params, signal)).data.data;
}

export async function getLimitRangeList(params: NamespacedListParams, signal?: AbortSignal) {
  if (getDataSourceMode("limit") === "mock") {
    return mockGetLimitRangeList({ namespace: params.namespace }).data.data;
  }
  return (await limitRangeApi.getLimitRangeList(params, signal)).data.data ?? [];
}

export async function getLimitRangeDetail(params: NamespacedResourceParams, signal?: AbortSignal) {
  if (getDataSourceMode("limit") === "mock") {
    return mockGetLimitRangeDetail(params.Name, params.Namespace).data.data;
  }
  return (await limitRangeApi.getLimitRangeDetail(params, signal)).data.data;
}

export async function getServiceAccountList(params: NamespacedListParams, signal?: AbortSignal) {
  if (getDataSourceMode("sa") === "mock") {
    return mockGetServiceAccountList({ namespace: params.namespace }).data.data;
  }
  return (await serviceAccountApi.getServiceAccountList(params, signal)).data.data ?? [];
}

export async function getServiceAccountDetail(params: NamespacedResourceParams, signal?: AbortSignal) {
  if (getDataSourceMode("sa") === "mock") {
    return mockGetServiceAccountDetail(params.Name, params.Namespace).data.data;
  }
  return (await serviceAccountApi.getServiceAccountDetail(params, signal)).data.data;
}
