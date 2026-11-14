/**
 * 集群资源的取数描述 —— 工作负载（StatefulSet / DaemonSet / Job / CronJob）
 *
 * 约定同 cluster.ts：列表按默认档轮询、没有集群时不启用；
 * 详情的键带 namespace 与 name、目标不完整时不启用、不轮询。
 */
import { queryOptions } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import {
  getCronJobDetail,
  getCronJobList,
  getDaemonSetDetail,
  getDaemonSetList,
  getJobDetail,
  getJobList,
  getStatefulSetDetail,
  getStatefulSetList,
} from "@/datasource/cluster";
import { clusterKeys, type NamespacedTarget } from "./keys";
import { clusterDataAvailable } from "./availability";
import { REFRESH } from "./refresh";

// ============================================================
// StatefulSet
// ============================================================

export function statefulSetListQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("statefulset", clusterId, getDataSourceMode("statefulset")),
    queryFn: ({ signal }) => getStatefulSetList({ cluster_id: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("statefulset")),
  });
}

export function statefulSetDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("statefulset", clusterId, getDataSourceMode("statefulset"), target),
    queryFn: ({ signal }) =>
      getStatefulSetDetail({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("statefulset")) && target.namespace !== "" && target.name !== "",
  });
}

// ============================================================
// DaemonSet
// ============================================================

export function daemonSetListQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("daemonset", clusterId, getDataSourceMode("daemonset")),
    queryFn: ({ signal }) => getDaemonSetList({ cluster_id: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("daemonset")),
  });
}

export function daemonSetDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("daemonset", clusterId, getDataSourceMode("daemonset"), target),
    queryFn: ({ signal }) =>
      getDaemonSetDetail({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("daemonset")) && target.namespace !== "" && target.name !== "",
  });
}

// ============================================================
// Job
// ============================================================

export function jobListQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("job", clusterId, getDataSourceMode("job")),
    queryFn: ({ signal }) => getJobList({ cluster_id: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("job")),
  });
}

export function jobDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("job", clusterId, getDataSourceMode("job"), target),
    queryFn: ({ signal }) =>
      getJobDetail({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("job")) && target.namespace !== "" && target.name !== "",
  });
}

// ============================================================
// CronJob
// ============================================================

export function cronJobListQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("cronjob", clusterId, getDataSourceMode("cronjob")),
    queryFn: ({ signal }) => getCronJobList({ cluster_id: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("cronjob")),
  });
}

export function cronJobDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("cronjob", clusterId, getDataSourceMode("cronjob"), target),
    queryFn: ({ signal }) =>
      getCronJobDetail({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("cronjob")) && target.namespace !== "" && target.name !== "",
  });
}
