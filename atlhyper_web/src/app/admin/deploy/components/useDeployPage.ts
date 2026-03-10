import { useEffect, useState, useCallback } from "react";
import {
  mockGetDeployConfig,
  mockGetPathStatus,
  mockGetDeployHistory,
  mockGetAuthorizedRepos,
  mockGetKustomizePaths,
} from "@/mock/deploy";
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
  const [githubConnected, setGithubConnected] = useState(true);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const configData = mockGetDeployConfig();
      const statusData = mockGetPathStatus();
      const historyData = mockGetDeployHistory();
      const reposData = mockGetAuthorizedRepos();

      setConfig(configData);
      setStatusList(statusData);
      setHistory(historyData);
      setRepos(reposData);
      setGithubConnected(true);

      if (configData.repoUrl) {
        setKustomizePaths(mockGetKustomizePaths(configData.repoUrl));
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
      setKustomizePaths(mockGetKustomizePaths(config.repoUrl));
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
      // Config 仓库变更时，重新加载 kustomize 路径
      if (prev && prev.repoUrl !== newConfig.repoUrl) {
        setKustomizePaths(mockGetKustomizePaths(newConfig.repoUrl));
      }
      return newConfig;
    });
  }, []);

  const handleSaveConfig = useCallback(async () => {
    if (!editConfig) return;
    setSaving(true);
    await new Promise((r) => setTimeout(r, 500));
    // 过滤掉空路径
    const cleaned = { ...editConfig, paths: editConfig.paths.filter((p) => p !== "") };
    setConfig(cleaned);
    setEditing(false);
    setEditConfig(null);
    setSaving(false);
  }, [editConfig]);

  const handleTestConnection = useCallback(async () => {
    await new Promise((r) => setTimeout(r, 1000));
    return true;
  }, []);

  const handleSyncNow = useCallback(async (path: string) => {
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
