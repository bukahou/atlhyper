/**
 * 邮件通知表单的草稿：从已保存的配置生成、与已保存的配置比较
 */
import type { EmailConfig } from "@/api/notify";

const DEFAULT_SMTP_PORT = 587;

/** 表单里正在编辑的值（还没保存） */
export interface EmailFormDraft {
  enabled: boolean;
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

/** 用已保存的配置填好表单。presetHosts 是预设服务器的地址，用来判断已存的服务器是不是手填的 */
export function emailFormFromSaved(config: EmailConfig, enabled: boolean, presetHosts: string[]): EmailFormDraft {
  const smtpHost = config.smtpHost || "";
  return {
    enabled,
    smtpHost,
    smtpPort: config.smtpPort || DEFAULT_SMTP_PORT,
    smtpUser: config.smtpUser || "",
    smtpPassword: "",
    smtpTls: config.smtpTLS ?? true,
    recipients: config.toAddresses || [],
    isCustomSmtp: smtpHost !== "" && !presetHosts.includes(smtpHost),
  };
}

/** 草稿相对已保存的配置有没有改动。密码只要有输入就算改动 */
export function emailFormHasChanges(draft: EmailFormDraft, config: EmailConfig, enabled: boolean): boolean {
  const savedRecipients = config.toAddresses || [];
  const recipientsChanged =
    draft.recipients.length !== savedRecipients.length ||
    draft.recipients.some((recipient, index) => recipient !== savedRecipients[index]);

  return (
    draft.enabled !== enabled ||
    draft.smtpHost !== (config.smtpHost || "") ||
    draft.smtpPort !== (config.smtpPort || DEFAULT_SMTP_PORT) ||
    draft.smtpUser !== (config.smtpUser || "") ||
    draft.smtpPassword !== "" ||
    draft.smtpTls !== (config.smtpTLS ?? true) ||
    recipientsChanged
  );
}
