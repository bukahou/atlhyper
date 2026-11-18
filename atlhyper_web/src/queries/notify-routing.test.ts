import { QueryClient, type MutationOptions } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const api = vi.hoisted(() => ({
  getNotifyReceivers: vi.fn(),
  createNotifyReceiver: vi.fn(),
  updateNotifyReceiver: vi.fn(),
  deleteNotifyReceiver: vi.fn(),
  testNotifyReceiver: vi.fn(),
  getNotifyRules: vi.fn(),
  createNotifyRule: vi.fn(),
  updateNotifyRule: vi.fn(),
  deleteNotifyRule: vi.fn(),
  reorderNotifyRules: vi.fn(),
  matchNotifyRules: vi.fn(),
}));
vi.mock("@/api/notify", () => api);

import { sealCredential, type NotifyReceiverSettings } from "@/lib/notify-receiver-input";
import { notifyKeys } from "./keys";
import {
  deleteNotifyReceiverMutation,
  notifyReceiversQuery,
  saveNotifyReceiverMutation,
  testNotifyReceiverMutation,
} from "./notify-routing";

// 接收人的取数与写操作；路由规则的在 notify-rules.test.ts

const receiversKey = notifyKeys.receivers();
const rulesKey = notifyKeys.rules();
const slackReceiver = { id: 1, name: "值班频道", type: "slack", enabled: true, slack: { webhookConfigured: true } };
const slackSettings: NotifyReceiverSettings = { name: "值班频道", type: "slack", enabled: true };
const emailSettings: NotifyReceiverSettings & { type: "email" } = {
  name: "运维邮件",
  type: "email",
  enabled: true,
  email: { toAddresses: ["a@example.com"] },
};
// 测试用的假凭据：只要是在别处不会出现的字符串即可
const WEBHOOK_PLAINTEXT = "https://hooks.example.com/services/not-a-real-webhook";

function clientWithCachedLists(): QueryClient {
  const client = new QueryClient();
  client.setQueryData(receiversKey, [slackReceiver]);
  client.setQueryData(rulesKey, []);
  return client;
}

function isInvalidated(client: QueryClient, queryKey: readonly unknown[]): boolean {
  return client.getQueryState(queryKey)?.isInvalidated ?? false;
}

function runMutation<TResult, TVariables>(
  client: QueryClient,
  options: MutationOptions<TResult, Error, TVariables>,
  variables: TVariables,
) {
  return client.getMutationCache().build(client, options).execute(variables);
}

/** 把一个值里能走到的字符串全部收集出来（JSON.stringify 会跳过函数与循环引用，不够可靠） */
function collectStrings(value: unknown, seen = new Set<unknown>()): string[] {
  if (typeof value === "string") return [value];
  if (typeof value !== "object" || value === null || seen.has(value)) return [];
  seen.add(value);
  return Object.values(value).flatMap((inner) => collectStrings(inner, seen));
}

function everyCachedString(client: QueryClient): string {
  const fromMutations = client.getMutationCache().getAll().flatMap((mutation) => collectStrings(mutation.state));
  const fromQueries = client.getQueryCache().getAll().flatMap((query) => collectStrings(query.state));
  return [...fromMutations, ...fromQueries].join("\n");
}

beforeEach(() => {
  for (const call of Object.values(api)) call.mockReset();
});

describe("notifyReceiversQuery", () => {
  it("没有 notify.manage 时不启用", () => {
    expect(notifyReceiversQuery(false).enabled).toBe(false);
    expect(notifyReceiversQuery(true).enabled).toBe(true);
  });

  it("键固定；返回解包后的列表，带取消信号；没有接收人时统一成空数组", async () => {
    expect(notifyReceiversQuery(true).queryKey).toEqual(["notify", "receivers"]);

    api.getNotifyReceivers.mockResolvedValueOnce({ data: { message: "获取成功", data: [slackReceiver], total: 1 } });
    expect(await new QueryClient().fetchQuery(notifyReceiversQuery(true))).toEqual([slackReceiver]);
    expect(api.getNotifyReceivers.mock.calls[0][0]).toBeInstanceOf(AbortSignal);

    api.getNotifyReceivers.mockResolvedValueOnce({ data: { message: "获取成功", data: null, total: 0 } });
    expect(await new QueryClient().fetchQuery(notifyReceiversQuery(true))).toEqual([]);
  });
});

