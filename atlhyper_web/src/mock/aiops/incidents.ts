/**
 * AIOps 事件的演示数据（均为虚构）
 *
 * 结构与 docs/design/active/incident-notification-design.md「功能零」一致：事件以主体为单位，
 * 同一工作负载的 Pod 归到一个事件；严重度是峰值，关闭后保留；警告级的反复也计入复发次数。
 * 前六条覆盖每一种主体类型、每一种状态与两种关闭原因；后三条是拓扑归并的一组「源头 + 2 个受影响」
 * （关联关系在 incident-links.ts）。
 * 详情（功能四的结论、证据、成员状态、时间线展示形态）在 incident-detail.ts。
 */
import type { Incident, IncidentListParams, IncidentStats } from "@/api/aiops";
import { parseEntityKey } from "@/lib/entity-key";
import { linkOfIncident, linkSummaryOf, linksToRoot } from "./incident-links";

/** 相对现在往前若干分钟的时刻 */
export function minutesAgo(minutes: number): string {
  return new Date(Date.now() - minutes * 60_000).toISOString();
}

/** 一条演示事件：事件本身 + 它的成员实体（第一个是领头实体） */
export interface MockIncidentSeed {
  incident: Omit<
    Incident,
    | "clusterId" | "rootCause" | "createdAt" | "subjectNamespace" | "subjectName"
    | "rootIncidentId" | "linkedCount" | "impacted"
  >;
  /** 成员实体：键、类型、风险（百分制）。第一个是领头实体 */
  members: { entityKey: string; entityType: string; rLocal: number; rFinal: number }[];
}

export const INCIDENT_SEEDS: MockIncidentSeed[] = [
  {
    // Deployment 的三个 Pod 同时反复崩溃：一个事件，已升级
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
      subject: "_cluster/node/worker-1",
      subjectType: "node",
      peakRisk: 93.5,
      startedAt: minutesAgo(60 * 80),
      resolvedAt: minutesAgo(60 * 30),
      escalatedAt: minutesAgo(60 * 80 - 6),
      closeReason: "recovered",
      durationS: 2 * 60 * 60,
      recurrence: 1,
    },
    members: [{ entityKey: "_cluster/node/worker-1", entityType: "node", rLocal: 90, rFinal: 94 }],
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
  {
    // 拓扑归并的源头：数据库 StatefulSet 的两个 Pod 都起不来
    incident: {
      id: "inc-007",
      state: "incident",
      severity: "critical",
      subject: "shop/statefulset/db",
      subjectType: "statefulset",
      peakRisk: 91.0,
      startedAt: minutesAgo(18),
      resolvedAt: null,
      escalatedAt: minutesAgo(15),
      closeReason: "",
      durationS: 18 * 60,
      recurrence: 0,
    },
    members: [
      { entityKey: "shop/pod/db-0", entityType: "pod", rLocal: 88, rFinal: 91 },
      { entityKey: "shop/pod/db-1", entityType: "pod", rLocal: 80, rFinal: 84 },
    ],
  },
  {
    // 受影响：数据库的 Service，风险全部来自依赖传播
    incident: {
      id: "inc-008",
      state: "warning",
      severity: "medium",
      subject: "shop/service/db",
      subjectType: "service",
      peakRisk: 55.0,
      startedAt: minutesAgo(17),
      resolvedAt: null,
      escalatedAt: null,
      closeReason: "",
      durationS: 17 * 60,
      recurrence: 0,
    },
    members: [{ entityKey: "shop/service/db", entityType: "service", rLocal: 0, rFinal: 55 }],
  },
  {
    // 受影响：经链路调用数据库的 payment Service，错误率上升是它自己的本地信号，但根因在上游
    incident: {
      id: "inc-009",
      state: "warning",
      severity: "medium",
      subject: "shop/service/payment",
      subjectType: "service",
      peakRisk: 58.0,
      startedAt: minutesAgo(16),
      resolvedAt: null,
      escalatedAt: null,
      closeReason: "",
      durationS: 16 * 60,
      recurrence: 0,
    },
    members: [{ entityKey: "shop/service/payment", entityType: "service", rLocal: 40, rFinal: 58 }],
  },
];

export function incidentOf(seed: MockIncidentSeed, clusterId: string): Incident {
  // 真实接口里这两项由后端给出；演示数据从主体键里拆
  const { namespace, name } = parseEntityKey(seed.incident.subject);
  return {
    ...seed.incident,
    subjectNamespace: namespace,
    subjectName: name,
    clusterId,
    // 领头实体 = 成员里风险最高的那个
    rootCause: seed.members[0].entityKey,
    createdAt: seed.incident.startedAt,
    // 拓扑归并：平铺形态（impacted 只在 grouped=true 的列表里给）
    rootIncidentId: linkOfIncident(seed.incident.id)?.rootIncidentId ?? "",
    linkedCount: linksToRoot(seed.incident.id).length,
    impacted: [],
  };
}

/**
 * 列表。grouped=true 时照契约折叠：受影响事件不单独出现，源头带上 impacted。
 * 状态筛选作用在折叠后的行上（源头的 impacted 不按状态筛）——契约没写，见汇报。
 */
export function mockIncidents(params: IncidentListParams): Incident[] {
  const offset = params.offset ?? 0;
  const limit = params.limit ?? 20;
  const all = INCIDENT_SEEDS.map((seed) => incidentOf(seed, params.cluster));
  const rows = params.grouped
    ? all
        .filter((incident) => incident.rootIncidentId === "")
        .map((incident) => ({
          ...incident,
          impacted: all
            .filter((candidate) => candidate.rootIncidentId === incident.id)
            .map(linkSummaryOf),
        }))
    : all;
  return rows.filter((incident) => !params.state || incident.state === params.state).slice(offset, offset + limit);
}

export function mockIncidentStats(_cluster: string): IncidentStats {
  return {
    totalIncidents: INCIDENT_SEEDS.length,
    // 只数警告与事件中；恢复观察不算活跃
    activeIncidents: 6,
    mttr: 2700,
    recurrenceRate: 33.3,
    bySeverity: { low: 1, medium: 4, high: 2, critical: 2 },
    byState: { warning: 3, incident: 3, recovery: 1, stable: 2 },
    // 键是主体键，不是具体的 Pod
    topRootCauses: [
      { entityKey: "shop/deployment/api", count: 3 },
      { entityKey: "_cluster/node/worker-1", count: 2 },
    ],
  };
}
