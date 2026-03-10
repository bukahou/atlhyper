/**
 * GitHub 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 */

import { getDataSourceMode } from "@/config/data-source";
import * as mock from "@/mock/github";
import * as api from "@/api/github";

export async function getGitHubConnection() {
  if (getDataSourceMode("github") === "mock") {
    return mock.mockGetGitHubConnection();
  }
  const res = await api.getConnection();
  return res.data.data;
}

export async function connectGitHub() {
  const res = await api.connect();
  return res.data.data;
}

export async function callbackGitHub(code: string) {
  const res = await api.callback(code);
  return res.data.data;
}

export async function disconnectGitHub() {
  await api.disconnect();
}

export async function getAuthorizedRepos() {
  if (getDataSourceMode("github") === "mock") {
    return mock.mockGetAuthorizedRepos();
  }
  const res = await api.getRepos();
  return res.data.data;
}

export async function toggleRepoMapping(repo: string, enabled: boolean) {
  const res = await api.toggleMapping(repo, enabled);
  return res.data;
}

export async function getRepoDirs(repo: string) {
  if (getDataSourceMode("github") === "mock") {
    return mock.mockGetRepoDirs(repo);
  }
  const res = await api.getRepoDirs(repo);
  return res.data.data;
}
