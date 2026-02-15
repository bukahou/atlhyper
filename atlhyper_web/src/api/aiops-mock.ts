/**
 * AIOps Mock 数据生成器
 *
 * 基于实际 K8s 集群结构生成真实感的 mock 数据。
 * 后端就绪后，此文件将不再使用。
 */

import type {
  ClusterRisk,
  EntityRisk,
  EntityRiskDetail,
  AnomalyResult,
  CausalEntry,
  PropagationPath,
  DependencyGraph,
  GraphNode,
  GraphEdge,
  Incident,
  IncidentDetail,
  IncidentEntity,
  IncidentTimeline,
  IncidentStats,
  IncidentListParams,
} from "./aiops";

// ==================== 常量 ====================

const NODES = ["atlas-master", "atlas-worker-1", "atlas-worker-2"];
const NAMESPACES = ["default", "kube-system", "linkerd", "monitoring", "app"];
const SERVICES = [
  { ns: "default", name: "api-server" },
  { ns: "default", name: "web-frontend" },
  { ns: "default", name: "user-service" },
  { ns: "default", name: "payment-service" },
  { ns: "kube-system", name: "coredns" },
  { ns: "linkerd", name: "linkerd-proxy" },
  { ns: "monitoring", name: "prometheus" },
  { ns: "monitoring", name: "grafana" },
];
const PODS = [
  { ns: "default", name: "api-server-7b5d4f8c9-x2k4j" },
  { ns: "default", name: "api-server-7b5d4f8c9-m8n2p" },
  { ns: "default", name: "web-frontend-5c8d7e6f4-q9r3s" },
  { ns: "default", name: "user-service-3a2b1c0d9-t5u6v" },
  { ns: "default", name: "payment-service-8e7f6g5h4-w1x2y" },
  { ns: "kube-system", name: "coredns-5d78c9869d-abc12" },
  { ns: "kube-system", name: "etcd-atlas-master" },
  { ns: "linkerd", name: "linkerd-proxy-injector-6b7c8d9-z3a4b" },
  { ns: "monitoring", name: "prometheus-server-0" },
  { ns: "monitoring", name: "grafana-7f8e9d0c1-k5l6m" },
];
const INGRESSES = [
  { ns: "default", name: "api-ingress" },
  { ns: "default", name: "web-ingress" },
  { ns: "monitoring", name: "grafana-ingress" },
];

const RISK_LEVELS = ["healthy", "low", "warning", "critical"] as const;
const SEVERITIES = ["low", "medium", "high", "critical"] as const;
const INCIDENT_STATES = ["warning", "incident", "recovery", "stable"] as const;
const METRICS = [
  "cpu_usage_percent",
  "memory_usage_percent",
  "disk_io_util",
  "network_error_rate",
  "request_latency_p99",
  "error_rate_5xx",
  "pod_restart_count",
];

// ==================== 工具函数 ====================

function seededRandom(seed: string): () => number {
  let h = 0;
  for (let i = 0; i < seed.length; i++) {
    h = (Math.imul(31, h) + seed.charCodeAt(i)) | 0;
  }
  return () => {
    h = (Math.imul(h, 1103515245) + 12345) | 0;
    return ((h >>> 16) & 0x7fff) / 0x7fff;
  };
}

function riskLevelFromScore(score: number): string {
  if (score >= 80) return "critical";
  if (score >= 50) return "warning";
  if (score >= 20) return "low";
  return "healthy";
}

function isoNow(offsetMinutes = 0): string {
  return new Date(Date.now() - offsetMinutes * 60000).toISOString();
}

// ==================== Mock 生成器 ====================

export function mockClusterRisk(cluster: string): ClusterRisk {
  const rand = seededRandom(cluster + "risk");
  const risk = Math.round(rand() * 60 + 10); // 10-70
  const entities = buildEntityRisks(cluster, 20);

  return {
    clusterId: cluster,
    risk,
    level: riskLevelFromScore(risk),
    topEntities: entities.slice(0, 5),
    totalEntities: entities.length,
    anomalyCount: entities.filter((e) => e.riskLevel !== "healthy").length,
    updatedAt: Date.now(),
  };
}

