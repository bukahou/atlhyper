/**
 * Workload API (StatefulSet & DaemonSet)
 *
 * 适配 Master V2 API（嵌套结构）
 */

import { get } from "./request";

// ============================================================
// 通用类型
// ============================================================

interface UpdateStrategy {
  type?: string;
  partition?: number;
  maxUnavailable?: string;
  maxSurge?: string;
}

interface WorkloadCondition {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastUpdateTime?: string;
  lastTransitionTime?: string;
}

interface WorkloadRollout {
  phase: string;
  message?: string;
  badges?: string[];
}

interface LabelSelector {
  matchLabels?: Record<string, string>;
  matchExpressions?: { key: string; operator: string; values?: string[] }[];
}

interface ContainerDetail {
  name: string;
  image: string;
  imagePullPolicy?: string;
  command?: string[];
  args?: string[];
  workingDir?: string;
  ports?: { name?: string; containerPort: number; protocol?: string }[];
  envs?: { name: string; value?: string; valueFrom?: string }[];
  volumeMounts?: { name: string; mountPath: string; subPath?: string; readOnly?: boolean }[];
  requests?: Record<string, string>;
  limits?: Record<string, string>;
  livenessProbe?: ProbeSpec;
  readinessProbe?: ProbeSpec;
  startupProbe?: ProbeSpec;
}

interface ProbeSpec {
  type: string;
  path?: string;
  port?: number;
  command?: string;
  initialDelaySeconds?: number;
  periodSeconds?: number;
  timeoutSeconds?: number;
  successThreshold?: number;
  failureThreshold?: number;
}

interface VolumeSpec {
  name: string;
  type: string;
  source?: string;
}

interface Toleration {
  key?: string;
  operator?: string;
  value?: string;
  effect?: string;
  tolerationSeconds?: number;
}

interface PodTemplate {
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  containers: ContainerDetail[];
  volumes?: VolumeSpec[];
  serviceAccountName?: string;
  nodeSelector?: Record<string, string>;
  tolerations?: Toleration[];
  affinity?: { nodeAffinity?: string; podAffinity?: string; podAntiAffinity?: string };
  runtimeClassName?: string;
  imagePullSecrets?: string[];
  hostNetwork?: boolean;
  dnsPolicy?: string;
}

// ============================================================
// StatefulSet 类型
// ============================================================

interface StatefulSetApiItem {
  summary: {
    name: string;
    namespace: string;
    replicas: number;
    ready: number;
    current: number;
    updated: number;
    available: number;
    createdAt: string;
    age: string;
    serviceName?: string;
    selector?: string;
  };
  spec: {
    replicas?: number;
    serviceName?: string;
    podManagementPolicy?: string;
    updateStrategy?: UpdateStrategy;
    revisionHistoryLimit?: number;
    minReadySeconds?: number;
    persistentVolumeClaimRetentionPolicy?: {
      whenDeleted?: string;
      whenScaled?: string;
    };
    selector?: LabelSelector;
    volumeClaimTemplates?: {
      name: string;
      accessModes?: string[];
      storageClass?: string;
      storage?: string;
    }[];
  };
  template: PodTemplate;
  status: {
    observedGeneration?: number;
    replicas: number;
    readyReplicas?: number;
    currentReplicas?: number;
    updatedReplicas?: number;
    availableReplicas?: number;
    currentRevision?: string;
    updateRevision?: string;
    collisionCount?: number;
    conditions?: WorkloadCondition[];
  };
  rollout?: WorkloadRollout;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

interface StatefulSetListResponse {
  message: string;
  data: StatefulSetApiItem[];
  total: number;
}

interface StatefulSetDetailResponse {
  message: string;
  data: StatefulSetApiItem;
}

// ============================================================
// DaemonSet 类型
// ============================================================

interface DaemonSetApiItem {
  summary: {
    name: string;
    namespace: string;
    desiredNumberScheduled: number;
    currentNumberScheduled: number;
    numberReady: number;
    numberAvailable: number;
    numberUnavailable: number;
    numberMisscheduled: number;
    updatedNumberScheduled: number;
    createdAt: string;
    age: string;
    selector?: string;
  };
  spec: {
    updateStrategy?: UpdateStrategy;
    minReadySeconds?: number;
    revisionHistoryLimit?: number;
    selector?: LabelSelector;
  };
  template: PodTemplate;
  status: {
    observedGeneration?: number;
    desiredNumberScheduled: number;
    currentNumberScheduled: number;
    numberReady: number;
    numberAvailable?: number;
    numberUnavailable?: number;
    numberMisscheduled: number;
    updatedNumberScheduled?: number;
    collisionCount?: number;
    conditions?: WorkloadCondition[];
  };
  rollout?: WorkloadRollout;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

interface DaemonSetListResponse {
  message: string;
  data: DaemonSetApiItem[];
  total: number;
}

interface DaemonSetDetailResponse {
  message: string;
  data: DaemonSetApiItem;
}

// ============================================================
// 导出类型（供组件使用）
// ============================================================

export interface StatefulSetDetail {
  // Summary
  name: string;
  namespace: string;
  replicas: number;
  ready: number;
  current: number;
  updated: number;
  available: number;
  createdAt: string;
  age: string;
  serviceName?: string;
  selector?: string;

