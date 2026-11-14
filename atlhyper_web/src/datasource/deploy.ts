/**
 * Deploy 数据源代理
 *
 * 根据中心配置自动切换 mock / api。约定同 datasource/cluster.ts：
 * - 读函数返回解包后的数据，mock 分支返回相同形态
 * - 读函数最后一个参数是可选的 AbortSignal，原样交给 api 层，用于取消请求
 *
 * 这里不判断权限：由查询工厂的 enabled 把关。
 */

import { getDataSourceMode } from "@/config/data-source";
import * as mock from "@/mock/deploy";
import type { MockDeployConfig, MockDeployRecord, MockPathStatus } from "@/mock/deploy/data";
import * as api from "@/api/deploy";

/** 某个集群的部署配置；还没配置过时为 null */
export async function getDeployConfig(clusterId: string, signal?: AbortSignal): Promise<MockDeployConfig | null> {
  if (getDataSourceMode("deploy") === "mock") return mock.mockGetDeployConfig();
  return (await api.getConfig(clusterId, signal)).data.data;
}

/** 某个仓库里发现的全部 kustomization 路径 */
export async function getKustomizePaths(repo: string, signal?: AbortSignal): Promise<string[]> {
  if (getDataSourceMode("deploy") === "mock") return mock.mockGetKustomizePaths(repo);
  return (await api.getKustomizePaths(repo, signal)).data.data ?? [];
}

/** 各部署路径的同步状态 */
export async function getDeployStatus(signal?: AbortSignal): Promise<MockPathStatus[]> {
  if (getDataSourceMode("deploy") === "mock") return mock.mockGetPathStatus();
  return (await api.getStatus(signal)).data.data ?? [];
}

/** 某个集群的部署历史 */
export async function getDeployHistory(clusterId: string, signal?: AbortSignal): Promise<MockDeployRecord[]> {
  if (getDataSourceMode("deploy") === "mock") return mock.mockGetDeployHistory();
  // 列表接口只带概要字段，详情字段在弹窗里按有无显示
  return ((await api.getHistory({ clusterId }, signal)).data.data ?? []) as MockDeployRecord[];
}

// ============================================================
// 写操作 —— 一律走真实接口（没有演示数据）
// ============================================================

export async function saveDeployConfig(config: MockDeployConfig) {
  return (await api.saveConfig(config)).data;
}

/** 测试到仓库的连接是否可用 */
export async function testDeployConnection(): Promise<boolean> {
  return (await api.testConnection()).data.data.success;
}

/** 立即同步某个路径（只是提交同步任务，结果稍后才反映在状态与历史里） */
export async function syncDeployNow(path: string) {
  return (await api.syncNow(path)).data;
}
