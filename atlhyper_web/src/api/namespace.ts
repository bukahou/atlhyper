/**
 * Namespace API
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get, post } from "./request";
import type { NamespaceOverview, NamespaceDetail, ConfigMapDTO, SecretDTO, NamespaceItem } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

interface NamespaceListParams {
  cluster_id: string;
  limit?: number;
  offset?: number;
}

interface ConfigMapListParams {
  cluster_id: string;
  namespace: string;
  limit?: number;
  offset?: number;
}

// ============================================================
// 响应类型（后端返回扁平结构）
// ============================================================

interface NamespaceListResponse {
  message: string;
  data: NamespaceItem[];
  total: number;
}

interface ConfigMapListResponse {
  message: string;
  data: ConfigMapDTO[];
  total: number;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 Namespace 列表
 * GET /api/v2/namespaces?cluster_id=xxx
 */
export function getNamespaceList(params: NamespaceListParams) {
  return get<NamespaceListResponse>("/api/v2/namespaces", params);
}

/**
 * 获取 Namespace 详情
 * GET /api/v2/namespaces/{name}?cluster_id=xxx
 */
export async function getNamespaceDetail(data: { ClusterID: string; Namespace: string }) {
  return get<{ message: string; data: NamespaceDetail }>(
    `/api/v2/namespaces/${encodeURIComponent(data.Namespace)}`,
    { cluster_id: data.ClusterID }
  );
}

/**
 * 获取 ConfigMap 列表
 * GET /api/v2/configmaps?cluster_id=xxx&namespace=xxx
 */
export function getConfigMapList(params: ConfigMapListParams) {
  return get<ConfigMapListResponse>("/api/v2/configmaps", params);
}

// ============================================================
// 概览聚合（前端从扁平列表计算统计卡片）
// ============================================================

/**
 * 获取 Namespace 概览（包含统计卡片和列表）
 */
export async function getNamespaceOverview(data: { ClusterID: string }) {
  const response = await getNamespaceList({ cluster_id: data.ClusterID });
  const items = response.data.data || [];

  let activeCount = 0;
  let terminating = 0;
  let totalPods = 0;

  for (const item of items) {
    if (item.status === "Active") activeCount++;
    else if (item.status === "Terminating") terminating++;
    totalPods += item.podCount ?? 0;
  }

  const overview: NamespaceOverview = {
    cards: {
      totalNamespaces: items.length,
      activeCount,
      terminating,
      totalPods,
    },
    rows: items,
  };

  return {
    ...response,
    data: { data: overview },
  };
}

/** @deprecated 使用 getConfigMapList 替代 */
export function getConfigMaps(data: { ClusterID: string; Namespace: string }) {
  return getConfigMapList({ cluster_id: data.ClusterID, namespace: data.Namespace });
}

// ============================================================
// Secret 列表 (需要 Operator 权限)
// ============================================================

interface SecretListResponse {
  message: string;
  data: SecretDTO[];
  total: number;
}

/**
 * 获取 Secret 列表
 * GET /api/v2/secrets?cluster_id=xxx&namespace=xxx
 * 需要 Operator 权限 (role >= 2)
 */
export function getSecretList(params: { cluster_id: string; namespace: string }) {
  return get<SecretListResponse>("/api/v2/secrets", params);
}

/**
 * 获取 Secret 列表（简化参数）
 */
export function getSecrets(data: { ClusterID: string; Namespace: string }) {
  return getSecretList({ cluster_id: data.ClusterID, namespace: data.Namespace });
}

// ============================================================
// ConfigMap/Secret 数据获取 (需要 Operator 权限)
// ============================================================

interface ConfigMapDataResponse {
  message: string;
  data: string; // JSON 字符串，需要 parse
}

interface SecretDataResponse {
  message: string;
  data: string; // JSON 字符串，需要 parse
}

/**
 * 获取 ConfigMap 数据内容
 * POST /api/v2/ops/configmaps/data
 * 需要 Operator 权限 (role >= 2)
 */
export async function getConfigMapData(data: {
  ClusterID: string;
  Namespace: string;
  Name: string;
}): Promise<Record<string, string>> {
  const res = await post<ConfigMapDataResponse>("/api/v2/ops/configmaps/data", {
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    name: data.Name,
  });
  try {
    return JSON.parse(res.data.data || "{}");
  } catch {
    return {};
  }
}

/**
 * 获取 Secret 数据内容
 * POST /api/v2/ops/secrets/data
 * 需要 Operator 权限 (role >= 2)
 */
export async function getSecretData(data: {
  ClusterID: string;
  Namespace: string;
  Name: string;
}): Promise<Record<string, string>> {
  const res = await post<SecretDataResponse>("/api/v2/ops/secrets/data", {
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    name: data.Name,
  });
  try {
    return JSON.parse(res.data.data || "{}");
  } catch {
    return {};
  }
}
