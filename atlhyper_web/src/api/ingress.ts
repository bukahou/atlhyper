/**
 * Ingress API
 *
 * 后端已完成 model_v2 → model 扁平化转换（含行展开），前端直接使用
 */

import { get } from "./request";
import type { IngressOverview, IngressDetail, IngressItem } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

interface IngressListParams {
  cluster_id: string;
  namespace?: string;
  limit?: number;
  offset?: number;
}

// ============================================================
// 响应类型（后端返回已展开的扁平行）
// ============================================================

interface IngressListResponse {
  message: string;
  data: IngressItem[];
  total: number;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 Ingress 列表（已按 host×path 展开为多行）
 * GET /api/v2/ingresses?cluster_id=xxx&namespace=xxx
 */
export function getIngressList(params: IngressListParams) {
  return get<IngressListResponse>("/api/v2/ingresses", params);
}

/**
 * 获取 Ingress 详情
 * GET /api/v2/ingresses/{name}?cluster_id=xxx&namespace=xxx
 */
export async function getIngressDetail(data: { ClusterID: string; Namespace: string; Name: string }) {
  return get<{ message: string; data: IngressDetail }>(
    `/api/v2/ingresses/${encodeURIComponent(data.Name)}`,
    {
      cluster_id: data.ClusterID,
      namespace: data.Namespace,
    }
  );
}

// ============================================================
// 概览聚合（前端从扁平行列表计算统计卡片）
// ============================================================

/**
 * 获取 Ingress 概览（包含统计卡片和列表）
 */
export async function getIngressOverview(data: { ClusterID: string }) {
  const response = await getIngressList({ cluster_id: data.ClusterID });
  const rows = response.data.data || [];

  const hostsSet = new Set<string>();
  let tlsCerts = 0;
  let totalPaths = 0;
  const ingressNames = new Set<string>();

  for (const row of rows) {
    if (row.host && row.host !== "*") hostsSet.add(row.host);
    if (row.tls) tlsCerts++;
    if (row.path) totalPaths++;
    ingressNames.add(row.name);
  }

  const overview: IngressOverview = {
    cards: {
      totalIngresses: ingressNames.size,
      usedHosts: hostsSet.size,
      tlsCerts,
      totalPaths,
    },
    rows,
  };

  return {
    ...response,
    data: { data: overview },
  };
}
