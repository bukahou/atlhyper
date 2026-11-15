import { describe, expect, it } from "vitest";
import type { NotifyReceiver } from "@/types/notify";
import { receiverRequestBody, receiverSettingsOf, sealCredential } from "./notify-receiver-input";

const slackReceiver: NotifyReceiver = {
  id: 1,
  name: "值班频道",
  type: "slack",
  enabled: true,
  slack: { webhookConfigured: true },
  createdAt: "2026-11-01T00:00:00Z",
  updatedAt: "2026-11-01T00:00:00Z",
};

const emailReceiver: NotifyReceiver = {
  id: 2,
  name: "运维邮件",
  type: "email",
  enabled: false,
  email: {
    smtpHost: "smtp.example.com",
    smtpPort: 587,
    smtpUser: "ops@example.com",
    smtpTLS: true,
    fromAddress: "ops@example.com",
    toAddresses: ["a@example.com"],
    passwordConfigured: true,
  },
  createdAt: "2026-11-01T00:00:00Z",
  updatedAt: "2026-11-01T00:00:00Z",
};

describe("receiverSettingsOf", () => {
  it("Slack：只留名称、类型、启用状态", () => {
    expect(receiverSettingsOf(slackReceiver)).toEqual({ name: "值班频道", type: "slack", enabled: true });
  });

  it("邮件：带上 SMTP 各项与收件人，不带「密码是否已配置」这个只读标记", () => {
    expect(receiverSettingsOf(emailReceiver)).toEqual({
      name: "运维邮件",
      type: "email",
      enabled: false,
      email: {
        smtpHost: "smtp.example.com",
        smtpPort: 587,
        smtpUser: "ops@example.com",
        smtpTLS: true,
        fromAddress: "ops@example.com",
        toAddresses: ["a@example.com"],
      },
    });
  });

  it("邮件接收人缺 email 段时按空设置处理，不抛错", () => {
    const settings = receiverSettingsOf({ ...emailReceiver, email: undefined });
    expect(settings).toMatchObject({ type: "email", email: { smtpHost: "", smtpPort: 587, toAddresses: [] } });
  });
});

describe("receiverRequestBody", () => {
  it("Slack：凭据放进 slack.webhookUrl", () => {
    expect(receiverRequestBody(receiverSettingsOf(slackReceiver), "https://hooks.example.com/x")).toEqual({
      name: "值班频道",
      type: "slack",
      enabled: true,
      slack: { webhookUrl: "https://hooks.example.com/x" },
    });
  });

  it("邮件：凭据放进 email.smtpPassword，其余各项原样带上", () => {
    expect(receiverRequestBody(receiverSettingsOf(emailReceiver), "new-password")).toEqual({
      name: "运维邮件",
      type: "email",
      enabled: false,
      email: {
        smtpHost: "smtp.example.com",
        smtpPort: 587,
        smtpUser: "ops@example.com",
        smtpTLS: true,
        fromAddress: "ops@example.com",
        toAddresses: ["a@example.com"],
        smtpPassword: "new-password",
      },
    });
  });

  it("凭据为空串时照样带上空串：接口约定留空 = 保持原值", () => {
    expect(receiverRequestBody(receiverSettingsOf(slackReceiver), "")).toMatchObject({ slack: { webhookUrl: "" } });
    expect(receiverRequestBody(receiverSettingsOf(emailReceiver), "")).toMatchObject({ email: { smtpPassword: "" } });
  });
});

describe("sealCredential", () => {
  it("第一次取到明文，之后再取只有空串", () => {
    const take = sealCredential("s3cret");
    expect(take()).toBe("s3cret");
    expect(take()).toBe("");
    expect(take()).toBe("");
  });
});
