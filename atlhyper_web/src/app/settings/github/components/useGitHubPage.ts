import { useMutation, useQuery } from "@tanstack/react-query";
import * as mock from "@/mock/github";
import type { MockGitHubConnection, MockAuthorizedRepo } from "@/mock/github/data";
import { useCan } from "@/hooks/useCan";
import { useSessionReady } from "@/hooks/useSessionReady";
import { connectGitHubMutation, disconnectGitHubMutation, githubConnectionQuery, githubReposQuery } from "@/queries/gitops";

/** 连接状态取不到时按「未连接」显示（没有配置 GitHub App 的部署里，这些接口根本不存在） */
const NOT_CONNECTED: MockGitHubConnection = { connected: false, accountLogin: "", avatarUrl: "", installationId: 0 };
const NO_REPOS: MockAuthorizedRepo[] = [];

export function useGitHubPage() {
  // GitHub 连接是纯管理功能页：权限加载完才决定显示真实连接还是演示数据
  const sessionReady = useSessionReady();
  const canManageGitHub = useCan()("github.manage");
  const isDemo = sessionReady && !canManageGitHub;

  const connectionQuery = useQuery(githubConnectionQuery(canManageGitHub));
  const realConnection = connectionQuery.data ?? NOT_CONNECTED;
  const reposQuery = useQuery(githubReposQuery(canManageGitHub, realConnection.connected));

  const connect = useMutation(connectGitHubMutation());
  const disconnect = useMutation(disconnectGitHubMutation());

  const handleConnect = () => {
    connect.mutate(undefined, {
      // 已经装过 App 时后端直接返回 connected（工厂会让连接状态重新取数）；否则跳转到 GitHub 的授权页
      onSuccess: (connectResult) => {
        if (!connectResult?.connected && connectResult?.authUrl) window.location.href = connectResult.authUrl;
      },
      onError: (err) => console.error("Failed to initiate GitHub connection:", err),
    });
  };

  const handleDisconnect = () => {
    disconnect.mutate(undefined, {
      onError: (err) => console.error("Failed to disconnect:", err),
    });
  };

  return {
    isDemo,
    // 权限没加载完、或真实的连接状态还没回来
    loading: !sessionReady || (canManageGitHub && connectionQuery.isPending),
    connection: isDemo ? mock.mockGetGitHubConnection() : realConnection,
    repos: isDemo ? mock.mockGetAuthorizedRepos() : realConnection.connected ? (reposQuery.data ?? NO_REPOS) : NO_REPOS,
    handleConnect,
    handleDisconnect,
  };
}
