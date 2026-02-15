/**
 * AIOps API
 *
 * 类型定义对齐设计文档: docs/design/active/aiops-phase3-frontend.md §2
 * 当前使用 mock 数据，后端就绪后切换为真实 API 调用
 */

import { post } from "./request";
import {
  mockClusterRisk,
  mockClusterRiskTrend,
  mockEntityRisks,
  mockEntityRiskDetail,
  mockDependencyGraph,
  mockIncidents,
  mockIncidentDetail,
  mockIncidentStats,
} from "./aiops-mock";

// ==================== 类型定义 ====================

// 风险相关
export interface ClusterRisk {
  clusterId: string;
  risk: number; // [0, 100]
  level: string; // "healthy" | "low" | "warning" | "critical"
  topEntities: EntityRisk[];
  totalEntities: number;
  anomalyCount: number;
  updatedAt: number;
}

export interface EntityRisk {
  entityKey: string;
  entityType: string; // "service" | "pod" | "node" | "ingress"
  namespace: string;
  name: string;
  rLocal: number;
  wTime: number;
  rWeighted: number;
  rFinal: number;
  riskLevel: string;
  firstAnomaly: number;
}

export interface EntityRiskDetail extends EntityRisk {
  metrics: AnomalyResult[];
  propagation: PropagationPath[];
  causalChain: CausalEntry[];
}

export interface AnomalyResult {
  entityKey: string;
  metricName: string;
  currentValue: number;
  baseline: number;
  deviation: number;
  score: number;
  isAnomaly: boolean;
  detectedAt: number;
}

export interface PropagationPath {
  from: string;
  to: string;
  edgeType: string;
  contribution: number;
}

export interface CausalEntry {
  entityKey: string;
  metricName: string;
  deviation: number;
  detectedAt: number;
}

// 依赖图相关
export interface DependencyGraph {
  clusterId: string;
  nodes: Record<string, GraphNode>;
  edges: GraphEdge[];
  updatedAt: string;
}

export interface GraphNode {
  key: string;
  type: string;
  namespace: string;
  name: string;
  metadata: Record<string, string>;
}

export interface GraphEdge {
  from: string;
  to: string;
  type: string;
  weight: number;
}

// 事件相关
export interface Incident {
  id: string;
  clusterId: string;
  state: string;
  severity: string;
  rootCause: string;
  peakRisk: number;
  startedAt: string;
  resolvedAt: string | null;
  durationS: number;
  recurrence: number;
  createdAt: string;
}

export interface IncidentDetail extends Incident {
  entities: IncidentEntity[];
  timeline: IncidentTimeline[];
}

export interface IncidentEntity {
  incidentId: string;
  entityKey: string;
  entityType: string;
  rLocal: number;
  rFinal: number;
  role: string;
}

export interface IncidentTimeline {
  id: number;
  incidentId: string;
  timestamp: string;
  eventType: string;
  entityKey: string;
  detail: string;
}

export interface IncidentStats {
  totalIncidents: number;
  activeIncidents: number;
  mttr: number;
  recurrenceRate: number;
  bySeverity: Record<string, number>;
  byState: Record<string, number>;
  topRootCauses: { entityKey: string; count: number }[];
}

// AI 增强
export interface SummarizeResponse {
  incidentId: string;
  summary: string;
  rootCauseAnalysis: string;
  recommendations: Recommendation[];
  similarIncidents: SimilarMatch[];
  generatedAt: number;
}

export interface Recommendation {
  priority: number;
  action: string;
  reason: string;
  impact: string;
}

export interface SimilarMatch {
  incidentId: string;
  similarity: number;
  rootCause: string;
  occurredAt: string;
  durationS: number;
}

// 查询参数
export interface IncidentListParams {
  cluster: string;
  state?: string;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
}

// 风险趋势数据点
export interface RiskTrendPoint {
  timestamp: number;
  risk: number;
  level: string;
}

// ==================== API 方法 ====================

// 风险
export async function getClusterRisk(cluster: string): Promise<ClusterRisk> {
  // TODO: 后端就绪后切换 → return (await get<ClusterRisk>('/api/v2/aiops/risk/cluster', { cluster })).data
  return mockClusterRisk(cluster);
}

export async function getClusterRiskTrend(cluster: string, period = "24h"): Promise<RiskTrendPoint[]> {
  // TODO: 后端就绪后切换 → return (await get<RiskTrendPoint[]>('/api/v2/aiops/risk/cluster/trend', { cluster, period })).data
  void period;
  return mockClusterRiskTrend(cluster);
}

export async function getEntityRisks(cluster: string, sort = "r_final", limit = 20): Promise<EntityRisk[]> {
  // TODO: 后端就绪后切换 → return (await get<EntityRisk[]>('/api/v2/aiops/risk/entities', { cluster, sort, limit })).data
  void sort;
  return mockEntityRisks(cluster, limit);
}

export async function getEntityRiskDetail(cluster: string, entityKey: string): Promise<EntityRiskDetail> {
  // TODO: 后端就绪后切换 → return (await get<EntityRiskDetail>(`/api/v2/aiops/risk/entity/${encodeURIComponent(entityKey)}`, { cluster })).data
  return mockEntityRiskDetail(cluster, entityKey);
}

// 依赖图
export async function getGraph(cluster: string): Promise<DependencyGraph> {
  // TODO: 后端就绪后切换 → return (await get<DependencyGraph>('/api/v2/aiops/graph', { cluster })).data
  return mockDependencyGraph(cluster);
}

// 事件
export async function getIncidents(params: IncidentListParams): Promise<Incident[]> {
  // TODO: 后端就绪后切换 → return (await get<Incident[]>('/api/v2/aiops/incidents', params)).data
  return mockIncidents(params);
}

export async function getIncidentDetail(id: string): Promise<IncidentDetail> {
  // TODO: 后端就绪后切换 → return (await get<IncidentDetail>(`/api/v2/aiops/incidents/${encodeURIComponent(id)}`)).data
  return mockIncidentDetail(id);
}

export async function getIncidentStats(cluster: string, period = "7d"): Promise<IncidentStats> {
  // TODO: 后端就绪后切换 → return (await get<IncidentStats>('/api/v2/aiops/incidents/stats', { cluster, period })).data
  void period;
  return mockIncidentStats(cluster);
}

// AI 增强
export async function summarizeIncident(incidentId: string): Promise<SummarizeResponse> {
  return (await post<SummarizeResponse>("/api/v2/aiops/ai/summarize", { incidentId })).data;
}

export async function recommendActions(incidentId: string): Promise<SummarizeResponse> {
  return (await post<SummarizeResponse>("/api/v2/aiops/ai/recommend", { incidentId })).data;
}
