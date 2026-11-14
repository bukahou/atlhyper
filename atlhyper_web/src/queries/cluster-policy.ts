/**
 * 集群资源的取数描述 —— 策略（NetworkPolicy / ResourceQuota / LimitRange / ServiceAccount）
 *
 * 约定同 cluster.ts：列表按默认档轮询、没有集群时不启用；
 * 详情的键带 namespace 与 name、目标不完整时不启用、不轮询。
 */
import { queryOptions } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import {
  getLimitRangeDetail,
  getLimitRangeList,
  getNetworkPolicyDetail,
  getNetworkPolicyList,
  getResourceQuotaDetail,
  getResourceQuotaList,
  getServiceAccountDetail,
  getServiceAccountList,
} from "@/datasource/cluster";
import { clusterKeys, type NamespacedTarget } from "./keys";
import { clusterDataAvailable } from "./availability";
import { REFRESH } from "./refresh";

// ============================================================
// NetworkPolicy
// ============================================================

export function networkPolicyListQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("netpol", clusterId, getDataSourceMode("netpol")),
    queryFn: ({ signal }) => getNetworkPolicyList({ cluster_id: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("netpol")),
  });
}

export function networkPolicyDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("netpol", clusterId, getDataSourceMode("netpol"), target),
    queryFn: ({ signal }) =>
      getNetworkPolicyDetail({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("netpol")) && target.namespace !== "" && target.name !== "",
  });
}

// ============================================================
// ResourceQuota
// ============================================================

export function resourceQuotaListQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("quota", clusterId, getDataSourceMode("quota")),
    queryFn: ({ signal }) => getResourceQuotaList({ cluster_id: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("quota")),
  });
}

export function resourceQuotaDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("quota", clusterId, getDataSourceMode("quota"), target),
    queryFn: ({ signal }) =>
      getResourceQuotaDetail({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("quota")) && target.namespace !== "" && target.name !== "",
  });
}

// ============================================================
// LimitRange
// ============================================================

export function limitRangeListQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("limit", clusterId, getDataSourceMode("limit")),
    queryFn: ({ signal }) => getLimitRangeList({ cluster_id: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("limit")),
  });
}

export function limitRangeDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("limit", clusterId, getDataSourceMode("limit"), target),
    queryFn: ({ signal }) =>
      getLimitRangeDetail({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("limit")) && target.namespace !== "" && target.name !== "",
  });
}

// ============================================================
// ServiceAccount
// ============================================================

export function serviceAccountListQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("sa", clusterId, getDataSourceMode("sa")),
    queryFn: ({ signal }) => getServiceAccountList({ cluster_id: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("sa")),
  });
}

export function serviceAccountDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("sa", clusterId, getDataSourceMode("sa"), target),
    queryFn: ({ signal }) =>
      getServiceAccountDetail({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("sa")) && target.namespace !== "" && target.name !== "",
  });
}
