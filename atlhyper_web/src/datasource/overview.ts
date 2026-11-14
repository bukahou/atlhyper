/**
 * Overview 数据源代理
 *
 * 根据中心配置自动切换 mock / api。约定同 datasource/cluster.ts：
 * - 读函数返回解包后的数据，mock 分支返回相同形态
 * - 读函数最后一个参数是可选的 AbortSignal，原样交给 api 层，用于取消请求
 */

import { getDataSourceMode } from "@/config/data-source";
import * as mock from "@/mock/overview";
import * as api from "@/api/overview";
import type { ClusterOverview } from "@/types/overview";

/** 集群概览（健康卡片 / 工作负载 / 告警 / 节点用量） */
export async function getClusterOverview(clusterId: string, signal?: AbortSignal): Promise<ClusterOverview> {
  if (getDataSourceMode("overview") === "mock") return mock.mockGetClusterOverview();
  return (await api.getClusterOverview({ cluster_id: clusterId }, signal)).data;
}
