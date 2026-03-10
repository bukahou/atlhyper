/**
 * Deploy 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 */

import { getDataSourceMode } from "@/config/data-source";
import * as mock from "@/mock/deploy";
import * as api from "@/api/deploy";

export async function getDeployConfig(clusterId: string) {
  if (getDataSourceMode("deploy") === "mock") {
    return mock.mockGetDeployConfig();
  }
  const res = await api.getConfig(clusterId);
  return res.data.data;
}

export async function saveDeployConfig(data: {
  clusterId: string;
  repoUrl: string;
  paths: string[];
  intervalSec: number;
  autoDeploy: boolean;
}) {
  const res = await api.saveConfig(data);
  return res.data;
}

export async function getKustomizePaths(repo: string) {
  if (getDataSourceMode("deploy") === "mock") {
    return mock.mockGetKustomizePaths(repo);
  }
  const res = await api.getKustomizePaths(repo);
  return res.data.data;
}

export async function testDeployConnection() {
  const res = await api.testConnection();
  return res.data.data.success;
}

export async function getDeployHistory(params: { clusterId: string; path?: string }) {
  if (getDataSourceMode("deploy") === "mock") {
    return mock.mockGetDeployHistory();
  }
  const res = await api.getHistory(params);
  return res.data.data;
}

export async function getAuthorizedRepos() {
  if (getDataSourceMode("deploy") === "mock") {
    return mock.mockGetAuthorizedRepos();
  }
  // 部署页面的仓库列表也通过 deploy mock 获取
  const res = await api.getKustomizePaths("");
  return res.data.data;
}