function buildEntityRisks(cluster: string, limit: number): EntityRisk[] {
  const rand = seededRandom(cluster + "entities");
  const entities: EntityRisk[] = [];

  // 从所有实体中生成风险数据
  const allEntities = [
    ...NODES.map((n) => ({ key: `cluster/${cluster}/node/${n}`, type: "node", ns: "", name: n })),
    ...SERVICES.map((s) => ({ key: `${s.ns}/service/${s.name}`, type: "service", ns: s.ns, name: s.name })),
    ...PODS.map((p) => ({ key: `${p.ns}/pod/${p.name}`, type: "pod", ns: p.ns, name: p.name })),
    ...INGRESSES.map((i) => ({ key: `${i.ns}/ingress/${i.name}`, type: "ingress", ns: i.ns, name: i.name })),
  ];

  for (const e of allEntities) {
    const rLocal = Math.round(rand() * 80 * 10) / 10;
    const wTime = Math.round((0.5 + rand() * 0.5) * 100) / 100;
    const rWeighted = Math.round(rLocal * wTime * 10) / 10;
    const rFinal = Math.round((rWeighted + rand() * 10) * 10) / 10;

    entities.push({
      entityKey: e.key,
      entityType: e.type,
      namespace: e.ns,
      name: e.name,
      rLocal,
      wTime,
      rWeighted,
      rFinal: Math.min(100, rFinal),
      riskLevel: riskLevelFromScore(rFinal),
      firstAnomaly: rFinal > 20 ? Date.now() - Math.floor(rand() * 3600000) : 0,
    });
  }

  // 按 rFinal 降序排序
  entities.sort((a, b) => b.rFinal - a.rFinal);
  return entities.slice(0, limit);
}

export function mockEntityRisks(cluster: string, limit: number): EntityRisk[] {
  return buildEntityRisks(cluster, limit);
}

export function mockEntityRiskDetail(cluster: string, entityKey: string): EntityRiskDetail {
  const rand = seededRandom(cluster + entityKey);
  const parts = entityKey.split("/");
  const entityType = parts.length >= 3 ? parts[parts.length - 2] : "pod";
  const name = parts[parts.length - 1];
  const ns = parts.length >= 3 ? parts[0] : "";

  const rLocal = Math.round(rand() * 70 * 10) / 10 + 10;
  const wTime = Math.round((0.5 + rand() * 0.5) * 100) / 100;
  const rFinal = Math.min(100, Math.round((rLocal * wTime + rand() * 15) * 10) / 10);

  const metrics: AnomalyResult[] = METRICS.slice(0, 3 + Math.floor(rand() * 3)).map((m) => ({
    entityKey,
    metricName: m,
    currentValue: Math.round(rand() * 90 * 100) / 100,
    baseline: Math.round((20 + rand() * 30) * 100) / 100,
    deviation: Math.round((rand() * 4) * 100) / 100,
    score: Math.round(rand() * 100 * 10) / 10,
    isAnomaly: rand() > 0.4,
    detectedAt: Date.now() - Math.floor(rand() * 1800000),
  }));

  const causalChain: CausalEntry[] = [
    { entityKey, metricName: "memory_usage_percent", deviation: 3.2, detectedAt: Date.now() - 600000 },
    { entityKey: `${ns}/pod/${name}`, metricName: "cpu_usage_percent", deviation: 2.1, detectedAt: Date.now() - 480000 },
  ];

  const propagation: PropagationPath[] = [
    { from: entityKey, to: `${ns}/service/${name.split("-")[0]}`, edgeType: "serves", contribution: 0.8 },
  ];

  return {
    entityKey,
    entityType,
    namespace: ns,
    name,
    rLocal,
    wTime,
    rWeighted: Math.round(rLocal * wTime * 10) / 10,
    rFinal,
    riskLevel: riskLevelFromScore(rFinal),
    firstAnomaly: Date.now() - Math.floor(rand() * 3600000),
    metrics,
    propagation,
    causalChain,
  };
}

