/**
 * Deployment API
 *
 * 适配 Master V2 API（嵌套结构）
 */

import { get, post } from "./request";
import type { DeploymentOverview, DeploymentDetail, DeploymentItem } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

interface DeploymentListParams {
  cluster_id: string;
  namespace?: string;
  limit?: number;
  offset?: number;
}

// ============================================================
// 响应类型（后端返回嵌套结构）
// ============================================================

// 后端返回的 Deployment 格式（与 model_v2.Deployment 一致）
interface DeploymentApiItem {
  summary: {
    name: string;
    namespace: string;
    strategy: string;
    replicas: number;
    updated: number;
    ready: number;
    available: number;
    unavailable?: number;
    paused?: boolean;
    createdAt: string;
    age: string;
    selector?: string;
  };
  spec: {
    replicas?: number;
    selector?: {
      matchLabels?: Record<string, string>;
      matchExpressions?: { key: string; operator: string; values?: string[] }[];
    };
    strategy?: {
      type: string;
      rollingUpdate?: {
        maxUnavailable?: string;
        maxSurge?: string;
      };
    };
    minReadySeconds?: number;
    revisionHistoryLimit?: number;
    progressDeadlineSeconds?: number;
  };
  template: {
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
    containers: {
      name: string;
      image: string;
      imagePullPolicy?: string;
      command?: string[];
      args?: string[];
      workingDir?: string;
      ports?: { name?: string; containerPort: number; protocol?: string; hostPort?: number }[];
      envs?: { name: string; value?: string; valueFrom?: string }[];
      volumeMounts?: { name: string; mountPath: string; subPath?: string; readOnly?: boolean }[];
      requests?: Record<string, string>;
      limits?: Record<string, string>;
      livenessProbe?: ProbeApi;
      readinessProbe?: ProbeApi;
      startupProbe?: ProbeApi;
    }[];
    volumes?: { name: string; type: string; source?: string }[];
    serviceAccountName?: string;
    nodeSelector?: Record<string, string>;
    tolerations?: { key?: string; operator?: string; value?: string; effect?: string; tolerationSeconds?: number }[];
    affinity?: { nodeAffinity?: string; podAffinity?: string; podAntiAffinity?: string };
    runtimeClassName?: string;
    imagePullSecrets?: string[];
    hostNetwork?: boolean;
    dnsPolicy?: string;
  };
  status: {
    observedGeneration?: number;
    replicas: number;
    updatedReplicas?: number;
    readyReplicas?: number;
    availableReplicas?: number;
    unavailableReplicas?: number;
    collisionCount?: number;
    conditions?: {
      type: string;
      status: string;
      reason?: string;
      message?: string;
      lastUpdateTime?: string;
      lastTransitionTime?: string;
    }[];
  };
  rollout?: {
    phase: string;
    message?: string;
    badges?: string[];
  };
  replicaSets?: {
    name: string;
    namespace: string;
    revision?: string;
    replicas: number;
    ready: number;
    available: number;
    image?: string;
    createdAt: string;
    age: string;
  }[];
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

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

interface DeploymentListResponse {
  message: string;
  data: DeploymentApiItem[];
  total: number;
}

interface DeploymentDetailResponse {
  message: string;
  data: DeploymentApiItem;
}

// ============================================================
// 操作请求类型
// ============================================================

interface CommandResponse {
  message: string;
  command_id: string;
  status: string;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 Deployment 列表
 * GET /api/v2/deployments?cluster_id=xxx&namespace=xxx
 */
export function getDeploymentList(params: DeploymentListParams) {
  return get<DeploymentListResponse>("/api/v2/deployments", params);
}

/**
 * 获取 Deployment 详情
 * GET /api/v2/deployments/{name}?cluster_id=xxx&namespace=xxx
 */
export async function getDeploymentDetail(data: { ClusterID: string; Namespace: string; Name: string }) {
  const response = await get<DeploymentDetailResponse>(
    `/api/v2/deployments/${encodeURIComponent(data.Name)}`,
    {
      cluster_id: data.ClusterID,
      namespace: data.Namespace,
    }
  );

  const apiDeploy = response.data.data;
  if (!apiDeploy) {
    throw new Error("Deployment not found");
  }

  // 转换为前端详情格式
  const detail = transformToDeploymentDetail(apiDeploy);

  return {
    ...response,
    data: {
      data: detail,
    },
  };
}

/**
 * Deployment 扩缩容（需要 Operator 权限）
 * POST /api/v2/ops/deployments/scale
 */
export function scaleDeployment(data: { ClusterID: string; Namespace: string; Name: string; Kind?: string; Replicas: number }) {
  return post<CommandResponse>("/api/v2/ops/deployments/scale", {
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    name: data.Name,
    replicas: data.Replicas,
  });
}

/**
 * Deployment 滚动重启（需要 Operator 权限）
 * POST /api/v2/ops/deployments/restart
 */
export function restartDeployment(data: { ClusterID: string; Namespace: string; Name: string }) {
  return post<CommandResponse>("/api/v2/ops/deployments/restart", {
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    name: data.Name,
  });
}

/**
 * Deployment 更新镜像（需要 Operator 权限）
 * POST /api/v2/ops/deployments/image
 */
export function updateDeploymentImage(data: { ClusterID: string; Namespace: string; Name: string; Kind?: string; NewImage: string; OldImage?: string }) {
  return post<CommandResponse>("/api/v2/ops/deployments/image", {
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    name: data.Name,
    container: "main", // 默认容器名
    image: data.NewImage,
  });
}

// ============================================================
// 数据转换
// ============================================================

/**
 * 将后端 API 返回的 Deployment 数据转换为列表项格式
 */
function transformDeploymentItem(apiItem: DeploymentApiItem): DeploymentItem {
  const summary = apiItem.summary;
  const image = apiItem.template?.containers?.[0]?.image || "";

  return {
    name: summary.name,
    namespace: summary.namespace,
    image,
    replicas: `${summary.ready}/${summary.replicas}`,
    labelCount: Object.keys(apiItem.labels || {}).length,
    annoCount: Object.keys(apiItem.annotations || {}).length,
    createdAt: summary.createdAt,
  };
}

/**
 * 将后端 Deployment 转换为详情格式
 */
function transformToDeploymentDetail(apiDeploy: DeploymentApiItem): DeploymentDetail {
  const summary = apiDeploy.summary;
  const spec = apiDeploy.spec;
  const status = apiDeploy.status;
  const template = apiDeploy.template;

  return {
    // 基本信息
    name: summary.name,
    namespace: summary.namespace,
    strategy: summary.strategy,
    replicas: summary.replicas,
    updated: summary.updated,
    ready: summary.ready,
    available: summary.available,
    unavailable: summary.unavailable,
    paused: summary.paused,
    selector: summary.selector,
    createdAt: summary.createdAt,
    age: summary.age,

    // Spec
    spec: {
      replicas: spec.replicas,
      minReadySeconds: spec.minReadySeconds,
      revisionHistoryLimit: spec.revisionHistoryLimit,
      progressDeadlineSeconds: spec.progressDeadlineSeconds,
      strategyType: spec.strategy?.type,
      maxUnavailable: spec.strategy?.rollingUpdate?.maxUnavailable,
      maxSurge: spec.strategy?.rollingUpdate?.maxSurge,
      matchLabels: spec.selector?.matchLabels,
    },

    // Template
    template: {
      labels: template.labels,
      annotations: template.annotations,
      containers: template.containers.map((c) => ({
        name: c.name,
        image: c.image,
        imagePullPolicy: c.imagePullPolicy,
        command: c.command,
        args: c.args,
        workingDir: c.workingDir,
        ports: c.ports,
        envs: c.envs,
        volumeMounts: c.volumeMounts,
        requests: c.requests,
        limits: c.limits,
        livenessProbe: c.livenessProbe,
        readinessProbe: c.readinessProbe,
        startupProbe: c.startupProbe,
      })),
      volumes: template.volumes,
      serviceAccountName: template.serviceAccountName,
      nodeSelector: template.nodeSelector,
      tolerations: template.tolerations,
      affinity: template.affinity,
      hostNetwork: template.hostNetwork,
      dnsPolicy: template.dnsPolicy,
      runtimeClassName: template.runtimeClassName,
      imagePullSecrets: template.imagePullSecrets,
    },

    // Status
    status: {
      observedGeneration: status.observedGeneration,
      replicas: status.replicas,
      updatedReplicas: status.updatedReplicas,
      readyReplicas: status.readyReplicas,
      availableReplicas: status.availableReplicas,
      unavailableReplicas: status.unavailableReplicas,
      collisionCount: status.collisionCount,
    },

    // Conditions
    conditions: status.conditions?.map((c) => ({
      type: c.type,
      status: c.status,
      reason: c.reason,
      message: c.message,
      lastUpdateTime: c.lastUpdateTime,
      lastTransitionTime: c.lastTransitionTime,
    })),

    // Rollout
    rollout: apiDeploy.rollout,

    // ReplicaSets
    replicaSets: apiDeploy.replicaSets?.map((rs) => ({
      name: rs.name,
      replicas: rs.replicas,
      ready: rs.ready,
      available: rs.available,
      revision: rs.revision,
      image: rs.image,
      createdAt: rs.createdAt,
    })),

    // Labels & Annotations
    labels: apiDeploy.labels,
    annotations: apiDeploy.annotations,
  };
}

/**
 * 将 Deployment 列表转换为 DeploymentOverview 格式
 */
function transformToDeploymentOverview(apiDeployments: DeploymentApiItem[]): DeploymentOverview {
  const deployments = apiDeployments.map(transformDeploymentItem);
  const namespaceSet = new Set<string>();
  let totalReplicas = 0;
  let readyReplicas = 0;

  for (const api of apiDeployments) {
    if (api.summary.namespace) namespaceSet.add(api.summary.namespace);
    totalReplicas += api.summary.replicas;
    readyReplicas += api.summary.ready;
  }

  return {
    cards: {
      totalDeployments: deployments.length,
      namespaces: namespaceSet.size,
      totalReplicas,
      readyReplicas,
    },
    rows: deployments,
  };
}

// ============================================================
// 兼容旧接口
// ============================================================

/**
 * 获取 Deployment 概览（包含统计卡片和列表）
 */
export async function getDeploymentOverview(data: { ClusterID: string }) {
  const response = await getDeploymentList({ cluster_id: data.ClusterID });
  const deployments = response.data.data || [];
  const overview = transformToDeploymentOverview(deployments);

  return {
    ...response,
    data: {
      data: overview,
    },
  };
}
