/**
 * Node API
 *
 * 后端已完成 model_v2 → model 扁平化转换（含 CPU/内存单位转换），前端直接使用
 */

import { get, post } from "./request";
import type { NodeOverview, NodeDetail, NodeItem } from "@/types/cluster";

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
// 响应类型（后端返回扁平结构，单位已转换）
// ============================================================

interface NodeListResponse {
  message: string;
  data: NodeItem[];
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
  return get<{ message: string; data: NodeDetail }>(
    `/api/v2/nodes/${encodeURIComponent(data.NodeName)}`,
    { cluster_id: data.ClusterID }
  );
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
// 概览聚合（前端从扁平列表计算统计卡片）
// ============================================================

/**
 * 获取 Node 概览（包含统计卡片和列表）
 */
export async function getNodeOverview(data: { ClusterID: string }) {
  const response = await getNodeList({ cluster_id: data.ClusterID });
  const nodes = response.data.data || [];

  let readyNodes = 0;
  let totalCPU = 0;
  let totalMemoryGiB = 0;

  for (const n of nodes) {
    if (n.ready) readyNodes++;
    totalCPU += n.cpuCores || 0;
    totalMemoryGiB += n.memoryGiB || 0;
  }

  const overview: NodeOverview = {
    cards: { totalNodes: nodes.length, readyNodes, totalCPU, totalMemoryGiB },
    rows: nodes,
  };

  return {
    ...response,
    data: { data: overview },
  };
}
