/**
 * 事件详情：结论、影响数字、证据、成员状态的展示映射（纯函数）
 *
 * 后端只给结构化的码与数值（契约「功能四」），这里只做「码 → 文案 / 配色」的映射与模板填充，
 * 不做业务判断。文案模板都来自 t.aiops.detail，测试直接传入模板对象。
 */
import type { IncidentDetailTranslations } from "@/types/i18n-incident";
import type { IncidentDetailView, IncidentHeadline, IncidentImpact, IncidentSignal } from "@/types/incident-view";
import { formatDuration, formatShortTime } from "./incident-format";

const NO_IMPACT: IncidentImpact = { desiredReplicas: null, availableReplicas: null, restartsTotal: null };

/**
 * 补齐改版新增的四项：后端还没上线新字段（或老版本 Master）时，响应里没有它们，
 * 按「没有证据、没有影响数字、没有成员状态、没有时间线」显示，页面不至于崩。
 */
export function withDetailViewDefaults<T extends Partial<IncidentDetailView>>(detail: T): T & IncidentDetailView {
  return {
    ...detail,
    headline: detail.headline ?? null,
    // 老版本 Master 没有这个字段：有证据算 recorded，没有按「改版前」处理
    evidenceStatus: detail.evidenceStatus ?? ((detail.evidence ?? []).length > 0 ? "recorded" : "not_recorded"),
    impact: detail.impact ?? NO_IMPACT,
    evidence: detail.evidence ?? [],
    members: detail.members ?? [],
    timelineView: detail.timelineView ?? [],
  };
}

/** 把模板里的 {key} 换成对应的值；没提供的占位原样保留，方便发现漏填 */
export function fillTemplate(template: string, values: Record<string, string | number>): string {
  return template.replace(/\{(\w+)\}/g, (placeholder, key: string) =>
    key in values ? String(values[key]) : placeholder,
  );
}

/** 成员是 Pod 的主体类型（工作负载）：结论里按「N 个 Pod」说 */
const POD_MEMBER_SUBJECTS = new Set(["deployment", "statefulset", "daemonset", "job", "cronjob"]);

/** 信号的原因码 / 取值怎么显示：工作负载不可用的取值是百分比整数，其余原样 */
export function signalReasonText(signal: IncidentSignal, reason: string): string {
  if (reason === "") return "";
  return signal === "workload_unavailable" ? `${reason}%` : reason;
}

export interface HeadlineInput {
  headline: IncidentHeadline | null;
  impact: IncidentImpact;
  /** 主体类型的原值（deployment 等），用来决定成员的叫法 */
  subjectType: string;
  /** 主体类型的显示名（Deployment 等） */
  subjectTypeLabel: string;
  /** 时间线里有合并的检测记录：headline 为 null 时据此区分「改版前的事件」与「只有依赖传播」 */
  hasDetections: boolean;
}

/** 结论的一句话：主证据 + 中招成员数 + 主体类型，按模板拼；headline 为 null 时给出「没有本地异常信号」的说明 */
export function headlineSentence(input: HeadlineInput, texts: IncidentDetailTranslations): string {
  const { headline, impact, subjectType, subjectTypeLabel, hasDetections } = input;
  const tpl = texts.headline;
  if (headline === null) return hasDetections ? tpl.legacy : tpl.none;

  let sentence: string;
  if (headline.signal === "workload_unavailable") {
    sentence =
      headline.reason === ""
        ? fillTemplate(tpl.workloadUnavailableNoPercent, { type: subjectTypeLabel })
        : fillTemplate(tpl.workloadUnavailable, { type: subjectTypeLabel, percent: headline.reason });
  } else {
    const phrase = texts.signalPhrase[headline.signal] ?? texts.signalPhrase.other;
    const what = headline.reason === "" ? phrase : fillTemplate(tpl.withReason, { what: phrase, reason: headline.reason });
    const noun = POD_MEMBER_SUBJECTS.has(subjectType) ? tpl.podNoun : tpl.memberNoun;
    if (headline.totalMembers <= 1) {
      sentence = fillTemplate(tpl.single, { type: subjectTypeLabel, what });
    } else if (headline.affectedMembers >= headline.totalMembers) {
      sentence = fillTemplate(tpl.allMembers, { total: headline.totalMembers, noun, what });
    } else {
      sentence = fillTemplate(tpl.someMembers, {
        total: headline.totalMembers,
        affected: headline.affectedMembers,
        noun,
        what,
      });
    }
    // 主证据是原因（如容器崩溃）时，补半句影响：先看当前快照（副本数），对象已删除等拿不到时用证据里记下的不可用比例
    const percent = headline.unavailablePercent ?? null;
    if (impact.desiredReplicas !== null && impact.desiredReplicas > 0 && impact.availableReplicas === 0) {
      sentence += fillTemplate(tpl.noAvailableReplicas, { type: subjectTypeLabel });
    } else if (percent !== null && percent >= 100) {
      sentence += fillTemplate(tpl.noAvailableReplicas, { type: subjectTypeLabel });
    } else if (percent !== null && percent > 0) {
      sentence += fillTemplate(tpl.unavailablePercent, { type: subjectTypeLabel, percent });
    }
  }
  return sentence + tpl.end;
}

