/**
 * AIOps 事件的演示数据（均为虚构）
 *
 * 结构与 docs/design/active/incident-notification-design.md「功能零」一致：事件以主体为单位，
 * 同一工作负载的 Pod 归到一个事件；严重度是峰值，关闭后保留；警告级的反复也计入复发次数。
 * 六条覆盖每一种主体类型、每一种状态与两种关闭原因。
 */
import type { Incident, IncidentDetail, IncidentEntity, IncidentListParams, IncidentStats, IncidentTimeline } from "@/api/aiops";

/** 相对现在往前若干分钟的时刻 */
function minutesAgo(minutes: number): string {
  return new Date(Date.now() - minutes * 60_000).toISOString();
}

/** 一条演示事件：事件本身 + 它的成员实体（第一个是领头实体） */
interface MockIncidentSeed {
  incident: Omit<Incident, "clusterId" | "rootCause" | "createdAt">;
  /** 成员实体：键、类型、风险（百分制）。第一个是领头实体 */
  members: { entityKey: string; entityType: string; rLocal: number; rFinal: number }[];
}

const INCIDENT_SEEDS: MockIncidentSeed[] = [
  {
    // Deployment 的三个 Pod 同时探针失败：一个事件，已升级
    incident: {
      id: "inc-001",
      state: "incident",
      severity: "high",
      subject: "shop/deployment/api",
      subjectType: "deployment",
      peakRisk: 82.5,
      startedAt: minutesAgo(42),
      resolvedAt: null,
      escalatedAt: minutesAgo(35),
      closeReason: "",
      durationS: 42 * 60,
      recurrence: 2,
    },
    members: [
      { entityKey: "shop/pod/api-7b5d4f8c9-x2k4j", entityType: "pod", rLocal: 78, rFinal: 83 },
      { entityKey: "shop/pod/api-7b5d4f8c9-m8n2p", entityType: "pod", rLocal: 70, rFinal: 74 },
      { entityKey: "shop/pod/api-7b5d4f8c9-q9r3s", entityType: "pod", rLocal: 61, rFinal: 66 },
    ],
  },
  {
    // 只到「警告」就回落：进入恢复观察，没有升级过；警告级的反复也计入复发
    incident: {
      id: "inc-002",
      state: "recovery",
      severity: "medium",
      subject: "shop/statefulset/orders-db",
      subjectType: "statefulset",
      peakRisk: 41.0,
      startedAt: minutesAgo(190),
      resolvedAt: null,
      escalatedAt: null,
      closeReason: "",
      durationS: 95 * 60,
      recurrence: 3,
    },
    members: [{ entityKey: "shop/pod/orders-db-0", entityType: "pod", rLocal: 38, rFinal: 41 }],
  },
  {
    incident: {
      id: "inc-003",
      state: "warning",
      severity: "low",
      subject: "kube-system/daemonset/node-exporter",
      subjectType: "daemonset",
      peakRisk: 28.0,
      startedAt: minutesAgo(12),
      resolvedAt: null,
      escalatedAt: null,
      closeReason: "",
      durationS: 12 * 60,
      recurrence: 0,
    },
    members: [
      { entityKey: "kube-system/pod/node-exporter-8xk2p", entityType: "pod", rLocal: 26, rFinal: 28 },
      { entityKey: "kube-system/pod/node-exporter-j4m9t", entityType: "pod", rLocal: 22, rFinal: 23 },
    ],
  },
  {
    // 观察期满未复发：正常关闭，严重度保留
    incident: {
      id: "inc-004",
      state: "stable",
      severity: "critical",
      subject: "cluster/demo/node/worker-1",
      subjectType: "node",
      peakRisk: 93.5,
      startedAt: minutesAgo(60 * 80),
      resolvedAt: minutesAgo(60 * 30),
      escalatedAt: minutesAgo(60 * 80 - 6),
      closeReason: "recovered",
      durationS: 2 * 60 * 60,
      recurrence: 1,
    },
    members: [{ entityKey: "cluster/demo/node/worker-1", entityType: "node", rLocal: 90, rFinal: 94 }],
  },
  {
    // CronJob 的 Pod 已经不存在：失联关闭，并不是真的恢复了
    incident: {
      id: "inc-005",
      state: "stable",
      severity: "medium",
      subject: "shop/cronjob/report",
      subjectType: "cronjob",
      peakRisk: 47.0,
      startedAt: minutesAgo(60 * 26),
      resolvedAt: minutesAgo(60 * 25),
      escalatedAt: null,
      closeReason: "stale",
      durationS: 40 * 60,
      recurrence: 0,
    },
    members: [{ entityKey: "shop/pod/report-28871-x2k4j", entityType: "pod", rLocal: 44, rFinal: 47 }],
  },
  {
    // 主体就是实体自身：服务
    incident: {
      id: "inc-006",
      state: "incident",
      severity: "high",
      subject: "shop/service/checkout",
      subjectType: "service",
      peakRisk: 71.0,
      startedAt: minutesAgo(25),
      resolvedAt: null,
      escalatedAt: minutesAgo(19),
      closeReason: "",
      durationS: 25 * 60,
      recurrence: 0,
    },
    members: [{ entityKey: "shop/service/checkout", entityType: "service", rLocal: 66, rFinal: 71 }],
  },
];

