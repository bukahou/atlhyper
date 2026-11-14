/**
 * 通知设置页的取数与写操作描述
 *
 * 没有 datasource：工厂直接调用 api 层。需要 notify.manage —— 有没有权限由调用方用 useCan 得到后传进来；
 * 没有权限时页面整页显示演示数据（由页面自己取，不经过这里）。
 */
import { mutationOptions, queryOptions, type QueryClient } from "@tanstack/react-query";
import {
  listChannels,
  testChannel,
  updateEmail,
  updateSlack,
  type ChannelType,
  type EmailUpdateData,
  type NotifyChannel,
} from "@/api/notify";
import { notifyKeys } from "./keys";

/** 全部通知渠道（Slack / 邮件）。凭据不回显，是否已设置看 configuredSecrets */
export function notifyChannelsQuery(canManageNotifications: boolean) {
  return queryOptions({
    queryKey: notifyKeys.channels(),
    queryFn: async ({ signal }) => (await listChannels(signal)).data.channels ?? [],
    enabled: canManageNotifications,
  });
}

/** 接口带回了保存后的渠道：直接换进列表，不必再请求一次 */
function replaceChannel(client: QueryClient, saved: NotifyChannel): void {
  client.setQueryData<NotifyChannel[]>(notifyKeys.channels(), (channels) =>
    channels?.map((channel) => (channel.type === saved.type ? saved : channel)),
  );
}

/** 保存 Slack 配置。不带 webhookUrl = 保持已存的地址不变 */
export function updateSlackChannelMutation() {
  return mutationOptions({
    mutationFn: async (changes: { enabled?: boolean; webhookUrl?: string }) => (await updateSlack(changes)).data,
    onSuccess: (saved, _changes, _onMutateResult, context) => replaceChannel(context.client, saved),
  });
}

/** 保存邮件配置。不带 smtpPassword = 保持已存的密码不变 */
export function updateEmailChannelMutation() {
  return mutationOptions({
    mutationFn: async (changes: EmailUpdateData) => (await updateEmail(changes)).data,
    onSuccess: (saved, _changes, _onMutateResult, context) => replaceChannel(context.client, saved),
  });
}

/** 按已保存的配置发一条测试消息。发送失败不抛错，结果里带原因 */
export function testNotifyChannelMutation() {
  return mutationOptions({
    mutationFn: (channelType: ChannelType) => testChannel(channelType),
  });
}
