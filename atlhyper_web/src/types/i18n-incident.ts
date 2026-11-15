/**
 * 事件详情弹窗（结论、关键证据、下一步、成员、时间线、历史与 AI 分析）的翻译类型
 *
 * 从 i18n.ts 拆出来单放：i18n.ts 已经很长。挂在 t.aiops.detail 下。
 * 带 {xxx} 的文案是模板，由 components/incident-*.ts 里的纯函数填入。
 */
import type { IncidentSignal, IncidentTimelineKind } from "./incident-view";

export interface IncidentDetailTranslations {
  clusterLabel: string;

  // 结论
  ongoingFor: string;
  lastedFor: string;
  startedAtShort: string;
  escalatedAtShort: string;
  notEscalated: string;
  noRecurrence: string;
  recurredTimes: string;
  headline: {
    /** 成员全部中招：{total} {noun} {what} */
    allMembers: string;
    /** 部分成员中招：{total} {noun} {affected} {what} */
    someMembers: string;
    /** 单个成员的主体（Pod / Node / Service 等）：{type} {what} */
    single: string;
    /** 工作负载不可用这类主体级信号：{type} {percent} */
    workloadUnavailable: string;
    workloadUnavailableNoPercent: string;
    /** 主证据带原因码时的写法：{what} {reason} */
    withReason: string;
    /** 期望副本 > 0 而可用为 0 时补一句：{type} */
    noAvailableReplicas: string;
    /** 一句话的结尾标点 */
    end: string;
    /** headline 为 null */
    none: string;
    /** headline 为 null、但时间线里有检测记录：改版前的事件 */
    legacy: string;
    podNoun: string;
    memberNoun: string;
  };
  impact: {
    replicas: string;
    restarts: string;
    peakRisk: string;
  };

  // 关键证据
  evidenceTitle: string;
  /** 证据卡标题用的信号名称 */
  signalName: Record<IncidentSignal, string>;
  /** 结论句里用的信号说法（放进 {what}） */
  signalPhrase: Record<IncidentSignal, string>;
  evidenceAffected: string;
  evidenceFirstSeen: string;
  evidenceLastSeen: string;
  evidenceHits: string;
  evidenceLegacy: string;
  evidenceNone: string;

  // 下一步
  nextStepsTitle: string;
  viewSubject: string;
  viewPods: string;
  viewNodePods: string;
  relatedEvents: string;
  containerLogs: string;
  aiAnalysis: string;

  // 成员
  membersTitle: string;
  colName: string;
  colStatus: string;
  colRestarts: string;
  colRisk: string;
  leadBadge: string;
  leadBadgeHint: string;
  memberGone: string;
  noMembers: string;

  // 时间线
  timelineTitle: string;
  timelineSummary: string;
  timelineExpand: string;
  timelineCollapse: string;
  timelineKind: Record<IncidentTimelineKind, string>;
  timelineUnknownKind: string;
  /** recovery_started 且原因是对象已删除时的文案 */
  timelineRecoverySubjectDeleted: string;
  timelineSeverityRaisedTo: string;
  timelineSeverity: string;
  timelineRisk: string;
  timelineAnomalyMerged: string;
  timelineEmpty: string;

  // 底部
  historyTitle: string;
}

/**
 * 拓扑归并（事件列表的按组折叠、详情的源头与影响范围）的翻译类型，挂在 t.aiops.grouping 下。
 * 契约：docs/design/active/aiops-topology-grouping-design.md
 */
export interface IncidentGroupingTranslations {
  // 列表
  groupedView: string;
  flatView: string;
  viewToggleHint: string;
  /** 源头行的「关联 N 个」：{count} */
  linkedCount: string;
  linkedCountHint: string;
  /** 平铺时源头行的「关联 N 个」不能展开（受影响事件各自单独列出） */
  linkedCountFlatHint: string;
  /** 展开后受影响事件行前的小标签 */
  impactedBadge: string;
  /** 平铺时受影响事件的小标签：{subject} */
  rootTag: string;
  rootTagHint: string;

  // 详情：受影响事件
  rootLabel: string;
  via: string;
  rootLinkHint: string;
  /** 归入时刻：{time} */
  linkedAt: string;

  // 详情：源头事件
  /** {count} */
  impactTitle: string;
  impactHint: string;
  impactOpenHint: string;

  // 依赖路径
  /** 边类型的文案；起点（空串）没有边 */
  edge: Record<"calls" | "selects" | "runs_on" | "routes_to", string>;
  /** 折叠起来的中间实体：{count} */
  pathGap: string;
  pathExpand: string;
  pathCollapse: string;
  pathEmpty: string;
}
