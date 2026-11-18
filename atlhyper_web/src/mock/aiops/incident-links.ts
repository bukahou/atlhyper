/**
 * AIOps 拓扑归并的演示数据（均为虚构）
 *
 * 结构与 types/incident-grouping.ts 一致：
 * 一组「源头 + 2 个受影响」——数据库 StatefulSet（inc-007）挂了，
 * 它的 Service（inc-008，经「选中」）与经链路调用它的 payment Service（inc-009，经两次「调用」再「选中」）归到它下面。
 * inc-009 的路径有 4 个实体，用来演示路径过长时的折叠。
 * 路径从受影响事件的成员出发，沿依赖方向走到源头的成员，第一步的 edge 为空串。
 */
import type { Incident } from "@/api/aiops";
import type { IncidentLinkDetail, IncidentLinkSummary, IncidentPathStep } from "@/types/incident-grouping";

/** 一条关联：受影响事件 → 源头事件 + 归组依据 */
export interface MockIncidentLinkSeed {
  incidentId: string;
  rootIncidentId: string;
  path: IncidentPathStep[];
  /** 归入时刻：多少分钟前 */
  linkedMin: number;
}

export const INCIDENT_LINK_SEEDS: MockIncidentLinkSeed[] = [
  {
    incidentId: "inc-008",
    rootIncidentId: "inc-007",
    path: [
      { entityKey: "shop/service/db", edge: "" },
      { entityKey: "shop/pod/db-0", edge: "selects" },
    ],
    linkedMin: 16,
  },
  {
    incidentId: "inc-009",
    rootIncidentId: "inc-007",
    path: [
      { entityKey: "shop/service/payment", edge: "" },
      { entityKey: "shop/service/orders", edge: "calls" },
      { entityKey: "shop/service/db", edge: "calls" },
      { entityKey: "shop/pod/db-0", edge: "selects" },
    ],
    linkedMin: 15,
  },
];

/** 这个事件归到的源头；不是受影响事件时为 undefined */
export function linkOfIncident(incidentId: string): MockIncidentLinkSeed | undefined {
  return INCIDENT_LINK_SEEDS.find((link) => link.incidentId === incidentId);
}

/** 归到这个源头下面的关联 */
export function linksToRoot(rootIncidentId: string): MockIncidentLinkSeed[] {
  return INCIDENT_LINK_SEEDS.filter((link) => link.rootIncidentId === rootIncidentId);
}

/** 事件 → 关联另一端的概要 */
export function linkSummaryOf(incident: Incident): IncidentLinkSummary {
  return {
    incidentId: incident.id,
    subject: incident.subject,
    subjectType: incident.subjectType,
    state: incident.state,
    severity: incident.severity,
  };
}

/** 事件 + 关联 → 详情里的关联条目 */
export function linkDetailOf(incident: Incident, link: MockIncidentLinkSeed, linkedAt: string): IncidentLinkDetail {
  return { ...linkSummaryOf(incident), path: link.path, linkedAt };
}
