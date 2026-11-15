/**
 * 通知设置页的取数与写操作描述：接收人、路由规则
 *
 * 约定同 notify.ts。接收人的凭据（Webhook 地址、SMTP 密码）只写不读：
 * 接口不返回，这里也不让明文留在查询缓存或 mutation 缓存里。
 */
import { mutationOptions, queryOptions, type QueryClient } from "@tanstack/react-query";
import {
  createNotifyReceiver,
  createNotifyRule,
  deleteNotifyReceiver,
  deleteNotifyRule,
  getNotifyReceivers,
  getNotifyRules,
  matchNotifyRules,
  reorderNotifyRules,
  testNotifyReceiver,
  updateNotifyReceiver,
  updateNotifyRule,
} from "@/api/notify";
import { receiverRequestBody, type NotifyReceiverSettings } from "@/lib/notify-receiver-input";
import type { NotifyRuleBody, NotifyRuleMatchRequest } from "@/types/notify";
import { notifyKeys } from "./keys";

// ============================================================
// 接收人
// ============================================================

/** 让接收人列表重新取数。不等它取完：写操作已经成功，按钮不必为了刷新继续转圈 */
function invalidateReceivers(client: QueryClient): void {
  void client.invalidateQueries({ queryKey: notifyKeys.receivers() });
}

/** 全部接收人。凭据不回显，只有「是否已配置」 */
export function notifyReceiversQuery(canManageNotifications: boolean) {
  return queryOptions({
    queryKey: notifyKeys.receivers(),
    queryFn: async ({ signal }) => (await getNotifyReceivers(signal)).data.data ?? [],
    enabled: canManageNotifications,
  });
}

export interface NotifyReceiverSave {
  /** 要修改的接收人；不带 = 新建 */
  receiverId?: number;
  /** 不含凭据的部分 */
  settings: NotifyReceiverSettings;
  /**
   * 取凭据明文（Slack 的 Webhook 地址 / 邮件的 SMTP 密码）；修改时取到空串 = 保持原值。
   * 只在发请求的那一刻被调用一次。用 lib/notify-receiver-input 的 sealCredential 生成：取过即清空。
   */
  takeCredential: () => string;
}

/**
 * 新建或修改接收人；返回保存后的那一行（不含凭据）。名称重复时后端返回 409，错误原样抛给调用方。
 *
 * ⛔ 凭据明文不直接作为 mutation 的参数：参数会留在 MutationCache 里（页面还挂着就一直在），
 * 开发者工具与任何拿得到 QueryClient 的代码都读得到。所以参数里只有取值函数，
 * 明文在取数函数里拼进请求体，用完即弃。
 */
export function saveNotifyReceiverMutation() {
  return mutationOptions({
    mutationFn: async ({ receiverId, settings, takeCredential }: NotifyReceiverSave) => {
      const body = receiverRequestBody(settings, takeCredential());
      const response =
        receiverId === undefined ? await createNotifyReceiver(body) : await updateNotifyReceiver(receiverId, body);
      return response.data.data;
    },
    onSuccess: (_saved, _save, _onMutateResult, context) => invalidateReceivers(context.client),
  });
}

/** 删除接收人。仍被规则或默认策略引用时后端返回 409 并写明被谁引用，错误原样抛给调用方 */
export function deleteNotifyReceiverMutation() {
  return mutationOptions({
    mutationFn: async (receiverId: number) => {
      await deleteNotifyReceiver(receiverId);
    },
    onSuccess: (_nothing, _receiverId, _onMutateResult, context) => invalidateReceivers(context.client),
  });
}

/** 按已保存的配置给某个接收人发一条测试消息。发不出去时后端返回 502 并带原因，错误原样抛给调用方 */
export function testNotifyReceiverMutation() {
  return mutationOptions({
    mutationFn: async (receiverId: number) => {
      await testNotifyReceiver(receiverId);
    },
  });
}

// ============================================================
// 路由规则
// ============================================================

function invalidateRules(client: QueryClient): void {
  void client.invalidateQueries({ queryKey: notifyKeys.rules() });
}

/**
 * 全部路由规则，按 position 从小到大。
 *
 * 匹配顺序就是这个顺序，上移 / 下移也按它算，所以这里按 position 排一次：
 * 不依赖接口返回的先后（契约没有写明）。
 */
export function notifyRulesQuery(canManageNotifications: boolean) {
  return queryOptions({
    queryKey: notifyKeys.rules(),
    queryFn: async ({ signal }) => {
      const rules = (await getNotifyRules(signal)).data.data ?? [];
      return [...rules].sort((earlier, later) => earlier.position - later.position);
    },
    enabled: canManageNotifications,
  });
}

export interface NotifyRuleSave {
  /** 要修改的规则；不带 = 新建（排在最后） */
  ruleId?: number;
  rule: NotifyRuleBody;
}

/** 新建或修改规则；返回保存后的那一条 */
export function saveNotifyRuleMutation() {
  return mutationOptions({
    mutationFn: async ({ ruleId, rule }: NotifyRuleSave) => {
      const response = ruleId === undefined ? await createNotifyRule(rule) : await updateNotifyRule(ruleId, rule);
      return response.data.data;
    },
    onSuccess: (_saved, _save, _onMutateResult, context) => invalidateRules(context.client),
  });
}

/** 删除规则 */
export function deleteNotifyRuleMutation() {
  return mutationOptions({
    mutationFn: async (ruleId: number) => {
      await deleteNotifyRule(ruleId);
    },
    onSuccess: (_nothing, _ruleId, _onMutateResult, context) => invalidateRules(context.client),
  });
}

/** 调整规则顺序。orderedRuleIds 必须恰好是全部规则的一个排列（用 lib/notify-rule-input 的 movedRuleIds 得到） */
export function reorderNotifyRulesMutation() {
  return mutationOptions({
    mutationFn: async (orderedRuleIds: number[]) => {
      await reorderNotifyRules(orderedRuleIds);
    },
    onSuccess: (_nothing, _orderedRuleIds, _onMutateResult, context) => invalidateRules(context.client),
  });
}

/** 试匹配：假想一条告警，看它命中哪条规则、会发给谁。只读，但每次都要现算，所以不进查询缓存 */
export function matchNotifyRulesMutation() {
  return mutationOptions({
    mutationFn: async (alert: NotifyRuleMatchRequest) => (await matchNotifyRules(alert)).data.data,
  });
}
