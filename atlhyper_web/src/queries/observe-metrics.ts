/**
 * 指标页的取数描述
 *
 * - 快照（汇总 + 各节点当前读数）与硬件健康矩阵：两个独立的查询并行取，硬件失败不拖垮快照
 * - 节点历史：每个「节点 + 时间范围」一个查询，各自缓存。
 *   集群趋势图（top 节点）与节点展开面板用同一个工厂，同一个节点只请求一份
 */
import { queryOptions } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import { getClusterNodeMetrics, getHardwareHealth, getNodeMetricsHistory } from "@/datasource/metrics";
import { clusterDataAvailable } from "./availability";
import { metricsKeys } from "./keys";
import { REFRESH } from "./refresh";

/** 集群节点指标快照（实时） */
export function metricsSnapshotQuery(clusterId: string) {
  const mode = getDataSourceMode("metrics");
  return queryOptions({
    queryKey: metricsKeys.snapshot(clusterId, mode),
    queryFn: ({ signal }) => getClusterNodeMetrics(clusterId, signal),
    refetchInterval: REFRESH.fast,
    staleTime: REFRESH.fast,
    enabled: clusterDataAvailable(clusterId, mode),
  });
}

/** 硬件健康矩阵（实时；阈值判定在后端完成） */
export function hardwareHealthQuery(clusterId: string) {
  const mode = getDataSourceMode("metrics");
  return queryOptions({
    queryKey: metricsKeys.hardware(clusterId, mode),
    queryFn: ({ signal }) => getHardwareHealth(clusterId, signal),
    refetchInterval: REFRESH.fast,
    staleTime: REFRESH.fast,
    enabled: clusterDataAvailable(clusterId, mode),
  });
}

/**
 * 单个节点最近 hours 小时的历史时序。
 *
 * 不跟着快照的 10 秒轮询走：趋势点是分钟级的，一分钟刷新一次就看得到新点。
 * 快照每刷新一轮，top 节点的名单可能变，但名单里没变的节点键不变，不会重新请求。
 */
export function nodeMetricsHistoryQuery(clusterId: string, nodeName: string, hours: number) {
  const mode = getDataSourceMode("metrics");
  return queryOptions({
    queryKey: metricsKeys.nodeHistory(clusterId, mode, nodeName, hours),
    queryFn: ({ signal }) => getNodeMetricsHistory(clusterId, nodeName, hours, signal),
    refetchInterval: REFRESH.minute,
    staleTime: REFRESH.minute,
    enabled: clusterDataAvailable(clusterId, mode) && nodeName !== "",
  });
}
