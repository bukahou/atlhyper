/**
 * Pod API
 *
 * 适配 Master V2 API（嵌套结构）
 */

import { get, post } from "./request";
import type { PodOverview, PodDetail, PodItem } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

interface PodListParams {
  cluster_id: string;
  namespace?: string;
  node?: string;
  phase?: string;
  limit?: number;
  offset?: number;
}

// ============================================================
// 后端返回类型（Master V2 API 嵌套结构）
// ============================================================

// 后端容器端口
interface ContainerPortApi {
  name?: string;
  containerPort: number;
  protocol?: string;
  hostPort?: number;
}

// 后端环境变量
interface EnvVarApi {
  name: string;
  value?: string;
  valueFrom?: string;
}

// 后端卷挂载
interface VolumeMountApi {
  name: string;
  mountPath: string;
  subPath?: string;
  readOnly?: boolean;
}

// 后端探针
interface ProbeApi {
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

// 后端容器详情
interface PodContainerApi {
  name: string;
  image: string;
  imagePullPolicy?: string;
  command?: string[];
  args?: string[];
  workingDir?: string;
  ports?: ContainerPortApi[];
  envs?: EnvVarApi[];
  volumeMounts?: VolumeMountApi[];
  requests?: Record<string, string>;
  limits?: Record<string, string>;
  livenessProbe?: ProbeApi;
  readinessProbe?: ProbeApi;
  startupProbe?: ProbeApi;
  // 运行状态
  state?: string;
  stateReason?: string;
  stateMessage?: string;
  ready?: boolean;
  restartCount?: number;
  lastTerminationReason?: string;
  lastTerminationMessage?: string;
  lastTerminationTime?: string;
}

// 后端卷定义
interface VolumeSpecApi {
  name: string;
  type: string;
  source?: string;
}

// 后端 Pod Condition
interface PodConditionApi {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
}

// 后端 Toleration
interface TolerationApi {
  key?: string;
  operator?: string;
  value?: string;
  effect?: string;
  tolerationSeconds?: number;
}

// 后端 Affinity（简化）
interface AffinityApi {
  nodeAffinity?: string;
  podAffinity?: string;
  podAntiAffinity?: string;
}

// 后端 Pod 结构（嵌套）
interface PodApiItem {
  summary: {
    name: string;
    namespace: string;
    nodeName?: string;
    ownerKind?: string;
    ownerName?: string;
    createdAt: string;
    age: string;
  };
  spec: {
    restartPolicy?: string;
    serviceAccountName?: string;
    nodeSelector?: Record<string, string>;
    tolerations?: TolerationApi[];
    affinity?: AffinityApi;
    dnsPolicy?: string;
    hostNetwork?: boolean;
    runtimeClassName?: string;
    priorityClassName?: string;
    terminationGracePeriodSeconds?: number;
    imagePullSecrets?: string[];
  };
  status: {
    phase: string;
    ready: string;
    restarts: number;
    qosClass?: string;
    podIP?: string;
    podIPs?: string[];
    hostIP?: string;
    reason?: string;
    message?: string;
    conditions?: PodConditionApi[];
    cpuUsage?: string;
    memoryUsage?: string;
  };
  containers: PodContainerApi[];
  initContainers?: PodContainerApi[];
  volumes?: VolumeSpecApi[];
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

interface PodListResponse {
  message: string;
  data: PodApiItem[];
  total: number;
}

interface PodDetailResponse {
  message: string;
  data: PodApiItem;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 Pod 列表
 * GET /api/v2/pods?cluster_id=xxx&namespace=xxx
 */
export function getPodList(params: PodListParams) {
  return get<PodListResponse>("/api/v2/pods", params);
}

/**
 * 获取 Pod 详情
 * GET /api/v2/pods/{name}?cluster_id=xxx&namespace=xxx
 */
export async function getPodDetail(params: {
  ClusterID: string;
  Namespace: string;
  PodName: string;
}) {
  const response = await get<PodDetailResponse>(
    `/api/v2/pods/${encodeURIComponent(params.PodName)}`,
    {
      cluster_id: params.ClusterID,
      namespace: params.Namespace,
    }
  );

  const apiPod = response.data.data;
  if (!apiPod) {
    throw new Error("Pod not found");
  }

  const detail = transformPodToDetail(apiPod);

  return {
    ...response,
    data: { data: detail },
  };
}

/**
 * 将 PodApiItem 转换为 PodDetail 格式
 */
function transformPodToDetail(pod: PodApiItem): PodDetail {
  // 推断 controller
  let controller = "";
  if (pod.summary.ownerKind && pod.summary.ownerName) {
    controller = `${pod.summary.ownerKind}/${pod.summary.ownerName}`;
  }

  return {
    // 基本信息
    name: pod.summary.name,
    namespace: pod.summary.namespace,
    controller,
    phase: pod.status.phase,
    ready: pod.status.ready,
    restarts: pod.status.restarts,
    startTime: pod.summary.createdAt,
    age: pod.summary.age,
    node: pod.summary.nodeName || "",
    podIP: pod.status.podIP,
    hostIP: pod.status.hostIP,
    qosClass: pod.status.qosClass,
    reason: pod.status.reason,
    message: pod.status.message,

    // 调度/策略
    restartPolicy: pod.spec.restartPolicy,
    priorityClassName: pod.spec.priorityClassName,
    runtimeClassName: pod.spec.runtimeClassName,
    terminationGracePeriodSeconds: pod.spec.terminationGracePeriodSeconds,
    tolerations: pod.spec.tolerations,
    affinity: pod.spec.affinity,
    nodeSelector: pod.spec.nodeSelector,

    // 网络
    hostNetwork: pod.spec.hostNetwork,
    podIPs: pod.status.podIPs,
    dnsPolicy: pod.spec.dnsPolicy,
    serviceAccountName: pod.spec.serviceAccountName,

    // 指标
    cpuUsage: pod.status.cpuUsage,
    memUsage: pod.status.memoryUsage,

    // 容器
    containers: (pod.containers || []).map((c) => ({
      name: c.name,
      image: c.image,
      imagePullPolicy: c.imagePullPolicy,
      ports: c.ports?.map((p) => ({
        containerPort: p.containerPort,
        protocol: p.protocol || "TCP",
        name: p.name,
      })),
      envs: c.envs?.map((e) => ({
        name: e.name,
        value: e.value,
      })),
      volumeMounts: c.volumeMounts?.map((v) => ({
        name: v.name,
        mountPath: v.mountPath,
        readOnly: v.readOnly,
        subPath: v.subPath,
      })),
      requests: c.requests,
      limits: c.limits,
      readinessProbe: c.readinessProbe,
      livenessProbe: c.livenessProbe,
      startupProbe: c.startupProbe,
      state: c.state,
      restartCount: c.restartCount,
      lastTerminatedReason: c.lastTerminationReason,
    })),

    // 存储卷
    volumes: (pod.volumes || []).map((v) => ({
      name: v.name,
      type: v.type,
      sourceBrief: v.source,
    })),
  };
}

/**
 * 获取 Pod 日志（需要 Operator 权限）
 * POST /api/v2/ops/pods/logs
 */
interface PodLogsResponse {
  message: string;
  data?: {
    logs: string;
  };
}

export function getPodLogs(data: {
  ClusterID: string;
  Namespace: string;
  Pod: string;
  Container?: string;
  TailLines?: number;
  TimeoutSeconds?: number;
}) {
  return post<PodLogsResponse>("/api/v2/ops/pods/logs", {
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    name: data.Pod,
    container: data.Container,
    tail_lines: data.TailLines,
  });
}

/**
 * 重启 Pod（需要 Operator 权限）
 * POST /api/v2/ops/pods/restart
 */
interface CommandResponse {
  message: string;
  command_id: string;
  status: string;
}

export function restartPod(data: {
  ClusterID: string;
  Namespace: string;
  Pod: string;
}) {
  return post<CommandResponse>("/api/v2/ops/pods/restart", {
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    name: data.Pod,
  });
}

// ============================================================
// 数据转换函数（列表用）
// ============================================================

/**
 * 将后端 Pod 转换为前端 PodItem 格式
 */
function transformPod(pod: PodApiItem): PodItem {
  // 确定 deployment 名称（从 owner 或 labels 推断）
  let deployment = "";
  if (pod.summary.ownerKind === "ReplicaSet" && pod.summary.ownerName) {
    // ReplicaSet 通常命名为 deployment-name-hash
    const parts = pod.summary.ownerName.split("-");
    if (parts.length > 1) {
      deployment = parts.slice(0, -1).join("-");
    } else {
      deployment = pod.summary.ownerName;
    }
  } else if (pod.summary.ownerKind === "Deployment" && pod.summary.ownerName) {
    deployment = pod.summary.ownerName;
  } else if (pod.labels?.["app"]) {
    deployment = pod.labels["app"];
  } else if (pod.labels?.["app.kubernetes.io/name"]) {
    deployment = pod.labels["app.kubernetes.io/name"];
  }

  // 解析 metrics
  const cpuText = pod.status.cpuUsage || "-";
  const memoryText = pod.status.memoryUsage || "-";

  return {
    name: pod.summary.name,
    namespace: pod.summary.namespace,
    deployment,
    ready: pod.status.ready,
    phase: pod.status.phase,
    restarts: pod.status.restarts,
    cpu: 0,
    cpuPercent: 0,
    memory: 0,
    memPercent: 0,
    cpuText,
    cpuPercentText: "-",
    memoryText,
    memPercentText: "-",
    startTime: pod.summary.createdAt,
    node: pod.summary.nodeName || "",
    age: pod.summary.age,
  };
}

/**
 * 将 Pod 列表转换为 PodOverview 格式
 */
function transformPodListToOverview(pods: PodApiItem[]): PodOverview {
  // 计算统计
  let running = 0;
  let pending = 0;
  let failed = 0;
  let unknown = 0;

  for (const pod of pods) {
    switch (pod.status.phase) {
      case "Running":
        running++;
        break;
      case "Pending":
        pending++;
        break;
      case "Failed":
        failed++;
        break;
      case "Succeeded":
        // Succeeded 不计入，或可以单独统计
        break;
      default:
        unknown++;
    }
  }

  return {
    cards: {
      running,
      pending,
      failed,
      unknown,
    },
    pods: pods.map(transformPod),
  };
}

/**
 * 获取 Pod 概览（包含统计卡片和列表）
 */
export async function getPodOverview(data: { ClusterID: string }) {
  const response = await getPodList({ cluster_id: data.ClusterID });
  const pods = response.data.data || [];
  const overview = transformPodListToOverview(pods);

  return {
    ...response,
    data: {
      data: overview,
    },
  };
}
