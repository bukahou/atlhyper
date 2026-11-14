/**
 * 查询键工厂 —— AIOps（风险 / 依赖图 / 事件 / AI 报告）与 AI 对话
 *
 * 约定同 keys.ts。AIOps 没有演示数据，键里不带 mode；
 * 事件详情、AI 报告按 ID 取数，ID 全局唯一，键里不带集群。
 */

/** 依赖追踪的方向：沿依赖图往上游还是下游走 */
export type GraphTraceDirection = "upstream" | "downstream";

export const aiopsKeys = {
  /** 整个 AIOps 域的前缀：页头的手动刷新按它匹配 */
  all: ["aiops"] as const,

  clusterRisk: (clusterId: string) => ["aiops", "cluster-risk", { clusterId }] as const,
  /** 条数进键：风险页看前 20 / 50 / 100，拓扑页固定看前 100，各用各的缓存 */
  entityRisks: (clusterId: string, limit: number) => ["aiops", "entity-risks", { clusterId, limit }] as const,
  entityRiskDetail: (clusterId: string, entityKey: string) =>
    ["aiops", "entity-risk-detail", { clusterId, entityKey }] as const,
  entityBaseline: (clusterId: string, entityKey: string) =>
    ["aiops", "entity-baseline", { clusterId, entityKey }] as const,

  graph: (clusterId: string) => ["aiops", "graph", { clusterId }] as const,
  graphTrace: (clusterId: string, entityKey: string, direction: GraphTraceDirection) =>
    ["aiops", "graph-trace", { clusterId, entityKey, direction }] as const,

  /** state 为空串 = 不按状态过滤 */
  incidents: (clusterId: string, state: string) => ["aiops", "incidents", { clusterId, state }] as const,
  incidentStats: (clusterId: string) => ["aiops", "incident-stats", { clusterId }] as const,
  incidentDetail: (incidentId: string) => ["aiops", "incident-detail", { incidentId }] as const,
  incidentPatterns: (clusterId: string, entityKey: string) =>
    ["aiops", "incident-patterns", { clusterId, entityKey }] as const,

  /** 某个事件的 AI 报告列表 */
  aiReports: (incidentId: string) => ["aiops", "ai-reports", { incidentId }] as const,
  aiReportDetail: (reportId: number) => ["aiops", "ai-report-detail", { reportId }] as const,
  /** 某个事件最近一次深度分析的状态 */
  analysisStatus: (incidentId: string) => ["aiops", "analysis-status", { incidentId }] as const,
};

export const aiChatKeys = {
  conversations: () => ["ai-chat", "conversations"] as const,
  messages: (conversationId: number) => ["ai-chat", "messages", { conversationId }] as const,
};
