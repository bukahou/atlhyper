import { useEffect, useState, useCallback } from "react";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { useAuthStore } from "@/store/authStore";
import * as githubDS from "@/datasource/github";
import * as mock from "@/mock/github";
import type { MockGitHubConnection, MockAuthorizedRepo } from "@/mock/github/data";

export function useGitHubPage() {
  const { isAuthenticated } = useAuthStore();
  const [loading, setLoading] = useState(true);
  const [connection, setConnection] = useState<MockGitHubConnection | null>(null);
  const [repos, setRepos] = useState<MockAuthorizedRepo[]>([]);

  const fetchData = useCallback(async () => {
    if (!isAuthenticated) {
      setConnection(mock.mockGetGitHubConnection());
      setRepos(mock.mockGetAuthorizedRepos());
      setLoading(false);
      return;
    }
    try {
      const connData = await githubDS.getGitHubConnection();
      setConnection(connData);

      if (!connData?.connected) {
        setRepos([]);
        return;
      }

      const reposData = await githubDS.getAuthorizedRepos();
      setRepos(reposData);
    } catch (err) {
      console.error("Failed to load GitHub data:", err);
      setConnection({ connected: false, accountLogin: "", avatarUrl: "", installationId: 0 });
      setRepos([]);
    } finally {
      setLoading(false);
    }
  }, [isAuthenticated]);

  useAutoRefresh(fetchData);

  const handleConnect = useCallback(async () => {
    try {
      const data = await githubDS.connectGitHub();
      if (data?.connected) {
        await fetchData();
      } else if (data?.authUrl) {
        window.location.href = data.authUrl;
      }
    } catch (err) {
      console.error("Failed to initiate GitHub connection:", err);
    }
  }, [fetchData]);

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
  }, []);

  return {
    loading,
    connection,
    repos,
    handleConnect,
    handleDisconnect,
  };
}
