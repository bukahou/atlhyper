import { describe, expect, it } from "vitest";
import type { NotifyReceiver } from "@/types/notify";
import {
  receiverCredentialFromForm,
  receiverFormFrom,
  receiverFormProblems,
  receiverSettingsFromForm,
} from "./receiver-form";

const slackReceiver: NotifyReceiver = {
  id: 1,
  name: "值班频道",
  type: "slack",
  enabled: false,
  slack: { webhookConfigured: true },
  createdAt: "2026-11-01T00:00:00Z",
  updatedAt: "2026-11-01T00:00:00Z",
};

const emailReceiver: NotifyReceiver = {
  id: 2,
  name: "运维邮件",
  type: "email",
  enabled: true,
  email: { toAddresses: ["a@example.com"] },
  createdAt: "2026-11-01T00:00:00Z",
  updatedAt: "2026-11-01T00:00:00Z",
};

describe("receiverFormFrom", () => {
  it("新建：默认 Slack、启用，凭据与收件人为空", () => {
    expect(receiverFormFrom(null)).toEqual({
      name: "",
      type: "slack",
      enabled: true,
      webhookUrl: "",
      recipients: [],
    });
  });

  it("编辑：填入已保存的各项；凭据不回显", () => {
    expect(receiverFormFrom(slackReceiver)).toMatchObject({
      name: "值班频道",
      type: "slack",
      enabled: false,
      webhookUrl: "",
    });
    expect(receiverFormFrom(emailReceiver)).toMatchObject({
      name: "运维邮件",
      type: "email",
      recipients: ["a@example.com"],
    });
  });

  it("邮件接收人缺 email 段时收件人为空，不抛错", () => {
    expect(receiverFormFrom({ ...emailReceiver, email: undefined }).recipients).toEqual([]);
  });
});

describe("receiverFormProblems", () => {
  it("新建 Slack：名称与 Webhook 地址必填", () => {
    expect(receiverFormProblems(receiverFormFrom(null), null)).toEqual(["nameRequired", "webhookRequired"]);
  });

  it("编辑 Slack：Webhook 已配置时可以留空（保持不变）", () => {
    expect(receiverFormProblems(receiverFormFrom(slackReceiver), slackReceiver)).toEqual([]);
  });

  it("编辑 Slack：之前没配置过 Webhook 的，仍然必填", () => {
    const neverConfigured = { ...slackReceiver, slack: { webhookConfigured: false } };
    expect(receiverFormProblems(receiverFormFrom(neverConfigured), neverConfigured)).toEqual(["webhookRequired"]);
  });

  it("新建邮件：只要求名称与至少一位收件人（发件设置是全局的，不在这里）", () => {
    const draft = { ...receiverFormFrom(null), name: "运维邮件", type: "email" as const };
    expect(receiverFormProblems(draft, null)).toEqual(["recipientsRequired"]);
    expect(receiverFormProblems({ ...draft, recipients: ["a@example.com"] }, null)).toEqual([]);
  });

  it("编辑邮件：已有收件人即合规", () => {
    expect(receiverFormProblems(receiverFormFrom(emailReceiver), emailReceiver)).toEqual([]);
  });

  it("只检查当前类型的字段：选了 Slack 时不管收件人填没填", () => {
    const draft = { ...receiverFormFrom(null), name: "值班频道", webhookUrl: "https://hooks.example.com/x" };
    expect(receiverFormProblems(draft, null)).toEqual([]);
  });
});

describe("receiverSettingsFromForm / receiverCredentialFromForm", () => {
  it("Slack：设置里没有 Webhook 地址，地址单独作为凭据取出（去掉首尾空白）", () => {
    const draft = { ...receiverFormFrom(null), name: " 值班频道 ", webhookUrl: " https://hooks.example.com/x " };
    expect(receiverSettingsFromForm(draft)).toEqual({ name: "值班频道", type: "slack", enabled: true });
    expect(receiverCredentialFromForm(draft)).toBe("https://hooks.example.com/x");
  });

  it("邮件：设置里只有收件人；没有凭据", () => {
    const draft = { ...receiverFormFrom(emailReceiver), webhookUrl: "https://hooks.example.com/leftover" };
    expect(receiverSettingsFromForm(draft)).toEqual({
      name: "运维邮件",
      type: "email",
      enabled: true,
      email: { toAddresses: ["a@example.com"] },
    });
    // 切换过类型时 Slack 那一半留着的地址不会被当成邮件的凭据发出去
    expect(receiverCredentialFromForm(draft)).toBe("");
  });

  it("编辑 Slack 时凭据留空：取出空串（接口约定 = 保持原值）", () => {
    expect(receiverCredentialFromForm(receiverFormFrom(slackReceiver))).toBe("");
  });
});
