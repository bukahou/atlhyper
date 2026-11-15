/**
 * 路由规则表单的草稿、校验与调序（纯函数）
 */
import { isIntegerInRange, NOTIFY_POLICY_RANGES, parseInteger } from "@/lib/notify-policy-input";
import type { NotifyAlertSource, NotifyRule, NotifyRuleBody, NotifySeverity } from "@/types/notify";

/** 规则可以覆盖的三个时间参数 */
export const RULE_TIMING_FIELDS = ["groupWaitSeconds", "groupIntervalSeconds", "repeatIntervalSeconds"] as const;
export type RuleTimingField = (typeof RULE_TIMING_FIELDS)[number];

/** 表单里正在编辑的规则。三个时间参数以文本保存，空串 = 不覆盖，用策略里的全局值 */
export interface NotifyRuleDraft {
  name: string;
  enabled: boolean;
  clusterIds: string[];
  sources: NotifyAlertSource[];
  minSeverity: NotifySeverity;
  receiverIds: number[];
  groupWaitSeconds: string;
  groupIntervalSeconds: string;
  repeatIntervalSeconds: string;
}

function timingText(seconds: number | null): string {
  return seconds === null ? "" : String(seconds);
}

/** 用已有的规则填表单；不传 = 新建规则的空表单（启用、不限集群与来源、最低严重度 low） */
export function ruleDraftFrom(rule?: NotifyRule): NotifyRuleDraft {
  if (!rule) {
    return {
      name: "",
      enabled: true,
      clusterIds: [],
      sources: [],
      minSeverity: "low",
      receiverIds: [],
      groupWaitSeconds: "",
      groupIntervalSeconds: "",
      repeatIntervalSeconds: "",
    };
  }
  return {
    name: rule.name,
    enabled: rule.enabled,
    clusterIds: rule.clusterIds ?? [],
    sources: rule.sources ?? [],
    minSeverity: rule.minSeverity,
    receiverIds: rule.receiverIds ?? [],
    groupWaitSeconds: timingText(rule.groupWaitSeconds),
    groupIntervalSeconds: timingText(rule.groupIntervalSeconds),
    repeatIntervalSeconds: timingText(rule.repeatIntervalSeconds),
  };
}

export type NotifyRuleProblem = "nameRequired" | RuleTimingField;

/**
 * 规则表单哪里不合规（空数组 = 合规）。
 *
 * 时间参数留空是合法的（用全局值）；填了就按策略里同名项的范围检查。
 */
export function ruleDraftProblems(draft: NotifyRuleDraft): NotifyRuleProblem[] {
  const problems: NotifyRuleProblem[] = [];
  if (draft.name.trim() === "") problems.push("nameRequired");
  for (const field of RULE_TIMING_FIELDS) {
    if (draft[field].trim() !== "" && !isIntegerInRange(draft[field], NOTIFY_POLICY_RANGES[field])) {
      problems.push(field);
    }
  }
  return problems;
}

/** 把校验过的草稿变成请求体。空的时间参数发 null */
export function ruleBodyFromDraft(draft: NotifyRuleDraft): NotifyRuleBody {
  return {
    name: draft.name.trim(),
    enabled: draft.enabled,
    clusterIds: draft.clusterIds,
    sources: draft.sources,
    minSeverity: draft.minSeverity,
    receiverIds: draft.receiverIds,
    groupWaitSeconds: parseInteger(draft.groupWaitSeconds),
    groupIntervalSeconds: parseInteger(draft.groupIntervalSeconds),
    repeatIntervalSeconds: parseInteger(draft.repeatIntervalSeconds),
  };
}

export type RuleMoveDirection = "up" | "down";

/**
 * 把一条规则上移 / 下移一位后的完整 id 顺序（交给 PUT /rules/order）。
 *
 * orderedRuleIds 是当前顺序。已经在最前 / 最后、或 id 不在列表里时返回 null：没有可做的移动。
 */
export function movedRuleIds(orderedRuleIds: number[], ruleId: number, direction: RuleMoveDirection): number[] | null {
  const from = orderedRuleIds.indexOf(ruleId);
  const to = direction === "up" ? from - 1 : from + 1;
  if (from < 0 || to < 0 || to >= orderedRuleIds.length) return null;
  const reordered = [...orderedRuleIds];
  [reordered[from], reordered[to]] = [reordered[to], reordered[from]];
  return reordered;
}
