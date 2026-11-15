/**
 * 通知设置页的取数与写操作描述：策略、当前告警、送达记录、静默
 *
 * 没有 datasource：工厂直接调用 api 层。需要 notify.manage —— 有没有权限由调用方用 useCan 得到后传进来；
 * 没有权限时页面整页显示演示数据（由页面自己取，不经过这里）。
 *
 * 接收人与路由规则在 notify-routing.ts。
 */
import { keepPreviousData, mutationOptions, queryOptions, type QueryClient } from "@tanstack/react-query";
import {
  createNotifySilence,
  expireNotifySilence,
  getNotifyAlerts,
  getNotifyDeliveries,
  getNotifyPolicy,
  getNotifySilences,
  updateNotifyPolicy,
  type NotifyAlertListParams,
  type NotifyDeliveryListParams,
} from "@/api/notify";
import type { NotifyPolicy, NotifySilenceBody, NotifySilenceState } from "@/types/notify";
import { notifyKeys } from "./keys";
import { REFRESH } from "./refresh";

// ============================================================
// 策略
// ============================================================

/** 全局通知策略。只有人改它才会变，不轮询 */
export function notifyPolicyQuery(canManageNotifications: boolean) {
  return queryOptions({
    queryKey: notifyKeys.policy(),
    queryFn: async ({ signal }) => (await getNotifyPolicy(signal)).data.data,
    enabled: canManageNotifications,
  });
}

/** 保存策略。接口带回了保存后的策略：直接换进缓存，不必再请求一次 */
export function saveNotifyPolicyMutation() {
  return mutationOptions({
    mutationFn: async (policy: NotifyPolicy) => (await updateNotifyPolicy(policy)).data.data,
    onSuccess: (saved, _policy, _onMutateResult, context) => {
      context.client.setQueryData(notifyKeys.policy(), saved);
    },
  });
}

// ============================================================
// 当前告警与送达记录
// ============================================================

/**
 * 告警列表（按状态筛选 + 分页）。
 *
 * 慢速轮询：告警会自己触发、结束、被再次通知。翻页、换状态时先留着上一份结果，界面不闪。
 */
export function notifyAlertsQuery(filter: NotifyAlertListParams, canManageNotifications: boolean) {
  return queryOptions({
    queryKey: notifyKeys.alerts(filter),
    queryFn: async ({ signal }) => {
      const page = (await getNotifyAlerts(filter, signal)).data;
      return { items: page.data ?? [], total: page.total ?? 0 };
    },
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
    placeholderData: keepPreviousData,
    enabled: canManageNotifications,
  });
}

/** 送达记录（按状态筛选 + 分页）。慢速轮询：等待重试的记录会自己变成成功 / 已放弃 */
export function notifyDeliveriesQuery(filter: NotifyDeliveryListParams, canManageNotifications: boolean) {
  return queryOptions({
    queryKey: notifyKeys.deliveries(filter),
    queryFn: async ({ signal }) => {
      const page = (await getNotifyDeliveries(filter, signal)).data;
      return { items: page.data ?? [], total: page.total ?? 0 };
    },
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
    placeholderData: keepPreviousData,
    enabled: canManageNotifications,
  });
}

// ============================================================
// 静默
// ============================================================

/** 静默变了（新建 / 提前结束），让各个状态下的静默列表重新取数 */
function invalidateSilences(client: QueryClient): void {
  void client.invalidateQueries({ queryKey: notifyKeys.silenceLists });
}

/** 某个状态下的静默。慢速轮询：状态由时间算出，未开始的会自己变成生效中、再变成已过期 */
export function notifySilencesQuery(state: NotifySilenceState, canManageNotifications: boolean) {
  return queryOptions({
    queryKey: notifyKeys.silences(state),
    queryFn: async ({ signal }) => (await getNotifySilences(state, signal)).data.data ?? [],
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
    enabled: canManageNotifications,
  });
}

/** 新建静默；返回建好的那一条。起止时间、时长、备注不合规时后端返回 400，错误原样抛给调用方 */
export function createNotifySilenceMutation() {
  return mutationOptions({
    mutationFn: async (silence: NotifySilenceBody) => (await createNotifySilence(silence)).data.data,
    onSuccess: (_created, _silence, _onMutateResult, context) => invalidateSilences(context.client),
  });
}

/** 提前结束某个静默（结束时间置为现在） */
export function expireNotifySilenceMutation() {
  return mutationOptions({
    mutationFn: async (silenceId: number) => {
      await expireNotifySilence(silenceId);
    },
    onSuccess: (_nothing, _silenceId, _onMutateResult, context) => invalidateSilences(context.client),
  });
}
