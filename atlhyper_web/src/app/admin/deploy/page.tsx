"use client";

import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { AlertTriangle, ExternalLink } from "lucide-react";
import Link from "next/link";

import { ConfigCard, StatusCard, HistoryCard, useDeployPage } from "./components";

export default function DeployPage() {
  const { t } = useI18n();
  const dt = t.deployPage;

  const {
    loading,
    githubConnected,
    config,
    editing,
    editConfig,
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
    syncingPaths,
    refresh,
    intervalSeconds,
  } = useDeployPage();

  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center py-12">
          <LoadingSpinner />
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={dt.pageTitle} description={dt.pageDescription} autoRefreshSeconds={intervalSeconds} onRefresh={refresh} />

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
            <Link
              href="/settings/github"
              className="flex items-center gap-1 px-3 py-1.5 text-sm rounded-lg bg-amber-600 text-white hover:bg-amber-700 transition-colors"
            >
              {dt.connectGithub}
              <ExternalLink className="w-3 h-3" />
            </Link>
          </div>
        )}

        {/* Config 仓库配置 */}
        <ConfigCard
          config={config}
          editing={editing}
          editConfig={editConfig}
          repos={repos}
          kustomizePaths={kustomizePaths}
          saving={saving}
          onStartEdit={handleStartEdit}
          onCancelEdit={handleCancelEdit}
          onSave={handleSaveConfig}
          onUpdateConfig={handleUpdateConfig}
          onTestConnection={handleTestConnection}
        />

        {/* 同步状态 */}
        <StatusCard statusList={statusList} onSyncNow={handleSyncNow} syncingPaths={syncingPaths} />

        {/* 部署历史 */}
        <HistoryCard history={history} />
      </div>
    </Layout>
  );
}
