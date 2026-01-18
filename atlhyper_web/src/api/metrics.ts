import { post } from "./request";
import type { ClusterRequest } from "@/types/common";
import type { MetricsOverview, NodeMetricsDetail, MetricsDetailRequest } from "@/types/cluster";

interface MetricsOverviewResponse {
  message: string;
  data: MetricsOverview;
}

interface NodeMetricsDetailResponse {
  message: string;
  data: NodeMetricsDetail;
}

/**
 * 获取节点指标概览
 */
export function getMetricsOverview(data: ClusterRequest) {
  return post<MetricsOverviewResponse>("/uiapi/metrics/overview", data);
}

/**
 * 获取节点指标详情
 */
export function getNodeMetricsDetail(data: MetricsDetailRequest) {
  return post<NodeMetricsDetailResponse>("/uiapi/metrics/node/detail", data);
}
