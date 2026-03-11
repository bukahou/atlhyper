import { useEffect, useState, useCallback, useRef } from "react";
import * as githubDS from "@/datasource/github";
import * as clusterDS from "@/datasource/cluster";
import { useClusterStore } from "@/store/clusterStore";
import type { MockGitHubConnection, MockRepoMapping, MockAuthorizedRepo } from "@/mock/github/data";

export function useGitHubPage() {
  const clusterId = useClusterStore((s) => s.currentClusterId);
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
      const mappingData = await githubDS.getMappings();

      const dirs: Record<string, string[]> = {};
      const repoNs: Record<string, string[]> = {};
      for (const repo of reposData.filter((r: MockAuthorizedRepo) => r.mappingEnabled)) {
        dirs[repo.fullName] = await githubDS.getRepoDirs(repo.fullName);
        repoNs[repo.fullName] = await githubDS.getRepoNamespaces(repo.fullName);
      }

      // 从集群加载 namespace 和 deployment 列表（用于映射下拉菜单）
      if (clusterId) {
        try {
          const nsResp = await clusterDS.getNamespaceList({ cluster_id: clusterId });
          const nsList = (nsResp.data?.data || []).map((ns: { name: string }) => ns.name);
          setNamespaces(nsList);

          const deployResp = await clusterDS.getDeploymentList({ cluster_id: clusterId });
          const deployList = (deployResp.data?.data || []).map((d: { name: string; namespace: string; image: string }) => ({
            name: d.name,
            namespace: d.namespace,
            image: d.image,
          }));
          setDeployments(deployList);
        } catch (clusterErr) {
          console.error("Failed to load cluster data:", clusterErr);
          setNamespaces([]);
          setDeployments([]);
        }
      }

      setRepos(reposData);
      setMappings(mappingData);
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
  }, [clusterId]);

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
      githubDS.getRepoNamespaces(fullName).then((ns) => {
        setRepoNamespaces((prev) => ({ ...prev, [fullName]: ns }));
      });
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
      if (data?.connected) {
        // 自动检测到已有安装，直接刷新数据
        await loadData();
      } else if (data?.authUrl) {
        window.location.href = data.authUrl;
      }
    } catch (err) {
      console.error("Failed to initiate GitHub connection:", err);
    }
  }, [loadData]);

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
    try {
      await githubDS.confirmMappingAPI(id);
    } catch (err) {
      console.error("Failed to confirm mapping:", err);
    }
    setMappings((prev) =>
      prev.map((m) => (m.id === id ? { ...m, confirmed: true } : m))
    );
  }, []);

  const handleUpdateMapping = useCallback(async (id: number, field: string, value: string) => {
    try {
      await githubDS.updateMapping(id, { [field]: value });
    } catch (err) {
      console.error("Failed to update mapping:", err);
    }
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
    const toConfirm = mappings.filter(
      (m) => m.namespace && m.deployment && m.sourcePath && !m.confirmed
    );
    try {
      await Promise.all(toConfirm.map((m) => githubDS.confirmMappingAPI(m.id)));
    } catch (err) {
      console.error("Failed to confirm all mappings:", err);
    }
    setMappings((prev) =>
      prev.map((m) => {
        if (m.namespace && m.deployment && m.sourcePath) {
          return { ...m, confirmed: true };
        }
        return m;
      })
    );
  }, [mappings]);

  const handleAddMapping = useCallback(async (repo: string) => {
    try {
      const created = await githubDS.createMapping({
        clusterId,
        repo,
        namespace: "",
        deployment: "",
      });
      setMappings((prev) => [...prev, created as MockRepoMapping]);
    } catch (err) {
      console.error("Failed to create mapping:", err);
      // Fallback: 本地创建
      const newId = nextIdRef.current++;
      const newMapping: MockRepoMapping = {
        id: newId,
        clusterId,
        repo,
        namespace: "",
        deployment: "",
        container: "",
        imagePrefix: "",
        sourcePath: "",
        confirmed: false,
      };
      setMappings((prev) => [...prev, newMapping]);
    }
  }, [clusterId]);

  const handleDeleteMapping = useCallback(async (id: number) => {
    try {
      await githubDS.deleteMappingAPI(id);
    } catch (err) {
      console.error("Failed to delete mapping:", err);
    }
    setMappings((prev) => prev.filter((m) => m.id !== id));
  }, []);

  const handleAddRepoNamespace = useCallback(async (repo: string, ns: string) => {
    try {
      await githubDS.addRepoNamespace(repo, ns);
    } catch (err) {
      console.error("Failed to add namespace:", err);
    }
    setRepoNamespaces((prev) => {
      const current = prev[repo] || [];
      if (current.includes(ns)) return prev;
      return { ...prev, [repo]: [...current, ns] };
    });

    // 自动填充：如果该 repo+namespace 没有映射记录，为该 NS 下所有 Deployment 创建预填行
    const hasExisting = mappings.some((m) => m.repo === repo && m.namespace === ns);
    if (!hasExisting) {
      const nsDeployments = deployments.filter((d) => d.namespace === ns);
      if (nsDeployments.length > 0) {
        const created: MockRepoMapping[] = [];
        for (const dep of nsDeployments) {
          try {
            const result = await githubDS.createMapping({
              clusterId,
              repo,
              namespace: ns,
              deployment: dep.name,
            });
            created.push(result as MockRepoMapping);
          } catch {
            // Fallback: 本地创建
            const newId = nextIdRef.current++;
            created.push({
              id: newId,
              clusterId,
              repo,
              namespace: ns,
              deployment: dep.name,
              container: "",
              imagePrefix: "",
              sourcePath: "",
              confirmed: false,
            });
          }
        }
        if (created.length > 0) {
          setMappings((prev) => [...prev, ...created]);
        }
      }
    }
  }, [mappings, deployments, clusterId]);

  const handleRemoveRepoNamespace = useCallback(async (repo: string, ns: string) => {
    try {
      await githubDS.removeRepoNamespace(repo, ns);
    } catch (err) {
      console.error("Failed to remove namespace:", err);
    }
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
