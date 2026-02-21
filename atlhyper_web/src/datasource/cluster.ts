/**
 * Cluster 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 */

import { getDataSourceMode } from "@/config/data-source";

// ============================================================
// Mock imports
// ============================================================

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
  mockGetStatefulSetList,
  mockGetStatefulSetDetail,
  mockGetDaemonSetList,
  mockGetDaemonSetDetail,
  mockGetJobList,
  mockGetJobDetail,
  mockGetCronJobList,
  mockGetCronJobDetail,
  mockGetPVList,
  mockGetPVDetail,
  mockGetPVCList,
  mockGetPVCDetail,
  mockGetNetworkPolicyList,
  mockGetNetworkPolicyDetail,
  mockGetResourceQuotaList,
  mockGetResourceQuotaDetail,
  mockGetLimitRangeList,
  mockGetLimitRangeDetail,
  mockGetServiceAccountList,
  mockGetServiceAccountDetail,
} from "@/mock/cluster";

// ============================================================
// API imports
// ============================================================

import {
  getPodList as apiGetPodList,
  getPodOverview as apiGetPodOverview,
  getPodDetail as apiGetPodDetail,
} from "@/api/pod";

import {
  getNodeList as apiGetNodeList,
  getNodeOverview as apiGetNodeOverview,
  getNodeDetail as apiGetNodeDetail,
} from "@/api/node";

import {
  getDeploymentList as apiGetDeploymentList,
  getDeploymentOverview as apiGetDeploymentOverview,
  getDeploymentDetail as apiGetDeploymentDetail,
} from "@/api/deployment";

import {
  getServiceList as apiGetServiceList,
  getServiceOverview as apiGetServiceOverview,
  getServiceDetail as apiGetServiceDetail,
} from "@/api/service";

import {
  getNamespaceList as apiGetNamespaceList,
  getNamespaceOverview as apiGetNamespaceOverview,
  getNamespaceDetail as apiGetNamespaceDetail,
  getConfigMapList as apiGetConfigMapList,
  getSecretList as apiGetSecretList,
} from "@/api/namespace";

import {
  getIngressList as apiGetIngressList,
  getIngressOverview as apiGetIngressOverview,
  getIngressDetail as apiGetIngressDetail,
} from "@/api/ingress";

import {
  getEventList as apiGetEventList,
  getEventOverview as apiGetEventOverview,
} from "@/api/event";

import {
  getStatefulSetList as apiGetStatefulSetList,
  getStatefulSetDetail as apiGetStatefulSetDetail,
  getDaemonSetList as apiGetDaemonSetList,
  getDaemonSetDetail as apiGetDaemonSetDetail,
} from "@/api/workload";

import {
  getJobList as apiGetJobList,
  getJobDetail as apiGetJobDetail,
  getCronJobList as apiGetCronJobList,
  getCronJobDetail as apiGetCronJobDetail,
  getPVList as apiGetPVList,
  getPVDetail as apiGetPVDetail,
  getPVCList as apiGetPVCList,
  getPVCDetail as apiGetPVCDetail,
  getNetworkPolicyList as apiGetNetworkPolicyList,
  getNetworkPolicyDetail as apiGetNetworkPolicyDetail,
  getResourceQuotaList as apiGetResourceQuotaList,
  getResourceQuotaDetail as apiGetResourceQuotaDetail,
  getLimitRangeList as apiGetLimitRangeList,
  getLimitRangeDetail as apiGetLimitRangeDetail,
  getServiceAccountList as apiGetServiceAccountList,
  getServiceAccountDetail as apiGetServiceAccountDetail,
} from "@/api/cluster-resources";

// ============================================================
// Pod
// ============================================================

export function getPodList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("pod") === "mock") return mockGetPodList({ namespace: params.namespace });
  return apiGetPodList(params);
}

export function getPodOverview(params: { ClusterID: string }) {
  if (getDataSourceMode("pod") === "mock") return mockGetPodOverview();
  return apiGetPodOverview(params);
}

export function getPodDetail(params: { ClusterID: string; Namespace: string; PodName: string }) {
  if (getDataSourceMode("pod") === "mock") return mockGetPodDetail(params.PodName, params.Namespace);
  return apiGetPodDetail(params);
}

