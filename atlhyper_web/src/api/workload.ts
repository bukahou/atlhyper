/**
 * Workload API (StatefulSet & DaemonSet)
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get } from "./request";

// ============================================================
// 嵌套子结构类型（匹配 model_v2 JSON 结构）
// ============================================================

interface WorkloadRollout {
  phase: string;
  message?: string;
  badges?: string[];
}

interface WorkloadCondition {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastUpdateTime?: string;
  lastTransitionTime?: string;
}

interface UpdateStrategy {
  type?: string;
  partition?: number;
  maxUnavailable?: string;
  maxSurge?: string;
}

interface ContainerPort {
  name?: string;
  containerPort: number;
  protocol?: string;
}

interface Probe {
  type: string;
  path?: string;
  port?: number;
  command?: string;
}

interface ContainerDetail {
  name: string;
  image: string;
  imagePullPolicy?: string;
  ports?: ContainerPort[];
  requests?: Record<string, string>;
  limits?: Record<string, string>;
  livenessProbe?: Probe;
  readinessProbe?: Probe;
  startupProbe?: Probe;
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
}

interface PodTemplate {
  containers: ContainerDetail[];
  volumes?: VolumeSpec[];
  nodeSelector?: Record<string, string>;
  tolerations?: Toleration[];
}

// StatefulSet 专用

interface PVCRetentionPolicy {
  whenDeleted?: string;
  whenScaled?: string;
}

interface VolumeClaimTemplate {
  name: string;
  accessModes?: string[];
  storageClass?: string;
  storage?: string;
}

interface StatefulSetSpec {
  podManagementPolicy?: string;
  updateStrategy?: UpdateStrategy;
  minReadySeconds?: number;
  revisionHistoryLimit?: number;
  persistentVolumeClaimRetentionPolicy?: PVCRetentionPolicy;
  volumeClaimTemplates?: VolumeClaimTemplate[];
}

interface StatefulSetStatus {
  currentRevision?: string;
  updateRevision?: string;
}

// DaemonSet 专用

interface DaemonSetSpec {
  updateStrategy?: UpdateStrategy;
  minReadySeconds?: number;
  revisionHistoryLimit?: number;
}

// ============================================================
// 导出类型（供组件使用）
// ============================================================

export interface StatefulSetDetail {
  // 扁平顶层（从 summary 提取）
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

  // 嵌套子结构（后端透传 model_v2 原始结构）
  spec: StatefulSetSpec;
  template?: PodTemplate;
  status: StatefulSetStatus;
  conditions?: WorkloadCondition[];
  rollout?: WorkloadRollout;

  // 元数据
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

export interface DaemonSetDetail {
  // 扁平顶层（从 summary 提取，字段已重命名）
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

  // 嵌套子结构（后端透传 model_v2 原始结构）
  spec: DaemonSetSpec;
  template?: PodTemplate;
  conditions?: WorkloadCondition[];
  rollout?: WorkloadRollout;

  // 元数据
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

// ============================================================
// 列表项类型（从 API 嵌套结构中提取 summary 字段）
// ============================================================

export interface DaemonSetListItem {
  name: string;
  namespace: string;
  desired: number;
  current: number;
  ready: number;
  available: number;
  misscheduled: number;
  createdAt: string;
  age: string;
}

export interface StatefulSetListItem {
  name: string;
  namespace: string;
  replicas: number;
  ready: number;
  current: number;
  updated: number;
  available: number;
  createdAt: string;
  age: string;
  serviceName: string;
}

// ============================================================
// 响应类型
// ============================================================

interface ListParams {
  cluster_id: string;
  namespace?: string;
}

// StatefulSet list still returns raw model_v2 (no list item conversion)
interface StatefulSetListResponse {
  message: string;
  data: unknown[];
  total: number;
}

interface DaemonSetListResponse {
  message: string;
  data: unknown[];
  total: number;
}

// ============================================================
// StatefulSet API
// ============================================================

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
  return get<{ message: string; data: StatefulSetDetail }>(
    `/api/v2/statefulsets/${encodeURIComponent(data.Name)}`,
    {
      cluster_id: data.ClusterID,
      namespace: data.Namespace,
    }
  );
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
  return get<{ message: string; data: DaemonSetDetail }>(
    `/api/v2/daemonsets/${encodeURIComponent(data.Name)}`,
    {
      cluster_id: data.ClusterID,
      namespace: data.Namespace,
    }
  );
}

// ============================================================
// 列表解析函数（从嵌套 model_v2 结构提取 summary 为扁平列表项）
// ============================================================

interface RawItem {
  summary?: Record<string, unknown>;
  [key: string]: unknown;
}

export function parseDaemonSetList(raw: unknown[]): DaemonSetListItem[] {
  return (raw as RawItem[]).map((item) => {
    const s = item.summary || item;
    return {
      name: String(s.name || ""),
      namespace: String(s.namespace || ""),
      desired: Number(s.desiredNumberScheduled || 0),
      current: Number(s.currentNumberScheduled || 0),
      ready: Number(s.numberReady || 0),
      available: Number(s.numberAvailable || 0),
      misscheduled: Number(s.numberMisscheduled || 0),
      createdAt: String(s.createdAt || ""),
      age: String(s.age || ""),
    };
  });
}

export function parseStatefulSetList(raw: unknown[]): StatefulSetListItem[] {
  return (raw as RawItem[]).map((item) => {
    const s = item.summary || item;
    return {
      name: String(s.name || ""),
      namespace: String(s.namespace || ""),
      replicas: Number(s.replicas || 0),
      ready: Number(s.ready || 0),
      current: Number(s.current || 0),
      updated: Number(s.updated || 0),
      available: Number(s.available || 0),
      createdAt: String(s.createdAt || ""),
      age: String(s.age || ""),
      serviceName: String(s.serviceName || ""),
    };
  });
}
