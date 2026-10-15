"use client";

import { useEffect, useState } from "react";
import { X, LogIn, Loader2, KeyRound } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { useClusterStore } from "@/store/clusterStore";
import { login, getAuthProviders, oidcStartURL } from "@/api/auth";
import type { AuthProvider } from "@/types/auth";

export function LoginDialog() {
  const { t } = useI18n();
  const { isLoginDialogOpen, closeLoginDialog, setLoginData, executePendingAction } =
    useAuthStore();
  const { setClusterIds } = useClusterStore();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [providers, setProviders] = useState<AuthProvider[]>([]);

  // 外部登录方式：未启用时为空列表，按钮不显示
  useEffect(() => {
    if (!isLoginDialogOpen) return;
    getAuthProviders()
      .then((res) => setProviders(res.data.data ?? []))
      .catch(() => setProviders([]));
  }, [isLoginDialogOpen]);

  // 整页跳转发起外部登录；登录成功后回到当前页
  const startExternalLogin = () => {
    const next = window.location.pathname + window.location.search;
    window.location.assign(oidcStartURL(next));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      // Login - 响应包含 token, user, cluster_ids
      const loginRes = await login({ username, password });
      setLoginData(loginRes.data.data);

      // Sync cluster_ids to clusterStore
      if (loginRes.data.data.cluster_ids?.length > 0) {
        setClusterIds(loginRes.data.data.cluster_ids);
      }

      // Close dialog and execute pending action
      closeLoginDialog();
      executePendingAction();

      // Reset form
      setUsername("");
      setPassword("");
    } catch (err) {
      setError(err instanceof Error ? err.message : t.login.loginFailed);
    } finally {
      setLoading(false);
    }
  };

  if (!isLoginDialogOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 z-50"
        onClick={closeLoginDialog}
      />

      {/* Dialog */}
      <div className="fixed inset-0 flex items-center justify-center z-50 p-4">
        <div
          className="bg-card rounded-xl shadow-2xl w-full max-w-md"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b border-[var(--border-color)]">
            <h2 className="text-xl font-semibold text-default">
              {t.common.login}
            </h2>
            <button
              onClick={closeLoginDialog}
              className="p-2 rounded-lg hover-bg"
            >
              <X className="w-5 h-5 text-muted" />
            </button>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="p-6 space-y-4">
            {error && (
              <div className="p-3 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 rounded-lg text-sm">
                {error}
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-secondary mb-1">
                {t.common.username}
              </label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full px-4 py-2 rounded-lg border border-[var(--border-color)] bg-card text-default focus:ring-2 focus:ring-primary focus:border-transparent outline-none transition-all"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-secondary mb-1">
                {t.common.password}
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-4 py-2 rounded-lg border border-[var(--border-color)] bg-card text-default focus:ring-2 focus:ring-primary focus:border-transparent outline-none transition-all"
                required
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-primary hover:bg-primary-hover text-white font-medium rounded-lg transition-colors flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? (
                <Loader2 className="w-5 h-5 animate-spin" />
              ) : (
                <LogIn className="w-5 h-5" />
              )}
              {t.common.login}
            </button>

            {providers.length > 0 && (
              <>
                <div className="flex items-center gap-3 text-xs text-muted">
                  <div className="flex-1 h-px bg-[var(--border-color)]" />
                  {t.login.orDivider}
                  <div className="flex-1 h-px bg-[var(--border-color)]" />
                </div>
                {providers.map((p) => (
                  <button
                    key={p.id}
                    type="button"
                    onClick={startExternalLogin}
                    className="w-full py-3 border border-[var(--border-color)] hover-bg text-default font-medium rounded-lg transition-colors flex items-center justify-center gap-2 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
                  >
                    <KeyRound className="w-5 h-5" />
                    {t.login.continueWith.replace("{name}", p.displayName)}
                  </button>
                ))}
              </>
            )}
          </form>
        </div>
      </div>
    </>
  );
}
