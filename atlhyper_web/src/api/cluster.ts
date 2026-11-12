/**
 * 集群信息 API
 *
 * 适配 Master V2 API
 */

import { get } from "./request";
import type { ClusterInfo } from "@/types/cluster";

// ============================================================
// Master V2 响应类型
// ============================================================

// 集群列表响应
interface ClusterListResponse {
  clusters: ClusterInfo[];
  total: number;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取集群列表
 * GET /api/v2/clusters
 */
export function getClusterList() {
  return get<ClusterListResponse>("/api/v2/clusters");
}

/**
 * 获取集群信息（兼容旧接口名称）
 * @deprecated 使用 getClusterList 替代
 */
export function getClusterInfo() {
  return getClusterList();
}
