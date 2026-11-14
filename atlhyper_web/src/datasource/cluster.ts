/**
 * Cluster 数据源代理 —— 核心资源（Pod / Node / Deployment / Service / Namespace / Ingress / Event）
 *
 * 根据中心配置自动切换 mock / api。约定：
 * - 读函数返回解包后的数据（查询工厂与页面不碰 axios 响应），mock 分支返回相同形态
 * - 读函数最后一个参数是可选的 AbortSignal，原样交给 api 层，用于取消请求
 *
 * 工作负载 / 存储 / 策略类资源按侧边栏分组放在 cluster-*.ts，从这里统一再导出，
 * 所以 `@/datasource/cluster` 仍然是集群数据的唯一入口。
 */

import { getDataSourceMode } from "@/config/data-source";
import {
  mockGetPodList,
  mockGetPodOverview,
  mockGetPodDetail,
  mockGetNodeList,
  mockGetNodeOverview,
  mockGetNodeDetail,
  mockGetDeploymentList,
  mockGetDeploymentOverview,
  mockGetDeploymentDetail,
  mockGetServiceList,
  mockGetServiceOverview,
  mockGetServiceDetail,
  mockGetNamespaceList,
  mockGetNamespaceOverview,
  mockGetNamespaceDetail,
  mockGetConfigMapList,
  mockGetSecretList,
  mockGetIngressList,
  mockGetIngressOverview,
  mockGetIngressDetail,
  mockGetEventList,
  mockGetEventOverview,
} from "@/mock/cluster";
import * as podApi from "@/api/pod";
import * as nodeApi from "@/api/node";
import * as deploymentApi from "@/api/deployment";
import * as serviceApi from "@/api/service";
import * as namespaceApi from "@/api/namespace";
import * as ingressApi from "@/api/ingress";
import * as eventApi from "@/api/event";
import type { NamespacedListParams, NamespacedResourceParams } from "./cluster-params";

export * from "./cluster-workload";
export * from "./cluster-storage";
export * from "./cluster-policy";

// ============================================================
// Pod
// ============================================================

export async function getPodList(params: NamespacedListParams, signal?: AbortSignal) {
  if (getDataSourceMode("pod") === "mock") return mockGetPodList({ namespace: params.namespace }).data.data;
  return (await podApi.getPodList(params, signal)).data.data ?? [];
}

export async function getPodOverview(params: { ClusterID: string }, signal?: AbortSignal) {
  if (getDataSourceMode("pod") === "mock") return mockGetPodOverview().data.data;
  return (await podApi.getPodOverview(params, signal)).data.data;
}

/** Pod 详情。演示数据里找不到该 Pod 时返回 null */
export async function getPodDetail(
  params: { ClusterID: string; Namespace: string; PodName: string },
  signal?: AbortSignal,
) {
  if (getDataSourceMode("pod") === "mock") return mockGetPodDetail(params.PodName, params.Namespace).data.data;
  return (await podApi.getPodDetail(params, signal)).data.data;
}

/**
 * Pod 日志文本（没有日志时为空串）。
 *
 * 没有演示数据：日志由 Agent 现场读取容器输出，经指令通道同步返回。
 */
export async function getPodLogs(
  params: { ClusterID: string; Namespace: string; Pod: string; Container?: string; TailLines?: number },
  signal?: AbortSignal,
) {
  return (await podApi.getPodLogs(params, signal)).data.data?.logs ?? "";
}

// ============================================================
// Node
// ============================================================

export async function getNodeList(params: { cluster_id: string }, signal?: AbortSignal) {
  if (getDataSourceMode("node") === "mock") return mockGetNodeList().data.data;
  return (await nodeApi.getNodeList(params, signal)).data.data ?? [];
}

export async function getNodeOverview(params: { ClusterID: string }, signal?: AbortSignal) {
  if (getDataSourceMode("node") === "mock") return mockGetNodeOverview().data.data;
  return (await nodeApi.getNodeOverview(params, signal)).data.data;
}

export async function getNodeDetail(params: { ClusterID: string; NodeName: string }, signal?: AbortSignal) {
  if (getDataSourceMode("node") === "mock") return mockGetNodeDetail(params.NodeName).data.data;
  return (await nodeApi.getNodeDetail(params, signal)).data.data;
}

// ============================================================
// Deployment
// ============================================================

export async function getDeploymentList(params: NamespacedListParams, signal?: AbortSignal) {
  if (getDataSourceMode("deployment") === "mock") {
    return mockGetDeploymentList({ namespace: params.namespace }).data.data;
  }
  return (await deploymentApi.getDeploymentList(params, signal)).data.data ?? [];
}

export async function getDeploymentOverview(params: { ClusterID: string }, signal?: AbortSignal) {
  if (getDataSourceMode("deployment") === "mock") return mockGetDeploymentOverview().data.data;
  return (await deploymentApi.getDeploymentOverview(params, signal)).data.data;
}

export async function getDeploymentDetail(params: NamespacedResourceParams, signal?: AbortSignal) {
  if (getDataSourceMode("deployment") === "mock") {
    return mockGetDeploymentDetail(params.Name, params.Namespace).data.data;
  }
  return (await deploymentApi.getDeploymentDetail(params, signal)).data.data;
}

// ============================================================
// Service
// ============================================================

