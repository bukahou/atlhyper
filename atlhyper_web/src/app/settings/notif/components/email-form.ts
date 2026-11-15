/**
 * 邮件接收人表单的草稿：从已保存的设置生成、转回要保存的设置
 */
import type { NotifyEmailSettings } from "@/types/notify";

const DEFAULT_SMTP_PORT = 587;

/** 表单里正在编辑的值（还没保存） */
export interface EmailFormDraft {
  smtpHost: string;
  smtpPort: number;
  smtpUser: string;
  /** 只放用户新输入的密码：已存的密码不回显，留空 = 保持不变 */
  smtpPassword: string;
  smtpTls: boolean;
  recipients: string[];
  /** SMTP 服务器是手填的（不在预设列表里） */
  isCustomSmtp: boolean;
}

/** 用已保存的设置填好表单。presetHosts 是预设服务器的地址，用来判断已存的服务器是不是手填的 */
export function emailFormFromSaved(saved: NotifyEmailSettings, presetHosts: string[]): EmailFormDraft {
  const smtpHost = saved.smtpHost || "";
  return {
    smtpHost,
    smtpPort: saved.smtpPort || DEFAULT_SMTP_PORT,
    smtpUser: saved.smtpUser || "",
    smtpPassword: "",
    smtpTls: saved.smtpTLS ?? true,
    recipients: saved.toAddresses || [],
    isCustomSmtp: smtpHost !== "" && !presetHosts.includes(smtpHost),
  };
}

/**
 * 把草稿转成要保存的设置（不含密码）。
 *
 * 表单里没有单独的发件人地址：邮箱账号同时作为发件人。但已保存的发件人若与账号不同
 * （经接口单独设过），且账号没有改动，就保留它，不悄悄改写。
 */
export function emailSettingsFromDraft(draft: EmailFormDraft, saved: NotifyEmailSettings | null): NotifyEmailSettings {
  const smtpUser = draft.smtpUser.trim();
  const keepsSavedSender =
    saved !== null && saved.fromAddress !== "" && saved.fromAddress !== saved.smtpUser && saved.smtpUser === smtpUser;
  return {
    smtpHost: draft.smtpHost.trim(),
    smtpPort: draft.smtpPort,
    smtpUser,
    smtpTLS: draft.smtpTls,
    fromAddress: keepsSavedSender ? saved.fromAddress : smtpUser,
    toAddresses: draft.recipients,
  };
}
