/**
 * 节点硬件指标 API
 *
 * 后端已返回 camelCase 响应，前端直接使用
 */

import { get } from "./request";
import type { NodeMetricsSnapshot, MetricsDataPoint } from "@/types/node-metrics";

// ============================================================================
// API 响应类型（与后端 camelCase 对齐）
// ============================================================================

export interface ClusterMetricsSummary {
  totalNodes: number;
  onlineNodes: number;
  offlineNodes: number;
  avgCPUUsage: number;
  avgMemoryUsage: number;
  avgDiskUsage: number;
  maxCPUUsage: number;
  maxMemoryUsage: number;
  maxDiskUsage: number;
  avgCPUTemp: number;
  maxCPUTemp: number;
  totalMemory: number;
  usedMemory: number;
  totalDisk: number;
  usedDisk: number;
  totalNetworkRx: number;
  totalNetworkTx: number;
}

export interface ClusterNodeMetricsResult {
  summary: ClusterMetricsSummary;
  nodes: NodeMetricsSnapshot[];
}

export interface NodeMetricsHistoryResult {
  nodeName: string;
  start: Date;
  end: Date;
  data: MetricsDataPoint[];
}

// ============================================================================
// API 函数
// ============================================================================

/**
 * 获取集群所有节点指标（含汇总）
 * @param clusterId 集群 ID
 */
export async function getClusterNodeMetrics(clusterId: string): Promise<ClusterNodeMetricsResult> {
  const response = await get<ClusterNodeMetricsResult>("/api/v2/node-metrics", {
    cluster_id: clusterId,
  });
  return response.data;
}

/**
 * 获取单节点详情
 * @param clusterId 集群 ID
 * @param nodeName 节点名称
 */
export async function getNodeMetricsDetail(
  clusterId: string,
  nodeName: string
): Promise<NodeMetricsSnapshot> {
  const response = await get<NodeMetricsSnapshot>(
    `/api/v2/node-metrics/${encodeURIComponent(nodeName)}`,
    { cluster_id: clusterId }
  );
  return response.data;
}

/**
 * 获取节点历史数据
 * @param clusterId 集群 ID
 * @param nodeName 节点名称
 * @param hours 时间范围（小时），默认 24
 */
export async function getNodeMetricsHistory(
  clusterId: string,
  nodeName: string,
  hours: number = 24
): Promise<NodeMetricsHistoryResult> {
  const response = await get<{ nodeName: string; start: string; end: string; data: MetricsDataPoint[] }>(
    `/api/v2/node-metrics/${encodeURIComponent(nodeName)}/history`,
    { cluster_id: clusterId, hours }
  );

  const data = response.data;
  return {
    nodeName: data.nodeName,
    start: new Date(data.start),
    end: new Date(data.end),
    data: data.data || [],
  };
}
