/**
 * SLO 页的取数描述
 *
 * window 是 SLO 的预聚合窗口（1h / 6h / 24h / 3d / 7d），由页面把全局时间范围贴合后传进来
 * （见 hooks/useObserveTimeRange 的 sloWindows）。集群由调用方用 useCurrentCluster 得到。
 */
import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import {
  getSLODomainHistory,
  getSLODomains,
  getSLOLatencyDistribution,
  upsertSLOTarget,
} from "@/datasource/slo";
import { clusterDataAvailable } from "./availability";
import { sloKeys } from "./keys";
import { REFRESH } from "./refresh";

/** 域名 SLO 清单 + 汇总 */
export function sloDomainsQuery(clusterId: string, window: string) {
  const mode = getDataSourceMode("slo");
  return queryOptions({
    queryKey: sloKeys.domains(clusterId, mode, window),
    queryFn: ({ signal }) => getSLODomains({ clusterId, timeRange: window }, signal),
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
    enabled: clusterDataAvailable(clusterId, mode),
  });
}

/** 某个域名的 SLO 历史点（展开行的「预算」标签页用；只在标签页打开时取） */
export function sloDomainHistoryQuery(clusterId: string, host: string, window: string) {
  const mode = getDataSourceMode("slo");
  return queryOptions({
    queryKey: sloKeys.domainHistory(clusterId, mode, host, window),
    queryFn: async ({ signal }) =>
      (await getSLODomainHistory({ clusterId, host, timeRange: window }, signal)).history ?? [],
    enabled: clusterDataAvailable(clusterId, mode) && host !== "",
  });
}

/** 某个域名的延迟分布（展开行的「延迟」标签页用；只在标签页打开时取） */
export function sloLatencyDistributionQuery(clusterId: string, domain: string, window: string) {
  const mode = getDataSourceMode("slo");
  return queryOptions({
    queryKey: sloKeys.latency(clusterId, mode, domain, window),
    queryFn: ({ signal }) => getSLOLatencyDistribution({ clusterId, domain, timeRange: window }, signal),
    enabled: clusterDataAvailable(clusterId, mode) && domain !== "",
  });
}

/** 一个域名的 SLO 目标（滚动窗口 + 可用率 + P95） */
export interface SLOTargetChange {
  host: string;
  windowDays: number;
  availabilityTarget: number;
  p95LatencyTarget: number;
}

/**
 * 保存域名的 SLO 目标。
 *
 * 目标变了，错误预算、燃烧率、状态都跟着变，所以成功后让全部 SLO 查询重新取数。
 * 不等它们取完：目标已经存好，保存按钮不必为了刷新继续转圈。
 */
export function upsertSLOTargetMutation(clusterId: string) {
  return mutationOptions({
    mutationFn: (target: SLOTargetChange) => upsertSLOTarget({ clusterId, ...target }),
    onSuccess: (_saved, _target, _onMutateResult, context) => {
      void context.client.invalidateQueries({ queryKey: sloKeys.all });
    },
  });
}
