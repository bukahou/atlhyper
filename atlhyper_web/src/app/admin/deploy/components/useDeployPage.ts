import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as deployMock from "@/mock/deploy";
import * as githubMock from "@/mock/github";
import type { MockDeployConfig, MockPathStatus, MockDeployRecord } from "@/mock/deploy/data";
import { useCan } from "@/hooks/useCan";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { createDelayedTaskScheduler } from "@/hooks/useDelayedInvalidate";
import { useManualRefetch } from "@/hooks/useManualRefetch";
import { useSessionReady } from "@/hooks/useSessionReady";
import { deployKeys } from "@/queries/keys";
import {
  deployConfigQuery,
  deployHistoryQuery,
  deployStatusQuery,
  githubConnectionQuery,
  githubReposQuery,
  kustomizePathsQuery,
  saveDeployConfigMutation,
  syncDeployNowMutation,
  testDeployConnectionMutation,
} from "@/queries/gitops";
import { REFRESH } from "@/queries/refresh";

/** 同步任务从提交到在状态与历史里看得到结果，大约需要的时间 */
const SYNC_SETTLE_DELAY_MS = 5000;

const NO_STATUS: MockPathStatus[] = [];
const NO_HISTORY: MockDeployRecord[] = [];
const NO_PATHS: string[] = [];

type RepoOption = { fullName: string; defaultBranch: string; private: boolean };
const NO_REPOS: RepoOption[] = [];

/** 还没有部署配置时表单里的空配置 */
function emptyDeployConfig(clusterId: string): MockDeployConfig {
  return { repoUrl: "", paths: [], intervalSec: 60, autoDeploy: true, clusterId };
}

export function useDeployPage() {
  const { clusterId } = useCurrentCluster();
  const queryClient = useQueryClient();
  // 部署是纯管理功能页：权限加载完才决定显示真实数据还是演示数据
  const sessionReady = useSessionReady();
  const can = useCan();
  const canManageDeploy = can("deploy.manage");
  const canManageGitHub = can("github.manage");
  const isDemo = sessionReady && !canManageDeploy;

  // GitHub 连接与仓库：与连接页共用同一份查询
  const connectionQuery = useQuery(githubConnectionQuery(canManageDeploy && canManageGitHub));
  const realConnected = connectionQuery.data?.connected ?? false;
  const reposQuery = useQuery(githubReposQuery(canManageDeploy && canManageGitHub, realConnected));

  // 接口取不到（如没有配置 GitHub App 的部署里这些路由不存在）按「还没有配置」处理，页面照常可用
  const configQuery = useQuery(deployConfigQuery(clusterId, canManageDeploy));
  const realConfig = configQuery.data ?? null;
  const statusQuery = useQuery(deployStatusQuery(canManageDeploy, realConfig !== null));
  const historyQuery = useQuery(deployHistoryQuery(clusterId, canManageDeploy));

  const config = isDemo ? deployMock.mockGetDeployConfig() : realConfig;
  // 权限没加载完、或真实的配置还在取（没有选定集群时配置查询不启用，不算在取）
  const loading = !sessionReady || (canManageDeploy && configQuery.isLoading);

  // 编辑中的草稿。还没有配置时自动进入编辑（首次设置），用一张空配置起步
  const [draft, setDraft] = useState<MockDeployConfig | null>(null);
  const isFirstSetup = !loading && config === null;
  const editing = draft !== null || isFirstSetup;
  const editConfig = draft ?? (isFirstSetup ? emptyDeployConfig(clusterId) : null);

  // 可选的 kustomize 路径跟着表单里选的仓库走；不在编辑时跟着已保存的配置走
  const selectedRepo = (editConfig ?? config)?.repoUrl ?? "";
  const pathsQuery = useQuery(kustomizePathsQuery(selectedRepo, canManageDeploy));

  const saveConfig = useMutation(saveDeployConfigMutation());
  const testConnection = useMutation(testDeployConnectionMutation());
  const syncNow = useMutation(syncDeployNowMutation());
  const { refetch: refresh } = useManualRefetch(deployKeys.all);

  const handleStartEdit = () => {
    setDraft(config ? { ...config, paths: [...config.paths] } : emptyDeployConfig(clusterId));
  };

  const handleCancelEdit = () => setDraft(null);

  const handleSaveConfig = () => {
    if (!editConfig) return;
    const cleaned = { ...editConfig, paths: editConfig.paths.filter((p) => p !== "") };
    // 工厂在保存成功后重取配置，取完才算结束：退出编辑时界面上已经是新配置
    saveConfig.mutate(cleaned, {
      onSuccess: () => setDraft(null),
      onError: (err) => console.error("Failed to save deploy config:", err),
    });
  };

  const handleTestConnection = async () => {
    try {
      return await testConnection.mutateAsync();
    } catch {
      return false;
    }
  };

  // 正在同步的路径：从点下去到状态刷新完为止，防重复点击
  const [syncingPaths, setSyncingPaths] = useState<Set<string>>(new Set());
  const [settleScheduler] = useState(createDelayedTaskScheduler);
  // 离开页面时不再需要那次延时刷新；再回来时查询会自己重新取数
  useEffect(() => () => settleScheduler.cancelAll(), [settleScheduler]);

  const handleSyncNow = (path: string) => {
    if (syncingPaths.has(path)) return;
    const markSyncing = (syncing: boolean) =>
      setSyncingPaths((current) => {
        const next = new Set(current);
        if (syncing) next.add(path);
        else next.delete(path);
        return next;
      });

    markSyncing(true);
    syncNow.mutate(path, {
      // 同步请求只是提交了任务，过几秒再刷新真实的状态与历史
      onSuccess: () => {
        settleScheduler.schedule(() => {
          void Promise.all([
            queryClient.invalidateQueries({ queryKey: deployKeys.statuses }),
            queryClient.invalidateQueries({ queryKey: deployKeys.histories }),
          ]).finally(() => markSyncing(false));
        }, SYNC_SETTLE_DELAY_MS);
      },
      onError: (err) => {
        console.error("Failed to sync:", err);
        markSyncing(false);
      },
    });
  };

  return {
    isDemo,
    loading,
    githubConnected: isDemo ? (githubMock.mockGetGitHubConnection()?.connected ?? false) : realConnected,
    config,
    editing,
    editConfig,
    kustomizePaths: isDemo ? deployMock.mockGetKustomizePaths(selectedRepo) : (pathsQuery.data ?? NO_PATHS),
    statusList: isDemo ? deployMock.mockGetPathStatus() : realConfig ? (statusQuery.data ?? NO_STATUS) : NO_STATUS,
    history: isDemo ? deployMock.mockGetDeployHistory() : (historyQuery.data ?? NO_HISTORY),
    repos: isDemo ? deployMock.mockGetAuthorizedRepos() : realConnected ? (reposQuery.data ?? NO_REPOS) : NO_REPOS,
    saving: saveConfig.isPending,
    handleStartEdit,
    handleCancelEdit,
    // 表单里每改一处都交回完整的配置
    handleUpdateConfig: setDraft,
    handleSaveConfig,
    handleTestConnection,
    handleSyncNow,
    syncingPaths,
    refresh,
    intervalSeconds: REFRESH.slow / 1000,
  };
}
