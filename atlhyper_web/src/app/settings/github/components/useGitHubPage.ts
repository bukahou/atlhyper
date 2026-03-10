import { useEffect, useState, useCallback, useRef } from "react";
import * as githubDS from "@/datasource/github";
import {
  mockGetRepoMappings,
  mockGetNamespaces,
  mockGetDeployments,
  mockGetRepoNamespaces,
} from "@/mock/github";
import type { MockGitHubConnection, MockRepoMapping, MockAuthorizedRepo } from "@/mock/github/data";

export function useGitHubPage() {
  const [loading, setLoading] = useState(true);
  const [connection, setConnection] = useState<MockGitHubConnection | null>(null);
  const [repos, setRepos] = useState<MockAuthorizedRepo[]>([]);
  const [mappings, setMappings] = useState<MockRepoMapping[]>([]);
  const [namespaces, setNamespaces] = useState<string[]>([]);
  const [deployments, setDeployments] = useState<{ name: string; namespace: string; image: string }[]>([]);
  const [repoDirs, setRepoDirs] = useState<Record<string, string[]>>({});
  const [repoNamespaces, setRepoNamespaces] = useState<Record<string, string[]>>({});
  const nextIdRef = useRef(100);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const connData = await githubDS.getGitHubConnection();
      setConnection(connData);

      // 未连接时不加载仓库数据
      if (!connData?.connected) {
        setRepos([]);
        setMappings([]);
        setNamespaces([]);
        setDeployments([]);
        setRepoDirs({});
        setRepoNamespaces({});
        return;
      }

      const reposData = await githubDS.getAuthorizedRepos();
      // Phase 2 功能：映射/Namespace/Deployment 暂用 mock
      const mappingData = mockGetRepoMappings();
      const nsData = mockGetNamespaces();
      const deplData = mockGetDeployments();

      const dirs: Record<string, string[]> = {};
      const repoNs: Record<string, string[]> = {};
      for (const repo of reposData.filter((r: MockAuthorizedRepo) => r.mappingEnabled)) {
        dirs[repo.fullName] = await githubDS.getRepoDirs(repo.fullName);
        repoNs[repo.fullName] = mockGetRepoNamespaces(repo.fullName);
      }

      setRepos(reposData);
      setMappings(mappingData);
      setNamespaces(nsData);
      setDeployments(deplData);
      setRepoDirs(dirs);
      setRepoNamespaces(repoNs);
    } catch (err) {
      console.error("Failed to load GitHub data:", err);
      // API 不可用时显示未连接状态
      setConnection({ connected: false, accountLogin: "", avatarUrl: "", installationId: 0 });
      setRepos([]);
      setMappings([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleToggleMapping = useCallback(async (fullName: string) => {
    const repo = repos.find((r) => r.fullName === fullName);
    if (!repo) return;
    const newEnabled = !repo.mappingEnabled;

    try {
      await githubDS.toggleRepoMapping(fullName, newEnabled);
    } catch (err) {
      console.error("Failed to toggle mapping:", err);
    }

    setRepos((prev) =>
      prev.map((r) =>
        r.fullName === fullName ? { ...r, mappingEnabled: newEnabled } : r
      )
    );
    if (newEnabled) {
      githubDS.getRepoDirs(fullName).then((dirs) => {
        setRepoDirs((prev) => ({ ...prev, [fullName]: dirs }));
      });
      setRepoNamespaces((prev) => ({ ...prev, [fullName]: mockGetRepoNamespaces(fullName) }));
    } else {
      setRepoDirs((prev) => {
        const next = { ...prev };
        delete next[fullName];
        return next;
      });
      setRepoNamespaces((prev) => {
        const next = { ...prev };
        delete next[fullName];
        return next;
      });
    }
  }, [repos]);

  // 只显示启用了映射的仓库的映射数据
  const enabledRepos = repos.filter((r) => r.mappingEnabled).map((r) => r.fullName);
  const filteredMappings = mappings.filter((m) => enabledRepos.includes(m.repo));

  const handleConnect = useCallback(async () => {
    try {
      const data = await githubDS.connectGitHub();
      if (data?.authUrl) {
        window.location.href = data.authUrl;
      }
    } catch (err) {
      console.error("Failed to initiate GitHub connection:", err);
    }
  }, []);

  const handleDisconnect = useCallback(async () => {
    try {
      await githubDS.disconnectGitHub();
    } catch (err) {
      console.error("Failed to disconnect:", err);
    }
    setConnection({
      connected: false,
      accountLogin: "",
      avatarUrl: "",
      installationId: 0,
    });
    setRepos([]);
    setMappings([]);
  }, []);

  const handleConfirmMapping = useCallback(async (id: number) => {
    setMappings((prev) =>
      prev.map((m) => (m.id === id ? { ...m, confirmed: true } : m))
    );
  }, []);

  const handleUpdateMapping = useCallback((id: number, field: string, value: string) => {
    setMappings((prev) =>
      prev.map((m) => {
        if (m.id !== id) return m;
        const updated = { ...m, [field]: value, confirmed: false };
        if (field === "namespace") {
          updated.deployment = "";
        }
        return updated;
      })
    );
  }, []);

  const handleConfirmAll = useCallback(async () => {
    setMappings((prev) =>
      prev.map((m) => {
        if (m.namespace && m.deployment && m.sourcePath) {
          return { ...m, confirmed: true };
        }
        return m;
      })
    );
  }, []);

  const handleAddMapping = useCallback((repo: string) => {
    const newId = nextIdRef.current++;
    const newMapping: MockRepoMapping = {
      id: newId,
      clusterId: "zgmf-x10a",
      repo,
      namespace: "",
      deployment: "",
      container: "",
      imagePrefix: "",
      sourcePath: "",
      confirmed: false,
    };
    setMappings((prev) => [...prev, newMapping]);
  }, []);

  const handleDeleteMapping = useCallback((id: number) => {
    setMappings((prev) => prev.filter((m) => m.id !== id));
  }, []);

  const handleAddRepoNamespace = useCallback((repo: string, ns: string) => {
    setRepoNamespaces((prev) => {
      const current = prev[repo] || [];
      if (current.includes(ns)) return prev;
      return { ...prev, [repo]: [...current, ns] };
    });
  }, []);

  const handleRemoveRepoNamespace = useCallback((repo: string, ns: string) => {
    setRepoNamespaces((prev) => {
      const current = prev[repo] || [];
      return { ...prev, [repo]: current.filter((n) => n !== ns) };
    });
    setMappings((prev) =>
      prev.map((m) => {
        if (m.repo === repo && m.namespace === ns) {
          return { ...m, namespace: "", deployment: "", confirmed: false };
        }
        return m;
      })
    );
  }, []);

  return {
    loading,
    connection,
    repos,
    mappings: filteredMappings,
    namespaces,
    deployments,
    repoDirs,
    repoNamespaces,
    handleToggleMapping,
    handleConnect,
    handleDisconnect,
    handleUpdateMapping,
    handleConfirmMapping,
    handleConfirmAll,
    handleAddMapping,
    handleDeleteMapping,
    handleAddRepoNamespace,
    handleRemoveRepoNamespace,
    loadData,
  };
}
