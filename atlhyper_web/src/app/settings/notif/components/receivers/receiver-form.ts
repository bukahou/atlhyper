/**
 * 接收人表单的草稿与校验（纯函数）
 *
 * 邮件接收人只填收件人：发件服务器、账号、密码是全局发件设置（见 ../email-form.ts）。
 */
import type { NotifyReceiverSettings } from "@/lib/notify-receiver-input";
import type { NotifyReceiverProblemKey } from "@/types/i18n-notify";
import type { NotifyReceiver, NotifyReceiverType } from "@/types/notify";

/** 表单里正在编辑的接收人。两种类型的字段都留着：切换类型时已填的内容不丢 */
export interface ReceiverFormDraft {
  name: string;
  type: NotifyReceiverType;
  enabled: boolean;
  /** 只放用户新输入的 Webhook 地址：已存的不回显，留空 = 保持不变 */
  webhookUrl: string;
  /** 邮件收件人 */
  recipients: string[];
}

/** 用已有的接收人填表单；editing 为 null = 新建（默认 Slack、启用）。凭据恒为空 */
export function receiverFormFrom(editing: NotifyReceiver | null): ReceiverFormDraft {
  return {
    name: editing?.name ?? "",
    type: editing?.type ?? "slack",
    enabled: editing?.enabled ?? true,
    webhookUrl: "",
    recipients: editing?.email?.toAddresses ?? [],
  };
}

/**
 * 表单哪里不合规（空数组 = 合规）。
 *
 * Webhook 地址在新建时必填；编辑时已配置过就可以留空（保持原值）。
 */
export function receiverFormProblems(draft: ReceiverFormDraft, editing: NotifyReceiver | null): NotifyReceiverProblemKey[] {
  const problems: NotifyReceiverProblemKey[] = [];
  if (draft.name.trim() === "") problems.push("nameRequired");

  if (draft.type === "slack") {
    const webhookAlreadySet = editing?.slack?.webhookConfigured === true;
    if (draft.webhookUrl.trim() === "" && !webhookAlreadySet) problems.push("webhookRequired");
    return problems;
  }

  if (draft.recipients.length === 0) problems.push("recipientsRequired");
  return problems;
}

/** 草稿里不含凭据的部分：交给保存接收人的 mutation */
export function receiverSettingsFromForm(draft: ReceiverFormDraft): NotifyReceiverSettings {
  const name = draft.name.trim();
  if (draft.type === "slack") return { name, type: "slack", enabled: draft.enabled };
  return { name, type: "email", enabled: draft.enabled, email: { toAddresses: draft.recipients } };
}

/** 草稿里的凭据明文：Slack 的 Webhook 地址（去掉首尾空白）；邮件接收人没有凭据，为空串 */
export function receiverCredentialFromForm(draft: ReceiverFormDraft): string {
  return draft.type === "slack" ? draft.webhookUrl.trim() : "";
}
