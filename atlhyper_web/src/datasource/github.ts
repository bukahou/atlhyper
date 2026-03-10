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

export async function getRepoNamespaces(repo: string) {
  if (getDataSourceMode("github") === "mock") {
    const { mockGetRepoNamespaces } = await import("@/mock/github");
    return mockGetRepoNamespaces(repo);
  }
  const res = await api.getRepoNamespaces(repo);
  return res.data.data;
}

export async function addRepoNamespace(repo: string, namespace: string) {
  const res = await api.addRepoNamespace(repo, namespace);
  return res.data.data;
}

export async function removeRepoNamespace(repo: string, namespace: string) {
  await api.removeRepoNamespace(repo, namespace);
}

export async function getMappings() {
  if (getDataSourceMode("github") === "mock") {
    const { mockGetRepoMappings } = await import("@/mock/github");
    return mockGetRepoMappings();
  }
  const res = await api.getMappings();
  return res.data.data;
}

export async function createMapping(data: {
  clusterId: string;
  repo: string;
  namespace: string;
  deployment: string;
  container?: string;
  imagePrefix?: string;
  sourcePath?: string;
}) {
  const res = await api.createMapping(data);
  return res.data.data;
}

export async function updateMapping(id: number, data: {
  namespace?: string;
  deployment?: string;
  container?: string;
  imagePrefix?: string;
  sourcePath?: string;
}) {
  const res = await api.updateMapping(id, data);
  return res.data.data;
}

export async function confirmMappingAPI(id: number) {
  await api.confirmMapping(id);
}

export async function deleteMappingAPI(id: number) {
  await api.deleteMapping(id);
}
