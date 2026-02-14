/**
 * Deployment API
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get, post } from "./request";
import type { DeploymentOverview, DeploymentDetail, DeploymentItem } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

interface DeploymentListParams {
  cluster_id: string;
  namespace?: string;
  limit?: number;
  offset?: number;
}

// ============================================================
// 响应类型（后端返回扁平结构）
// ============================================================

interface DeploymentListResponse {
  message: string;
  data: DeploymentItem[];
  total: number;
}

// ============================================================
// 操作请求类型
// ============================================================

interface CommandResponse {
  message: string;
  command_id: string;
  status: string;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 Deployment 列表
 * GET /api/v2/deployments?cluster_id=xxx&namespace=xxx
 */
export function getDeploymentList(params: DeploymentListParams) {
  return get<DeploymentListResponse>("/api/v2/deployments", params);
}

/**
 * 获取 Deployment 详情
 * GET /api/v2/deployments/{name}?cluster_id=xxx&namespace=xxx
 */
export async function getDeploymentDetail(data: { ClusterID: string; Namespace: string; Name: string }) {
  return get<{ message: string; data: DeploymentDetail }>(
    `/api/v2/deployments/${encodeURIComponent(data.Name)}`,
    {
      cluster_id: data.ClusterID,
      namespace: data.Namespace,
    }
  );
}

/**
 * Deployment 扩缩容（需要 Operator 权限）
 * POST /api/v2/ops/deployments/scale
 */
export function scaleDeployment(data: { ClusterID: string; Namespace: string; Name: string; Kind?: string; Replicas: number }) {
  return post<CommandResponse>("/api/v2/ops/deployments/scale", {
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    name: data.Name,
    replicas: data.Replicas,
  });
}

/**
 * Deployment 滚动重启（需要 Operator 权限）
 * POST /api/v2/ops/deployments/restart
 */
export function restartDeployment(data: { ClusterID: string; Namespace: string; Name: string }) {
  return post<CommandResponse>("/api/v2/ops/deployments/restart", {
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    name: data.Name,
  });
}

/**
 * Deployment 更新镜像（需要 Operator 权限）
 * POST /api/v2/ops/deployments/image
 */
export function updateDeploymentImage(data: { ClusterID: string; Namespace: string; Name: string; Kind?: string; ContainerName?: string; NewImage: string; OldImage?: string }) {
  return post<CommandResponse>("/api/v2/ops/deployments/image", {
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    name: data.Name,
    container: data.ContainerName || "",
    image: data.NewImage,
  });
}

// ============================================================
// 概览聚合（前端从扁平列表计算统计卡片）
// ============================================================

/**
 * 获取 Deployment 概览（包含统计卡片和列表）
 */
export async function getDeploymentOverview(data: { ClusterID: string }) {
  const response = await getDeploymentList({ cluster_id: data.ClusterID });
  const items = response.data.data || [];

  const namespaceSet = new Set<string>();
  let totalReplicas = 0;
  let readyReplicas = 0;

  for (const item of items) {
    if (item.namespace) namespaceSet.add(item.namespace);
    // replicas 格式: "2/3" → 解析 ready 和 total
    const parts = item.replicas.split("/");
    if (parts.length === 2) {
      readyReplicas += parseInt(parts[0], 10) || 0;
      totalReplicas += parseInt(parts[1], 10) || 0;
    }
  }

  const overview: DeploymentOverview = {
    cards: {
      totalDeployments: items.length,
      namespaces: namespaceSet.size,
      totalReplicas,
      readyReplicas,
    },
    rows: items,
  };

  return {
    ...response,
    data: { data: overview },
  };
}
