/**
 * Namespace API
 *
 * 适配 Master V2 API（嵌套结构）
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
// 后端返回类型（Master V2 API 嵌套结构）
// ============================================================

// 后端 ResourceQuota
interface ResourceQuotaApi {
  name: string;
  namespace: string;
  createdAt: string;
  age: string;
  scopes?: string[];
  hard?: Record<string, string>;
  used?: Record<string, string>;
}

// 后端 LimitRangeItem
interface LimitRangeItemApi {
  type: string;
  default?: Record<string, string>;
  defaultRequest?: Record<string, string>;
  max?: Record<string, string>;
  min?: Record<string, string>;
  maxLimitRequestRatio?: Record<string, string>;
}

// 后端 LimitRange
interface LimitRangeApi {
  name: string;
  namespace: string;
  createdAt: string;
  age: string;
  items: LimitRangeItemApi[];
}

// 后端 Namespace 资源统计
interface NamespaceResourcesApi {
  pods: number;
  podsRunning: number;
  podsPending: number;
  podsFailed: number;
  podsSucceeded: number;
  deployments: number;
  statefulSets: number;
  daemonSets: number;
  replicaSets: number;
  jobs: number;
  cronJobs: number;
  services: number;
  ingresses: number;
  networkPolicies: number;
  configMaps: number;
  secrets: number;
  serviceAccounts: number;
  pvcs: number;
}

// 后端 Namespace 结构（嵌套）
interface NamespaceApiItem {
  summary: {
    name: string;
    createdAt: string;
    age: string;
  };
  status: {
    phase: string;
  };
  resources: NamespaceResourcesApi;
  quotas?: ResourceQuotaApi[];
  limitRanges?: LimitRangeApi[];
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

interface NamespaceListResponse {
  message: string;
  data: NamespaceApiItem[];
  total: number;
}

interface NamespaceDetailResponse {
  message: string;
  data: NamespaceDetail;
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
  const response = await getNamespaceList({
    cluster_id: data.ClusterID,
  });

  const apiNamespaces = response.data.data || [];
  const target = apiNamespaces.find((ns) => ns.summary.name === data.Namespace);

  if (!target) {
    throw new Error("Namespace not found");
  }

  // 转换为前端格式
  const detail = transformNamespaceToDetail(target);

  return {
    ...response,
    data: {
      data: detail,
    },
  };
}

/**
 * 获取 ConfigMap 列表
 * GET /api/v2/configmaps?cluster_id=xxx&namespace=xxx
 */
export function getConfigMapList(params: ConfigMapListParams) {
  return get<ConfigMapListResponse>("/api/v2/configmaps", params);
}

// ============================================================
// 数据转换
// ============================================================

/**
 * 将后端 Namespace 转换为前端 NamespaceItem 格式（列表用）
 */
function transformNamespaceItem(apiItem: NamespaceApiItem): NamespaceItem {
  const labelCount = apiItem.labels ? Object.keys(apiItem.labels).length : 0;
  const annotationCount = apiItem.annotations ? Object.keys(apiItem.annotations).length : 0;

  return {
    name: apiItem.summary.name || "",
    status: apiItem.status.phase || "Unknown",
    podCount: apiItem.resources?.pods ?? 0,
    labelCount,
    annotationCount,
    createdAt: apiItem.summary.createdAt || "",
  };
}

/**
 * 将后端 Namespace 转换为前端 NamespaceDetail 格式（详情用）
 */
function transformNamespaceToDetail(apiItem: NamespaceApiItem): NamespaceDetail {
  const labelCount = apiItem.labels ? Object.keys(apiItem.labels).length : 0;
  const annotationCount = apiItem.annotations ? Object.keys(apiItem.annotations).length : 0;
  const res = apiItem.resources || {} as NamespaceResourcesApi;

  return {
    // 基本信息
    name: apiItem.summary.name,
    phase: apiItem.status.phase,
    createdAt: apiItem.summary.createdAt,
    age: apiItem.summary.age,

    // 标签和注解
    labels: apiItem.labels,
    annotations: apiItem.annotations,
    labelCount,
    annotationCount,

    // 资源计数
    pods: res.pods ?? 0,
    podsRunning: res.podsRunning ?? 0,
    podsPending: res.podsPending ?? 0,
    podsFailed: res.podsFailed ?? 0,
    podsSucceeded: res.podsSucceeded ?? 0,
    deployments: res.deployments ?? 0,
    statefulSets: res.statefulSets ?? 0,
    daemonSets: res.daemonSets ?? 0,
    jobs: res.jobs ?? 0,
    cronJobs: res.cronJobs ?? 0,
    services: res.services ?? 0,
    ingresses: res.ingresses ?? 0,
    configMaps: res.configMaps ?? 0,
    secrets: res.secrets ?? 0,
    persistentVolumeClaims: res.pvcs ?? 0,
    networkPolicies: res.networkPolicies ?? 0,
    serviceAccounts: res.serviceAccounts ?? 0,

    // 配额和限制
    quotas: apiItem.quotas?.map((q) => ({
      name: q.name,
      scopes: q.scopes,
      hard: q.hard,
      used: q.used,
    })),
    limitRanges: apiItem.limitRanges?.map((lr) => ({
      name: lr.name,
      items: lr.items.map((item) => ({
        type: item.type,
        default: item.default,
        defaultRequest: item.defaultRequest,
        max: item.max,
        min: item.min,
        maxLimitRequestRatio: item.maxLimitRequestRatio,
      })),
    })),
  };
}

/**
 * 将 Namespace 列表转换为 NamespaceOverview 格式
 */
function transformToNamespaceOverview(apiNamespaces: NamespaceApiItem[]): NamespaceOverview {
  const namespaces = apiNamespaces.map(transformNamespaceItem);
  let activeCount = 0;
  let terminating = 0;
  let totalPods = 0;

  for (const apiNs of apiNamespaces) {
    if (apiNs.status.phase === "Active") {
      activeCount++;
    } else if (apiNs.status.phase === "Terminating") {
      terminating++;
    }
    totalPods += apiNs.resources?.pods ?? 0;
  }

  return {
    cards: {
      totalNamespaces: namespaces.length,
      activeCount,
      terminating,
      totalPods,
    },
    rows: namespaces,
  };
}

// ============================================================
// 兼容旧接口
// ============================================================

/**
 * 获取 Namespace 概览（包含统计卡片和列表）
 */
export async function getNamespaceOverview(data: { ClusterID: string }) {
  const response = await getNamespaceList({ cluster_id: data.ClusterID });
  const namespaces = response.data.data || [];
  const overview = transformToNamespaceOverview(namespaces);

  return {
    ...response,
    data: {
      data: overview,
    },
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
  // 解析 JSON 字符串
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
  // 解析 JSON 字符串
  try {
    return JSON.parse(res.data.data || "{}");
  } catch {
    return {};
  }
}