export function mockDependencyGraph(cluster: string): DependencyGraph {
  const nodes: Record<string, GraphNode> = {};
  const edges: GraphEdge[] = [];

  // 添加节点
  for (const n of NODES) {
    const key = `cluster/${cluster}/node/${n}`;
    nodes[key] = { key, type: "node", namespace: "", name: n, metadata: { role: n.includes("master") ? "control-plane" : "worker" } };
  }
  for (const s of SERVICES) {
    const key = `${s.ns}/service/${s.name}`;
    nodes[key] = { key, type: "service", namespace: s.ns, name: s.name, metadata: {} };
  }
  for (const p of PODS) {
    const key = `${p.ns}/pod/${p.name}`;
    nodes[key] = { key, type: "pod", namespace: p.ns, name: p.name, metadata: {} };
  }
  for (const i of INGRESSES) {
    const key = `${i.ns}/ingress/${i.name}`;
    nodes[key] = { key, type: "ingress", namespace: i.ns, name: i.name, metadata: {} };
  }

  // 添加边: ingress → service → pod → node
  edges.push(
    { from: "default/ingress/api-ingress", to: "default/service/api-server", type: "routes_to", weight: 1 },
    { from: "default/ingress/web-ingress", to: "default/service/web-frontend", type: "routes_to", weight: 1 },
    { from: "monitoring/ingress/grafana-ingress", to: "monitoring/service/grafana", type: "routes_to", weight: 1 },
    { from: "default/service/api-server", to: "default/pod/api-server-7b5d4f8c9-x2k4j", type: "selects", weight: 1 },
    { from: "default/service/api-server", to: "default/pod/api-server-7b5d4f8c9-m8n2p", type: "selects", weight: 1 },
    { from: "default/service/web-frontend", to: "default/pod/web-frontend-5c8d7e6f4-q9r3s", type: "selects", weight: 1 },
    { from: "default/service/user-service", to: "default/pod/user-service-3a2b1c0d9-t5u6v", type: "selects", weight: 1 },
    { from: "default/service/payment-service", to: "default/pod/payment-service-8e7f6g5h4-w1x2y", type: "selects", weight: 1 },
    // service → service 依赖
    { from: "default/service/web-frontend", to: "default/service/api-server", type: "calls", weight: 0.9 },
    { from: "default/service/api-server", to: "default/service/user-service", type: "calls", weight: 0.7 },
    { from: "default/service/api-server", to: "default/service/payment-service", type: "calls", weight: 0.5 },
    // pod → node
    { from: "default/pod/api-server-7b5d4f8c9-x2k4j", to: `cluster/${cluster}/node/atlas-worker-1`, type: "runs_on", weight: 1 },
    { from: "default/pod/api-server-7b5d4f8c9-m8n2p", to: `cluster/${cluster}/node/atlas-worker-2`, type: "runs_on", weight: 1 },
    { from: "default/pod/web-frontend-5c8d7e6f4-q9r3s", to: `cluster/${cluster}/node/atlas-worker-1`, type: "runs_on", weight: 1 },
    { from: "default/pod/user-service-3a2b1c0d9-t5u6v", to: `cluster/${cluster}/node/atlas-worker-2`, type: "runs_on", weight: 1 },
    { from: "default/pod/payment-service-8e7f6g5h4-w1x2y", to: `cluster/${cluster}/node/atlas-worker-1`, type: "runs_on", weight: 1 },
    { from: "kube-system/pod/coredns-5d78c9869d-abc12", to: `cluster/${cluster}/node/atlas-master`, type: "runs_on", weight: 1 },
    { from: "kube-system/pod/etcd-atlas-master", to: `cluster/${cluster}/node/atlas-master`, type: "runs_on", weight: 1 },
  );

  return {
    clusterId: cluster,
    nodes,
    edges,
    updatedAt: new Date().toISOString(),
  };
}

export function mockIncidents(params: IncidentListParams): Incident[] {
  const rand = seededRandom(params.cluster + "incidents");
  const incidents: Incident[] = [];
  const count = 12;

  for (let i = 0; i < count; i++) {
    const state = INCIDENT_STATES[Math.floor(rand() * INCIDENT_STATES.length)];
    const severity = SEVERITIES[Math.floor(rand() * SEVERITIES.length)];
    const startMinutesAgo = Math.floor(rand() * 10080); // 最近 7 天
    const durationS = Math.floor(rand() * 7200) + 60; // 1 分钟 ~ 2 小时
    const resolved = state === "stable" || state === "recovery";

    incidents.push({
      id: `inc-${String(1000 + i).slice(1)}`,
      clusterId: params.cluster,
      state,
      severity,
      rootCause: [
        `cluster/${params.cluster}/node/atlas-worker-1`,
        "default/pod/api-server-7b5d4f8c9-x2k4j",
        "default/service/payment-service",
        "kube-system/pod/coredns-5d78c9869d-abc12",
      ][Math.floor(rand() * 4)],
      peakRisk: Math.round((40 + rand() * 55) * 10) / 10,
      startedAt: isoNow(startMinutesAgo),
      resolvedAt: resolved ? isoNow(startMinutesAgo - durationS / 60) : null,
      durationS,
      recurrence: Math.floor(rand() * 4),
      createdAt: isoNow(startMinutesAgo),
    });
  }

  // 过滤
  let filtered = incidents;
  if (params.state) {
    filtered = filtered.filter((inc) => inc.state === params.state);
  }

  // 排序: 按开始时间降序
  filtered.sort((a, b) => new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime());

  // 分页
  const offset = params.offset ?? 0;
  const limit = params.limit ?? 20;
  return filtered.slice(offset, offset + limit);
}

