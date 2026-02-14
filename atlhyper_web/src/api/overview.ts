/**
 * 集群概览 API
 *
 * 后端已返回 camelCase 响应，前端直接使用
 */

import { get } from "./request";
import type { ClusterOverview } from "@/types/overview";

// ============================================================
// API 方法
// ============================================================

/**
 * 获取集群概览
 * GET /api/v2/overview?cluster_id=xxx
 */
export async function getClusterOverview(params: { cluster_id: string }) {
  const response = await get<ClusterOverview>("/api/v2/overview", params);
  return {
    ...response,
    data: {
      data: response.data,
    },
  };
}

/** @deprecated 使用新参数格式 */
export function getClusterOverviewLegacy(data: { ClusterID: string }) {
  return getClusterOverview({ cluster_id: data.ClusterID });
}
