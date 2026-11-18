/**
 * GitHub 连接与 GitOps 部署的取数与写操作描述（设置 → GitHub & GitOps 页）
 *
 * 读需要权限（github.manage / deploy.manage）：有没有权限由调用方用 useCan 得到后传进来；
 * 没有权限时页面整页显示演示数据（由页面自己取，不经过这里）。
 */
import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import {
  getDeployConfig,
  getDeployHistory,
  getDeployStatus,
  getKustomizePaths,
  saveDeployConfig,
  syncDeployNow,
  testDeployConnection,
} from "@/datasource/deploy";
import {
  completeGitHubInstallation,
  connectGitHub,
  disconnectGitHub,
  getAuthorizedRepos,
  getGitHubConnection,
} from "@/datasource/github";
import type { MockDeployConfig } from "@/mock/deploy/data";
import { clusterDataAvailable } from "./availability";
import { deployKeys, githubKeys } from "./keys";
import { REFRESH } from "./refresh";

// ============================================================
// GitHub 连接
// ============================================================

/** GitHub App 的连接状态。连接页与部署页共用这一份 */
export function githubConnectionQuery(canManageGitHub: boolean) {
  return queryOptions({
    queryKey: githubKeys.connection(getDataSourceMode("github")),
    queryFn: ({ signal }) => getGitHubConnection(signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: canManageGitHub,
  });
}

/** 已授权的仓库。connected 为假（还没连接 GitHub）时不取 */
export function githubReposQuery(canManageGitHub: boolean, connected: boolean) {
  return queryOptions({
    queryKey: githubKeys.repos(getDataSourceMode("github")),
    queryFn: ({ signal }) => getAuthorizedRepos(signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: canManageGitHub && connected,
  });
}

/** 发起连接。已经装过 App 时后端直接返回 connected：让连接状态重新取数；否则由调用方跳转到授权地址 */
export function connectGitHubMutation() {
  return mutationOptions({
    mutationFn: () => connectGitHub(),
    onSuccess: (connectResult, _variables, _onMutateResult, context) => {
      if (connectResult?.connected) void context.client.invalidateQueries({ queryKey: githubKeys.all });
    },
  });
}

/** 断开连接。无论成败都重新取连接状态：界面显示后端的真实状态 */
export function disconnectGitHubMutation() {
  return mutationOptions({
    mutationFn: () => disconnectGitHub(),
    onSettled: (_result, _error, _variables, _onMutateResult, context) =>
      context.client.invalidateQueries({ queryKey: githubKeys.all }),
  });
}

export interface GitHubInstallation {
  installationId: number;
  setupAction: string;
}

/** GitHub App 安装完成后的回跳：把 installation 交给后端登记 */
export function completeGitHubInstallationMutation() {
  return mutationOptions({
    mutationFn: ({ installationId, setupAction }: GitHubInstallation) =>
      completeGitHubInstallation(installationId, setupAction),
    onSuccess: (_connection, _installation, _onMutateResult, context) => {
      void context.client.invalidateQueries({ queryKey: githubKeys.all });
    },
  });
}

// ============================================================
// 部署
// ============================================================

/** 某个集群的部署配置（还没配置过时为 null） */
export function deployConfigQuery(clusterId: string, canManageDeploy: boolean) {
  const mode = getDataSourceMode("deploy");
  return queryOptions({
    queryKey: deployKeys.config(clusterId, mode),
    queryFn: ({ signal }) => getDeployConfig(clusterId, signal),
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
    enabled: canManageDeploy && clusterDataAvailable(clusterId, mode),
  });
}

/** 各部署路径的同步状态。hasConfig 为假（还没有部署配置）时不取 */
export function deployStatusQuery(canManageDeploy: boolean, hasConfig: boolean) {
  return queryOptions({
    queryKey: deployKeys.status(getDataSourceMode("deploy")),
    queryFn: ({ signal }) => getDeployStatus(signal),
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
    enabled: canManageDeploy && hasConfig,
  });
}

/** 某个集群的部署历史 */
export function deployHistoryQuery(clusterId: string, canManageDeploy: boolean) {
  const mode = getDataSourceMode("deploy");
  return queryOptions({
    queryKey: deployKeys.history(clusterId, mode),
    queryFn: ({ signal }) => getDeployHistory(clusterId, signal),
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
    enabled: canManageDeploy && clusterDataAvailable(clusterId, mode),
  });
}

/** 某个仓库里的 kustomization 路径（配置表单里选路径用）。还没选仓库时不取 */
export function kustomizePathsQuery(repo: string, canManageDeploy: boolean) {
  return queryOptions({
    queryKey: deployKeys.kustomizePaths(getDataSourceMode("deploy"), repo),
    queryFn: ({ signal }) => getKustomizePaths(repo, signal),
    enabled: canManageDeploy && repo !== "",
  });
}

/**
 * 保存部署配置。成功后等配置、状态、历史重新取完才算结束 ——
 * 保存按钮转圈停下时，界面上已经是新配置。
 */
export function saveDeployConfigMutation() {
  return mutationOptions({
    mutationFn: (config: MockDeployConfig) => saveDeployConfig(config),
    onSuccess: (_saved, _config, _onMutateResult, context) =>
      context.client.invalidateQueries({ queryKey: deployKeys.all }),
  });
}

/** 测试到仓库的连接是否可用 */
export function testDeployConnectionMutation() {
  return mutationOptions({ mutationFn: () => testDeployConnection() });
}

/**
 * 立即同步某个路径。
 *
 * 接口只是提交同步任务，所以这里不刷新：调用方等几秒后再让状态与历史失效（见 useDeployPage）。
 */
export function syncDeployNowMutation() {
  return mutationOptions({ mutationFn: (path: string) => syncDeployNow(path) });
}
