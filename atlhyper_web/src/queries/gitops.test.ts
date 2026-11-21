import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const github = vi.hoisted(() => ({
  getGitHubConnection: vi.fn(),
  getAuthorizedRepos: vi.fn(),
  connectGitHub: vi.fn(),
  disconnectGitHub: vi.fn(),
  completeGitHubInstallation: vi.fn(),
}));
const dataSourceMode = vi.hoisted(() => ({ current: "api" }));
vi.mock("@/datasource/github", () => github);
vi.mock("@/config/data-source", () => ({ getDataSourceMode: () => dataSourceMode.current }));

import {
  completeGitHubInstallationMutation,
  connectGitHubMutation,
  disconnectGitHubMutation,
  githubConnectionQuery,
  githubReposQuery,
} from "./gitops";
import { githubKeys } from "./keys";

const connectionKey = githubKeys.connection("api");

function invalidated(client: QueryClient, key: readonly unknown[]): boolean {
  return client.getQueryState(key)?.isInvalidated === true;
}

beforeEach(() => {
  for (const call of Object.values(github)) call.mockReset();
  dataSourceMode.current = "api";
});

describe("启用条件", () => {
  it("没有权限时一律不发请求（页面此时整页显示演示数据）", () => {
    expect(githubConnectionQuery(false).enabled).toBe(false);
    expect(githubReposQuery(false, true).enabled).toBe(false);
  });

  it("仓库要先连接 GitHub", () => {
    expect(githubConnectionQuery(true).enabled).toBe(true);
    expect(githubReposQuery(true, false).enabled).toBe(false);
    expect(githubReposQuery(true, true).enabled).toBe(true);
  });
});

describe("键与信号", () => {
  it("键里带数据源", () => {
    expect(githubConnectionQuery(true).queryKey).toEqual(["github", "connection", { mode: "api" }]);
  });

  it("把取消信号交给数据源", async () => {
    github.getGitHubConnection.mockResolvedValue({ connected: true });
    const client = new QueryClient();

    await client.fetchQuery(githubConnectionQuery(true));

    expect(github.getGitHubConnection.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
  });
});

describe("写操作", () => {
  it("发起连接：已经连上时让连接状态重新取数；需要跳转授权时不动缓存", async () => {
    const client = new QueryClient();
    client.setQueryData(connectionKey, { connected: false });

    github.connectGitHub.mockResolvedValue({ authUrl: "https://example.com/authorize" });
    await client.getMutationCache().build(client, connectGitHubMutation()).execute();
    expect(invalidated(client, connectionKey)).toBe(false);

    github.connectGitHub.mockResolvedValue({ connected: true });
    await client.getMutationCache().build(client, connectGitHubMutation()).execute();
    expect(invalidated(client, connectionKey)).toBe(true);
  });

  it("断开连接：无论成败都重新取连接状态", async () => {
    github.disconnectGitHub.mockRejectedValue(new Error("502"));
    const client = new QueryClient();
    client.setQueryData(connectionKey, { connected: true });

    await expect(client.getMutationCache().build(client, disconnectGitHubMutation()).execute()).rejects.toThrow("502");
    expect(invalidated(client, connectionKey)).toBe(true);
  });

  it("安装回跳：登记成功后让连接状态重新取数", async () => {
    github.completeGitHubInstallation.mockResolvedValue({ connected: true });
    const client = new QueryClient();
    client.setQueryData(connectionKey, { connected: false });

    await client
      .getMutationCache()
      .build(client, completeGitHubInstallationMutation())
      .execute({ installationId: 42, setupAction: "install" });

    expect(github.completeGitHubInstallation).toHaveBeenCalledWith(42, "install");
    expect(invalidated(client, connectionKey)).toBe(true);
  });
});
