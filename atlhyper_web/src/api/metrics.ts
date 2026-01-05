import { post } from "./request";
import type { ClusterRequest } from "@/types/common";
import type { MetricsOverview, NodeMetricsDetail, MetricsDetailRequest } from "@/types/cluster";

/**
 * 获取节点指标概览
 */
export function getMetricsOverview(data: ClusterRequest) {
  return post<MetricsOverview, ClusterRequest>("/uiapi/metrics/overview", data);
}

/**
 * 获取节点指标详情
 */
export function getNodeMetricsDetail(data: MetricsDetailRequest) {
  return post<NodeMetricsDetail, MetricsDetailRequest>("/uiapi/metrics/node/detail", data);
}
