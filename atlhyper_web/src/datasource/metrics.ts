/**
 * Metrics 数据源代理
 *
 * 根据中心配置自动切换 mock / api。约定同 datasource/cluster.ts：
 * - 读函数返回解包后的数据，mock 分支返回相同形态
 * - 读函数最后一个参数是可选的 AbortSignal，原样交给 api 层，用于取消请求
 *
 * 实时数据（summary + nodes + 硬件）用 observe/metrics 端点（ClickHouse），
 * 历史数据用 node-metrics 端点（SQLite，快速直读）。
 */

import { getDataSourceMode } from "@/config/data-source";
import * as mock from "@/mock/metrics";
import * as observe from "@/api/observe-metrics";
import * as nodeMetrics from "@/api/node-metrics";
import type { NodeMetrics, Point, Summary } from "@/types/node-metrics";
import type { HardwareHealth } from "@/types/hardware";
import type { FreshnessResponse } from "@/types/observe";

/** 集群节点指标快照：汇总 + 每个节点的当前读数 */
export interface ClusterNodeMetricsSnapshot {
  summary: Summary;
  nodes: NodeMetrics[];
}

/** 某个节点的历史时序，按指标分组：{ cpu: Point[], memory: Point[], ... } */
export type NodeMetricsHistory = Record<string, Point[]>;

export async function getClusterNodeMetrics(
  clusterId: string,
  signal?: AbortSignal,
): Promise<ClusterNodeMetricsSnapshot> {
  if (getDataSourceMode("metrics") === "mock") return mock.mockGetClusterNodeMetrics();

  // summary 与 nodes 来自同一份快照，并行取
  const [summaryResponse, nodesResponse] = await Promise.all([
    observe.getMetricsSummary(clusterId, signal),
    observe.getMetricsNodes(clusterId, signal),
  ]);
  return { summary: summaryResponse.data.data, nodes: nodesResponse.data.data ?? [] };
}

/** 硬件健康矩阵（阈值判定在后端完成） */
export async function getHardwareHealth(clusterId: string, signal?: AbortSignal): Promise<HardwareHealth> {
  if (getDataSourceMode("metrics") === "mock") return mock.mockGetHardwareHealth();
  return (await observe.getMetricsHardware(clusterId, signal)).data.data;
}

/**
 * 各信号的数据新鲜度。
 *
 * 没有演示数据 —— 演示模式下数据是编的，谈新鲜度没有意义，返回 null 让徽章显示「未知」。
 */
export async function getSignalFreshness(clusterId: string, signal?: AbortSignal): Promise<FreshnessResponse | null> {
  if (getDataSourceMode("metrics") === "mock") return null;
  return (await observe.getSignalFreshness(clusterId, signal)).data.data ?? null;
}

/** 单个节点最近 hours 小时的历史时序 */
export async function getNodeMetricsHistory(
  clusterId: string,
  nodeName: string,
  hours: number,
  signal?: AbortSignal,
): Promise<NodeMetricsHistory> {
  if (getDataSourceMode("metrics") === "mock") return mock.mockGetNodeMetricsHistory(nodeName, hours).data;
  return (await nodeMetrics.getNodeMetricsHistory(clusterId, nodeName, hours, signal)).data;
}
