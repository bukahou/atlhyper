import { QueryClient, type MutationOptions } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const api = vi.hoisted(() => ({
  listProviders: vi.fn(),
  getRolesOverview: vi.fn(),
  getBudgets: vi.fn(),
  getAIReports: vi.fn(),
  createProvider: vi.fn(),
  updateProvider: vi.fn(),
  deleteProvider: vi.fn(),
  updateAISettings: vi.fn(),
  updateProviderRoles: vi.fn(),
  updateBudget: vi.fn(),
}));
vi.mock("@/api/ai-provider", () => api);

import type { AIProvider } from "@/api/ai-provider";
import {
  aiBudgetsQuery,
  aiProvidersQuery,
  aiRolesOverviewQuery,
  aiUsageReportsQuery,
  assignAIRoleMutation,
  createAIProviderMutation,
  deleteAIProviderMutation,
  updateAIBudgetMutation,
  updateAIGlobalSettingsMutation,
  updateAIProviderMutation,
  USAGE_REPORT_PAGE_SIZE,
} from "./ai-settings";
import { aiSettingsKeys } from "./keys";

const assignmentKeys = [aiSettingsKeys.providers(), aiSettingsKeys.roles(), aiSettingsKeys.chatReadiness()];

function clientWithCache(): QueryClient {
  const client = new QueryClient();
  for (const key of [...assignmentKeys, aiSettingsKeys.budgets()]) client.setQueryData(key, { cached: true });
  return client;
}

function invalidated(client: QueryClient, key: readonly unknown[]): boolean {
  return client.getQueryState(key)?.isInvalidated === true;
}

function runMutation<TResult, TVariables>(
  client: QueryClient,
  options: MutationOptions<TResult, Error, TVariables>,
  variables: TVariables,
) {
  return client.getMutationCache().build(client, options).execute(variables);
}

beforeEach(() => {
  for (const call of Object.values(api)) {
    call.mockReset();
    call.mockResolvedValue({ data: {} });
  }
});

