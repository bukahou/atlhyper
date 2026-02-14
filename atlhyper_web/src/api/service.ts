/**
 * Service API
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get } from "./request";
import type { ServiceOverview, ServiceDetail, ServiceItem } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

interface ServiceListParams {
  cluster_id: string;
  namespace?: string;
  limit?: number;
  offset?: number;
}

// ============================================================
// 响应类型（后端返回扁平结构）
// ============================================================

interface ServiceListResponse {
  message: string;
  data: ServiceItem[];
  total: number;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 Service 列表
 * GET /api/v2/services?cluster_id=xxx&namespace=xxx
 */
export function getServiceList(params: ServiceListParams) {
  return get<ServiceListResponse>("/api/v2/services", params);
}

/**
 * 获取 Service 详情
 * GET /api/v2/services/{name}?cluster_id=xxx&namespace=xxx
 */
export async function getServiceDetail(data: { ClusterID: string; Namespace: string; Name: string }) {
  return get<{ message: string; data: ServiceDetail }>(
    `/api/v2/services/${encodeURIComponent(data.Name)}`,
    {
      cluster_id: data.ClusterID,
      namespace: data.Namespace,
    }
  );
}

// ============================================================
// 概览聚合（前端从扁平列表计算统计卡片）
// ============================================================

/**
 * 获取 Service 概览（包含统计卡片和列表）
 */
export async function getServiceOverview(data: { ClusterID: string }) {
  const response = await getServiceList({ cluster_id: data.ClusterID });
  const items = response.data.data || [];

  let externalServices = 0;
  let internalServices = 0;
  let headlessServices = 0;

  for (const item of items) {
    const type = item.type?.toLowerCase() || "";
    const clusterIP = item.clusterIP || "";

    if (type === "loadbalancer" || type === "nodeport") {
      externalServices++;
    } else if (type === "clusterip" && clusterIP === "None") {
      headlessServices++;
    } else {
      internalServices++;
    }
  }

  const overview: ServiceOverview = {
    cards: {
      totalServices: items.length,
      externalServices,
      internalServices,
      headlessServices,
    },
    rows: items,
  };

  return {
    ...response,
    data: { data: overview },
  };
}