export async function getServiceList(params: NamespacedListParams, signal?: AbortSignal) {
  if (getDataSourceMode("service") === "mock") return mockGetServiceList({ namespace: params.namespace }).data.data;
  return (await serviceApi.getServiceList(params, signal)).data.data ?? [];
}

export async function getServiceOverview(params: { ClusterID: string }, signal?: AbortSignal) {
  if (getDataSourceMode("service") === "mock") return mockGetServiceOverview().data.data;
  return (await serviceApi.getServiceOverview(params, signal)).data.data;
}

export async function getServiceDetail(params: NamespacedResourceParams, signal?: AbortSignal) {
  if (getDataSourceMode("service") === "mock") return mockGetServiceDetail(params.Name, params.Namespace).data.data;
  return (await serviceApi.getServiceDetail(params, signal)).data.data;
}

// ============================================================
// Namespace（含其下的 ConfigMap / Secret）
// ============================================================

export async function getNamespaceList(params: { cluster_id: string }, signal?: AbortSignal) {
  if (getDataSourceMode("namespace") === "mock") return mockGetNamespaceList().data.data;
  return (await namespaceApi.getNamespaceList(params, signal)).data.data ?? [];
}

export async function getNamespaceOverview(params: { ClusterID: string }, signal?: AbortSignal) {
  if (getDataSourceMode("namespace") === "mock") return mockGetNamespaceOverview().data.data;
  return (await namespaceApi.getNamespaceOverview(params, signal)).data.data;
}

export async function getNamespaceDetail(params: { ClusterID: string; Namespace: string }, signal?: AbortSignal) {
  if (getDataSourceMode("namespace") === "mock") return mockGetNamespaceDetail(params.Namespace).data.data;
  return (await namespaceApi.getNamespaceDetail(params, signal)).data.data;
}

export async function getConfigMapList(params: { cluster_id: string; namespace: string }, signal?: AbortSignal) {
  if (getDataSourceMode("namespace") === "mock") return mockGetConfigMapList(params.namespace).data.data;
  return (await namespaceApi.getConfigMapList(params, signal)).data.data ?? [];
}

export async function getSecretList(params: { cluster_id: string; namespace: string }, signal?: AbortSignal) {
  if (getDataSourceMode("namespace") === "mock") return mockGetSecretList(params.namespace).data.data;
  return (await namespaceApi.getSecretList(params, signal)).data.data ?? [];
}

/** ConfigMap 的键值内容。没有演示数据：内容经指令通道由 Agent 现场读取 */
export function getConfigMapData(params: NamespacedResourceParams, signal?: AbortSignal) {
  return namespaceApi.getConfigMapData(params, signal);
}

/** Secret 的键值内容。没有演示数据：内容经指令通道由 Agent 现场读取 */
export function getSecretData(params: NamespacedResourceParams, signal?: AbortSignal) {
  return namespaceApi.getSecretData(params, signal);
}

// ============================================================
// Ingress
// ============================================================

export async function getIngressList(params: NamespacedListParams, signal?: AbortSignal) {
  if (getDataSourceMode("ingress") === "mock") return mockGetIngressList({ namespace: params.namespace }).data.data;
  return (await ingressApi.getIngressList(params, signal)).data.data ?? [];
}

export async function getIngressOverview(params: { ClusterID: string }, signal?: AbortSignal) {
  if (getDataSourceMode("ingress") === "mock") return mockGetIngressOverview().data.data;
  return (await ingressApi.getIngressOverview(params, signal)).data.data;
}

export async function getIngressDetail(params: NamespacedResourceParams, signal?: AbortSignal) {
  if (getDataSourceMode("ingress") === "mock") return mockGetIngressDetail(params.Name, params.Namespace).data.data;
  return (await ingressApi.getIngressDetail(params, signal)).data.data;
}

// ============================================================
// Event
// ============================================================

export async function getEventList(
  params: { cluster_id: string; namespace?: string; type?: string },
  signal?: AbortSignal,
) {
  if (getDataSourceMode("event") === "mock") {
    return mockGetEventList({ namespace: params.namespace, type: params.type }).data.events;
  }
  return (await eventApi.getEventList(params, signal)).data.events ?? [];
}

/** 事件概览。返回解包后的数据；signal 用于取消请求 */
export async function getEventOverview(
  params: { ClusterID: string; Namespace?: string; Type?: string },
  signal?: AbortSignal,
) {
  if (getDataSourceMode("event") === "mock") return mockGetEventOverview().data.data;
  return (await eventApi.getEventOverview(params, signal)).data.data;
}

/**
 * 某个资源自己的事件时间线（资源详情里的「事件」标签页用）。
 *
 * 没有演示数据：演示用的事件不按资源归档。
 */
export async function getEventsByResource(
  params: { cluster_id: string; kind: string; namespace?: string; name: string },
  signal?: AbortSignal,
) {
  return (await eventApi.getEventsByResource(params, signal)).data.events ?? [];
}

// ============================================================
// 写操作 —— 一律走真实接口（指令下发，没有演示数据）
// ============================================================

export { restartPod } from "@/api/pod";
export { cordonNode, uncordonNode } from "@/api/node";
export { scaleDeployment, restartDeployment, updateDeploymentImage } from "@/api/deployment";