  // Spec
  spec: {
    replicas?: number;
    serviceName?: string;
    podManagementPolicy?: string;
    updateStrategy?: UpdateStrategy;
    revisionHistoryLimit?: number;
    minReadySeconds?: number;
    pvcRetentionPolicy?: {
      whenDeleted?: string;
      whenScaled?: string;
    };
    matchLabels?: Record<string, string>;
    volumeClaimTemplates?: {
      name: string;
      accessModes?: string[];
      storageClass?: string;
      storage?: string;
    }[];
  };

  // Template
  template: PodTemplate;

  // Status
  status: {
    observedGeneration?: number;
    replicas: number;
    readyReplicas?: number;
    currentReplicas?: number;
    updatedReplicas?: number;
    availableReplicas?: number;
    currentRevision?: string;
    updateRevision?: string;
    collisionCount?: number;
  };

  // Conditions
  conditions?: WorkloadCondition[];

  // Rollout
  rollout?: WorkloadRollout;

  // Labels & Annotations
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

export interface DaemonSetDetail {
  // Summary
  name: string;
  namespace: string;
  desired: number;
  current: number;
  ready: number;
  available: number;
  unavailable: number;
  misscheduled: number;
  updatedScheduled: number;
  createdAt: string;
  age: string;
  selector?: string;

  // Spec
  spec: {
    updateStrategy?: UpdateStrategy;
    minReadySeconds?: number;
    revisionHistoryLimit?: number;
    matchLabels?: Record<string, string>;
  };

  // Template
  template: PodTemplate;

  // Status
  status: {
    observedGeneration?: number;
    desiredNumberScheduled: number;
    currentNumberScheduled: number;
    numberReady: number;
    numberAvailable?: number;
    numberUnavailable?: number;
    numberMisscheduled: number;
    updatedNumberScheduled?: number;
    collisionCount?: number;
  };

  // Conditions
  conditions?: WorkloadCondition[];

  // Rollout
  rollout?: WorkloadRollout;