/** 影响数字：为 null 的项不显示；峰值风险来自事件本身，始终显示 */
export function impactItems(impact: IncidentImpact, peakRisk: number, texts: IncidentDetailTranslations): string[] {
  const items: string[] = [];
  if (impact.desiredReplicas !== null && impact.availableReplicas !== null) {
    items.push(
      fillTemplate(texts.impact.replicas, { available: impact.availableReplicas, desired: impact.desiredReplicas }),
    );
  }
  if (impact.restartsTotal !== null) {
    items.push(fillTemplate(texts.impact.restarts, { count: impact.restartsTotal }));
  }
  items.push(fillTemplate(texts.impact.peakRisk, { risk: Math.round(peakRisk) }));
  return items;
}

/** 结论第一行用到的事件字段 */
export interface VerdictMetaSource {
  state: string;
  durationS: number;
  startedAt: string;
  escalatedAt: string | null;
  recurrence: number;
}

/**
 * 结论第一行状态徽标之后的说明：已持续 · 开始 · 升级 · 复发。
 * 时长直接用后端的 durationS（恢复观察与关闭后不再增长），所以只有警告 / 事件中说「已持续」。
 */
export function verdictMetaParts(
  incident: VerdictMetaSource,
  texts: IncidentDetailTranslations,
  minuteLabel: string,
): string[] {
  const ongoing = incident.state === "warning" || incident.state === "incident";
  const duration = formatDuration(incident.durationS, minuteLabel);
  return [
    fillTemplate(ongoing ? texts.ongoingFor : texts.lastedFor, { duration }),
    fillTemplate(texts.startedAtShort, { time: formatShortTime(incident.startedAt) }),
    incident.escalatedAt === null
      ? texts.notEscalated
      : fillTemplate(texts.escalatedAtShort, { time: formatShortTime(incident.escalatedAt) }),
    incident.recurrence > 0 ? fillTemplate(texts.recurredTimes, { count: incident.recurrence }) : texts.noRecurrence,
  ];
}

/** 证据卡左侧色条：直接表明「坏了」的信号红色，指标类橙色，旁证灰色 */
export type EvidenceTone = "red" | "amber" | "gray";

const EVIDENCE_TONES: Record<IncidentSignal, EvidenceTone> = {
  container_crash: "red",
  workload_unavailable: "red",
  not_ready: "red",
  node_pressure: "red",
  restarts: "amber",
  resource: "amber",
  apm_errors: "amber",
  apm_latency: "amber",
  log_errors: "amber",
  k8s_event: "gray",
  other: "gray",
};

export function evidenceTone(signal: IncidentSignal): EvidenceTone {
  return EVIDENCE_TONES[signal] ?? "gray";
}

/** 成员状态徽标的配色分组：K8s 原样的状态字符串 → 正常 / 过渡中 / 异常 */
export type MemberStatusTone = "ok" | "pending" | "bad";

const OK_STATUSES = new Set(["Running", "Ready", "Succeeded", "Completed", "Active"]);
const PENDING_STATUSES = new Set(["Pending", "ContainerCreating", "PodInitializing", "Terminating"]);

export function memberStatusTone(status: string): MemberStatusTone {
  if (OK_STATUSES.has(status)) return "ok";
  if (PENDING_STATUSES.has(status)) return "pending";
  return "bad";
}
