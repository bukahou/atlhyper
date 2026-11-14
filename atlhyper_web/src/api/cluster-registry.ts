/**
 * 集群注册表与 Agent 令牌 API（需要 clusters.manage）
 *
 * 集群是管理员登记的对象；每个集群一枚令牌，Agent 用它向 Master 证明自己属于哪个集群。
 * 接口规格见 docs/static/reference/master-api-reference.md §3.6。
 */

import { del, get, post, put } from "./request";

// ============================================================
// 类型
// ============================================================

/** 集群的在线状态：正在上报 / 上报过但现在离线 / 登记了但从未上报 */
export type ClusterRegistryStatus = "online" | "offline" | "never";

/** 集群管理页的一行：登记信息 + 在线状态 + 认证状态 */
export interface ClusterRegistryItem {
  clusterId: string;
  /** 显示名；未登记的集群为空串 */
  name: string;
  description: string;
  /** 为假表示这一行来自正在上报但尚未登记的集群，可以直接登记收编 */
  registered: boolean;
  tokenIssued: boolean;
  /** 令牌开头的几个字符，供辨认；未签发时为空串 */
  tokenPrefix: string;
  /** RFC 3339（UTC）；未签发时为空串 */
  tokenIssuedAt: string;
  status: ClusterRegistryStatus;
  /** 最近一次心跳或快照的时间，RFC 3339（UTC）；从未上报时为空串 */
  lastSeen: string;
  /** 该集群最近一次上报是否带了有效令牌 */
  authenticated: boolean;
}

/** 登记集群的请求体 */
export interface ClusterRegistration {
  clusterId: string;
  name: string;
  description: string;
}

/** 集群的显示名与描述 */
export interface ClusterProfile {
  name: string;
  description: string;
}

/**
 * 一次令牌签发的结果。
 *
 * token 是明文，只在签发的这一次响应里出现，之后任何接口都取不回它。
 * ⛔ 不得写进查询缓存、localStorage 或日志。
 */
export interface ClusterTokenIssued {
  token: string;
  tokenPrefix: string;
  tokenIssuedAt: string;
}

interface ClusterRegistryListResponse {
  message: string;
  data: ClusterRegistryItem[] | null;
  total: number;
}

interface ClusterRegistryItemResponse {
  message: string;
  data: ClusterRegistryItem;
}

// ============================================================
// API 方法
// ============================================================

/** 集群在路径里的那一段。集群 ID 的字符集本身不需要转义，这里转义是不信任调用方传来的值 */
function clusterPath(clusterId: string): string {
  return `/api/v2/admin/clusters/${encodeURIComponent(clusterId)}`;
}

/**
 * 集群列表：已登记的 + 正在上报但未登记的
 * GET /api/v2/admin/clusters
 */
export function getClusterRegistry(signal?: AbortSignal) {
  return get<ClusterRegistryListResponse>("/api/v2/admin/clusters", undefined, signal);
}

/**
 * 登记集群（成功 201；集群 ID 已登记 409；输入不合规 400）
 * POST /api/v2/admin/clusters
 */
export function registerCluster(registration: ClusterRegistration) {
  return post<ClusterRegistryItemResponse>("/api/v2/admin/clusters", registration);
}

/**
 * 修改显示名与描述（集群未登记 404；输入不合规 400）
 * PUT /api/v2/admin/clusters/{clusterId}
 */
export function updateClusterProfile(clusterId: string, profile: ClusterProfile) {
  return put<ClusterRegistryItemResponse>(clusterPath(clusterId), profile);
}

/**
 * 签发令牌；已有令牌时即为轮换，旧令牌立即失效（集群未登记 404）
 * POST /api/v2/admin/clusters/{clusterId}/token
 */
export function issueClusterToken(clusterId: string) {
  return post<{ message: string; data: ClusterTokenIssued }>(`${clusterPath(clusterId)}/token`);
}

/**
 * 删除登记：令牌随之失效，历史数据保留（集群未登记 404）
 * DELETE /api/v2/admin/clusters/{clusterId}
 */
export function unregisterCluster(clusterId: string) {
  return del<{ message: string }>(clusterPath(clusterId));
}
