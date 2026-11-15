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
  email: { toAddresses: ["a@example.com"] },
  createdAt: "2026-11-01T00:00:00Z",
  updatedAt: "2026-11-01T00:00:00Z",
};

describe("receiverSettingsOf", () => {
  it("Slack：只留名称、类型、启用状态", () => {
    expect(receiverSettingsOf(slackReceiver)).toEqual({ name: "值班频道", type: "slack", enabled: true });
  });

  it("邮件：只带收件人", () => {
    expect(receiverSettingsOf(emailReceiver)).toEqual({
      name: "运维邮件",
      type: "email",
      enabled: false,
      email: { toAddresses: ["a@example.com"] },
    });
  });

  it("邮件接收人缺 email 段时收件人为空数组，不抛错", () => {
    const settings = receiverSettingsOf({ ...emailReceiver, email: undefined });
    expect(settings).toMatchObject({ type: "email", email: { toAddresses: [] } });
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

  it("邮件：请求体的 email 只有收件人；传进来的凭据不会出现在请求体里", () => {
    const body = receiverRequestBody(receiverSettingsOf(emailReceiver), "should-not-appear");
    expect(body).toEqual({
      name: "运维邮件",
      type: "email",
      enabled: false,
      email: { toAddresses: ["a@example.com"] },
    });
    expect(JSON.stringify(body)).not.toContain("should-not-appear");
  });

  it("Slack 凭据为空串时照样带上空串：接口约定留空 = 保持原值", () => {
    expect(receiverRequestBody(receiverSettingsOf(slackReceiver), "")).toMatchObject({ slack: { webhookUrl: "" } });
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
