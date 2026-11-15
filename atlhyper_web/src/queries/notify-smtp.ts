/**
 * 通知设置页的取数与写操作描述：全局发件设置（邮件接收人共用的 SMTP）
 *
 * 约定同 notify.ts。SMTP 密码只写不读：接口不返回，这里也不让明文留在查询缓存或 mutation 缓存里
 * （做法与接收人的 Webhook 地址相同，见 notify-routing.ts）。
 */
import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { getNotifySMTP, updateNotifySMTP } from "@/api/notify";
import type { NotifySMTPBody, NotifySMTPSettings } from "@/types/notify";
import { notifyKeys } from "./keys";

/** 全局发件设置。只有人改它才会变，不轮询；密码不回显，只有「是否已配置」 */
export function notifySMTPQuery(canManageNotifications: boolean) {
  return queryOptions({
    queryKey: notifyKeys.smtp(),
    queryFn: async ({ signal }) => (await getNotifySMTP(signal)).data.data,
    enabled: canManageNotifications,
  });
}

export interface NotifySMTPSave {
  /** 不含密码的部分 */
  settings: NotifySMTPSettings;
  /**
   * 取密码明文；取到空串 = 保持原值。只在发请求的那一刻被调用一次。
   * 用 lib/notify-receiver-input 的 sealCredential 生成：取过即清空。
   */
  takePassword: () => string;
}

/** 拼请求体：只挑契约里的五项（调用方传来的对象可能带着 configured 之类的只读字段） */
function smtpRequestBody(settings: NotifySMTPSettings, password: string): NotifySMTPBody {
  const { host, port, user, tls, fromAddress } = settings;
  return { host, port, user, tls, fromAddress, password };
}

/**
 * 保存全局发件设置；返回保存后的设置（不含密码）。第一次保存没填密码时后端返回 400，错误原样抛给调用方。
 *
 * ⛔ 密码明文不直接作为 mutation 的参数（理由同 saveNotifyReceiverMutation）。
 * 接口带回了保存后的设置，里面没有密码：直接换进缓存，不必再请求一次。
 */
export function saveNotifySMTPMutation() {
  return mutationOptions({
    mutationFn: async ({ settings, takePassword }: NotifySMTPSave) =>
      (await updateNotifySMTP(smtpRequestBody(settings, takePassword()))).data.data,
    onSuccess: (saved, _save, _onMutateResult, context) => {
      context.client.setQueryData(notifyKeys.smtp(), saved);
    },
  });
}