  // Labels & Annotations
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

// ============================================================
// StatefulSet API
// ============================================================

interface ListParams {
  cluster_id: string;
  namespace?: string;
}

/**
 * 获取 StatefulSet 列表
 */
export function getStatefulSetList(params: ListParams) {
  return get<StatefulSetListResponse>("/api/v2/statefulsets", params);
}

/**
 * 获取 StatefulSet 详情
 */
export async function getStatefulSetDetail(data: { ClusterID: string; Namespace: string; Name: string }) {
  const response = await get<StatefulSetDetailResponse>(
    `/api/v2/statefulsets/${encodeURIComponent(data.Name)}`,
    {
      cluster_id: data.ClusterID,
      namespace: data.Namespace,
    }
  );

  const api = response.data.data;
  if (!api) {
    throw new Error("StatefulSet not found");
  }

  const detail = transformStatefulSetDetail(api);

  return {
    ...response,
    data: { data: detail },
  };
}

function transformStatefulSetDetail(api: StatefulSetApiItem): StatefulSetDetail {
  return {
    name: api.summary.name,
    namespace: api.summary.namespace,
    replicas: api.summary.replicas,
    ready: api.summary.ready,
    current: api.summary.current,
    updated: api.summary.updated,
    available: api.summary.available,
    createdAt: api.summary.createdAt,
    age: api.summary.age,
    serviceName: api.summary.serviceName,
    selector: api.summary.selector,

    spec: {
      replicas: api.spec.replicas,
      serviceName: api.spec.serviceName,
      podManagementPolicy: api.spec.podManagementPolicy,
      updateStrategy: api.spec.updateStrategy,
      revisionHistoryLimit: api.spec.revisionHistoryLimit,
      minReadySeconds: api.spec.minReadySeconds,
      pvcRetentionPolicy: api.spec.persistentVolumeClaimRetentionPolicy,
      matchLabels: api.spec.selector?.matchLabels,
      volumeClaimTemplates: api.spec.volumeClaimTemplates,
    },

    template: api.template,

    status: {
      observedGeneration: api.status.observedGeneration,
      replicas: api.status.replicas,
      readyReplicas: api.status.readyReplicas,
      currentReplicas: api.status.currentReplicas,
      updatedReplicas: api.status.updatedReplicas,
      availableReplicas: api.status.availableReplicas,
      currentRevision: api.status.currentRevision,
      updateRevision: api.status.updateRevision,
      collisionCount: api.status.collisionCount,
    },

    conditions: api.status.conditions,
    rollout: api.rollout,
    labels: api.labels,
    annotations: api.annotations,
  };
}

// ============================================================
// DaemonSet API
// ============================================================

/**
 * 获取 DaemonSet 列表
 */
export function getDaemonSetList(params: ListParams) {
  return get<DaemonSetListResponse>("/api/v2/daemonsets", params);
}

/**
 * 获取 DaemonSet 详情
 */
export async function getDaemonSetDetail(data: { ClusterID: string; Namespace: string; Name: string }) {
  const response = await get<DaemonSetDetailResponse>(
    `/api/v2/daemonsets/${encodeURIComponent(data.Name)}`,
    {
      cluster_id: data.ClusterID,
      namespace: data.Namespace,
    }
  );

  const api = response.data.data;
  if (!api) {
    throw new Error("DaemonSet not found");
  }

  const detail = transformDaemonSetDetail(api);

  return {
    ...response,
    data: { data: detail },
  };
}

function transformDaemonSetDetail(api: DaemonSetApiItem): DaemonSetDetail {
  return {
    name: api.summary.name,
    namespace: api.summary.namespace,
    desired: api.summary.desiredNumberScheduled,
    current: api.summary.currentNumberScheduled,
    ready: api.summary.numberReady,
    available: api.summary.numberAvailable,
    unavailable: api.summary.numberUnavailable,
    misscheduled: api.summary.numberMisscheduled,
    updatedScheduled: api.summary.updatedNumberScheduled,
    createdAt: api.summary.createdAt,
    age: api.summary.age,
    selector: api.summary.selector,

    spec: {
      updateStrategy: api.spec.updateStrategy,
      minReadySeconds: api.spec.minReadySeconds,
      revisionHistoryLimit: api.spec.revisionHistoryLimit,
      matchLabels: api.spec.selector?.matchLabels,
    },

    template: api.template,

    status: {
      observedGeneration: api.status.observedGeneration,
      desiredNumberScheduled: api.status.desiredNumberScheduled,
      currentNumberScheduled: api.status.currentNumberScheduled,
      numberReady: api.status.numberReady,
      numberAvailable: api.status.numberAvailable,
      numberUnavailable: api.status.numberUnavailable,
      numberMisscheduled: api.status.numberMisscheduled,
      updatedNumberScheduled: api.status.updatedNumberScheduled,
      collisionCount: api.status.collisionCount,
    },

    conditions: api.status.conditions,
    rollout: api.rollout,
    labels: api.labels,
    annotations: api.annotations,
  };
}
