/**
 * AIOps 事件详情的演示数据（均为虚构）
 *
 * 结构与 types/incident-view.ts 一致：
 * 原有字段（事件本身 + entities + timeline）之外，补上 headline / impact / evidence / members / timelineView。
 * 六个事件分别覆盖：成员全部中招、部分中招、只有一个成员、节点主体、改版前的事件（没有证据、成员已不存在）、
 * 只有依赖传播（headline 为 null）；后三个是拓扑归并的一组（源头带 impacted，受影响带 root，见 incident-links.ts）。
 */
import type { IncidentDetail, IncidentEntity, IncidentTimeline } from "@/api/aiops";
import { parseEntityKey } from "@/lib/entity-key";
import type {
  IncidentEvidence,
  IncidentImpact,
  IncidentMember,
  IncidentTimelineItem,
} from "@/types/incident-view";
import { INCIDENT_SEEDS, incidentOf, minutesAgo } from "./incidents";
import { linkDetailOf, linkOfIncident, linksToRoot } from "./incident-links";

/** 证据的种子：时刻用「多少分钟前」表示 */
type EvidenceSeed = Omit<IncidentEvidence, "firstSeen" | "lastSeen"> & { firstMin: number; lastMin: number };

/** 成员在当前快照里的样子；没列出的成员按「已不存在」处理 */
type MemberState = Pick<IncidentMember, "status" | "restarts" | "ready">;

interface DetailSeed {
  evidence: EvidenceSeed[];
  impact: IncidentImpact;
  memberStates: Record<string, MemberState>;
  /** 合并进时间线的检测记录条数；0 = 没有检测记录（只有依赖传播） */
  anomalyCount: number;
}

const NO_IMPACT: IncidentImpact = { desiredReplicas: null, availableReplicas: null, restartsTotal: null };

