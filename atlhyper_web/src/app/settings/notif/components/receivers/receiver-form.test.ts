import { describe, expect, it } from "vitest";
import type { NotifyReceiver } from "@/types/notify";
import {
  receiverCredentialFromForm,
  receiverFormFrom,
  receiverFormProblems,
  receiverSettingsFromForm,
} from "./receiver-form";

const PRESET_HOSTS = ["smtp.gmail.com"];

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

describe("receiverFormFrom", () => {
  it("新建：默认 Slack、启用，凭据为空", () => {
    expect(receiverFormFrom(null, PRESET_HOSTS)).toMatchObject({
      name: "",
      type: "slack",
      enabled: true,
      webhookUrl: "",
      email: { smtpHost: "", smtpPort: 587, smtpPassword: "", recipients: [] },
    });
  });

  it("编辑：填入已保存的各项；凭据不回显", () => {
    expect(receiverFormFrom(slackReceiver, PRESET_HOSTS)).toMatchObject({
      name: "值班频道",
      type: "slack",
      enabled: false,
      webhookUrl: "",
    });
    expect(receiverFormFrom(emailReceiver, PRESET_HOSTS)).toMatchObject({
      name: "运维邮件",
      type: "email",
      email: { smtpHost: "smtp.example.com", smtpUser: "ops@example.com", smtpPassword: "", isCustomSmtp: true },
    });
  });
});

describe("receiverFormProblems", () => {
  it("新建 Slack：名称与 Webhook 地址必填", () => {
    expect(receiverFormProblems(receiverFormFrom(null, PRESET_HOSTS), null)).toEqual([
      "nameRequired",
      "webhookRequired",
    ]);
  });

  it("编辑 Slack：Webhook 已配置时可以留空（保持不变）", () => {
    expect(receiverFormProblems(receiverFormFrom(slackReceiver, PRESET_HOSTS), slackReceiver)).toEqual([]);
  });

  it("编辑 Slack：之前没配置过 Webhook 的，仍然必填", () => {
    const neverConfigured = { ...slackReceiver, slack: { webhookConfigured: false } };
    expect(receiverFormProblems(receiverFormFrom(neverConfigured, PRESET_HOSTS), neverConfigured)).toEqual([
      "webhookRequired",
    ]);
  });

  it("新建邮件：服务器、账号、密码、收件人必填", () => {
    const draft = { ...receiverFormFrom(null, PRESET_HOSTS), name: "运维邮件", type: "email" as const };
    expect(receiverFormProblems(draft, null)).toEqual([
      "smtpHostRequired",
      "smtpUserRequired",
      "passwordRequired",
      "recipientsRequired",
    ]);
  });

  it("编辑邮件：密码已配置时可以留空", () => {
    expect(receiverFormProblems(receiverFormFrom(emailReceiver, PRESET_HOSTS), emailReceiver)).toEqual([]);
  });

  it.each([0, -1, 65536, 587.5, Number.NaN])("端口 %s 不合规", (smtpPort) => {
    const draft = receiverFormFrom(emailReceiver, PRESET_HOSTS);
    expect(receiverFormProblems({ ...draft, email: { ...draft.email, smtpPort } }, emailReceiver)).toEqual([
      "smtpPortInvalid",
    ]);
  });

  it("只检查当前类型的字段：选了 Slack 时不管邮件那一半填没填", () => {
    const draft = { ...receiverFormFrom(null, PRESET_HOSTS), name: "值班频道", webhookUrl: "https://hooks.example.com/x" };
    expect(receiverFormProblems(draft, null)).toEqual([]);
  });
});

describe("receiverSettingsFromForm / receiverCredentialFromForm", () => {
  it("Slack：设置里没有 Webhook 地址，地址单独作为凭据取出（去掉首尾空白）", () => {
    const draft = {
      ...receiverFormFrom(null, PRESET_HOSTS),
      name: " 值班频道 ",
      webhookUrl: " https://hooks.example.com/x ",
    };
    expect(receiverSettingsFromForm(draft, null)).toEqual({ name: "值班频道", type: "slack", enabled: true });
    expect(receiverCredentialFromForm(draft)).toBe("https://hooks.example.com/x");
  });

  it("邮件：设置里没有密码，密码单独作为凭据原样取出", () => {
    const base = receiverFormFrom(emailReceiver, PRESET_HOSTS);
    const draft = { ...base, email: { ...base.email, smtpPassword: " pass word " } };
    const settings = receiverSettingsFromForm(draft, emailReceiver);
    expect(settings).toEqual({
      name: "运维邮件",
      type: "email",
      enabled: true,
      email: {
        smtpHost: "smtp.example.com",
        smtpPort: 587,
        smtpUser: "ops@example.com",
        smtpTLS: true,
        fromAddress: "ops@example.com",
        toAddresses: ["a@example.com"],
      },
    });
    expect(JSON.stringify(settings)).not.toContain("pass word");
    expect(receiverCredentialFromForm(draft)).toBe(" pass word ");
  });

  it("编辑时凭据留空：取出空串（接口约定 = 保持原值）", () => {
    expect(receiverCredentialFromForm(receiverFormFrom(slackReceiver, PRESET_HOSTS))).toBe("");
    expect(receiverCredentialFromForm(receiverFormFrom(emailReceiver, PRESET_HOSTS))).toBe("");
  });
});
