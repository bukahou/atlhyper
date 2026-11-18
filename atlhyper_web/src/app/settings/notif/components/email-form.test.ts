import { describe, expect, it } from "vitest";
import type { NotifySMTPSettings } from "@/types/notify";
import { smtpFormFromSaved, smtpFormProblems, smtpSettingsFromDraft } from "./email-form";

const PRESET_HOSTS = ["smtp.gmail.com", "smtp.qq.com"];

const savedSettings: NotifySMTPSettings = {
  host: "smtp.gmail.com",
  port: 465,
  user: "alerts@example.com",
  tls: false,
  fromAddress: "noreply@example.com",
};

describe("smtpFormFromSaved", () => {
  it("用已保存的设置填表单；密码不回显，恒为空", () => {
    expect(smtpFormFromSaved(savedSettings, PRESET_HOSTS)).toEqual({
      host: "smtp.gmail.com",
      port: 465,
      user: "alerts@example.com",
      password: "",
      tls: false,
      fromAddress: "noreply@example.com",
      isCustomHost: false,
    });
  });

  it("还没配置过时用默认值：端口 587、不算手填服务器", () => {
    const emptySettings: NotifySMTPSettings = { host: "", port: 0, user: "", tls: true, fromAddress: "" };
    expect(smtpFormFromSaved(emptySettings, PRESET_HOSTS)).toMatchObject({
      host: "",
      port: 587,
      tls: true,
      fromAddress: "",
      isCustomHost: false,
    });
  });

  it("已存的服务器不在预设列表里：算手填", () => {
    const customSettings = { ...savedSettings, host: "mail.example.com" };
    expect(smtpFormFromSaved(customSettings, PRESET_HOSTS).isCustomHost).toBe(true);
  });
});

describe("smtpFormProblems", () => {
  const draft = smtpFormFromSaved(savedSettings, PRESET_HOSTS);

  it("还没配置过：服务器、账号、密码必填", () => {
    const emptyDraft = smtpFormFromSaved({ host: "", port: 0, user: "", tls: true, fromAddress: "" }, PRESET_HOSTS);
    expect(smtpFormProblems(emptyDraft, false)).toEqual(["hostRequired", "userRequired", "passwordRequired"]);
  });

  it("密码已配置时可以留空（保持原值）", () => {
    expect(smtpFormProblems(draft, true)).toEqual([]);
  });

  it("密码没配置过：其余都填了也要填密码", () => {
    expect(smtpFormProblems(draft, false)).toEqual(["passwordRequired"]);
    expect(smtpFormProblems({ ...draft, password: "typed" }, false)).toEqual([]);
  });

  it.each([0, -1, 65536, 587.5, Number.NaN])("端口 %s 不合规", (port) => {
    expect(smtpFormProblems({ ...draft, port }, true)).toEqual(["portInvalid"]);
  });

  it("发件地址可以留空；填了就必须像邮箱地址", () => {
    expect(smtpFormProblems({ ...draft, fromAddress: "" }, true)).toEqual([]);
    expect(smtpFormProblems({ ...draft, fromAddress: "not-an-address" }, true)).toEqual(["fromAddressInvalid"]);
  });
});

describe("smtpSettingsFromDraft", () => {
  const draft = smtpFormFromSaved(savedSettings, PRESET_HOSTS);

  it("结果里没有密码与「手填服务器」这类表单状态", () => {
    expect(smtpSettingsFromDraft({ ...draft, password: "typed-password" })).toEqual(savedSettings);
  });

  it("服务器地址、账号、发件地址去掉首尾空白", () => {
    const padded = { ...draft, host: " mail.example.com ", user: " alerts@example.com ", fromAddress: " a@example.com " };
    expect(smtpSettingsFromDraft(padded)).toMatchObject({
      host: "mail.example.com",
      user: "alerts@example.com",
      fromAddress: "a@example.com",
    });
  });
});
