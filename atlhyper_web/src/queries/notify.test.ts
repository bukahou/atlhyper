import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const api = vi.hoisted(() => ({
  listChannels: vi.fn(),
  updateSlack: vi.fn(),
  updateEmail: vi.fn(),
  testChannel: vi.fn(),
}));
vi.mock("@/api/notify", () => api);

import { notifyKeys } from "./keys";
import {
  notifyChannelsQuery,
  testNotifyChannelMutation,
  updateEmailChannelMutation,
  updateSlackChannelMutation,
} from "./notify";

const slack = { id: 1, type: "slack", enabled: false };
const email = { id: 2, type: "email", enabled: false };

beforeEach(() => {
  for (const call of Object.values(api)) call.mockReset();
});

describe("notifyChannelsQuery", () => {
  it("没有 notify.manage 权限时不发请求（页面此时整页显示演示数据）", () => {
    expect(notifyChannelsQuery(false).enabled).toBe(false);
    expect(notifyChannelsQuery(true).enabled).toBe(true);
  });

  it("返回解包后的渠道列表，带取消信号", async () => {
    api.listChannels.mockResolvedValue({ data: { channels: [slack, email], total: 2 } });

    expect(await new QueryClient().fetchQuery(notifyChannelsQuery(true))).toEqual([slack, email]);
    expect(api.listChannels.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
  });
});

describe("保存渠道配置", () => {
  it("Slack：把接口带回的渠道换进列表，不再请求一次", async () => {
    const savedSlack = { ...slack, enabled: true };
    api.updateSlack.mockResolvedValue({ data: savedSlack });
    const client = new QueryClient();
    client.setQueryData(notifyKeys.channels(), [slack, email]);

    await client.getMutationCache().build(client, updateSlackChannelMutation()).execute({ enabled: true });

    expect(api.updateSlack).toHaveBeenCalledWith({ enabled: true });
    expect(client.getQueryData(notifyKeys.channels())).toEqual([savedSlack, email]);
    expect(api.listChannels).not.toHaveBeenCalled();
  });

  it("邮件：同样换进列表", async () => {
    const savedEmail = { ...email, enabled: true };
    api.updateEmail.mockResolvedValue({ data: savedEmail });
    const client = new QueryClient();
    client.setQueryData(notifyKeys.channels(), [slack, email]);

    await client.getMutationCache().build(client, updateEmailChannelMutation()).execute({ smtpHost: "smtp.example.com" });

    expect(api.updateEmail).toHaveBeenCalledWith({ smtpHost: "smtp.example.com" });
    expect(client.getQueryData(notifyKeys.channels())).toEqual([slack, savedEmail]);
  });

  it("保存失败不动列表", async () => {
    api.updateSlack.mockRejectedValue(new Error("bad webhook"));
    const client = new QueryClient();
    client.setQueryData(notifyKeys.channels(), [slack, email]);

    await expect(
      client.getMutationCache().build(client, updateSlackChannelMutation()).execute({ webhookUrl: "x" }),
    ).rejects.toThrow("bad webhook");
    expect(client.getQueryData(notifyKeys.channels())).toEqual([slack, email]);
  });
});

describe("testNotifyChannelMutation", () => {
  it("按渠道类型发测试消息，原样返回结果", async () => {
    api.testChannel.mockResolvedValue({ success: false, message: "smtp refused" });
    const client = new QueryClient();

    const result = await client.getMutationCache().build(client, testNotifyChannelMutation()).execute("email");

    expect(api.testChannel).toHaveBeenCalledWith("email");
    expect(result).toEqual({ success: false, message: "smtp refused" });
  });
});