// ============================================================
// Node
// ============================================================

export function getNodeList(params: { cluster_id: string }) {
  if (getDataSourceMode("node") === "mock") return mockGetNodeList();
  return apiGetNodeList(params);
}

export function getNodeOverview(params: { ClusterID: string }) {
  if (getDataSourceMode("node") === "mock") return mockGetNodeOverview();
  return apiGetNodeOverview(params);
}

export function getNodeDetail(params: { ClusterID: string; NodeName: string }) {
  if (getDataSourceMode("node") === "mock") return mockGetNodeDetail(params.NodeName);
  return apiGetNodeDetail(params);
}

// ============================================================
// Deployment
// ============================================================

export function getDeploymentList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("deployment") === "mock") return mockGetDeploymentList({ namespace: params.namespace });
  return apiGetDeploymentList(params);
}

export function getDeploymentOverview(params: { ClusterID: string }) {
  if (getDataSourceMode("deployment") === "mock") return mockGetDeploymentOverview();
  return apiGetDeploymentOverview(params);
}

export function getDeploymentDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  if (getDataSourceMode("deployment") === "mock") return mockGetDeploymentDetail(params.Name, params.Namespace);
  return apiGetDeploymentDetail(params);
}

// ============================================================
// Service
// ============================================================

export function getServiceList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("service") === "mock") return mockGetServiceList({ namespace: params.namespace });
  return apiGetServiceList(params);
}

export function getServiceOverview(params: { ClusterID: string }) {
  if (getDataSourceMode("service") === "mock") return mockGetServiceOverview();
  return apiGetServiceOverview(params);
}

export function getServiceDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  if (getDataSourceMode("service") === "mock") return mockGetServiceDetail(params.Name, params.Namespace);
  return apiGetServiceDetail(params);
}

// ============================================================
// Namespace
// ============================================================

export function getNamespaceList(params: { cluster_id: string }) {
  if (getDataSourceMode("namespace") === "mock") return mockGetNamespaceList();
  return apiGetNamespaceList(params);
}

export function getNamespaceOverview(params: { ClusterID: string }) {
  if (getDataSourceMode("namespace") === "mock") return mockGetNamespaceOverview();
  return apiGetNamespaceOverview(params);
}

export function getNamespaceDetail(params: { ClusterID: string; Namespace: string }) {
  if (getDataSourceMode("namespace") === "mock") return mockGetNamespaceDetail(params.Namespace);
  return apiGetNamespaceDetail(params);
}

export function getConfigMapList(params: { cluster_id: string; namespace: string }) {
  if (getDataSourceMode("namespace") === "mock") return mockGetConfigMapList(params.namespace);
  return apiGetConfigMapList(params);
}

export function getSecretList(params: { cluster_id: string; namespace: string }) {
  if (getDataSourceMode("namespace") === "mock") return mockGetSecretList(params.namespace);
  return apiGetSecretList(params);
}

// ============================================================
// Ingress
// ============================================================

export function getIngressList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("ingress") === "mock") return mockGetIngressList({ namespace: params.namespace });
  return apiGetIngressList(params);
}

export function getIngressOverview(params: { ClusterID: string }) {
  if (getDataSourceMode("ingress") === "mock") return mockGetIngressOverview();
  return apiGetIngressOverview(params);
}

export function getIngressDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  if (getDataSourceMode("ingress") === "mock") return mockGetIngressDetail(params.Name, params.Namespace);
  return apiGetIngressDetail(params);
}

// ============================================================
// Event
// ============================================================

export function getEventList(params: { cluster_id: string; namespace?: string; type?: string }) {
  if (getDataSourceMode("event") === "mock") return mockGetEventList({ namespace: params.namespace, type: params.type });
  return apiGetEventList(params);
}

export function getEventOverview(params: { ClusterID: string; Namespace?: string; Type?: string }) {
  if (getDataSourceMode("event") === "mock") return mockGetEventOverview();
  return apiGetEventOverview(params);
}

// ============================================================
// StatefulSet / DaemonSet
// ============================================================

