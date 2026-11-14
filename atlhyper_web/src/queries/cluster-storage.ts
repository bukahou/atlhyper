/**
 * 集群资源的取数描述 —— 存储（PV / PVC）
 *
 * 约定同 cluster.ts：列表按默认档轮询、没有集群时不启用；
 * 详情的键带 namespace 与 name、目标不完整时不启用、不轮询。
 */
import { queryOptions } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import {
  getPVCDetail,
  getPVCList,
  getPVDetail,
  getPVList,
} from "@/datasource/cluster";
import { clusterKeys, type NamespacedTarget } from "./keys";
import { clusterDataAvailable } from "./availability";
import { REFRESH } from "./refresh";

// ============================================================
// PersistentVolume
// ============================================================

export function pvListQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("pv", clusterId, getDataSourceMode("pv")),
    queryFn: ({ signal }) => getPVList({ cluster_id: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("pv")),
  });
}

/** PV 是集群级资源，没有 namespace */
export function pvDetailQuery(clusterId: string, pvName: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("pv", clusterId, getDataSourceMode("pv"), { name: pvName }),
    queryFn: ({ signal }) => getPVDetail({ ClusterID: clusterId, Name: pvName }, signal),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("pv")) && pvName !== "",
  });
}

// ============================================================
// PersistentVolumeClaim
// ============================================================

export function pvcListQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("pvc", clusterId, getDataSourceMode("pvc")),
    queryFn: ({ signal }) => getPVCList({ cluster_id: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("pvc")),
  });
}

export function pvcDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("pvc", clusterId, getDataSourceMode("pvc"), target),
    queryFn: ({ signal }) =>
      getPVCDetail({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("pvc")) && target.namespace !== "" && target.name !== "",
  });
}
