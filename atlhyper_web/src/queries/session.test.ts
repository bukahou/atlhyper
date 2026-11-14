import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const authApi = vi.hoisted(() => ({ getAuthProviders: vi.fn(), login: vi.fn(), oidcStartURL: vi.fn() }));
const clusterApi = vi.hoisted(() => ({ getClusterList: vi.fn() }));
vi.mock("@/api/auth", () => authApi);
vi.mock("@/api/cluster", () => clusterApi);

import { authProvidersQuery, loginMutation } from "./session";

beforeEach(() => {
  authApi.getAuthProviders.mockReset();
  authApi.login.mockReset();
});

describe("authProvidersQuery", () => {
  it("只在登录框打开时取；一次会话里不重取", () => {
    expect(authProvidersQuery(false).enabled).toBe(false);
    expect(authProvidersQuery(true).enabled).toBe(true);
    expect(authProvidersQuery(true).staleTime).toBe(Infinity);
  });

  it("返回解包后的列表，带取消信号", async () => {
    authApi.getAuthProviders.mockResolvedValue({ data: { data: [{ id: "oidc", displayName: "SSO" }] } });

    expect(await new QueryClient().fetchQuery(authProvidersQuery(true))).toEqual([{ id: "oidc", displayName: "SSO" }]);
    expect(authApi.getAuthProviders.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
  });
});

describe("loginMutation", () => {
  it("把用户名密码交给接口，返回解包后的登录结果", async () => {
    const loginData = { token: "demo-token", user: { id: 1 }, cluster_ids: [] };
    authApi.login.mockResolvedValue({ data: { data: loginData } });
    const client = new QueryClient();

    const result = await client
      .getMutationCache()
      .build(client, loginMutation())
      .execute({ username: "u", password: "p" });

    expect(authApi.login).toHaveBeenCalledWith({ username: "u", password: "p" });
    expect(result).toBe(loginData);
  });
});
