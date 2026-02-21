/**
 * Metrics 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 */

import { getDataSourceMode } from "@/config/data-source";
import * as mock from "@/mock/metrics";
import * as api from "@/api/node-metrics";

export type { ClusterMetricsSummary } from "@/api/node-metrics";
export type { MockClusterNodeMetricsResult, MockNodeMetricsHistoryResult } from "@/mock/metrics";

export async function getClusterNodeMetrics(clusterId: string) {
  if (getDataSourceMode("metrics") === "mock") return mock.mockGetClusterNodeMetrics();
  return api.getClusterNodeMetrics(clusterId);
}

export async function getNodeMetricsHistory(clusterId: string, nodeName: string, hours?: number) {
  if (getDataSourceMode("metrics") === "mock") return mock.mockGetNodeMetricsHistory(nodeName, hours);
  return api.getNodeMetricsHistory(clusterId, nodeName, hours);
}
