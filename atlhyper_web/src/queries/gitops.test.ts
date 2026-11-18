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
const deploy = vi.hoisted(() => ({
  getDeployConfig: vi.fn(),
  getDeployStatus: vi.fn(),
  getDeployHistory: vi.fn(),
  getKustomizePaths: vi.fn(),
  saveDeployConfig: vi.fn(),
  testDeployConnection: vi.fn(),
  syncDeployNow: vi.fn(),
}));
const dataSourceMode = vi.hoisted(() => ({ current: "api" }));
vi.mock("@/datasource/github", () => github);
vi.mock("@/datasource/deploy", () => deploy);
vi.mock("@/config/data-source", () => ({ getDataSourceMode: () => dataSourceMode.current }));

import {
  completeGitHubInstallationMutation,
  connectGitHubMutation,
  deployConfigQuery,
  deployHistoryQuery,
  deployStatusQuery,
  disconnectGitHubMutation,
  githubConnectionQuery,
  githubReposQuery,
  kustomizePathsQuery,
  saveDeployConfigMutation,
  syncDeployNowMutation,
} from "./gitops";
import { deployKeys, githubKeys } from "./keys";

const connectionKey = githubKeys.connection("api");
const configKey = deployKeys.config("c1", "api");
const statusKey = deployKeys.status("api");
const historyKey = deployKeys.history("c1", "api");

function invalidated(client: QueryClient, key: readonly unknown[]): boolean {
  return client.getQueryState(key)?.isInvalidated === true;
}

beforeEach(() => {
  for (const call of [...Object.values(github), ...Object.values(deploy)]) call.mockReset();
  dataSourceMode.current = "api";
});

describe("启用条件", () => {
  it("没有权限时一律不发请求（页面此时整页显示演示数据）", () => {
    expect(githubConnectionQuery(false).enabled).toBe(false);
    expect(githubReposQuery(false, true).enabled).toBe(false);
    expect(deployConfigQuery("c1", false).enabled).toBe(false);
    expect(deployStatusQuery(false, true).enabled).toBe(false);
    expect(deployHistoryQuery("c1", false).enabled).toBe(false);
    expect(kustomizePathsQuery("org/repo", false).enabled).toBe(false);
  });

  it("仓库要先连接 GitHub；同步状态要先有部署配置；路径要先选仓库", () => {
    expect(githubConnectionQuery(true).enabled).toBe(true);
    expect(githubReposQuery(true, false).enabled).toBe(false);
    expect(githubReposQuery(true, true).enabled).toBe(true);
    expect(deployStatusQuery(true, false).enabled).toBe(false);
    expect(deployStatusQuery(true, true).enabled).toBe(true);
    expect(kustomizePathsQuery("", true).enabled).toBe(false);
    expect(kustomizePathsQuery("org/repo", true).enabled).toBe(true);
  });

  it("部署配置与历史按集群取：没有选定集群时不启用，演示数据不依赖集群", () => {
    expect(deployConfigQuery("", true).enabled).toBe(false);
    expect(deployHistoryQuery("", true).enabled).toBe(false);
    expect(deployConfigQuery("c1", true).enabled).toBe(true);

    dataSourceMode.current = "mock";
    expect(deployConfigQuery("", true).enabled).toBe(true);
  });
});

describe("键与信号", () => {
  it("键里带集群与数据源", () => {
    expect(deployConfigQuery("c1", true).queryKey).toEqual(["deploy", "config", { clusterId: "c1", mode: "api" }]);
    expect(deployConfigQuery("c1", true).queryKey).not.toEqual(deployConfigQuery("c2", true).queryKey);
    expect(kustomizePathsQuery("org/a", true).queryKey).not.toEqual(kustomizePathsQuery("org/b", true).queryKey);
    expect(githubConnectionQuery(true).queryKey).toEqual(["github", "connection", { mode: "api" }]);
  });

  it("把参数与取消信号交给数据源", async () => {
    github.getGitHubConnection.mockResolvedValue({ connected: true });
    deploy.getDeployConfig.mockResolvedValue(null);
    deploy.getDeployHistory.mockResolvedValue([]);
    deploy.getKustomizePaths.mockResolvedValue([]);
    const client = new QueryClient();

    await client.fetchQuery(githubConnectionQuery(true));
    await client.fetchQuery(deployConfigQuery("c1", true));
    await client.fetchQuery(deployHistoryQuery("c1", true));
    await client.fetchQuery(kustomizePathsQuery("org/repo", true));

    expect(github.getGitHubConnection.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
    expect(deploy.getDeployConfig.mock.calls[0][0]).toBe("c1");
    expect(deploy.getDeployConfig.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
    expect(deploy.getDeployHistory.mock.calls[0][0]).toBe("c1");
    expect(deploy.getDeployHistory.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
    expect(deploy.getKustomizePaths.mock.calls[0][0]).toBe("org/repo");
    expect(deploy.getKustomizePaths.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
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

  it("保存部署配置：成功后配置、状态、历史都重新取数；失败不动", async () => {
    const config = { repoUrl: "org/repo", paths: ["a"], intervalSec: 60, autoDeploy: true, clusterId: "c1" };
    const client = new QueryClient();
    for (const key of [configKey, statusKey, historyKey]) client.setQueryData(key, { cached: true });

    deploy.saveDeployConfig.mockRejectedValueOnce(new Error("invalid"));
    await expect(client.getMutationCache().build(client, saveDeployConfigMutation()).execute(config)).rejects.toThrow(
      "invalid",
    );
    expect([configKey, statusKey, historyKey].map((key) => invalidated(client, key))).toEqual([false, false, false]);

    deploy.saveDeployConfig.mockResolvedValue({ message: "ok" });
    await client.getMutationCache().build(client, saveDeployConfigMutation()).execute(config);
    expect(deploy.saveDeployConfig).toHaveBeenLastCalledWith(config);
    expect([configKey, statusKey, historyKey].map((key) => invalidated(client, key))).toEqual([true, true, true]);
  });

  it("立即同步只提交任务，不立刻刷新（由调用方延时刷新）", async () => {
    deploy.syncDeployNow.mockResolvedValue({ message: "queued" });
    const client = new QueryClient();
    client.setQueryData(statusKey, []);

    await client.getMutationCache().build(client, syncDeployNowMutation()).execute("apps/web");

    expect(deploy.syncDeployNow).toHaveBeenCalledWith("apps/web");
    expect(invalidated(client, statusKey)).toBe(false);
  });
});
