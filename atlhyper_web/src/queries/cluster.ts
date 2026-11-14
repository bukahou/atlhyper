/**
 * 集群资源的取数描述
 *
 * 每个工厂返回一份 queryOptions：键、取数函数、轮询间隔、启用条件。
 * 没有选定集群（clusterId 为空）时一律不启用。
 */
import { queryOptions } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import { getEventOverview } from "@/datasource/cluster";
import { clusterKeys } from "./keys";
import { REFRESH } from "./refresh";

/** 集群事件概览（实时） */
export function eventOverviewQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.events(clusterId, getDataSourceMode("event")),
    queryFn: ({ signal }) => getEventOverview({ ClusterID: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    enabled: clusterId !== "",
  });
}
