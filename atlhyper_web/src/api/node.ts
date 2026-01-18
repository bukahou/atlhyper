/**
 * Node API
 *
 * 适配 Master V2 API（嵌套结构）
 */

import { get, post } from "./request";
import type { NodeOverview, NodeDetail, NodeItem, NodeCondition, NodeTaint } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

interface NodeListParams {
  cluster_id: string;
  status?: string;
  limit?: number;
  offset?: number;
}

// ============================================================
// 响应类型（后端返回嵌套结构）
// ============================================================

// 后端返回的 Node 格式（与 model_v2.Node 一致）
interface NodeApiItem {
  summary: {
    name: string;
    roles?: string[];
    ready: string; // "True", "False", "Unknown"
    schedulable: boolean;
    age: string;
    creationTime: string;
    badges?: string[];
    reason?: string;
    message?: string;
  };
  spec: {
    podCIDRs?: string[];
    providerID?: string;
    unschedulable?: boolean;
  };
  capacity: {
    cpu?: string;
    memory?: string;
    pods?: string;
    ephemeralStorage?: string;
  };
  allocatable: {
    cpu?: string;
    memory?: string;
    pods?: string;
    ephemeralStorage?: string;
  };
  addresses: {
    hostname?: string;
    internalIP?: string;
    externalIP?: string;
  };
  info: {
    osImage?: string;
    operatingSystem?: string;
    architecture?: string;
    kernelVersion?: string;
    containerRuntimeVersion?: string;
    kubeletVersion?: string;
    kubeProxyVersion?: string;
  };
  conditions?: Array<{
    type: string;
    status: string;
    reason?: string;
    message?: string;
    lastHeartbeatTime?: string;
    lastTransitionTime?: string;
  }>;
  taints?: Array<{
    key: string;
    value?: string;
    effect: string;
    timeAdded?: string;
  }>;
  labels?: Record<string, string>;
  metrics?: {
    cpu: {
      usage: string;
      allocatable?: string;
      capacity?: string;
      utilPct?: number;
    };
    memory: {
      usage: string;
      allocatable?: string;
      capacity?: string;
      utilPct?: number;
    };
    pods: {
      used: number;
      capacity: number;
      utilPct?: number;
    };
    pressure?: {
      memoryPressure?: boolean;
      diskPressure?: boolean;
      pidPressure?: boolean;
      networkUnavailable?: boolean;
    };
  };
}

