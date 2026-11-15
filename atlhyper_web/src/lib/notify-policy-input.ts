/**
 * 通知策略与规则里数值项的校验（纯函数）
 *
 * 范围与设计文档（incident-notification-design.md 功能一）一致。前端先拦一道只是为了就地提示；
 * 后端仍然是唯一的裁决方。
 */
import type { NotifyPolicy } from "@/types/notify";

export interface IntegerRange {
  min: number;
  max: number;
}

/** 策略里各数值项的取值范围（含两端） */
export const NOTIFY_POLICY_RANGES = {
  warningFlapThreshold: { min: 0, max: 20 },
  groupWaitSeconds: { min: 0, max: 600 },
  groupIntervalSeconds: { min: 30, max: 86_400 },
  repeatIntervalSeconds: { min: 300, max: 604_800 },
} as const satisfies Record<string, IntegerRange>;

export type NotifyPolicyNumberField = keyof typeof NOTIFY_POLICY_RANGES;

export const NOTIFY_POLICY_NUMBER_FIELDS = Object.keys(NOTIFY_POLICY_RANGES) as NotifyPolicyNumberField[];

/** 表单里数值项以文本保存：输入框清空、输到一半都要能表示 */
export type NotifyPolicyNumberDraft = Record<NotifyPolicyNumberField, string>;

/**
 * 把输入框里的文本读成整数；不是整数返回 null。
 *
 * 只认十进制整数：「1.5」「1e3」「 」「abc」都不算，不做四舍五入之类的自动修正。
 */
export function parseInteger(text: string): number | null {
  const trimmed = text.trim();
  if (!/^-?\d+$/.test(trimmed)) return null;
  const parsed = Number(trimmed);
  return Number.isSafeInteger(parsed) ? parsed : null;
}

/** 文本是不是落在范围内的整数 */
export function isIntegerInRange(text: string, range: IntegerRange): boolean {
  const parsed = parseInteger(text);
  return parsed !== null && parsed >= range.min && parsed <= range.max;
}

/** 策略表单里哪些数值项不合规（空数组 = 全部合规） */
export function policyNumberProblems(draft: NotifyPolicyNumberDraft): NotifyPolicyNumberField[] {
  return NOTIFY_POLICY_NUMBER_FIELDS.filter((field) => !isIntegerInRange(draft[field], NOTIFY_POLICY_RANGES[field]));
}

/** 用已保存的策略填数值项的输入框 */
export function policyNumberDraft(policy: NotifyPolicy): NotifyPolicyNumberDraft {
  return {
    warningFlapThreshold: String(policy.warningFlapThreshold),
    groupWaitSeconds: String(policy.groupWaitSeconds),
    groupIntervalSeconds: String(policy.groupIntervalSeconds),
    repeatIntervalSeconds: String(policy.repeatIntervalSeconds),
  };
}

/**
 * 把校验过的数值项并回策略。调用前必须先确认 policyNumberProblems 为空；
 * 读不出整数的项保持原值（不会走到，只是不让类型里出现 null）。
 */
export function policyWithNumbers(policy: NotifyPolicy, draft: NotifyPolicyNumberDraft): NotifyPolicy {
  const merged = { ...policy };
  for (const field of NOTIFY_POLICY_NUMBER_FIELDS) {
    merged[field] = parseInteger(draft[field]) ?? policy[field];
  }
  return merged;
}
