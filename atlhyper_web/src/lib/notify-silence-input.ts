/**
 * 静默表单的草稿与校验（纯函数）
 *
 * 规则与后端校验一致：
 * 结束晚于开始、结束晚于现在、时长不超过 30 天、备注必填且不超过 256 字。
 */
import type { NotifySilenceBody } from "@/types/notify";

export const SILENCE_MAX_DURATION_MS = 30 * 24 * 60 * 60 * 1000;
export const SILENCE_COMMENT_MAX_CHARS = 256;
/** 新建静默时默认的时长：两小时（一次常见的维护窗口） */
const DEFAULT_SILENCE_DURATION_MS = 2 * 60 * 60 * 1000;

/** 表单里正在编辑的静默。起止时间是 datetime-local 输入框的值（本地时间，YYYY-MM-DDTHH:mm） */
export interface NotifySilenceDraft {
  clusterId: string;
  namespace: string;
  subject: string;
  startsAtLocal: string;
  endsAtLocal: string;
  comment: string;
}

function twoDigits(part: number): string {
  return String(part).padStart(2, "0");
}

/** 把一个时刻写成 datetime-local 输入框认的形式（本地时间，精确到分钟） */
export function toLocalInputValue(moment: Date): string {
  const day = `${moment.getFullYear()}-${twoDigits(moment.getMonth() + 1)}-${twoDigits(moment.getDate())}`;
  return `${day}T${twoDigits(moment.getHours())}:${twoDigits(moment.getMinutes())}`;
}

/** 读 datetime-local 输入框的值（按本地时间）；空或读不出来返回 null */
export function parseLocalInputValue(localValue: string): Date | null {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?$/.test(localValue)) return null;
  const parsed = new Date(localValue);
  return Number.isNaN(parsed.getTime()) ? null : parsed;
}

/** 新建静默的空表单：从现在开始、两小时后结束，范围不限 */
export function newSilenceDraft(now: Date): NotifySilenceDraft {
  return {
    clusterId: "",
    namespace: "",
    subject: "",
    startsAtLocal: toLocalInputValue(now),
    endsAtLocal: toLocalInputValue(new Date(now.getTime() + DEFAULT_SILENCE_DURATION_MS)),
    comment: "",
  };
}

export type NotifySilenceProblem =
  | "startInvalid"
  | "endInvalid"
  | "endNotAfterStart"
  | "endInPast"
  | "tooLong"
  | "commentRequired"
  | "commentTooLong";

/** 按字符数（码点）计长度：一个汉字算一个字 */
function characterCount(text: string): number {
  return Array.from(text).length;
}

/** 静默表单哪里不合规（空数组 = 合规）。now 由调用方传入，便于测试 */
export function silenceDraftProblems(draft: NotifySilenceDraft, now: Date): NotifySilenceProblem[] {
  const problems: NotifySilenceProblem[] = [];
  const startsAt = parseLocalInputValue(draft.startsAtLocal);
  const endsAt = parseLocalInputValue(draft.endsAtLocal);
  if (!startsAt) problems.push("startInvalid");
  if (!endsAt) problems.push("endInvalid");
  if (startsAt && endsAt) {
    if (endsAt.getTime() <= startsAt.getTime()) problems.push("endNotAfterStart");
    else if (endsAt.getTime() - startsAt.getTime() > SILENCE_MAX_DURATION_MS) problems.push("tooLong");
  }
  if (endsAt && endsAt.getTime() <= now.getTime()) problems.push("endInPast");

  const comment = draft.comment.trim();
  if (comment === "") problems.push("commentRequired");
  else if (characterCount(comment) > SILENCE_COMMENT_MAX_CHARS) problems.push("commentTooLong");
  return problems;
}

/**
 * 把校验过的草稿变成请求体：起止时间转成 RFC 3339（UTC），文本去掉首尾空白。
 * 调用前必须先确认 silenceDraftProblems 为空；起止时间读不出来时返回 null。
 */
export function silenceBodyFromDraft(draft: NotifySilenceDraft): NotifySilenceBody | null {
  const startsAt = parseLocalInputValue(draft.startsAtLocal);
  const endsAt = parseLocalInputValue(draft.endsAtLocal);
  if (!startsAt || !endsAt) return null;
  return {
    clusterId: draft.clusterId.trim(),
    namespace: draft.namespace.trim(),
    subject: draft.subject.trim(),
    startsAt: startsAt.toISOString(),
    endsAt: endsAt.toISOString(),
    comment: draft.comment.trim(),
  };
}
