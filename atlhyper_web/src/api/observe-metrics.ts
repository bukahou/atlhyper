/**
 * Metrics 信号域 API
 */

import { get } from "./request";
import type { ObserveResponse } from "./observe-common";
import type { NodeMetricsView, Summary } from "@/types/node-metrics";
import type { HardwareHealth } from "@/types/hardware";
import type { FreshnessResponse } from "@/types/observe";

/** 获取指标汇总 */
export function getMetricsSummary(clusterId: string, signal?: AbortSignal) {
  return get<ObserveResponse<Summary>>("/api/v2/observe/metrics/summary", { cluster_id: clusterId }, signal);
}

/** 获取所有节点指标列表（每项带后端判定的等级） */
export function getMetricsNodes(clusterId: string, signal?: AbortSignal) {
  return get<ObserveResponse<NodeMetricsView[]>>("/api/v2/observe/metrics/nodes", { cluster_id: clusterId }, signal);
}

/** 获取硬件健康矩阵（阈值判定已在后端完成） */
export function getMetricsHardware(clusterId: string, signal?: AbortSignal) {
  return get<ObserveResponse<HardwareHealth>>("/api/v2/observe/metrics/hardware", { cluster_id: clusterId }, signal);
}

/** 获取各信号的数据新鲜度（用于区分「没有流量」和「采集异常」） */
export function getSignalFreshness(clusterId: string, signal?: AbortSignal) {
  return get<ObserveResponse<FreshnessResponse>>("/api/v2/observe/freshness", { cluster_id: clusterId }, signal);
}
