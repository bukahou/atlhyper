"use client";

import { useI18n } from "@/i18n/context";
import { Github, CheckCircle, XCircle, ExternalLink } from "lucide-react";
import type { MockGitHubConnection } from "@/mock/github/data";

interface ConnectionCardProps {
  connection: MockGitHubConnection | null;
  onConnect: () => void;
  onDisconnect: () => void;
}

export function ConnectionCard({
  connection,
  onConnect,
  onDisconnect,
}: ConnectionCardProps) {
  const { t } = useI18n();
  const gt = t.githubPage;

  const connected = connection?.connected ?? false;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="flex items-center gap-2 px-6 py-4 border-b border-[var(--border-color)]">
        <Github className="w-5 h-5 text-muted" />
        <h3 className="text-lg font-medium text-default">GitHub App</h3>
      </div>

      <div className="p-6">
        {connected ? (
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-emerald-500" />
                <span className="text-sm font-medium text-emerald-600 dark:text-emerald-400">
                  {gt.statusConnected}
                </span>
              </div>
              <div className="flex items-center gap-2">
                {connection?.avatarUrl && (
                  <img
                    src={connection.avatarUrl}
                    alt={connection.accountLogin}
                    className="w-6 h-6 rounded-full"
                  />
                )}
                <span className="text-sm text-default">
                  {gt.connectedAs}: <strong>{connection?.accountLogin}</strong>
                </span>
              </div>
            </div>
            <button
              onClick={() => {
                if (confirm(gt.disconnectConfirm)) {
                  onDisconnect();
                }
              }}
              className="px-3 py-1.5 text-sm rounded-lg border border-red-200 text-red-600 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-900/20 transition-colors"
            >
              {gt.disconnectButton}
            </button>
          </div>
        ) : (
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <XCircle className="w-5 h-5 text-gray-400" />
              <span className="text-sm text-muted">{gt.statusNotConnected}</span>
            </div>
            <button
              onClick={onConnect}
              className="flex items-center gap-2 px-4 py-2 text-sm rounded-lg bg-gray-900 text-white hover:bg-gray-800 dark:bg-gray-100 dark:text-gray-900 dark:hover:bg-gray-200 transition-colors"
            >
              <Github className="w-4 h-4" />
              {gt.connectButton}
              <ExternalLink className="w-3 h-3" />
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
