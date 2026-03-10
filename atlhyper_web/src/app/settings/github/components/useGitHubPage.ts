import { useEffect, useState, useCallback, useRef } from "react";
import {
  mockGetGitHubConnection,
  mockGetAuthorizedRepos,
  mockGetRepoMappings,
  mockGetNamespaces,
  mockGetDeployments,
  mockGetRepoDirs,
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
  // 每个仓库已配置的 Namespace 列表（用于缩小映射范围）
  const [repoNamespaces, setRepoNamespaces] = useState<Record<string, string[]>>({});
  const nextIdRef = useRef(100);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const connData = mockGetGitHubConnection();
      const reposData = mockGetAuthorizedRepos();
      const mappingData = mockGetRepoMappings();

      const nsData = mockGetNamespaces();
      const deplData = mockGetDeployments();
      const dirs: Record<string, string[]> = {};
      const repoNs: Record<string, string[]> = {};
      for (const repo of reposData.filter((r) => r.mappingEnabled)) {
        dirs[repo.fullName] = mockGetRepoDirs(repo.fullName);
        repoNs[repo.fullName] = mockGetRepoNamespaces(repo.fullName);
      }

      setConnection(connData);
      setRepos(reposData);
      setMappings(mappingData);
      setNamespaces(nsData);
      setDeployments(deplData);
      setRepoDirs(dirs);
      setRepoNamespaces(repoNs);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleToggleMapping = useCallback((fullName: string) => {
    setRepos((prev) =>
      prev.map((r) =>
        r.fullName === fullName ? { ...r, mappingEnabled: !r.mappingEnabled } : r
      )
    );
    setRepoDirs((prev) => {
      if (prev[fullName]) {
        const next = { ...prev };
        delete next[fullName];
        return next;
      }
      return { ...prev, [fullName]: mockGetRepoDirs(fullName) };
    });
    setRepoNamespaces((prev) => {
      if (prev[fullName]) {
        const next = { ...prev };
        delete next[fullName];
        return next;
      }
      return { ...prev, [fullName]: mockGetRepoNamespaces(fullName) };
    });
  }, []);

  // 只显示启用了映射的仓库的映射数据
  const enabledRepos = repos.filter((r) => r.mappingEnabled).map((r) => r.fullName);
  const filteredMappings = mappings.filter((m) => enabledRepos.includes(m.repo));

  const handleConnect = useCallback(() => {
    console.log("Redirect to GitHub OAuth...");
  }, []);

  const handleDisconnect = useCallback(async () => {
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

  // 为仓库添加 Namespace
  const handleAddRepoNamespace = useCallback((repo: string, ns: string) => {
    setRepoNamespaces((prev) => {
      const current = prev[repo] || [];
      if (current.includes(ns)) return prev;
      return { ...prev, [repo]: [...current, ns] };
    });
  }, []);

  // 移除仓库的 Namespace
  const handleRemoveRepoNamespace = useCallback((repo: string, ns: string) => {
    setRepoNamespaces((prev) => {
      const current = prev[repo] || [];
      return { ...prev, [repo]: current.filter((n) => n !== ns) };
    });
    // 同时清除该仓库下使用了此 NS 的映射行的 NS 和 Deployment
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
