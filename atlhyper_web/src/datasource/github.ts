/**
 * GitHub 数据源代理
 *
 * 根据中心配置自动切换 mock / api。约定同 datasource/cluster.ts：
 * - 读函数返回解包后的数据，mock 分支返回相同形态
 * - 读函数最后一个参数是可选的 AbortSignal，原样交给 api 层，用于取消请求
 *
 * 这里不判断权限：由查询工厂的 enabled 把关。
 */

import { getDataSourceMode } from "@/config/data-source";
import * as mock from "@/mock/github";
import type { MockAuthorizedRepo, MockGitHubConnection } from "@/mock/github/data";
import * as api from "@/api/github";

/** GitHub App 的连接状态 */
export async function getGitHubConnection(signal?: AbortSignal): Promise<MockGitHubConnection> {
  if (getDataSourceMode("github") === "mock") return mock.mockGetGitHubConnection();
  return (await api.getConnection(signal)).data.data;
}

/** 已授权给 GitHub App 的仓库 */
export async function getAuthorizedRepos(signal?: AbortSignal): Promise<MockAuthorizedRepo[]> {
  if (getDataSourceMode("github") === "mock") return mock.mockGetAuthorizedRepos();
  return (await api.getRepos(signal)).data.data ?? [];
}

// ============================================================
// 写操作 —— 一律走真实接口（没有演示数据）
// ============================================================

/** 发起连接：已经装过 App 时直接返回 connected，否则返回要跳转去的授权地址 */
export async function connectGitHub() {
  return (await api.connect()).data.data;
}

/** GitHub App 安装完成后的回跳：把 installation 交给后端登记 */
export async function completeGitHubInstallation(installationId: number, setupAction: string) {
  return (await api.callback(installationId, setupAction)).data.data;
}

export async function disconnectGitHub(): Promise<void> {
  await api.disconnect();
}
