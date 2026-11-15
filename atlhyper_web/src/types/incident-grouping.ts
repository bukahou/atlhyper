/**
 * 拓扑归并新增的响应字段（事件列表与事件详情）
 *
 * 契约：docs/design/active/aiops-topology-grouping-design.md「实施方案 → 接口（契约）」。
 * 依赖链上同一次故障的事件归为一组：一个源头事件，其余是受影响事件（事件照旧各自存在，另记归属关系）。
 * 只给当前有效的关联；边类型等码由前端按 i18n 显示。
 */

/** 依赖路径上一步的边类型：空串 = 起点；其余是依赖图的边。没见过的取值原样显示 */
export type IncidentLinkEdge = "" | "calls" | "selects" | "runs_on" | "routes_to";

/** 依赖路径的一步：实体键 + 从上一步走到这一步经过的边 */
export interface IncidentPathStep {
  entityKey: string;
  /** 取值见 IncidentLinkEdge；后端将来新增边类型时可能出现没见过的取值，所以类型放宽为 string */
  edge: string;
}

/** 关联的另一端事件的概要：列表里源头的 impacted 条目 */
export interface IncidentLinkSummary {
  incidentId: string;
  /** 主体键，如 shop/service/api */
  subject: string;
  subjectType: string;
  state: string;
  severity: string;
}

/** 详情里的关联：概要 + 归组依据（依赖路径）+ 归入时刻 */
export interface IncidentLinkDetail extends IncidentLinkSummary {
  /** 从受影响事件的成员沿依赖走到源头的成员；第一步的 edge 为空串 */
  path: IncidentPathStep[];
  linkedAt: string;
}

/** 列表每条事件新增的三项 */
export interface IncidentListGrouping {
  /** 受影响事件：归到的源头事件号；不是受影响事件时为空串 */
  rootIncidentId: string;
  /** 源头事件：当前归到它下面的受影响事件数；否则 0 */
  linkedCount: number;
  /** 只在 grouped=true 且本条是源头时给出，否则为空数组 */
  impacted: IncidentLinkSummary[];
}

/** 详情新增的两项 */
export interface IncidentDetailGrouping {
  /** 本事件归到某个源头时给出，否则 null */
  root: IncidentLinkDetail | null;
  /** 本事件是源头时：归到它下面的受影响事件；否则空数组 */
  impacted: IncidentLinkDetail[];
}
