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

import type { NotifyRuleBody } from "@/types/notify";
import { notifyKeys } from "./keys";
import {
  deleteNotifyRuleMutation,
  matchNotifyRulesMutation,
  notifyRulesQuery,
  reorderNotifyRulesMutation,
  saveNotifyRuleMutation,
} from "./notify-routing";

const receiversKey = notifyKeys.receivers();
const rulesKey = notifyKeys.rules();
const slackReceiver = { id: 1, name: "值班频道", type: "slack", enabled: true, slack: { webhookConfigured: true } };
const ruleBody: NotifyRuleBody = {
  name: "生产高严重度",
  enabled: true,
  clusterIds: ["prod"],
  sources: ["incident"],
  minSeverity: "high",
  receiverIds: [1],
  groupWaitSeconds: null,
  groupIntervalSeconds: null,
  repeatIntervalSeconds: 3600,
};
const savedRule = { ...ruleBody, id: 1, position: 1 };

function clientWithCachedLists(): QueryClient {
  const client = new QueryClient();
  client.setQueryData(receiversKey, [slackReceiver]);
  client.setQueryData(rulesKey, [savedRule]);
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

beforeEach(() => {
  for (const call of Object.values(api)) call.mockReset();
});

describe("notifyRulesQuery", () => {
  it("没有 notify.manage 时不启用；键固定", () => {
    expect(notifyRulesQuery(false).enabled).toBe(false);
    expect(notifyRulesQuery(true).queryKey).toEqual(["notify", "rules"]);
  });

  it("按 position 从小到大返回：匹配顺序与上移 / 下移都按它算", async () => {
    const third = { ...savedRule, id: 3, position: 3 };
    const second = { ...savedRule, id: 2, position: 2 };
    api.getNotifyRules.mockResolvedValue({ data: { message: "获取成功", data: [third, savedRule, second], total: 3 } });

    const rules = await new QueryClient().fetchQuery(notifyRulesQuery(true));

    expect(rules.map((rule) => rule.id)).toEqual([1, 2, 3]);
    expect(api.getNotifyRules.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
  });

  it("没有规则时统一成空数组", async () => {
    api.getNotifyRules.mockResolvedValue({ data: { message: "获取成功", data: null, total: 0 } });

    expect(await new QueryClient().fetchQuery(notifyRulesQuery(true))).toEqual([]);
  });
});

describe("规则的写操作", () => {
  it("不带 id = 新建，带 id = 修改；成功后规则列表重新取数", async () => {
    api.createNotifyRule.mockResolvedValue({ data: { message: "已创建", data: savedRule } });
    api.updateNotifyRule.mockResolvedValue({ data: { message: "已保存", data: savedRule } });
    const client = clientWithCachedLists();

    expect(await runMutation(client, saveNotifyRuleMutation(), { rule: ruleBody })).toEqual(savedRule);
    expect(api.createNotifyRule).toHaveBeenCalledWith(ruleBody);
    expect(isInvalidated(client, rulesKey)).toBe(true);

    await runMutation(client, saveNotifyRuleMutation(), { ruleId: 1, rule: ruleBody });
    expect(api.updateNotifyRule).toHaveBeenCalledWith(1, ruleBody);
  });

  it("保存失败不动列表缓存", async () => {
    api.createNotifyRule.mockRejectedValue(new Error("bad request"));
    const client = clientWithCachedLists();

    await expect(runMutation(client, saveNotifyRuleMutation(), { rule: ruleBody })).rejects.toThrow("bad request");
    expect(isInvalidated(client, rulesKey)).toBe(false);
  });

  it("删除后规则列表重新取数", async () => {
    api.deleteNotifyRule.mockResolvedValue({ data: { message: "已删除" } });
    const client = clientWithCachedLists();

    await runMutation(client, deleteNotifyRuleMutation(), 1);

    expect(api.deleteNotifyRule).toHaveBeenCalledWith(1);
    expect(isInvalidated(client, rulesKey)).toBe(true);
  });

  it("调序：把完整的 id 排列交给接口，成功后列表重新取数；失败不动缓存", async () => {
    api.reorderNotifyRules.mockResolvedValueOnce({ data: { message: "已保存" } });
    const client = clientWithCachedLists();

    await runMutation(client, reorderNotifyRulesMutation(), [3, 1, 2]);
    expect(api.reorderNotifyRules).toHaveBeenCalledWith([3, 1, 2]);
    expect(isInvalidated(client, rulesKey)).toBe(true);

    const untouched = clientWithCachedLists();
    api.reorderNotifyRules.mockRejectedValueOnce(new Error("not a permutation"));
    await expect(runMutation(untouched, reorderNotifyRulesMutation(), [1])).rejects.toThrow("not a permutation");
    expect(isInvalidated(untouched, rulesKey)).toBe(false);
  });

  it("规则的写操作不让接收人列表重新取数", async () => {
    api.deleteNotifyRule.mockResolvedValue({ data: { message: "已删除" } });
    const client = clientWithCachedLists();

    await runMutation(client, deleteNotifyRuleMutation(), 1);

    expect(isInvalidated(client, receiversKey)).toBe(false);
  });
});

describe("matchNotifyRulesMutation", () => {
  it("把假想的告警交给接口，返回解包后的匹配结果，不动任何缓存", async () => {
    const result = {
      matchedRule: { id: 1, name: "生产高严重度" },
      receivers: [{ id: 1, name: "值班频道", type: "slack", enabled: true }],
      belowMinSeverity: false,
    };
    api.matchNotifyRules.mockResolvedValue({ data: { message: "匹配完成", data: result } });
    const client = clientWithCachedLists();
    const alert = { clusterId: "prod", source: "incident", severity: "high" } as const;

    expect(await runMutation(client, matchNotifyRulesMutation(), alert)).toEqual(result);

    expect(api.matchNotifyRules).toHaveBeenCalledWith(alert);
    expect(isInvalidated(client, rulesKey)).toBe(false);
  });
});