describe("saveNotifyReceiverMutation", () => {
  it("不带 id = 新建：凭据拼进请求体，成功后列表重新取数", async () => {
    api.createNotifyReceiver.mockResolvedValue({ data: { message: "已创建", data: slackReceiver } });
    const client = clientWithCachedLists();

    const saved = await runMutation(client, saveNotifyReceiverMutation(), {
      settings: slackSettings,
      takeCredential: sealCredential(WEBHOOK_PLAINTEXT),
    });

    expect(saved).toEqual(slackReceiver);
    expect(api.createNotifyReceiver).toHaveBeenCalledWith({ ...slackSettings, slack: { webhookUrl: WEBHOOK_PLAINTEXT } });
    expect(api.updateNotifyReceiver).not.toHaveBeenCalled();
    expect(isInvalidated(client, receiversKey)).toBe(true);
  });

  it("带 id = 修改：Slack 凭据留空则请求体里是空串（保持原值）", async () => {
    api.updateNotifyReceiver.mockResolvedValue({ data: { message: "已保存", data: slackReceiver } });
    const client = clientWithCachedLists();

    await runMutation(client, saveNotifyReceiverMutation(), {
      receiverId: 1,
      settings: slackSettings,
      takeCredential: sealCredential(""),
    });

    expect(api.updateNotifyReceiver).toHaveBeenCalledWith(1, { ...slackSettings, slack: { webhookUrl: "" } });
    expect(api.createNotifyReceiver).not.toHaveBeenCalled();
    expect(isInvalidated(client, receiversKey)).toBe(true);
  });

  it("邮件接收人：请求体的 email 只有收件人", async () => {
    api.updateNotifyReceiver.mockResolvedValue({ data: { message: "已保存", data: slackReceiver } });
    const client = clientWithCachedLists();

    await runMutation(client, saveNotifyReceiverMutation(), {
      receiverId: 2,
      settings: emailSettings,
      takeCredential: sealCredential(""),
    });

    expect(api.updateNotifyReceiver).toHaveBeenCalledWith(2, {
      name: "运维邮件",
      type: "email",
      enabled: true,
      email: { toAddresses: ["a@example.com"] },
    });
    expect(api.createNotifyReceiver).not.toHaveBeenCalled();
    expect(isInvalidated(client, receiversKey)).toBe(true);
  });

  it("凭据明文不留在任何缓存里：mutation 还在缓存里时，查询缓存与 mutation 缓存里都找不到", async () => {
    api.createNotifyReceiver.mockResolvedValue({ data: { message: "已创建", data: slackReceiver } });
    api.updateNotifyReceiver.mockResolvedValue({ data: { message: "已保存", data: slackReceiver } });
    const client = clientWithCachedLists();

    await runMutation(client, saveNotifyReceiverMutation(), {
      settings: slackSettings,
      takeCredential: sealCredential(WEBHOOK_PLAINTEXT),
    });
    await runMutation(client, saveNotifyReceiverMutation(), {
      receiverId: 1,
      settings: slackSettings,
      takeCredential: sealCredential(WEBHOOK_PLAINTEXT),
    });

    // 两条 mutation 此刻仍在缓存里（页面挂着时就是这样），正是要检查的状态
    expect(client.getMutationCache().getAll()).toHaveLength(2);
    const cached = everyCachedString(client);
    expect(cached).toContain("值班频道");
    expect(cached).not.toContain(WEBHOOK_PLAINTEXT);
  });

  it("保存失败（如 409 名称重复）不动列表缓存，缓存里同样没有明文", async () => {
    api.createNotifyReceiver.mockRejectedValue(new Error("conflict"));
    const client = clientWithCachedLists();

    await expect(
      runMutation(client, saveNotifyReceiverMutation(), {
        settings: slackSettings,
        takeCredential: sealCredential(WEBHOOK_PLAINTEXT),
      }),
    ).rejects.toThrow("conflict");

    expect(isInvalidated(client, receiversKey)).toBe(false);
    expect(everyCachedString(client)).not.toContain(WEBHOOK_PLAINTEXT);
  });
});

describe("deleteNotifyReceiverMutation / testNotifyReceiverMutation", () => {
  it("删除后列表重新取数", async () => {
    api.deleteNotifyReceiver.mockResolvedValue({ data: { message: "已删除" } });
    const client = clientWithCachedLists();

    await runMutation(client, deleteNotifyReceiverMutation(), 1);

    expect(api.deleteNotifyReceiver).toHaveBeenCalledWith(1);
    expect(isInvalidated(client, receiversKey)).toBe(true);
  });

  it("删除被拒绝（409 仍被引用）不动列表缓存，错误原样抛给调用方", async () => {
    api.deleteNotifyReceiver.mockRejectedValue(new Error("referenced"));
    const client = clientWithCachedLists();

    await expect(runMutation(client, deleteNotifyReceiverMutation(), 1)).rejects.toThrow("referenced");
    expect(isInvalidated(client, receiversKey)).toBe(false);
  });

  it("测试发送：按 id 调接口，不动任何缓存；发不出去（502）时错误原样抛出", async () => {
    api.testNotifyReceiver.mockResolvedValueOnce({ data: { message: "测试消息已发送" } });
    const client = clientWithCachedLists();

    await runMutation(client, testNotifyReceiverMutation(), 1);
    expect(api.testNotifyReceiver).toHaveBeenCalledWith(1);
    expect(isInvalidated(client, receiversKey)).toBe(false);

    api.testNotifyReceiver.mockRejectedValueOnce(new Error("bad gateway"));
    await expect(runMutation(client, testNotifyReceiverMutation(), 1)).rejects.toThrow("bad gateway");
  });
});
