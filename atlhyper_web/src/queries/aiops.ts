/**
 * AIOps 的取数描述 —— 风险、依赖图、事件
 *
 * AIOps 没有演示数据，工厂直接调用 api 层（约定：没有 datasource 的域由工厂直接调 api）。
 * 集群由调用方用 useCurrentCluster 得到；没有选定集群时不启用。
 * AI 报告与深度分析在 aiops-ai.ts。
 */
import { queryOptions } from "@tanstack/react-query";
import {
  getClusterRisk,
  getEntityBaseline,
  getEntityRiskDetail,
  getEntityRisks,
  getGraph,
  getGraphTrace,
  getIncidentDetail,
  getIncidentPatterns,
  getIncidents,
  getIncidentStats,
} from "@/api/aiops";
import { aiopsKeys, type GraphTraceDirection } from "./keys";
import { REFRESH } from "./refresh";

/** 事件列表一次取多少条 */
const INCIDENT_PAGE_SIZE = 20;
/** 依赖追踪沿依赖图走几层 */
const GRAPH_TRACE_DEPTH = 3;

/**
 * 上一份查询是不是同一个集群的。
 *
 * 换筛选条件时先留着上一份结果（placeholderData），但只在同一个集群内：
 * 换了集群还显示旧集群的数据，等于把别的集群的东西当成当前集群的。
 */
export function isQueryOfCluster(query: { queryKey: readonly unknown[] } | undefined, clusterId: string): boolean {
  const scope = query?.queryKey[2] as { clusterId?: string } | undefined;
  return scope?.clusterId === clusterId;
}

// ============================================================
// 风险
// ============================================================

/** 集群整体风险（仪表盘） */
export function clusterRiskQuery(clusterId: string) {
  return queryOptions({
    queryKey: aiopsKeys.clusterRisk(clusterId),
    queryFn: ({ signal }) => getClusterRisk(clusterId, signal),
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
    enabled: clusterId !== "",
  });
}

/**
 * 风险最高的前 limit 个实体。
 *
 * pollIntervalMs 由页面给：风险页 30 秒；拓扑页 10 秒，暂停自动刷新时传 false。
 */
export function entityRisksQuery(clusterId: string, limit: number, pollIntervalMs: number | false) {
  return queryOptions({
    queryKey: aiopsKeys.entityRisks(clusterId, limit),
    queryFn: ({ signal }) => getEntityRisks(clusterId, "r_final", limit, signal),
    refetchInterval: pollIntervalMs,
    staleTime: pollIntervalMs === false ? 0 : pollIntervalMs,
    placeholderData: (previousRisks, previousQuery) =>
      isQueryOfCluster(previousQuery, clusterId) ? previousRisks : undefined,
    enabled: clusterId !== "",
  });
}

/** 某个实体的风险详情（异常指标、因果树、传播路径）。只在展开或选中时取 */
export function entityRiskDetailQuery(clusterId: string, entityKey: string) {
  return queryOptions({
    queryKey: aiopsKeys.entityRiskDetail(clusterId, entityKey),
    queryFn: ({ signal }) => getEntityRiskDetail(clusterId, entityKey, signal),
    enabled: clusterId !== "" && entityKey !== "",
  });
}

/** 某个实体的指标基线（EMA 学习状态） */
export function entityBaselineQuery(clusterId: string, entityKey: string) {
  return queryOptions({
    queryKey: aiopsKeys.entityBaseline(clusterId, entityKey),
    queryFn: ({ signal }) => getEntityBaseline(clusterId, entityKey, signal),
    enabled: clusterId !== "" && entityKey !== "",
  });
}

// ============================================================
// 依赖图
// ============================================================

/** 集群依赖图。pollIntervalMs：拓扑页 10 秒，暂停自动刷新时传 false */
export function dependencyGraphQuery(clusterId: string, pollIntervalMs: number | false) {
  return queryOptions({
    queryKey: aiopsKeys.graph(clusterId),
    queryFn: ({ signal }) => getGraph(clusterId, signal),
    refetchInterval: pollIntervalMs,
    staleTime: pollIntervalMs === false ? 0 : pollIntervalMs,
    enabled: clusterId !== "",
  });
}

/** 从某个实体出发沿依赖图往上游或下游走到的节点 */
export function graphTraceQuery(clusterId: string, entityKey: string, direction: GraphTraceDirection) {
  return queryOptions({
    queryKey: aiopsKeys.graphTrace(clusterId, entityKey, direction),
    queryFn: ({ signal }) => getGraphTrace(clusterId, entityKey, direction, GRAPH_TRACE_DEPTH, signal),
    enabled: clusterId !== "" && entityKey !== "",
  });
}

// ============================================================
// 事件
// ============================================================

/** 事件列表。state 为空串 = 全部状态 */
export function incidentsQuery(clusterId: string, state: string) {
  return queryOptions({
    queryKey: aiopsKeys.incidents(clusterId, state),
    queryFn: ({ signal }) =>
      getIncidents({ cluster: clusterId, state: state || undefined, limit: INCIDENT_PAGE_SIZE }, signal),
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
    placeholderData: (previousIncidents, previousQuery) =>
      isQueryOfCluster(previousQuery, clusterId) ? previousIncidents : undefined,
    enabled: clusterId !== "",
  });
}

/** 事件统计（近 7 天，由后端取默认周期） */
export function incidentStatsQuery(clusterId: string) {
  return queryOptions({
    queryKey: aiopsKeys.incidentStats(clusterId),
    queryFn: ({ signal }) => getIncidentStats(clusterId, undefined, signal),
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
    enabled: clusterId !== "",
  });
}

/** 事件详情（实体 + 时间线）。只在详情弹窗打开期间取 */
export function incidentDetailQuery(incidentId: string) {
  return queryOptions({
    queryKey: aiopsKeys.incidentDetail(incidentId),
    queryFn: ({ signal }) => getIncidentDetail(incidentId, signal),
    enabled: incidentId !== "",
  });
}

/**
 * 某个实体的历史事件模式（近 30 天，由后端取默认周期）：回答「这个问题是不是老毛病」。
 *
 * entityKey 取事件详情里的根因实体，详情还没回来时为空串、不启用。
 */
export function incidentPatternsQuery(clusterId: string, entityKey: string) {
  return queryOptions({
    queryKey: aiopsKeys.incidentPatterns(clusterId, entityKey),
    queryFn: ({ signal }) => getIncidentPatterns(clusterId, entityKey, undefined, signal),
    enabled: clusterId !== "" && entityKey !== "",
  });
}
