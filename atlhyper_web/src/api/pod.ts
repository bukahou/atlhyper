/**
 * Pod API
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get, post } from "./request";
import type { PodOverview, PodDetail, PodItem } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

interface PodListParams {
  cluster_id: string;
  namespace?: string;
  node?: string;
  phase?: string;
  limit?: number;
  offset?: number;
}

// ============================================================
// 响应类型（后端返回扁平结构）
// ============================================================

interface PodListResponse {
  message: string;
  data: PodItem[];
  total: number;
}

interface PodDetailResponse {
  message: string;
  data: PodDetail;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 Pod 列表
 * GET /api/v2/pods?cluster_id=xxx&namespace=xxx
 */
export function getPodList(params: PodListParams) {
  return get<PodListResponse>("/api/v2/pods", params);
}

/**
 * 获取 Pod 详情
 * GET /api/v2/pods/{name}?cluster_id=xxx&namespace=xxx
 */
export async function getPodDetail(params: {
  ClusterID: string;
  Namespace: string;
  PodName: string;
}) {
  return get<{ message: string; data: PodDetail }>(
    `/api/v2/pods/${encodeURIComponent(params.PodName)}`,
    {
      cluster_id: params.ClusterID,
      namespace: params.Namespace,
    }
  );
}

/**
 * 获取 Pod 日志（需要 Operator 权限）
 * POST /api/v2/ops/pods/logs
 */
interface PodLogsResponse {
  message: string;
  data?: {
    logs: string;
  };
}

export function getPodLogs(data: {
  ClusterID: string;
  Namespace: string;
  Pod: string;
  Container?: string;
  TailLines?: number;
  TimeoutSeconds?: number;
}) {
  return post<PodLogsResponse>("/api/v2/ops/pods/logs", {
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    name: data.Pod,
    container: data.Container,
    tail_lines: data.TailLines,
  });
}

/**
 * 重启 Pod（需要 Operator 权限）
 * POST /api/v2/ops/pods/restart
 */
interface CommandResponse {
  message: string;
  command_id: string;
  status: string;
}

export function restartPod(data: {
  ClusterID: string;
  Namespace: string;
  Pod: string;
}) {
  return post<CommandResponse>("/api/v2/ops/pods/restart", {
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    name: data.Pod,
  });
}

// ============================================================
// 概览聚合（前端从扁平列表计算统计卡片）
// ============================================================

/**
 * 获取 Pod 概览（包含统计卡片和列表）
 */
export async function getPodOverview(data: { ClusterID: string }) {
  const response = await getPodList({ cluster_id: data.ClusterID });
  const pods = response.data.data || [];

  let running = 0;
  let pending = 0;
  let failed = 0;
  let unknown = 0;

  for (const pod of pods) {
    switch (pod.phase) {
      case "Running": running++; break;
      case "Pending": pending++; break;
      case "Failed": failed++; break;
      case "Succeeded": break;
      default: unknown++;
    }
  }

  const overview: PodOverview = {
    cards: { running, pending, failed, unknown },
    pods,
  };

  return {
    ...response,
    data: { data: overview },
  };
}
