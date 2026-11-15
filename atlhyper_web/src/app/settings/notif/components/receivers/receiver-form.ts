/**
 * 接收人表单的草稿与校验（纯函数）
 */
import { EMPTY_EMAIL_SETTINGS, type NotifyReceiverSettings } from "@/lib/notify-receiver-input";
import type { NotifyReceiverProblemKey } from "@/types/i18n-notify";
import type { NotifyReceiver, NotifyReceiverType } from "@/types/notify";
import { emailFormFromSaved, emailSettingsFromDraft, type EmailFormDraft } from "../email-form";

const MAX_PORT = 65_535;

/** 表单里正在编辑的接收人。两种类型的字段都留着：切换类型时已填的内容不丢 */
export interface ReceiverFormDraft {
  name: string;
  type: NotifyReceiverType;
  enabled: boolean;
  /** 只放用户新输入的 Webhook 地址：已存的不回显，留空 = 保持不变 */
  webhookUrl: string;
  email: EmailFormDraft;
}

/** 用已有的接收人填表单；editing 为 null = 新建（默认 Slack、启用）。凭据恒为空 */
export function receiverFormFrom(editing: NotifyReceiver | null, presetHosts: string[]): ReceiverFormDraft {
  return {
    name: editing?.name ?? "",
    type: editing?.type ?? "slack",
    enabled: editing?.enabled ?? true,
    webhookUrl: "",
    email: emailFormFromSaved(editing?.email ?? EMPTY_EMAIL_SETTINGS, presetHosts),
  };
}

/**
 * 表单哪里不合规（空数组 = 合规）。
 *
 * 凭据在新建时必填；编辑时已配置过就可以留空（保持原值）。
 */
export function receiverFormProblems(draft: ReceiverFormDraft, editing: NotifyReceiver | null): NotifyReceiverProblemKey[] {
  const problems: NotifyReceiverProblemKey[] = [];
  if (draft.name.trim() === "") problems.push("nameRequired");

  if (draft.type === "slack") {
    const webhookAlreadySet = editing?.slack?.webhookConfigured === true;
    if (draft.webhookUrl.trim() === "" && !webhookAlreadySet) problems.push("webhookRequired");
    return problems;
  }

  const { smtpHost, smtpPort, smtpUser, smtpPassword, recipients } = draft.email;
  if (smtpHost.trim() === "") problems.push("smtpHostRequired");
  if (!Number.isInteger(smtpPort) || smtpPort < 1 || smtpPort > MAX_PORT) problems.push("smtpPortInvalid");
  if (smtpUser.trim() === "") problems.push("smtpUserRequired");
  const passwordAlreadySet = editing?.email?.passwordConfigured === true;
  if (smtpPassword === "" && !passwordAlreadySet) problems.push("passwordRequired");
  if (recipients.length === 0) problems.push("recipientsRequired");
  return problems;
}

/** 草稿里不含凭据的部分：交给保存接收人的 mutation */
export function receiverSettingsFromForm(draft: ReceiverFormDraft, editing: NotifyReceiver | null): NotifyReceiverSettings {
  const name = draft.name.trim();
  if (draft.type === "slack") return { name, type: "slack", enabled: draft.enabled };
  return {
    name,
    type: "email",
    enabled: draft.enabled,
    email: emailSettingsFromDraft(draft.email, editing?.email ?? null),
  };
}

/** 草稿里的凭据明文：Slack 的 Webhook 地址（去掉首尾空白）或邮件的 SMTP 密码（原样，密码里的空白有意义） */
export function receiverCredentialFromForm(draft: ReceiverFormDraft): string {
  return draft.type === "slack" ? draft.webhookUrl.trim() : draft.email.smtpPassword;
}
