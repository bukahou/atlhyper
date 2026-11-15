/**
 * 接收人的保存内容与请求体（纯函数）
 *
 * 凭据（Webhook 地址、SMTP 密码）只写不读：这里把「可以读回、可以留存的部分」与「凭据明文」分开，
 * 明文只在拼请求体的那一刻出现。
 */
import type { NotifyEmailSettings, NotifyReceiver, NotifyReceiverBody } from "@/types/notify";

/** 一个接收人里不含凭据的部分：可以放进表单初值、mutation 的参数与缓存 */
export type NotifyReceiverSettings =
  | { name: string; type: "slack"; enabled: boolean }
  | { name: string; type: "email"; enabled: boolean; email: NotifyEmailSettings };

/** 还没配置过邮件时的空设置：端口 587、开启 TLS */
export const EMPTY_EMAIL_SETTINGS: NotifyEmailSettings = {
  smtpHost: "",
  smtpPort: 587,
  smtpUser: "",
  smtpTLS: true,
  fromAddress: "",
  toAddresses: [],
};

/** 从列表里的一行取出不含凭据的设置（「是否已配置」这类只读标记不带） */
export function receiverSettingsOf(receiver: NotifyReceiver): NotifyReceiverSettings {
  if (receiver.type === "slack") {
    return { name: receiver.name, type: "slack", enabled: receiver.enabled };
  }
  const saved = receiver.email ?? EMPTY_EMAIL_SETTINGS;
  return {
    name: receiver.name,
    type: "email",
    enabled: receiver.enabled,
    email: {
      smtpHost: saved.smtpHost,
      smtpPort: saved.smtpPort,
      smtpUser: saved.smtpUser,
      smtpTLS: saved.smtpTLS,
      fromAddress: saved.fromAddress,
      toAddresses: saved.toAddresses ?? [],
    },
  };
}

/**
 * 拼出新建 / 修改接收人的请求体。
 *
 * credential 是凭据明文：Slack 的 Webhook 地址，或邮件的 SMTP 密码。
 * 修改时传空串 = 保持已存的凭据不变（接口约定）。
 */
export function receiverRequestBody(settings: NotifyReceiverSettings, credential: string): NotifyReceiverBody {
  if (settings.type === "slack") {
    return { name: settings.name, type: "slack", enabled: settings.enabled, slack: { webhookUrl: credential } };
  }
  return {
    name: settings.name,
    type: "email",
    enabled: settings.enabled,
    email: { ...settings.email, smtpPassword: credential },
  };
}

/**
 * 把凭据明文装进一次性的盒子：取一次即清空。
 *
 * mutation 的参数会留在 MutationCache 里（页面还挂着就一直在），所以参数里不直接放明文，
 * 放这个取值函数；请求发出后盒子就空了，缓存里只剩一个取不出东西的函数。
 */
export function sealCredential(plaintext: string): () => string {
  let held = plaintext;
  return () => {
    const taken = held;
    held = "";
    return taken;
  };
}
