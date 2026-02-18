/**
 * AIOps API
 *
 * 类型定义对齐设计文档: docs/design/active/aiops-phase3-frontend.md §2
 */

import { get, post } from "./request";

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

export interface CausalTreeNode {
  entityKey: string;
  entityType: string;
  rFinal: number;
  edgeType?: string;
  direction?: string;
  metrics?: AnomalyResult[];
  children?: CausalTreeNode[];
}

export interface EntityRiskDetail extends EntityRisk {
  metrics: AnomalyResult[];
  propagation: PropagationPath[];
  causalChain: CausalEntry[];
  causalTree?: CausalTreeNode[];
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

// ==================== API 方法 ====================

// 风险
export async function getClusterRisk(cluster: string): Promise<ClusterRisk> {
  const data = (await get<ClusterRisk | null>("/api/v2/aiops/risk/cluster", { cluster })).data;
  return data ?? {
    clusterId: cluster, risk: 0, level: "healthy",
    topEntities: [], totalEntities: 0, anomalyCount: 0, updatedAt: Date.now(),
  };
}

export async function getEntityRisks(cluster: string, sort = "r_final", limit = 20): Promise<EntityRisk[]> {
  return (await get<EntityRisk[] | null>("/api/v2/aiops/risk/entities", { cluster, sort, limit })).data ?? [];
}

export async function getEntityRiskDetail(cluster: string, entityKey: string): Promise<EntityRiskDetail> {
  return (await get<EntityRiskDetail>("/api/v2/aiops/risk/entity", { cluster, entity: entityKey })).data;
}

// 依赖图
export async function getGraph(cluster: string): Promise<DependencyGraph> {
  return (await get<DependencyGraph>("/api/v2/aiops/graph", { cluster })).data;
}

// 事件
export async function getIncidents(params: IncidentListParams): Promise<Incident[]> {
  const resp = (await get<{ data: Incident[]; total: number }>("/api/v2/aiops/incidents", params)).data;
  return resp?.data ?? [];
}

export async function getIncidentDetail(id: string): Promise<IncidentDetail> {
  return (await get<IncidentDetail>(`/api/v2/aiops/incidents/${encodeURIComponent(id)}`)).data;
}

export async function getIncidentStats(cluster: string, period = "7d"): Promise<IncidentStats> {
  return (await get<IncidentStats>("/api/v2/aiops/incidents/stats", { cluster, period })).data;
}

// AI 增强
export async function summarizeIncident(incidentId: string): Promise<SummarizeResponse> {
  return (await post<SummarizeResponse>("/api/v2/aiops/ai/summarize", { incidentId })).data;
}

export async function recommendActions(incidentId: string): Promise<SummarizeResponse> {
  return (await post<SummarizeResponse>("/api/v2/aiops/ai/recommend", { incidentId })).data;
}