interface NodeListResponse {
  message: string;
  data: NodeApiItem[];
  total: number;
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
 * 获取 Node 列表
 * GET /api/v2/nodes?cluster_id=xxx
 */
export function getNodeList(params: NodeListParams) {
  return get<NodeListResponse>("/api/v2/nodes", params);
}

/**
 * 获取 Node 详情
 * GET /api/v2/nodes/{name}?cluster_id=xxx
 */
export async function getNodeDetail(data: { ClusterID: string; NodeName: string }) {
  const response = await getNodeList({
    cluster_id: data.ClusterID,
  });

  const apiNodes = response.data.data || [];
  const target = apiNodes.find((n) => n.summary?.name === data.NodeName);

  if (!target) {
    throw new Error("Node not found");
  }

  // 转换为前端详情格式
  const detail = transformToNodeDetail(target);

  return {
    ...response,
    data: {
      data: detail,
    },
  };
}

/**
 * 封锁 Node（需要 Operator 权限）
 * POST /api/v2/ops/nodes/cordon
 */
export function cordonNode(data: { ClusterID: string; Node: string }) {
  return post<CommandResponse>("/api/v2/ops/nodes/cordon", {
    cluster_id: data.ClusterID,
    name: data.Node,
  });
}

/**
 * 解封 Node（需要 Operator 权限）
 * POST /api/v2/ops/nodes/uncordon
 */
export function uncordonNode(data: { ClusterID: string; Node: string }) {
  return post<CommandResponse>("/api/v2/ops/nodes/uncordon", {
    cluster_id: data.ClusterID,
    name: data.Node,
  });
}

// ============================================================
// 数据转换
// ============================================================

/**
 * 解析 Kubernetes 内存字符串为 GiB
 */
function parseMemoryToGiB(memStr?: string): number {
  if (!memStr) return 0;

  const match = memStr.match(/^(\d+(?:\.\d+)?)(Ki|Mi|Gi|Ti|K|M|G|T)?$/i);
  if (!match) return 0;

  const value = parseFloat(match[1]);
  const unit = (match[2] || "").toLowerCase();

  switch (unit) {
    case "ki": return value / (1024 * 1024);
    case "mi": return value / 1024;
    case "gi": return value;
    case "ti": return value * 1024;
    case "k": return value / (1000 * 1000 * 1000) * 1024;
    case "m": return value / (1000 * 1000) * 1024;
    case "g": return value * 1.073741824;
    case "t": return value * 1024 * 1.073741824;
    default: return value / (1024 * 1024 * 1024);
  }
}

/**
 * 解析 CPU 字符串为核心数
 */
function parseCPUCores(cpuStr?: string): number {
  if (!cpuStr) return 0;

  const num = parseFloat(cpuStr);
  if (!isNaN(num) && !cpuStr.includes("m") && !cpuStr.includes("n")) {
    return num;
  }

  if (cpuStr.endsWith("m")) {
    return parseFloat(cpuStr) / 1000;
  }

  if (cpuStr.endsWith("n")) {
    return parseFloat(cpuStr) / 1000000000;
  }

  return num || 0;
}

/**
 * 将后端 Node 转换为列表项格式
 */
function transformNodeItem(apiNode: NodeApiItem): NodeItem {
  const cpuCores = parseCPUCores(apiNode.capacity?.cpu);
  const memoryGiB = parseMemoryToGiB(apiNode.capacity?.memory);

  return {
    name: apiNode.summary?.name || "",
    ready: apiNode.summary?.ready === "True",
    internalIP: apiNode.addresses?.internalIP || "",
    osImage: apiNode.info?.osImage || "",
    architecture: apiNode.info?.architecture || "",
    cpuCores,
    memoryGiB,
    schedulable: apiNode.summary?.schedulable ?? true,
  };
}

/**
 * 将后端 Node 转换为详情格式
 */
function transformToNodeDetail(apiNode: NodeApiItem): NodeDetail {
  const cpuCapacity = parseCPUCores(apiNode.capacity?.cpu);
  const cpuAllocatable = parseCPUCores(apiNode.allocatable?.cpu);
  const memCapacity = parseMemoryToGiB(apiNode.capacity?.memory);
  const memAllocatable = parseMemoryToGiB(apiNode.allocatable?.memory);
  const ephemeralStorage = parseMemoryToGiB(apiNode.capacity?.ephemeralStorage);
  const podsCapacity = parseInt(apiNode.capacity?.pods || "0", 10);
  const podsAllocatable = parseInt(apiNode.allocatable?.pods || "0", 10);

  // 从 metrics 获取使用量
  const cpuUsage = parseCPUCores(apiNode.metrics?.cpu?.usage);
  const memUsage = parseMemoryToGiB(apiNode.metrics?.memory?.usage);
  const podsUsed = apiNode.metrics?.pods?.used || 0;

  // 计算使用率
  const cpuUtilPct = cpuAllocatable > 0 ? (cpuUsage / cpuAllocatable) * 100 : 0;
  const memUtilPct = memAllocatable > 0 ? (memUsage / memAllocatable) * 100 : 0;
  const podsUtilPct = podsAllocatable > 0 ? (podsUsed / podsAllocatable) * 100 : 0;

  // 转换 conditions
  const conditions: NodeCondition[] = (apiNode.conditions || []).map((c) => ({
    type: c.type,
    status: c.status,
    reason: c.reason,
    message: c.message,
    heartbeat: c.lastHeartbeatTime,
    changedAt: c.lastTransitionTime,
  }));

  // 转换 taints
  const taints: NodeTaint[] = (apiNode.taints || []).map((t) => ({
    key: t.key,
    value: t.value,
    effect: t.effect,
  }));

  return {
    // 基本信息
    name: apiNode.summary?.name || "",
    roles: apiNode.summary?.roles,
    ready: apiNode.summary?.ready === "True",
    schedulable: apiNode.summary?.schedulable ?? true,
    age: apiNode.summary?.age,
    createdAt: apiNode.summary?.creationTime || "",

    // 地址与系统
    hostname: apiNode.addresses?.hostname,
    internalIP: apiNode.addresses?.internalIP,
    externalIP: apiNode.addresses?.externalIP,
    osImage: apiNode.info?.osImage,
    os: apiNode.info?.operatingSystem,
    architecture: apiNode.info?.architecture,
    kernel: apiNode.info?.kernelVersion,
    cri: apiNode.info?.containerRuntimeVersion,
    kubelet: apiNode.info?.kubeletVersion,
    kubeProxy: apiNode.info?.kubeProxyVersion,

    // 资源容量
    cpuCapacityCores: cpuCapacity,
    cpuAllocatableCores: cpuAllocatable,
    memCapacityGiB: memCapacity,
    memAllocatableGiB: memAllocatable,
    podsCapacity,
    podsAllocatable,
    ephemeralStorageGiB: ephemeralStorage,

    // 当前指标
    cpuUsageCores: cpuUsage,
    cpuUtilPct,
    memUsageGiB: memUsage,
    memUtilPct,
    podsUsed,
    podsUtilPct,

    // 压力状态
    pressureMemory: apiNode.metrics?.pressure?.memoryPressure,
    pressureDisk: apiNode.metrics?.pressure?.diskPressure,
    pressurePID: apiNode.metrics?.pressure?.pidPressure,
    networkUnavailable: apiNode.metrics?.pressure?.networkUnavailable,

    // 调度相关
    podCIDRs: apiNode.spec?.podCIDRs,
    providerID: apiNode.spec?.providerID,

    // 条件/污点/标签
    conditions,
    taints,
    labels: apiNode.labels,

    // 诊断
    badges: apiNode.summary?.badges,
    reason: apiNode.summary?.reason,
    message: apiNode.summary?.message,
  };
}

/**
 * 将 Node 列表转换为 NodeOverview 格式
 */
function transformToNodeOverview(apiNodes: NodeApiItem[]): NodeOverview {
  const nodes = apiNodes.map(transformNodeItem);

  let readyNodes = 0;
  let totalCPU = 0;
  let totalMemoryGiB = 0;

  for (const n of nodes) {
    if (n.ready) readyNodes++;
    totalCPU += n.cpuCores || 0;
    totalMemoryGiB += n.memoryGiB || 0;
  }

  return {
    cards: {
      totalNodes: nodes.length,
      readyNodes,
      totalCPU,
      totalMemoryGiB,
    },
    rows: nodes,
  };
}

// ============================================================
// 兼容旧接口
// ============================================================

/**
 * 获取 Node 概览（包含统计卡片和列表）
 */
export async function getNodeOverview(data: { ClusterID: string }) {
  const response = await getNodeList({ cluster_id: data.ClusterID });
  const nodes = response.data.data || [];
  const overview = transformToNodeOverview(nodes);

  return {
    ...response,
    data: {
      data: overview,
    },
  };
}