const DETAIL_SEEDS: Record<string, DetailSeed> = {
  // 三个 Pod 全部反复崩溃，Deployment 已无可用副本
  "inc-001": {
    evidence: [
      { signal: "container_crash", reason: "CrashLoopBackOff", affectedMembers: 3, detail: "容器 api: CrashLoopBackOff（退出码 1）", hits: 15, maxScore: 0.9, firstMin: 42, lastMin: 1 },
      { signal: "workload_unavailable", reason: "100", affectedMembers: 3, detail: "Deployment shop/api 不可用 100%（期望 3，可用 0）", hits: 9, maxScore: 0.85, firstMin: 41, lastMin: 1 },
      { signal: "k8s_event", reason: "BackOff", affectedMembers: 3, detail: "Back-off restarting failed container api", hits: 21, maxScore: 0.6, firstMin: 42, lastMin: 2 },
    ],
    impact: { desiredReplicas: 3, availableReplicas: 0, restartsTotal: 18 },
    memberStates: {
      "shop/pod/api-7b5d4f8c9-x2k4j": { status: "CrashLoopBackOff", restarts: 6, ready: "0/1" },
      "shop/pod/api-7b5d4f8c9-m8n2p": { status: "CrashLoopBackOff", restarts: 6, ready: "0/1" },
      "shop/pod/api-7b5d4f8c9-q9r3s": { status: "CrashLoopBackOff", restarts: 6, ready: "0/1" },
    },
    anomalyCount: 15,
  },
  // 只有一个成员的 StatefulSet：重启次数异常，已回落进入恢复观察
  "inc-002": {
    evidence: [
      { signal: "restarts", reason: "", affectedMembers: 1, detail: "容器 postgres 重启 4 次", hits: 6, maxScore: 0.41, firstMin: 190, lastMin: 96 },
    ],
    impact: { desiredReplicas: 1, availableReplicas: 1, restartsTotal: 4 },
    memberStates: { "shop/pod/orders-db-0": { status: "Running", restarts: 4, ready: "1/1" } },
    anomalyCount: 6,
  },
  // 两个成员只有一个中招：部分成员的句式
  "inc-003": {
    evidence: [
      { signal: "resource", reason: "", affectedMembers: 1, detail: "memory_usage 偏离基线 3.4σ", hits: 3, maxScore: 0.28, firstMin: 12, lastMin: 2 },
    ],
    impact: { desiredReplicas: 2, availableReplicas: 2, restartsTotal: 0 },
    memberStates: {
      "kube-system/pod/node-exporter-8xk2p": { status: "Running", restarts: 0, ready: "1/1" },
      "kube-system/pod/node-exporter-j4m9t": { status: "Running", restarts: 0, ready: "1/1" },
    },
    anomalyCount: 3,
  },
  // 节点主体：没有副本与重启的概念，影响数字只剩峰值风险
  "inc-004": {
    evidence: [
      { signal: "node_pressure", reason: "MemoryPressure", affectedMembers: 1, detail: "节点 worker-1 MemoryPressure=True", hits: 12, maxScore: 0.94, firstMin: 60 * 80, lastMin: 60 * 78 },
      { signal: "resource", reason: "", affectedMembers: 1, detail: "memory_usage 97%", hits: 10, maxScore: 0.8, firstMin: 60 * 80, lastMin: 60 * 78 },
    ],
    impact: NO_IMPACT,
    memberStates: { "_cluster/node/worker-1": { status: "Ready", restarts: null, ready: "" } },
    anomalyCount: 22,
  },
  // 改版前的事件：没有证据；CronJob 的 Pod 已经不存在
  "inc-005": {
    evidence: [],
    impact: NO_IMPACT,
    memberStates: {},
    anomalyCount: 4,
  },
  // 只有依赖传播：本主体自己没有异常信号，headline 为 null
  "inc-006": {
    evidence: [],
    impact: NO_IMPACT,
    memberStates: { "shop/service/checkout": { status: "", restarts: null, ready: "" } },
    anomalyCount: 0,
  },
  // 拓扑归并的源头：数据库的两个 Pod 都被 OOMKill 后反复重启
  "inc-007": {
    evidence: [
      { signal: "container_crash", reason: "OOMKilled", affectedMembers: 2, detail: "容器 postgres: OOMKilled（退出码 137）", hits: 8, maxScore: 0.91, firstMin: 18, lastMin: 1 },
      { signal: "workload_unavailable", reason: "100", affectedMembers: 2, detail: "StatefulSet shop/db 不可用 100%（期望 2，可用 0）", hits: 6, maxScore: 0.88, firstMin: 17, lastMin: 1 },
    ],
    impact: { desiredReplicas: 2, availableReplicas: 0, restartsTotal: 9 },
    memberStates: {
      "shop/pod/db-0": { status: "CrashLoopBackOff", restarts: 5, ready: "0/1" },
      "shop/pod/db-1": { status: "CrashLoopBackOff", restarts: 4, ready: "0/1" },
    },
    anomalyCount: 8,
  },
  // 受影响：数据库的 Service，只有依赖传播
  "inc-008": {
    evidence: [],
    impact: NO_IMPACT,
    memberStates: { "shop/service/db": { status: "", restarts: null, ready: "" } },
    anomalyCount: 0,
  },
  // 受影响：调用方自己的错误率上升（本地信号），根因在上游的数据库
  "inc-009": {
    evidence: [
      { signal: "apm_errors", reason: "", affectedMembers: 1, detail: "错误率 12.4%（基线 0.3%）", hits: 5, maxScore: 0.58, firstMin: 16, lastMin: 1 },
    ],
    impact: NO_IMPACT,
    memberStates: { "shop/service/payment": { status: "", restarts: null, ready: "" } },
    anomalyCount: 5,
  },
};

const GONE: MemberState = { status: "", restarts: null, ready: "" };

