/**
 * 总览页的取数描述
 *
 * 总览页有两份数据：集群概览，以及一张 SLO 摘要卡片。两份各自成一个查询 ——
 * SLO 取不到（集群没接 OTel）不该拖累概览，反过来也一样。
 */
import { queryOptions } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import { getClusterOverview } from "@/datasource/overview";
import { getSLODomains } from "@/datasource/slo";
import { clusterDataAvailable } from "./availability";
import { overviewKeys, sloKeys } from "./keys";
import { REFRESH } from "./refresh";

/** 总览页 SLO 卡片用的窗口：不指定，由后端取默认窗口。进键时用这个名字占位 */
const BACKEND_DEFAULT_SLO_WINDOW = "default";

/** 集群概览 */
export function clusterOverviewQuery(clusterId: string) {
  const mode = getDataSourceMode("overview");
  return queryOptions({
    queryKey: overviewKeys.cluster(clusterId, mode),
    queryFn: ({ signal }) => getClusterOverview(clusterId, signal),
    refetchInterval: REFRESH.fast,
    staleTime: REFRESH.fast,
    enabled: clusterDataAvailable(clusterId, mode),
  });
}

/**
 * 总览页的 SLO 摘要（域名清单 + 汇总）。
 *
 * 键挂在 SLO 域下：在 SLO 页改了目标，这张卡片也跟着重新取数。
 */
export function overviewSLODomainsQuery(clusterId: string) {
  const mode = getDataSourceMode("slo");
  return queryOptions({
    queryKey: sloKeys.domains(clusterId, mode, BACKEND_DEFAULT_SLO_WINDOW),
    queryFn: ({ signal }) => getSLODomains({ clusterId }, signal),
    refetchInterval: REFRESH.fast,
    staleTime: REFRESH.fast,
    enabled: clusterDataAvailable(clusterId, mode),
  });
}
