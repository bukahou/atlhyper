/**
 * 集群资源的取数描述 —— 核心资源（Pod / Node / Deployment / Service / Namespace / Ingress / Event）
 *
 * 每个工厂返回一份 queryOptions：键、取数函数、轮询间隔、启用条件。
 * - 列表：按默认档轮询；没有选定集群（clusterId 为空）时不启用
 * - 详情：键里带 namespace 与 name，目标不完整时不启用；不轮询（只在弹窗打开期间存在）
 *
 * 工作负载 / 存储 / 策略类资源按侧边栏分组放在 cluster-*.ts；
 * 经指令通道读取的内容（日志、ConfigMap / Secret 的值）在 cluster-content.ts。
 */
import { queryOptions } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import {
  getConfigMapList,
  getDeploymentDetail,
  getDeploymentOverview,
  getEventOverview,
  getEventsByResource,
  getIngressDetail,
  getIngressOverview,
  getNamespaceDetail,
  getNamespaceOverview,
  getNodeDetail,
  getNodeOverview,
  getPodDetail,
  getPodOverview,
  getSecretList,
  getServiceDetail,
  getServiceOverview,
} from "@/datasource/cluster";
import { clusterKeys, type EventSubject, type NamespacedTarget } from "./keys";
import { REFRESH } from "./refresh";

// ============================================================
// Event
// ============================================================

/** 集群事件概览（实时） */
export function eventOverviewQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.events(clusterId, getDataSourceMode("event")),
    queryFn: ({ signal }) => getEventOverview({ ClusterID: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    enabled: clusterId !== "",
  });
}

/** 某个资源自己的事件时间线（详情弹窗的「事件」标签页）。只在标签页打开时取一次 */
export function resourceEventsQuery(clusterId: string, subject: EventSubject) {
  return queryOptions({
    queryKey: clusterKeys.resourceEvents(clusterId, getDataSourceMode("event"), subject),
    queryFn: ({ signal }) =>
      getEventsByResource(
        { cluster_id: clusterId, kind: subject.kind, namespace: subject.namespace, name: subject.name },
        signal,
      ),
    enabled: clusterId !== "" && subject.name !== "",
  });
}

// ============================================================
// Pod
// ============================================================

export function podOverviewQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("pod", clusterId, getDataSourceMode("pod")),
    queryFn: ({ signal }) => getPodOverview({ ClusterID: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    enabled: clusterId !== "",
  });
}

export function podDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("pod", clusterId, getDataSourceMode("pod"), target),
    queryFn: ({ signal }) =>
      getPodDetail({ ClusterID: clusterId, Namespace: target.namespace, PodName: target.name }, signal),
    enabled: clusterId !== "" && target.namespace !== "" && target.name !== "",
  });
}

// ============================================================
// Node
// ============================================================

export function nodeOverviewQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("node", clusterId, getDataSourceMode("node")),
    queryFn: ({ signal }) => getNodeOverview({ ClusterID: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    enabled: clusterId !== "",
  });
}

export function nodeDetailQuery(clusterId: string, nodeName: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("node", clusterId, getDataSourceMode("node"), { name: nodeName }),
    queryFn: ({ signal }) => getNodeDetail({ ClusterID: clusterId, NodeName: nodeName }, signal),
    enabled: clusterId !== "" && nodeName !== "",
  });
}

// ============================================================
// Deployment
// ============================================================

export function deploymentOverviewQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("deployment", clusterId, getDataSourceMode("deployment")),
    queryFn: ({ signal }) => getDeploymentOverview({ ClusterID: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    enabled: clusterId !== "",
  });
}

export function deploymentDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("deployment", clusterId, getDataSourceMode("deployment"), target),
    queryFn: ({ signal }) =>
      getDeploymentDetail({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterId !== "" && target.namespace !== "" && target.name !== "",
  });
}

// ============================================================
// Service
// ============================================================

export function serviceOverviewQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("service", clusterId, getDataSourceMode("service")),
    queryFn: ({ signal }) => getServiceOverview({ ClusterID: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    enabled: clusterId !== "",
  });
}

export function serviceDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("service", clusterId, getDataSourceMode("service"), target),
    queryFn: ({ signal }) =>
      getServiceDetail({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterId !== "" && target.namespace !== "" && target.name !== "",
  });
}

// ============================================================
// Ingress
// ============================================================

export function ingressOverviewQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("ingress", clusterId, getDataSourceMode("ingress")),
    queryFn: ({ signal }) => getIngressOverview({ ClusterID: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    enabled: clusterId !== "",
  });
}

export function ingressDetailQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("ingress", clusterId, getDataSourceMode("ingress"), target),
    queryFn: ({ signal }) =>
      getIngressDetail({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterId !== "" && target.namespace !== "" && target.name !== "",
  });
}

// ============================================================
// Namespace（含其下的 ConfigMap / Secret）
// ============================================================

export function namespaceOverviewQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceList("namespace", clusterId, getDataSourceMode("namespace")),
    queryFn: ({ signal }) => getNamespaceOverview({ ClusterID: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    enabled: clusterId !== "",
  });
}

export function namespaceDetailQuery(clusterId: string, namespaceName: string) {
  return queryOptions({
    queryKey: clusterKeys.resourceDetail("namespace", clusterId, getDataSourceMode("namespace"), {
      name: namespaceName,
    }),
    queryFn: ({ signal }) => getNamespaceDetail({ ClusterID: clusterId, Namespace: namespaceName }, signal),
    enabled: clusterId !== "" && namespaceName !== "",
  });
}

/** 某个 namespace 下的 ConfigMap 列表（只有名字与大小，不含内容） */
export function configMapListQuery(clusterId: string, namespace: string) {
  return queryOptions({
    queryKey: clusterKeys.configMapList(clusterId, getDataSourceMode("namespace"), namespace),
    queryFn: ({ signal }) => getConfigMapList({ cluster_id: clusterId, namespace }, signal),
    enabled: clusterId !== "" && namespace !== "",
  });
}

/**
 * 某个 namespace 下的 Secret 列表（只有名字与键名，不含内容）。
 *
 * canReadSecrets 由调用方用 useCan("secret.read") 得到：没有权限就不发请求，
 * 而不是发出去再被后端拒绝。
 */
export function secretListQuery(clusterId: string, namespace: string, canReadSecrets: boolean) {
  return queryOptions({
    queryKey: clusterKeys.secretList(clusterId, getDataSourceMode("namespace"), namespace),
    queryFn: ({ signal }) => getSecretList({ cluster_id: clusterId, namespace }, signal),
    enabled: clusterId !== "" && namespace !== "" && canReadSecrets,
  });
}
