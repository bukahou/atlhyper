import { describe, expect, it } from "vitest";
import type { EmailConfig } from "@/api/notify";
import { emailFormFromSaved, emailFormHasChanges } from "./email-form";

const PRESET_HOSTS = ["smtp.gmail.com", "smtp.qq.com"];

const savedConfig: EmailConfig = {
  smtpHost: "smtp.gmail.com",
  smtpPort: 465,
  smtpUser: "alerts@example.com",
  smtpTLS: false,
  fromAddress: "alerts@example.com",
  toAddresses: ["ops@example.com"],
};

describe("emailFormFromSaved", () => {
  it("用已保存的配置填表单；密码不回显，恒为空", () => {
    expect(emailFormFromSaved(savedConfig, true, PRESET_HOSTS)).toEqual({
      enabled: true,
      smtpHost: "smtp.gmail.com",
      smtpPort: 465,
      smtpUser: "alerts@example.com",
      smtpPassword: "",
      smtpTls: false,
      recipients: ["ops@example.com"],
      isCustomSmtp: false,
    });
  });

  it("还没配置过时用默认值：端口 587、开启 TLS、不算手填服务器", () => {
    const emptyConfig = { smtpHost: "", smtpPort: 0, smtpUser: "", fromAddress: "", toAddresses: [] };
    expect(emailFormFromSaved(emptyConfig, false, PRESET_HOSTS)).toMatchObject({
      smtpHost: "",
      smtpPort: 587,
      smtpTls: true,
      isCustomSmtp: false,
    });
  });

  it("已存的服务器不在预设列表里：算手填", () => {
    const customConfig = { ...savedConfig, smtpHost: "mail.example.com" };
    expect(emailFormFromSaved(customConfig, true, PRESET_HOSTS).isCustomSmtp).toBe(true);
  });
});

describe("emailFormHasChanges", () => {
  const untouched = emailFormFromSaved(savedConfig, true, PRESET_HOSTS);

  it("刚填好的表单没有改动", () => {
    expect(emailFormHasChanges(untouched, savedConfig, true)).toBe(false);
  });

  it("任何一项与已保存的不同都算改动", () => {
    expect(emailFormHasChanges({ ...untouched, enabled: false }, savedConfig, true)).toBe(true);
    expect(emailFormHasChanges({ ...untouched, smtpHost: "smtp.qq.com" }, savedConfig, true)).toBe(true);
    expect(emailFormHasChanges({ ...untouched, smtpPort: 587 }, savedConfig, true)).toBe(true);
    expect(emailFormHasChanges({ ...untouched, smtpUser: "other@example.com" }, savedConfig, true)).toBe(true);
    expect(emailFormHasChanges({ ...untouched, smtpTls: true }, savedConfig, true)).toBe(true);
  });

  it("密码只要有输入就算改动", () => {
    expect(emailFormHasChanges({ ...untouched, smtpPassword: "x" }, savedConfig, true)).toBe(true);
  });

  it("收件人增删、换顺序都算改动", () => {
    expect(emailFormHasChanges({ ...untouched, recipients: [] }, savedConfig, true)).toBe(true);
    expect(
      emailFormHasChanges({ ...untouched, recipients: ["ops@example.com", "sre@example.com"] }, savedConfig, true),
    ).toBe(true);
    const twoRecipients = { ...savedConfig, toAddresses: ["a@example.com", "b@example.com"] };
    expect(
      emailFormHasChanges({ ...untouched, recipients: ["b@example.com", "a@example.com"] }, twoRecipients, true),
    ).toBe(true);
  });
});
