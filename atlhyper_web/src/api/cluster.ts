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

// 集群详情响应（包含完整快照）
interface ClusterDetailResponse {
  cluster_id: string;
  status: string;
  last_heartbeat: string;
  last_snapshot: string;
  snapshot: {
    cluster_id: string;
    timestamp: string;
    summary: {
      total_pods: number;
      running_pods: number;
      total_nodes: number;
      ready_nodes: number;
      total_deployments: number;
      ready_deployments: number;
      total_services: number;
      total_namespaces: number;
      warning_events: number;
      normal_events: number;
    };
    pods: unknown[];
    nodes: unknown[];
    deployments: unknown[];
    services: unknown[];
    namespaces: unknown[];
    events: unknown[];
  };
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
 * 获取集群详情
 * GET /api/v2/clusters/{cluster_id}
 */
export function getClusterDetail(clusterId: string) {
  return get<ClusterDetailResponse>(`/api/v2/clusters/${clusterId}`);
}

/**
 * 获取集群信息（兼容旧接口名称）
 * @deprecated 使用 getClusterList 替代
 */
export function getClusterInfo() {
  return getClusterList();
}
