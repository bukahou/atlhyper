import { useEffect, useState, useCallback } from "react";
import * as deployDS from "@/datasource/deploy";
import * as githubDS from "@/datasource/github";
import type {
  MockDeployConfig,
  MockPathStatus,
  MockDeployRecord,
} from "@/mock/deploy/data";

export function useDeployPage() {
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

  const loadData = useCallback(async () => {
    setLoading(true);
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
      const clusterId = "zgmf-x10a";
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
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

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
        clusterId: "zgmf-x10a",
      });
    }
    setEditing(true);
  }, [config]);

  const handleCancelEdit = useCallback(() => {
    setEditing(false);
    setEditConfig(null);
  }, []);

  const handleUpdateConfig = useCallback((newConfig: MockDeployConfig) => {
    setEditConfig((prev) => {
      if (prev && prev.repoUrl !== newConfig.repoUrl && newConfig.repoUrl) {
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
    try {
      await deployDS.syncDeployNow(path);
    } catch (err) {
      console.error("Failed to sync:", err);
    }
    setStatusList((prev) =>
      prev.map((s) => (s.path === path ? { ...s, inSync: true } : s))
    );
  }, []);

  const isFirstSetup = !config && !editing;

  return {
    loading,
    githubConnected,
    config,
    editing: editing || isFirstSetup,
    editConfig: editing ? editConfig : isFirstSetup ? {
      repoUrl: "",
      paths: [],
      intervalSec: 60,
      autoDeploy: true,
      clusterId: "zgmf-x10a",
    } : null,
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
    loadData,
  };
}
