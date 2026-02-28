/**
 * Metrics 信号域 API
 */

import { get } from "./request";
import type { ObserveResponse } from "./observe-common";
import type { NodeMetrics, Summary, Point } from "@/types/node-metrics";

/** 获取指标汇总 */
export function getMetricsSummary(clusterId: string) {
  return get<ObserveResponse<Summary>>("/api/v2/observe/metrics/summary", {
    cluster_id: clusterId,
  });
}

/** 获取所有节点指标列表 */
export function getMetricsNodes(clusterId: string) {
  return get<ObserveResponse<NodeMetrics[]>>("/api/v2/observe/metrics/nodes", {
    cluster_id: clusterId,
  });
}

/** 获取单节点指标 */
export function getMetricsNode(clusterId: string, nodeName: string) {
  return get<ObserveResponse<NodeMetrics>>(
    `/api/v2/observe/metrics/nodes/${encodeURIComponent(nodeName)}`,
    { cluster_id: clusterId }
  );
}

/** 获取节点时序数据 */
export function getMetricsNodeSeries(clusterId: string, nodeName: string, metric: string, minutes?: number) {
  return get<ObserveResponse<Point[]>>(
    `/api/v2/observe/metrics/nodes/${encodeURIComponent(nodeName)}/series`,
    { cluster_id: clusterId, metric, ...(minutes ? { minutes } : {}) }
  );
}
