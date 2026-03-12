"use client";

import { useState } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { Eye, Github, Rocket, Clock, AlertTriangle, ExternalLink } from "lucide-react";
import Link from "next/link";

import { ConnectionCard, ReposCard, useGitHubPage } from "./components";
import { ConfigCard, StatusCard, HistoryCard, useDeployPage } from "@/app/admin/deploy/components";

type Tab = "connection" | "deploy" | "history";

export default function GitHubPage() {
  const { t } = useI18n();
  const gt = t.githubPage;
  const dt = t.deployPage;
  const { isAuthenticated } = useAuthStore();
  const [activeTab, setActiveTab] = useState<Tab>("connection");

  const {
    loading: githubLoading,
    connection,
    repos,
    handleConnect,
    handleDisconnect,
  } = useGitHubPage();

  const {
    loading: deployLoading,
    githubConnected,
    config,
    editing,
    editConfig,
    kustomizePaths,
    statusList,
    history,
    repos: deployRepos,
    saving,
    handleStartEdit,
    handleCancelEdit,
    handleUpdateConfig,
    handleSaveConfig,
    handleTestConnection,
    handleSyncNow,
    syncingPaths,
    refresh,
    intervalSeconds,
  } = useDeployPage();

  const loading = githubLoading || deployLoading;

  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center py-12">
          <LoadingSpinner />
        </div>
      </Layout>
    );
  }

  const tabs: { key: Tab; label: string; icon: typeof Github }[] = [
    { key: "connection", label: gt.tabConnection, icon: Github },
    { key: "deploy", label: gt.tabDeploy, icon: Rocket },
    { key: "history", label: gt.tabHistory, icon: Clock },
  ];

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title={gt.pageTitle}
          description={gt.pageDescription}
          autoRefreshSeconds={activeTab !== "connection" ? intervalSeconds : undefined}
          onRefresh={activeTab !== "connection" ? refresh : undefined}
        />

        {/* 演示模式提示 */}
        {!isAuthenticated && (
          <div className="flex items-center gap-3 p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-xl">
            <Eye className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0" />
            <div>
              <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
                {t.common.demoMode}
              </p>
              <p className="text-xs text-amber-600 dark:text-amber-400">
                {t.common.demoModeHint}
              </p>
            </div>
          </div>
        )}

        {/* Tab 切换 */}
        <div className="flex gap-1 p-1 bg-gray-100 dark:bg-gray-800 rounded-lg w-fit">
          {tabs.map(({ key, label, icon: Icon }) => (
            <button
              key={key}
              onClick={() => setActiveTab(key)}
              className={`flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                activeTab === key
                  ? "bg-white dark:bg-gray-700 text-default shadow-sm"
                  : "text-muted hover:text-default"
              }`}
            >
              <Icon className="w-4 h-4" />
              {label}
            </button>
          ))}
        </div>

        {/* Tab: 连接 & 仓库 */}
        {activeTab === "connection" && (
          <div className="space-y-6">
            <ConnectionCard
              connection={connection}
              onConnect={handleConnect}
              onDisconnect={handleDisconnect}
            />
            <ReposCard
              repos={repos}
              connected={connection?.connected ?? false}
            />
          </div>
        )}

        {/* Tab: 部署配置 */}
        {activeTab === "deploy" && (
          <div className="space-y-6">
            {/* GitHub 未连接提示 */}
            {!githubConnected && (
              <div className="flex items-center justify-between p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-xl">
                <div className="flex items-center gap-3">
                  <AlertTriangle className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0" />
                  <div>
                    <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
                      {dt.githubNotConnected}
                    </p>
                    <p className="text-xs text-amber-600 dark:text-amber-400">
                      {dt.githubNotConnectedHint}
                    </p>
                  </div>
                </div>
                <button
                  onClick={() => setActiveTab("connection")}
                  className="flex items-center gap-1 px-3 py-1.5 text-sm rounded-lg bg-amber-600 text-white hover:bg-amber-700 transition-colors"
                >
                  {dt.connectGithub}
                  <ExternalLink className="w-3 h-3" />
                </button>
              </div>
            )}

            <ConfigCard
              config={config}
              editing={editing}
              editConfig={editConfig}
              repos={deployRepos}
              kustomizePaths={kustomizePaths}
              saving={saving}
              onStartEdit={handleStartEdit}
              onCancelEdit={handleCancelEdit}
              onSave={handleSaveConfig}
              onUpdateConfig={handleUpdateConfig}
              onTestConnection={handleTestConnection}
            />
            <StatusCard statusList={statusList} onSyncNow={handleSyncNow} syncingPaths={syncingPaths} />
          </div>
        )}

        {/* Tab: 部署历史 */}
        {activeTab === "history" && (
          <HistoryCard history={history} />
        )}
      </div>
    </Layout>
  );
}