export function getStatefulSetList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("statefulset") === "mock") return mockGetStatefulSetList({ namespace: params.namespace });
  return apiGetStatefulSetList(params);
}

export function getStatefulSetDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  if (getDataSourceMode("statefulset") === "mock") return mockGetStatefulSetDetail(params.Name, params.Namespace);
  return apiGetStatefulSetDetail(params);
}

export function getDaemonSetList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("daemonset") === "mock") return mockGetDaemonSetList({ namespace: params.namespace });
  return apiGetDaemonSetList(params);
}

export function getDaemonSetDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  if (getDataSourceMode("daemonset") === "mock") return mockGetDaemonSetDetail(params.Name, params.Namespace);
  return apiGetDaemonSetDetail(params);
}

// ============================================================
// Job / CronJob
// ============================================================

export function getJobList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("job") === "mock") return mockGetJobList({ namespace: params.namespace });
  return apiGetJobList(params);
}

export function getJobDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  if (getDataSourceMode("job") === "mock") return mockGetJobDetail(params.Name, params.Namespace);
  return apiGetJobDetail(params);
}

export function getCronJobList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("cronjob") === "mock") return mockGetCronJobList({ namespace: params.namespace });
  return apiGetCronJobList(params);
}

export function getCronJobDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  if (getDataSourceMode("cronjob") === "mock") return mockGetCronJobDetail(params.Name, params.Namespace);
  return apiGetCronJobDetail(params);
}

// ============================================================
// Storage (PV / PVC)
// ============================================================

export function getPVList(params: { cluster_id: string }) {
  if (getDataSourceMode("pv") === "mock") return mockGetPVList();
  return apiGetPVList(params);
}

export function getPVDetail(params: { ClusterID: string; Name: string }) {
  if (getDataSourceMode("pv") === "mock") return mockGetPVDetail(params.Name);
  return apiGetPVDetail(params);
}

export function getPVCList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("pvc") === "mock") return mockGetPVCList({ namespace: params.namespace });
  return apiGetPVCList(params);
}

export function getPVCDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  if (getDataSourceMode("pvc") === "mock") return mockGetPVCDetail(params.Name, params.Namespace);
  return apiGetPVCDetail(params);
}

// ============================================================
// Policy (NetworkPolicy / ResourceQuota / LimitRange / ServiceAccount)
// ============================================================

export function getNetworkPolicyList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("netpol") === "mock") return mockGetNetworkPolicyList({ namespace: params.namespace });
  return apiGetNetworkPolicyList(params);
}

export function getNetworkPolicyDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  if (getDataSourceMode("netpol") === "mock") return mockGetNetworkPolicyDetail(params.Name, params.Namespace);
  return apiGetNetworkPolicyDetail(params);
}

export function getResourceQuotaList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("quota") === "mock") return mockGetResourceQuotaList({ namespace: params.namespace });
  return apiGetResourceQuotaList(params);
}

export function getResourceQuotaDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  if (getDataSourceMode("quota") === "mock") return mockGetResourceQuotaDetail(params.Name, params.Namespace);
  return apiGetResourceQuotaDetail(params);
}

export function getLimitRangeList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("limit") === "mock") return mockGetLimitRangeList({ namespace: params.namespace });
  return apiGetLimitRangeList(params);
}

export function getLimitRangeDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  if (getDataSourceMode("limit") === "mock") return mockGetLimitRangeDetail(params.Name, params.Namespace);
  return apiGetLimitRangeDetail(params);
}

export function getServiceAccountList(params: { cluster_id: string; namespace?: string }) {
  if (getDataSourceMode("sa") === "mock") return mockGetServiceAccountList({ namespace: params.namespace });
  return apiGetServiceAccountList(params);
}

export function getServiceAccountDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  if (getDataSourceMode("sa") === "mock") return mockGetServiceAccountDetail(params.Name, params.Namespace);
  return apiGetServiceAccountDetail(params);
}

// ============================================================
// Write operations — always use real API (no mock needed)
// ============================================================

export { restartPod } from "@/api/pod";
export { cordonNode, uncordonNode } from "@/api/node";
export { scaleDeployment, restartDeployment, updateDeploymentImage } from "@/api/deployment";
