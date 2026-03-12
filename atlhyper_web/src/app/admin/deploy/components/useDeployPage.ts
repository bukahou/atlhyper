import { useEffect, useState, useCallback } from "react";
import * as deployDS from "@/datasource/deploy";
import * as githubDS from "@/datasource/github";
import * as deployMock from "@/mock/deploy";
import * as githubMock from "@/mock/github";
import { useClusterStore } from "@/store/clusterStore";
import { useAuthStore } from "@/store/authStore";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import type {
  MockDeployConfig,
  MockPathStatus,
  MockDeployRecord,
} from "@/mock/deploy/data";

export function useDeployPage() {
  const clusterId = useClusterStore((s) => s.currentClusterId);
  const { isAuthenticated } = useAuthStore();
  const [loading, setLoading] = useState(true);
  const [config, setConfig] = useState<MockDeployConfig | null>(null);
  const [statusList, setStatusList] = useState<MockPathStatus[]>([]);
  const [history, setHistory] = useState<MockDeployRecord[]>([]);
  const [repos, setRepos] = useState<{ fullName: string; defaultBranch: string; private: boolean }[]>([]);
  const [kustomizePaths, setKustomizePaths] = useState<string[]>([]);

  const [editing, setEditing] = useState(false);
  const [editConfig, setEditConfig] = useState<MockDeployConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [githubConnected, setGithubConnected] = useState(false);
  const [syncingPaths, setSyncingPaths] = useState<Set<string>>(new Set());

  const loadData = useCallback(async () => {
    if (!isAuthenticated) {
      const connData = githubMock.mockGetGitHubConnection();
      setGithubConnected(connData?.connected ?? false);
      const configData = deployMock.mockGetDeployConfig();
      setConfig(configData);
      setStatusList(deployMock.mockGetPathStatus() as MockPathStatus[]);
      setHistory(deployMock.mockGetDeployHistory() as MockDeployRecord[]);
      setRepos(deployMock.mockGetAuthorizedRepos().map((r) => ({
        fullName: r.fullName,
        defaultBranch: r.defaultBranch,
        private: r.private,
      })));
      if (configData?.repoUrl) {
        setKustomizePaths(deployMock.mockGetKustomizePaths(configData.repoUrl));
      }
      setLoading(false);
      return;
    }

    try {
      // 检查 GitHub 连接状态
      let connected = false;
      try {
        const connData = await githubDS.getGitHubConnection();
        connected = connData?.connected ?? false;
      } catch {
        connected = false;
      }
      setGithubConnected(connected);

      // 加载部署配置

      let configData: MockDeployConfig | null = null;
      try {
        configData = await deployDS.getDeployConfig(clusterId);
      } catch {
        configData = null;
      }
      setConfig(configData);

      // 同步状态通过 datasource（支持 mock/api 切换）
      if (configData) {
        try {
          const statusData = await deployDS.getDeployStatus();
          setStatusList((statusData ?? []) as MockPathStatus[]);
        } catch {
          setStatusList([]);
        }
      } else {
        setStatusList([]);
      }

      // 加载部署历史
      try {
        const historyData = await deployDS.getDeployHistory({ clusterId });
        setHistory((historyData ?? []) as MockDeployRecord[]);
      } catch {
        setHistory([]);
      }

      // 加载可选仓库列表（从 GitHub datasource 获取）
      if (connected) {
        try {
          const reposData = await githubDS.getAuthorizedRepos();
          setRepos((reposData ?? []).map((r: { fullName: string; defaultBranch: string; private: boolean }) => ({
            fullName: r.fullName,
            defaultBranch: r.defaultBranch,
            private: r.private,
          })));
        } catch {
          setRepos([]);
        }
      } else {
        setRepos([]);
      }

      // 加载 kustomize 路径
      if (configData?.repoUrl) {
        try {
          const paths = await deployDS.getKustomizePaths(configData.repoUrl);
          setKustomizePaths(paths ?? []);
        } catch {
          setKustomizePaths([]);
        }
      }
    } finally {
      setLoading(false);
    }
  }, [clusterId, isAuthenticated]);

  const { refresh, intervalSeconds } = useAutoRefresh(loadData, { interval: 30000 });

  const handleStartEdit = useCallback(() => {
    if (config) {
      setEditConfig({ ...config, paths: [...config.paths] });
      deployDS.getKustomizePaths(config.repoUrl).then((paths) => {
        setKustomizePaths(paths ?? []);
      }).catch(() => setKustomizePaths([]));
    } else {
      setEditConfig({
        repoUrl: "",
        paths: [],
        intervalSec: 60,
        autoDeploy: true,
        clusterId,
      });
    }
    setEditing(true);
  }, [config, clusterId]);

  const handleCancelEdit = useCallback(() => {
    setEditing(false);
    setEditConfig(null);
  }, []);

  const handleUpdateConfig = useCallback((newConfig: MockDeployConfig) => {
    // 确保进入正式编辑模式（解决 isFirstSetup 下 editConfig 被硬编码覆盖的问题）
    setEditing(true);
    setEditConfig((prev) => {
      const prevRepoUrl = prev?.repoUrl ?? "";
      if (prevRepoUrl !== newConfig.repoUrl && newConfig.repoUrl) {
        deployDS.getKustomizePaths(newConfig.repoUrl).then((paths) => {
          setKustomizePaths(paths ?? []);
        }).catch(() => setKustomizePaths([]));
      }
      return newConfig;
    });
  }, []);

  const handleSaveConfig = useCallback(async () => {
    if (!editConfig) return;
    setSaving(true);
    try {
      const cleaned = { ...editConfig, paths: editConfig.paths.filter((p) => p !== "") };
      await deployDS.saveDeployConfig(cleaned);
      setConfig(cleaned);
      setEditing(false);
      setEditConfig(null);
    } catch (err) {
      console.error("Failed to save deploy config:", err);
    } finally {
      setSaving(false);
    }
  }, [editConfig]);

  const handleTestConnection = useCallback(async () => {
    try {
      return await deployDS.testDeployConnection();
    } catch {
      return false;
    }
  }, []);

  const handleSyncNow = useCallback(async (path: string) => {
    // 防重复点击
    if (syncingPaths.has(path)) return;
    setSyncingPaths((prev) => new Set(prev).add(path));

    try {
      await deployDS.syncDeployNow(path);
    } catch (err) {
      console.error("Failed to sync:", err);
      setSyncingPaths((prev) => {
        const next = new Set(prev);
        next.delete(path);
        return next;
      });
      return;
    }
    // 同步请求只是提交了任务，延迟后刷新真实状态
    setTimeout(async () => {
      try {
        const statusData = await deployDS.getDeployStatus();
        setStatusList((statusData ?? []) as MockPathStatus[]);
        const historyData = await deployDS.getDeployHistory({ clusterId });
        setHistory((historyData ?? []) as MockDeployRecord[]);
      } catch {
        // ignore
      } finally {
        setSyncingPaths((prev) => {
          const next = new Set(prev);
          next.delete(path);
          return next;
        });
      }
    }, 5000);
  }, [clusterId, syncingPaths]);

  // 首次设置时自动进入编辑模式
  const isFirstSetup = !config && !editing && !editConfig;
  useEffect(() => {
    if (isFirstSetup && !loading) {
      setEditing(true);
      setEditConfig({
        repoUrl: "",
        paths: [],
        intervalSec: 60,
        autoDeploy: true,
        clusterId,
      });
    }
  }, [isFirstSetup, loading, clusterId]);

  return {
    loading,
    githubConnected,
    config,
    editing,
    editConfig,
    kustomizePaths,
    statusList,
    history,
    repos,
    saving,
    handleStartEdit,
    handleCancelEdit,
    handleUpdateConfig,
    handleSaveConfig,
    handleTestConnection,
    handleSyncNow,
    syncingPaths,
    loadData,
    refresh,
    intervalSeconds,
  };
}
