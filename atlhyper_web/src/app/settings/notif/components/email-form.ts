/**
 * 邮件发件设置（全局 SMTP）表单的草稿与校验（纯函数）：从已保存的设置生成、校验、转回要保存的设置
 */
import type { NotifySMTPProblemKey } from "@/types/i18n-notify";
import type { NotifySMTPSettings } from "@/types/notify";
import { emailValidator } from "./TagInput";

export const DEFAULT_SMTP_PORT = 587;
const MAX_PORT = 65_535;

/** 表单里正在编辑的值（还没保存） */
export interface SmtpFormDraft {
  host: string;
  port: number;
  user: string;
  /** 只放用户新输入的密码：已存的密码不回显，留空 = 保持不变 */
  password: string;
  tls: boolean;
  /** 留空 = 用账号作为发件地址 */
  fromAddress: string;
  /** SMTP 服务器是手填的（不在预设列表里） */
  isCustomHost: boolean;
}

/** 用已保存的设置填好表单。presetHosts 是预设服务器的地址，用来判断已存的服务器是不是手填的 */
export function smtpFormFromSaved(saved: NotifySMTPSettings, presetHosts: string[]): SmtpFormDraft {
  const host = saved.host || "";
  return {
    host,
    // 从没保存过时后端给的端口可能是 0：按常用的 587 填
    port: saved.port || DEFAULT_SMTP_PORT,
    user: saved.user || "",
    password: "",
    tls: saved.tls ?? true,
    fromAddress: saved.fromAddress || "",
    isCustomHost: host !== "" && !presetHosts.includes(host),
  };
}

/**
 * 表单哪里不合规（空数组 = 合规）。
 *
 * 密码在还没配置过时必填（后端第一次保存不填会 400）；已配置过就可以留空（保持原值）。
 */
export function smtpFormProblems(draft: SmtpFormDraft, passwordConfigured: boolean): NotifySMTPProblemKey[] {
  const problems: NotifySMTPProblemKey[] = [];
  if (draft.host.trim() === "") problems.push("hostRequired");
  if (!Number.isInteger(draft.port) || draft.port < 1 || draft.port > MAX_PORT) problems.push("portInvalid");
  if (draft.user.trim() === "") problems.push("userRequired");
  if (draft.password === "" && !passwordConfigured) problems.push("passwordRequired");
  const fromAddress = draft.fromAddress.trim();
  if (fromAddress !== "" && !emailValidator(fromAddress)) problems.push("fromAddressInvalid");
  return problems;
}

/** 把草稿转成要保存的设置（不含密码：密码作为凭据单独取，见 SmtpSettingsCard） */
export function smtpSettingsFromDraft(draft: SmtpFormDraft): NotifySMTPSettings {
  return {
    host: draft.host.trim(),
    port: draft.port,
    user: draft.user.trim(),
    tls: draft.tls,
    fromAddress: draft.fromAddress.trim(),
  };
}
