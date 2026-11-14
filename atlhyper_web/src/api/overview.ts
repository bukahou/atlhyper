/**
 * 集群概览 API
 *
 * 后端已返回 camelCase 响应，前端直接使用
 */

import { get } from "./request";
import type { ClusterOverview } from "@/types/overview";

/**
 * 获取集群概览
 * GET /api/v2/overview?cluster_id=xxx
 */
export function getClusterOverview(params: { cluster_id: string }, signal?: AbortSignal) {
  return get<ClusterOverview>("/api/v2/overview", params, signal);
}