/** 时间线展示形态：按事件的字段排出状态与严重度的变化，检测记录合并成最后一条 */
function timelineViewOf(detail: ReturnType<typeof incidentOf>, anomalyCount: number): IncidentTimelineItem[] {
  // 各时刻都落在事件自己的时间范围里：异常从开始持续 durationS，之后进入恢复观察
  const startMs = Date.parse(detail.startedAt);
  const abnormalEnd = new Date(startMs + detail.durationS * 1000).toISOString();
  const recurredAt = new Date(startMs + detail.durationS * 600).toISOString();
  const items: IncidentTimelineItem[] = [
    { at: detail.startedAt, kind: "created", risk: Math.round(detail.peakRisk * 0.7), severity: "medium" },
  ];
  if (detail.severity === "high" || detail.severity === "critical") {
    items.push({ at: detail.escalatedAt ?? detail.startedAt, kind: "severity_raised", risk: Math.round(detail.peakRisk * 0.9), severity: detail.severity });
  }
  if (detail.escalatedAt) items.push({ at: detail.escalatedAt, kind: "escalated", risk: Math.round(detail.peakRisk) });
  if (detail.recurrence > 0) items.push({ at: recurredAt, kind: "recurred", risk: Math.round(detail.peakRisk * 0.8) });
  if (detail.state === "recovery" || detail.closeReason !== "") {
    items.push({ at: abnormalEnd, kind: "recovery_started", risk: 20 });
  }
  if (detail.closeReason !== "" && detail.resolvedAt) {
    items.push({ at: detail.resolvedAt, kind: detail.closeReason === "stale" ? "closed_stale" : "closed" });
  }
  if (anomalyCount > 0) {
    items.push({ at: detail.startedAt, kind: "anomaly", count: anomalyCount, firstAt: detail.startedAt, lastAt: abnormalEnd });
  }
  return items;
}

/** 演示集群里的事件（列表形态），按事件号取 */
function mockIncidentById(id: string) {
  const seed = INCIDENT_SEEDS.find((candidate) => candidate.incident.id === id);
  return seed ? incidentOf(seed, "demo") : undefined;
}

/** 拓扑归并的两项：受影响事件给 root，源头给 impacted（各带依赖路径） */
function groupingOf(incidentId: string): Pick<IncidentDetail, "root" | "impacted"> {
  const link = linkOfIncident(incidentId);
  const rootIncident = link ? mockIncidentById(link.rootIncidentId) : undefined;
  const impacted = linksToRoot(incidentId).flatMap((child) => {
    const childIncident = mockIncidentById(child.incidentId);
    return childIncident ? [linkDetailOf(childIncident, child, minutesAgo(child.linkedMin))] : [];
  });
  return {
    root: link && rootIncident ? linkDetailOf(rootIncident, link, minutesAgo(link.linkedMin)) : null,
    impacted,
  };
}

export function mockIncidentDetail(id: string): IncidentDetail {
  const seed = INCIDENT_SEEDS.find((candidate) => candidate.incident.id === id) ?? INCIDENT_SEEDS[0];
  const detailSeed = DETAIL_SEEDS[seed.incident.id];
  const incident = incidentOf(seed, "demo");
  const leadKey = incident.rootCause;

  const entities: IncidentEntity[] = seed.members.map((member, index) => ({
    incidentId: incident.id,
    ...member,
    namespace: parseEntityKey(member.entityKey).namespace,
    name: parseEntityKey(member.entityKey).name,
    role: index === 0 ? "root_cause" : "affected",
  }));

  const members: IncidentMember[] = entities.map((entity) => {
    const state = detailSeed.memberStates[entity.entityKey];
    return {
      entityKey: entity.entityKey,
      entityType: entity.entityType,
      namespace: entity.namespace,
      name: entity.name,
      role: entity.role,
      rFinal: entity.rFinal,
      isLead: entity.entityKey === leadKey,
      present: state !== undefined,
      ...(state ?? GONE),
    };
  });

  const evidence: IncidentEvidence[] = detailSeed.evidence.map(({ firstMin, lastMin, ...rest }) => ({
    ...rest,
    firstSeen: minutesAgo(firstMin),
    lastSeen: minutesAgo(lastMin),
  }));
  const lead = evidence[0];

  // 原始时间线：保留原有字段（已不在详情页展示）
  const timeline: IncidentTimeline[] = [
    { id: 1, incidentId: incident.id, timestamp: incident.startedAt, eventType: "state_change", entityKey: incident.subject, detail: "healthy → warning" },
  ];

  return {
    ...incident,
    entities,
    timeline,
    headline: lead
      ? { signal: lead.signal, reason: lead.reason, affectedMembers: lead.affectedMembers, totalMembers: members.length }
      : null,
    // 没有证据时：有检测记录 = 改版前的事件；连检测记录也没有 = 只有依赖传播
    evidenceStatus: evidence.length > 0 ? "recorded" : detailSeed.anomalyCount > 0 ? "not_recorded" : "propagation_only",
    impact: detailSeed.impact,
    evidence,
    members,
    timelineView: timelineViewOf(incident, detailSeed.anomalyCount),
    ...groupingOf(incident.id),
  };
}
