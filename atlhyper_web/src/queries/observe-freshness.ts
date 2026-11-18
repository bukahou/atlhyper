/**
 * 信号新鲜度的取数描述
 *
 * 四个观测页（Metrics / APM / Logs / SLO）各看其中一个信号，但接口一次返回全部信号，
 * 所以共用同一份查询：同一个集群只请求一份，页面间切换直接用缓存。
 */
import { queryOptions } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import { getSignalFreshness } from "@/datasource/metrics";
import { clusterDataAvailable } from "./availability";
import { observeKeys } from "./keys";
import { REFRESH } from "./refresh";

/** 各信号的数据新鲜度。变化很慢，一分钟一次足够，不跟着页面数据的刷新频率走 */
export function signalFreshnessQuery(clusterId: string) {
  // 新鲜度没有自己的数据源开关，跟着 metrics 模块走（见 datasource/metrics.ts）
  const mode = getDataSourceMode("metrics");
  return queryOptions({
    queryKey: observeKeys.signalFreshness(clusterId, mode),
    queryFn: ({ signal }) => getSignalFreshness(clusterId, signal),
    refetchInterval: REFRESH.minute,
    staleTime: REFRESH.minute,
    enabled: clusterDataAvailable(clusterId, mode),
  });
}