export function mockIncidentDetail(id: string): IncidentDetail {
  const rand = seededRandom(id);
  const state = INCIDENT_STATES[Math.floor(rand() * INCIDENT_STATES.length)];
  const severity = SEVERITIES[1 + Math.floor(rand() * 3)]; // medium+
  const startMinutesAgo = Math.floor(rand() * 1440) + 10;
  const durationS = Math.floor(rand() * 3600) + 120;
  const rootCauseEntity = `cluster/demo/node/atlas-worker-${1 + Math.floor(rand() * 2)}`;

  const entities: IncidentEntity[] = [
    { incidentId: id, entityKey: rootCauseEntity, entityType: "node", rLocal: 85, rFinal: 90, role: "root_cause" },
    { incidentId: id, entityKey: "default/pod/api-server-7b5d4f8c9-x2k4j", entityType: "pod", rLocal: 72, rFinal: 78, role: "affected" },
    { incidentId: id, entityKey: "default/service/api-server", entityType: "service", rLocal: 68, rFinal: 85, role: "symptom" },
    { incidentId: id, entityKey: "default/pod/web-frontend-5c8d7e6f4-q9r3s", entityType: "pod", rLocal: 45, rFinal: 52, role: "affected" },
  ];

  const baseTime = Date.now() - startMinutesAgo * 60000;
  const timeline: IncidentTimeline[] = [
    { id: 1, incidentId: id, timestamp: new Date(baseTime).toISOString(), eventType: "anomaly_detected", entityKey: rootCauseEntity, detail: "memory_usage_percent deviation 3.2\u03c3" },
    { id: 2, incidentId: id, timestamp: new Date(baseTime + 120000).toISOString(), eventType: "state_change", entityKey: rootCauseEntity, detail: "healthy \u2192 warning" },
    { id: 3, incidentId: id, timestamp: new Date(baseTime + 180000).toISOString(), eventType: "metric_spike", entityKey: "default/service/api-server", detail: "error_rate_5xx 3.2%" },
    { id: 4, incidentId: id, timestamp: new Date(baseTime + 240000).toISOString(), eventType: "root_cause_identified", entityKey: rootCauseEntity, detail: "causal chain confirmed" },
    { id: 5, incidentId: id, timestamp: new Date(baseTime + 360000).toISOString(), eventType: "state_change", entityKey: rootCauseEntity, detail: "warning \u2192 incident" },
  ];

  if (state === "recovery" || state === "stable") {
    timeline.push({
      id: 6,
      incidentId: id,
      timestamp: new Date(baseTime + durationS * 1000).toISOString(),
      eventType: "recovery_started",
      entityKey: rootCauseEntity,
      detail: "metrics returning to baseline",
    });
  }

  return {
    id,
    clusterId: "demo",
    state,
    severity,
    rootCause: rootCauseEntity,
    peakRisk: Math.round((70 + rand() * 25) * 10) / 10,
    startedAt: isoNow(startMinutesAgo),
    resolvedAt: state === "stable" ? isoNow(startMinutesAgo - durationS / 60) : null,
    durationS,
    recurrence: Math.floor(rand() * 3),
    createdAt: isoNow(startMinutesAgo),
    entities,
    timeline,
  };
}

export function mockIncidentStats(_cluster: string): IncidentStats {
  return {
    totalIncidents: 47,
    activeIncidents: 3,
    mttr: 2700, // 45 分钟（秒）
    recurrenceRate: 13.3,
    bySeverity: { low: 12, medium: 18, high: 11, critical: 6 },
    byState: { warning: 1, incident: 2, recovery: 0, stable: 44 },
    topRootCauses: [
      { entityKey: "cluster/demo/node/atlas-worker-1", count: 8 },
      { entityKey: "default/service/api-server", count: 5 },
      { entityKey: "default/pod/payment-service-8e7f6g5h4-w1x2y", count: 3 },
    ],
  };
}
