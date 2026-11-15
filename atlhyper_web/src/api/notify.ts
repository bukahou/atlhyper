/**
 * 通知 API（均需要 notify.manage）
 *
 * 策略、当前告警、送达记录、接收人、全局发件设置、路由规则、静默。
 * 接口规格见 docs/design/active/incident-notification-design.md 功能一 / 二（含「全局发件设置」）/ 三。
 * 类型在 types/notify.ts。
 */

import { del, get, post, put } from "./request";
import type {
  NotifyAlert,
  NotifyAlertStatus,
  NotifyDelivery,
  NotifyDeliveryStatus,
  NotifyPolicy,
  NotifyReceiver,
  NotifyReceiverBody,
  NotifyRule,
  NotifyRuleBody,
  NotifyRuleMatchRequest,
  NotifyRuleMatchResult,
  NotifySilence,
  NotifySilenceBody,
  NotifySilenceState,
  NotifySMTP,
  NotifySMTPBody,
} from "@/types/notify";

// ============================================================
// 查询参数
// ============================================================

/** 告警列表的筛选条件 + 分页 */
export interface NotifyAlertListParams {
  status: NotifyAlertStatus;
  limit: number;
  offset: number;
}

/** 送达记录的筛选条件 + 分页。status 为空串 = 全部状态 */
export interface NotifyDeliveryListParams {
  status: NotifyDeliveryStatus | "";
  limit: number;
  offset: number;
}

// ============================================================
// 响应类型
// ============================================================

interface ItemResponse<T> {
  message: string;
  data: T;
}

/** 没有任何记录时 Go 的 nil 切片会序列化成 null */
interface ListResponse<T> {
  message: string;
  data: T[] | null;
  total: number;
}

interface MessageResponse {
  message: string;
}

/** 路径里的数字 id。id 来自接口返回值，这里转义是不信任调用方传来的值 */
function idSegment(id: number): string {
  return encodeURIComponent(String(id));
}

// ============================================================
// 策略
// ============================================================

/** GET /api/v2/notify/policy */
export function getNotifyPolicy(signal?: AbortSignal) {
  return get<ItemResponse<NotifyPolicy>>("/api/v2/notify/policy", undefined, signal);
}

/** PUT /api/v2/notify/policy（取值超出范围 400）；返回保存后的策略 */
export function updateNotifyPolicy(policy: NotifyPolicy) {
  return put<ItemResponse<NotifyPolicy>>("/api/v2/notify/policy", policy);
}

// ============================================================
// 告警与送达记录
// ============================================================

/** GET /api/v2/notify/alerts?status=&limit=&offset= */
export function getNotifyAlerts(params: NotifyAlertListParams, signal?: AbortSignal) {
  return get<ListResponse<NotifyAlert>>("/api/v2/notify/alerts", params, signal);
}

/** GET /api/v2/notify/deliveries?status=&limit=&offset= */
export function getNotifyDeliveries(params: NotifyDeliveryListParams, signal?: AbortSignal) {
  const { status, limit, offset } = params;
  // 不筛选时不带 status：空值参数由后端按「不限」处理，干脆不发
  return get<ListResponse<NotifyDelivery>>(
    "/api/v2/notify/deliveries",
    status === "" ? { limit, offset } : { status, limit, offset },
    signal,
  );
}

// ============================================================
// 接收人
// ============================================================

/** GET /api/v2/notify/receivers（凭据不返回，只有「是否已配置」） */
export function getNotifyReceivers(signal?: AbortSignal) {
  return get<ListResponse<NotifyReceiver>>("/api/v2/notify/receivers", undefined, signal);
}

/** POST /api/v2/notify/receivers（成功 201；名称重复 409） */
export function createNotifyReceiver(receiver: NotifyReceiverBody) {
  return post<ItemResponse<NotifyReceiver>>("/api/v2/notify/receivers", receiver);
}

