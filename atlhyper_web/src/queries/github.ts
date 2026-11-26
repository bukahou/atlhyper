/**
 * GitHub 连接的取数与写操作描述（设置 → GitHub 页）
 *
 * 读需要权限（github.manage）：有没有权限由调用方用 useCan 得到后传进来；
 * 没有权限时页面整页显示演示数据（由页面自己取，不经过这里）。
 */
import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import {
  completeGitHubInstallation,
  connectGitHub,
  disconnectGitHub,
  getAuthorizedRepos,
  getGitHubConnection,
} from "@/datasource/github";
import { githubKeys } from "./keys";
import { REFRESH } from "./refresh";

/** GitHub App 的连接状态 */
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
