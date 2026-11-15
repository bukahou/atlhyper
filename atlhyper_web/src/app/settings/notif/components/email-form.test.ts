import { describe, expect, it } from "vitest";
import type { NotifyEmailSettings } from "@/types/notify";
import { emailFormFromSaved, emailSettingsFromDraft } from "./email-form";

const PRESET_HOSTS = ["smtp.gmail.com", "smtp.qq.com"];

const savedSettings: NotifyEmailSettings = {
  smtpHost: "smtp.gmail.com",
  smtpPort: 465,
  smtpUser: "alerts@example.com",
  smtpTLS: false,
  fromAddress: "alerts@example.com",
  toAddresses: ["ops@example.com"],
};

describe("emailFormFromSaved", () => {
  it("用已保存的设置填表单；密码不回显，恒为空", () => {
    expect(emailFormFromSaved(savedSettings, PRESET_HOSTS)).toEqual({
      smtpHost: "smtp.gmail.com",
      smtpPort: 465,
      smtpUser: "alerts@example.com",
      smtpPassword: "",
      smtpTls: false,
      recipients: ["ops@example.com"],
      isCustomSmtp: false,
    });
  });

  it("还没配置过时用默认值：端口 587、不算手填服务器", () => {
    const emptySettings = { ...savedSettings, smtpHost: "", smtpPort: 0, smtpUser: "", smtpTLS: true, toAddresses: [] };
    expect(emailFormFromSaved(emptySettings, PRESET_HOSTS)).toMatchObject({
      smtpHost: "",
      smtpPort: 587,
      smtpTls: true,
      isCustomSmtp: false,
    });
  });

  it("已存的服务器不在预设列表里：算手填", () => {
    const customSettings = { ...savedSettings, smtpHost: "mail.example.com" };
    expect(emailFormFromSaved(customSettings, PRESET_HOSTS).isCustomSmtp).toBe(true);
  });
});

describe("emailSettingsFromDraft", () => {
  const draft = emailFormFromSaved(savedSettings, PRESET_HOSTS);

  it("新建：邮箱账号同时作为发件人地址；结果里没有密码", () => {
    expect(emailSettingsFromDraft({ ...draft, smtpPassword: "typed-password" }, null)).toEqual({
      smtpHost: "smtp.gmail.com",
      smtpPort: 465,
      smtpUser: "alerts@example.com",
      smtpTLS: false,
      fromAddress: "alerts@example.com",
      toAddresses: ["ops@example.com"],
    });
  });

  it("服务器地址与账号去掉首尾空白", () => {
    const padded = { ...draft, smtpHost: " mail.example.com ", smtpUser: " alerts@example.com " };
    expect(emailSettingsFromDraft(padded, null)).toMatchObject({
      smtpHost: "mail.example.com",
      smtpUser: "alerts@example.com",
      fromAddress: "alerts@example.com",
    });
  });

  it("已保存的发件人与账号不同、账号没改：保留已保存的发件人", () => {
    const separateSender = { ...savedSettings, fromAddress: "noreply@example.com" };
    expect(emailSettingsFromDraft(draft, separateSender).fromAddress).toBe("noreply@example.com");
  });

  it("账号改了：发件人跟着新账号走", () => {
    const separateSender = { ...savedSettings, fromAddress: "noreply@example.com" };
    expect(emailSettingsFromDraft({ ...draft, smtpUser: "other@example.com" }, separateSender).fromAddress).toBe(
      "other@example.com",
    );
  });
});