/** PUT /api/v2/notify/receivers/{id}（webhookUrl 留空 = 保持原值） */
export function updateNotifyReceiver(receiverId: number, receiver: NotifyReceiverBody) {
  return put<ItemResponse<NotifyReceiver>>(`/api/v2/notify/receivers/${idSegment(receiverId)}`, receiver);
}

/** DELETE /api/v2/notify/receivers/{id}（仍被规则或默认策略引用 409，message 写明被谁引用） */
export function deleteNotifyReceiver(receiverId: number) {
  return del<MessageResponse>(`/api/v2/notify/receivers/${idSegment(receiverId)}`);
}

/** POST /api/v2/notify/receivers/{id}/test（按已保存的配置发一条测试消息；发送失败 502，带原因） */
export function testNotifyReceiver(receiverId: number) {
  return post<MessageResponse>(`/api/v2/notify/receivers/${idSegment(receiverId)}/test`);
}

// ============================================================
// 全局发件设置
// ============================================================

/** GET /api/v2/notify/smtp（密码不返回，只有「是否已配置」） */
export function getNotifySMTP(signal?: AbortSignal) {
  return get<ItemResponse<NotifySMTP>>("/api/v2/notify/smtp", undefined, signal);
}

/** PUT /api/v2/notify/smtp（password 留空 = 保持原值；第一次保存不填密码 400）；返回保存后的设置 */
export function updateNotifySMTP(smtp: NotifySMTPBody) {
  return put<ItemResponse<NotifySMTP>>("/api/v2/notify/smtp", smtp);
}

// ============================================================
// 路由规则
// ============================================================

/** GET /api/v2/notify/rules */
export function getNotifyRules(signal?: AbortSignal) {
  return get<ListResponse<NotifyRule>>("/api/v2/notify/rules", undefined, signal);
}

/** POST /api/v2/notify/rules（不带 id / position；新规则排在最后） */
export function createNotifyRule(rule: NotifyRuleBody) {
  return post<ItemResponse<NotifyRule>>("/api/v2/notify/rules", rule);
}

/** PUT /api/v2/notify/rules/{id} */
export function updateNotifyRule(ruleId: number, rule: NotifyRuleBody) {
  return put<ItemResponse<NotifyRule>>(`/api/v2/notify/rules/${idSegment(ruleId)}`, rule);
}

/** DELETE /api/v2/notify/rules/{id} */
export function deleteNotifyRule(ruleId: number) {
  return del<MessageResponse>(`/api/v2/notify/rules/${idSegment(ruleId)}`);
}

/** PUT /api/v2/notify/rules/order（ids 必须恰好是全部规则的一个排列） */
export function reorderNotifyRules(ruleIds: number[]) {
  return put<MessageResponse>("/api/v2/notify/rules/order", { ids: ruleIds });
}

/** POST /api/v2/notify/rules/match（试匹配：假想一条告警，看命中哪条规则、发给谁） */
export function matchNotifyRules(alert: NotifyRuleMatchRequest) {
  return post<ItemResponse<NotifyRuleMatchResult>>("/api/v2/notify/rules/match", alert);
}

// ============================================================
// 静默
// ============================================================

/** GET /api/v2/notify/silences?state=active|pending|expired */
export function getNotifySilences(state: NotifySilenceState, signal?: AbortSignal) {
  return get<ListResponse<NotifySilence>>("/api/v2/notify/silences", { state }, signal);
}

/** POST /api/v2/notify/silences（成功 201；起止时间、时长、备注不合规 400） */
export function createNotifySilence(silence: NotifySilenceBody) {
  return post<ItemResponse<NotifySilence>>("/api/v2/notify/silences", silence);
}

/** POST /api/v2/notify/silences/{id}/expire（提前结束：endsAt 置为现在） */
export function expireNotifySilence(silenceId: number) {
  return post<MessageResponse>(`/api/v2/notify/silences/${idSegment(silenceId)}/expire`);
}