describe("读：Provider / 角色 / 预算", () => {
  it("没有 ai.settings.read 权限时不发请求（页面此时整页显示演示数据）", () => {
    expect(aiProvidersQuery(false).enabled).toBe(false);
    expect(aiRolesOverviewQuery(false).enabled).toBe(false);
    expect(aiBudgetsQuery(false).enabled).toBe(false);
    expect(aiUsageReportsQuery("", false).enabled).toBe(false);
    expect(aiProvidersQuery(true).enabled).toBe(true);
  });

  it("Provider 列表全页共用一个键：多处使用只请求一次", async () => {
    api.listProviders.mockResolvedValue({ data: { providers: [], settings: { toolTimeout: 30 }, models: [] } });
    const client = new QueryClient();

    const [first, second] = await Promise.all([
      client.fetchQuery(aiProvidersQuery(true)),
      client.fetchQuery(aiProvidersQuery(true)),
    ]);

    expect(first).toBe(second);
    expect(api.listProviders).toHaveBeenCalledTimes(1);
    expect(api.listProviders.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
  });

  it("返回解包后的数据，带取消信号", async () => {
    api.getRolesOverview.mockResolvedValue({ data: { data: [{ role: "chat" }] } });
    api.getBudgets.mockResolvedValue({ data: { data: [{ role: "chat" }] } });
    const client = new QueryClient();

    expect(await client.fetchQuery(aiRolesOverviewQuery(true))).toEqual([{ role: "chat" }]);
    expect(await client.fetchQuery(aiBudgetsQuery(true))).toEqual([{ role: "chat" }]);
    expect(api.getRolesOverview.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
    expect(api.getBudgets.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
  });
});

describe("aiUsageReportsQuery", () => {
  it("角色进键：换角色从第一页重新开始", () => {
    expect(aiUsageReportsQuery("", true).queryKey).toEqual(["ai-settings", "usage-reports", { role: "" }]);
    expect(aiUsageReportsQuery("", true).queryKey).not.toEqual(aiUsageReportsQuery("chat", true).queryKey);
  });

  it("下一页从已加载的条数开始；加载完或拿到空页就没有下一页", () => {
    const { getNextPageParam, initialPageParam } = aiUsageReportsQuery("", true);
    const page = (count: number, total: number) => ({ reports: new Array(count).fill({}), total });
    expect(initialPageParam).toBe(0);
    expect(getNextPageParam(page(10, 25), [page(10, 25)], 0, [0])).toBe(10);
    expect(getNextPageParam(page(5, 25), [page(10, 25), page(10, 25), page(5, 25)], 20, [0, 10, 20])).toBeUndefined();
    expect(getNextPageParam(page(0, 25), [page(10, 25), page(0, 25)], 10, [0, 10])).toBeUndefined();
  });

  it("按页大小与偏移取数，空角色不带筛选，带取消信号", async () => {
    api.getAIReports.mockResolvedValue({ data: { data: [{ id: 1 }], total: 1 } });

    const result = await new QueryClient().fetchInfiniteQuery(aiUsageReportsQuery("", true));

    expect(result.pages).toEqual([{ reports: [{ id: 1 }], total: 1 }]);
    const [params, signal] = api.getAIReports.mock.calls[0];
    expect(params).toEqual({ role: undefined, limit: USAGE_REPORT_PAGE_SIZE, offset: 0 });
    expect(signal).toBeInstanceOf(AbortSignal);
  });
});

describe("写：Provider 与全局设置", () => {
  it.each([
    ["新增", () => createAIProviderMutation(), { name: "p", provider: "openai", apiKey: "k", model: "m" }],
    ["修改", () => updateAIProviderMutation(), { providerId: 3, changes: { name: "p2" } }],
    ["删除", () => deleteAIProviderMutation(), 3],
    ["全局设置", () => updateAIGlobalSettingsMutation(), { toolTimeout: 60 }],
  ] as const)("%s成功后，Provider 列表、角色总览、对话可用性都重新取数", async (_label, factory, variables) => {
    const client = clientWithCache();

    await runMutation(client, factory() as MutationOptions<unknown, Error, unknown>, variables);

    expect(assignmentKeys.map((key) => invalidated(client, key))).toEqual([true, true, true]);
    expect(invalidated(client, aiSettingsKeys.budgets())).toBe(false);
  });

  it("修改 Provider 时把 ID 与改动分开交给接口", async () => {
    await runMutation(clientWithCache(), updateAIProviderMutation(), { providerId: 3, changes: { name: "p2" } });
    expect(api.updateProvider).toHaveBeenCalledWith(3, { name: "p2" });
  });

  it("失败时不动缓存", async () => {
    api.deleteProvider.mockRejectedValue(new Error("in use"));
    const client = clientWithCache();

    await expect(runMutation(client, deleteAIProviderMutation(), 3)).rejects.toThrow("in use");
    expect(assignmentKeys.map((key) => invalidated(client, key))).toEqual([false, false, false]);
  });
});

describe("assignAIRoleMutation", () => {
  const providers = [
    { id: 1, roles: ["chat", "background"] },
    { id: 2, roles: [] },
  ] as unknown as AIProvider[];

  it("先从现在的持有者身上拿掉角色，再加到新的 Provider 上", async () => {
    await runMutation(clientWithCache(), assignAIRoleMutation(), { role: "chat", providerId: 2, providers });

    expect(api.updateProviderRoles.mock.calls).toEqual([
      [1, ["background"]],
      [2, ["chat"]],
    ]);
  });

  it("取消分配：只从持有者身上拿掉", async () => {
    await runMutation(clientWithCache(), assignAIRoleMutation(), { role: "chat", providerId: null, providers });
    expect(api.updateProviderRoles.mock.calls).toEqual([[1, ["background"]]]);
  });

  it("无论成败都重新取数：第二步失败时界面要如实显示该角色没人持有", async () => {
    api.updateProviderRoles.mockResolvedValueOnce({ data: {} }).mockRejectedValueOnce(new Error("conflict"));
    const client = clientWithCache();

    await expect(
      runMutation(client, assignAIRoleMutation(), { role: "chat", providerId: 2, providers }),
    ).rejects.toThrow("conflict");
    expect(assignmentKeys.map((key) => invalidated(client, key))).toEqual([true, true, true]);
  });
});

describe("updateAIBudgetMutation", () => {
  it("把角色与改动分开交给接口，成功后只让预算重新取数", async () => {
    const client = clientWithCache();

    await runMutation(client, updateAIBudgetMutation(), { role: "chat", changes: { dailyCallLimit: 100 } });

    expect(api.updateBudget).toHaveBeenCalledWith("chat", { dailyCallLimit: 100 });
    expect(invalidated(client, aiSettingsKeys.budgets())).toBe(true);
    expect(invalidated(client, aiSettingsKeys.providers())).toBe(false);
  });
});
