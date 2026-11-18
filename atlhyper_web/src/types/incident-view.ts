/**
 * 事件详情页改版新增的响应字段（GET /api/v2/aiops/incidents/{id}）
 *
 * 后端只给结构化的码与数值；一句话结论、信号名称、状态名都由前端按 i18n 渲染。
 * 风险一律 0–100 整数；时间一律 UTC（RFC 3339），前端按浏览器时区显示。
 */

/** 信号类型：由基线 / 风险模块的指标名归类（契约「信号类型」表） */
export type IncidentSignal =
  | "container_crash"
  | "workload_unavailable"
  | "k8s_event"
  | "restarts"
  | "not_ready"
  | "resource"
  | "node_pressure"
  | "apm_errors"
  | "apm_latency"
  | "log_errors"
  | "other";

/** 结论那一句话的素材：主证据（evidence 里分数最高的一条） */
export interface IncidentHeadline {
  signal: IncidentSignal;
  /** 信号的原因码 / 取值：container_crash 为原因码，workload_unavailable 为百分比整数，k8s_event 为事件原因；没有时为空串 */
  reason: string;
  affectedMembers: number;
  totalMembers: number;
  /**
   * 主证据是原因类信号（容器崩溃 / 未就绪 / 重启）且同时有工作负载不可用时的不可用比例（0–100），否则 null。
   * 老版本后端没有这个字段
   */
  unavailablePercent?: number | null;
}

/** 影响数字；不适用的项为 null */
export interface IncidentImpact {
  /** 只对 Deployment / StatefulSet / DaemonSet 主体 */
  desiredReplicas: number | null;
  /** 只对 Deployment / StatefulSet / DaemonSet 主体 */
  availableReplicas: number | null;
  /** 当前仍存在的成员 Pod 的重启次数之和 */
  restartsTotal: number | null;
}

/** 关键证据：按信号类型去重，按最高分数从高到低 */
export interface IncidentEvidence {
  signal: IncidentSignal;
  reason: string;
  affectedMembers: number;
  /** 最近一次的原文（已去掉 Pod UID 等噪音），作为次要文字展示 */
  detail: string;
  firstSeen: string;
  lastSeen: string;
  hits: number;
  /** 契约示例为 0.9，与「风险一律 0–100」不一致，单位待确认；前端不展示、不参与判断 */
  maxScore: number;
}

/** 成员：与 entities 一一对应，补上当前快照里的状态 */
export interface IncidentMember {
  entityKey: string;
  entityType: string;
  namespace: string;
  name: string;
  /** root_cause / affected：沿用实体表的角色，界面上不再叫「根因」 */
  role: string;
  /** @unit 百分制 [0, 100] */
  rFinal: number;
  /** 成员里风险最高的那个（界面标「风险最高」） */
  isLead: boolean;
  /** 当前快照里还在；false 时 status / restarts / ready 为空 */
  present: boolean;
  status: string;
  restarts: number | null;
  ready: string;
}

/** 时间线展示形态的条目类型 */
export type IncidentTimelineKind =
  | "created"
  | "escalated"
  | "severity_raised"
  | "recovery_started"
  | "recurred"
  | "closed"
  | "closed_stale"
  | "anomaly";

/** 时间线展示形态：状态与严重度的变化 + 合并后的检测记录 */
export interface IncidentTimelineItem {
  at: string;
  kind: IncidentTimelineKind;
  /** @unit 百分制 [0, 100]；anomaly 条目没有 */
  risk?: number | null;
  /** created / severity_raised 带：low / medium / high / critical */
  severity?: string | null;
  /** anomaly：合并了几条检测 */
  count?: number | null;
  /** anomaly：合并范围的首条时刻 */
  firstAt?: string | null;
  /** anomaly：合并范围的末条时刻 */
  lastAt?: string | null;
  /** recovery_started：subject_deleted = 对象已删除（不是风险回落） */
  reason?: string | null;
}

/**
 * 证据的情况：recorded = 有证据；propagation_only = 只有依赖传播、本主体没有本地异常；
 * not_recorded = 改版前的事件，当时没有记录证据
 */
export type IncidentEvidenceStatus = "recorded" | "propagation_only" | "not_recorded";

/** 详情响应在原有字段之外新增的几项 */
export interface IncidentDetailView {
  /** null = 没有证据（原因见 evidenceStatus） */
  headline: IncidentHeadline | null;
  evidenceStatus: IncidentEvidenceStatus;
  /** 主体的对象在当前快照里是否还在；false = 已删除；null / 缺省 = 不知道 */
  subjectPresent?: boolean | null;
  /** 后端在集群快照不可用时给 null；经 withDetailViewDefaults 补成全部为 null 的影响项 */
  impact: IncidentImpact;
  evidence: IncidentEvidence[];
  members: IncidentMember[];
  timelineView: IncidentTimelineItem[];
}