function incidentOf(seed: MockIncidentSeed, clusterId: string): Incident {
  return {
    ...seed.incident,
    clusterId,
    // 领头实体 = 成员里风险最高的那个
    rootCause: seed.members[0].entityKey,
    createdAt: seed.incident.startedAt,
  };
}

export function mockIncidents(params: IncidentListParams): Incident[] {
  const offset = params.offset ?? 0;
  const limit = params.limit ?? 20;
  return INCIDENT_SEEDS.map((seed) => incidentOf(seed, params.cluster))
    .filter((incident) => !params.state || incident.state === params.state)
    .slice(offset, offset + limit);
}

export function mockIncidentDetail(id: string): IncidentDetail {
  const seed = INCIDENT_SEEDS.find((candidate) => candidate.incident.id === id) ?? INCIDENT_SEEDS[0];
  const incident = incidentOf(seed, "demo");
  const leadKey = incident.rootCause;

  const entities: IncidentEntity[] = seed.members.map((member, index) => ({
    incidentId: incident.id,
    ...member,
    role: index === 0 ? "root_cause" : "affected",
  }));

  const timeline: IncidentTimeline[] = [
    { id: 1, incidentId: incident.id, timestamp: incident.startedAt, eventType: "anomaly_detected", entityKey: leadKey, detail: "deviation 3.2σ from baseline" },
    { id: 2, incidentId: incident.id, timestamp: incident.startedAt, eventType: "state_change", entityKey: incident.subject, detail: "healthy → warning" },
  ];
  if (incident.escalatedAt) {
    timeline.push({ id: 3, incidentId: incident.id, timestamp: incident.escalatedAt, eventType: "state_change", entityKey: incident.subject, detail: "warning → incident" });
  }
  if (incident.escalatedAt && (incident.severity === "high" || incident.severity === "critical")) {
    timeline.push({ id: 5, incidentId: incident.id, timestamp: incident.escalatedAt, eventType: "severity_raised", entityKey: incident.subject, detail: `medium \u2192 ${incident.severity}` });
  }
  if (incident.state === "recovery" || incident.closeReason === "recovered") {
    timeline.push({ id: 4, incidentId: incident.id, timestamp: incident.resolvedAt ?? minutesAgo(5), eventType: "recovery_started", entityKey: incident.subject, detail: "risk back under threshold" });
  }

  return { ...incident, entities, timeline };
}

export function mockIncidentStats(_cluster: string): IncidentStats {
  return {
    totalIncidents: INCIDENT_SEEDS.length,
    // 只数警告与事件中；恢复观察不算活跃
    activeIncidents: 3,
    mttr: 2700,
    recurrenceRate: 50.0,
    bySeverity: { low: 1, medium: 2, high: 2, critical: 1 },
    byState: { warning: 1, incident: 2, recovery: 1, stable: 2 },
    // 键是主体键，不是具体的 Pod
    topRootCauses: [
      { entityKey: "shop/deployment/api", count: 3 },
      { entityKey: "cluster/demo/node/worker-1", count: 2 },
    ],
  };
}
