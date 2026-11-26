import { QueryClient, type MutationOptions } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const api = vi.hoisted(() => ({
  getServiceAccounts: vi.fn(),
  createServiceAccount: vi.fn(),
  updateServiceAccount: vi.fn(),
  deleteServiceAccount: vi.fn(),
  issueAPIToken: vi.fn(),
  revokeAPIToken: vi.fn(),
}));
vi.mock("@/api/service-accounts", () => api);

import type { ServiceAccountLimits, ServiceAccountView } from "@/api/service-accounts";
import { adminKeys } from "./keys";
import {
  issueAPITokenMutation,
  revokeAPITokenMutation,
  serviceAccountsQuery,
  updateServiceAccountMutation,
} from "./service-accounts";

// 测试用的令牌：明显是占位值，不是真实令牌
const TOKEN_PLAINTEXT = "atlp_placeholder-not-a-real-token";

const account: ServiceAccountView = {
  id: 7,
  name: "haro",
  displayName: "Haro",
  role: 0,
  enabled: true,
  createdAt: "2026-11-20T00:00:00Z",
  lastUsedAt: "",
  canIssueToken: true,
  tokens: [],
};
const limits: ServiceAccountLimits = {
  assignableRoles: [0, 1, 2],
  maxActiveTokens: 10,
  maxExpiresInDays: 3650,
  defaultExpiresInDays: 90,
};

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

describe("serviceAccountsQuery", () => {
  it("没有 user.manage 时不启用", () => {
    expect(serviceAccountsQuery(false).enabled).toBe(false);
    expect(serviceAccountsQuery(true).enabled).toBe(true);
  });

  it("键固定；返回账号与业务参数，带取消信号", async () => {
    expect(serviceAccountsQuery(true).queryKey).toEqual(adminKeys.serviceAccounts());

    api.getServiceAccounts.mockResolvedValueOnce({ data: { message: "获取成功", data: [account], total: 1, limits } });
    expect(await new QueryClient().fetchQuery(serviceAccountsQuery(true))).toEqual({ accounts: [account], limits });
    expect(api.getServiceAccounts.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
  });
});

describe("issueAPITokenMutation", () => {
  it("明文只经 deliverToken 交给调用方一次，不留在任何缓存里；列表随后重新取数", async () => {
    api.issueAPIToken.mockResolvedValue({
      data: { message: "已签发", data: { token: TOKEN_PLAINTEXT, tokenPrefix: "atlp_pla", expiresAt: "" } },
    });
    const client = new QueryClient();
    client.setQueryData(adminKeys.serviceAccounts(), { accounts: [account], limits });
    const delivered: string[] = [];

    const summary = await runMutation(client, issueAPITokenMutation(), {
      accountId: 7,
      request: { name: "prod", expiresInDays: 90 },
      deliverToken: (token: string) => delivered.push(token),
    });

    expect(api.issueAPIToken).toHaveBeenCalledWith(7, { name: "prod", expiresInDays: 90 });
    expect(delivered).toEqual([TOKEN_PLAINTEXT]);
    expect(summary).toEqual({ tokenPrefix: "atlp_pla", expiresAt: "" });
    expect(client.getMutationCache().getAll()).toHaveLength(1);
    expect(everyCachedString(client)).not.toContain(TOKEN_PLAINTEXT);
    expect(client.getQueryState(adminKeys.serviceAccounts())?.isInvalidated).toBe(true);
  });

  it("签发失败不交出任何东西，错误原样抛给调用方", async () => {
    api.issueAPIToken.mockRejectedValue(new Error("conflict"));
    const delivered: string[] = [];
    await expect(
      runMutation(new QueryClient(), issueAPITokenMutation(), {
        accountId: 7,
        request: { name: "prod" },
        deliverToken: (token: string) => delivered.push(token),
      }),
    ).rejects.toThrow("conflict");
    expect(delivered).toEqual([]);
  });
});

describe("其余写操作", () => {
  it("修改：整体替换，三项都带上", async () => {
    api.updateServiceAccount.mockResolvedValue({ data: { message: "已更新", data: account } });
    await runMutation(new QueryClient(), updateServiceAccountMutation(), {
      accountId: 7,
      request: { displayName: "Haro", role: 1, enabled: false },
    });
    expect(api.updateServiceAccount).toHaveBeenCalledWith(7, { displayName: "Haro", role: 1, enabled: false });
  });

  it("吊销：按账号与令牌 id 调用，成功后列表重新取数", async () => {
    api.revokeAPIToken.mockResolvedValue({ data: { message: "已吊销" } });
    const client = new QueryClient();
    client.setQueryData(adminKeys.serviceAccounts(), { accounts: [account], limits });
    await runMutation(client, revokeAPITokenMutation(), { accountId: 7, tokenId: 3 });
    expect(api.revokeAPIToken).toHaveBeenCalledWith(7, 3);
    expect(client.getQueryState(adminKeys.serviceAccounts())?.isInvalidated).toBe(true);
  });
});
