"use client";

import { useState } from "react";
import { X, LogIn, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { useClusterStore } from "@/store/clusterStore";
import { login } from "@/api/auth";

export function LoginDialog() {
  const { t } = useI18n();
  const { isLoginDialogOpen, closeLoginDialog, setLoginData, executePendingAction } =
    useAuthStore();
  const { setClusterIds } = useClusterStore();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

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
      setError(err instanceof Error ? err.message : "登录失败");
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
          </form>
        </div>
      </div>
    </>
  );
}
