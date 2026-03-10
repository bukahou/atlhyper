"use client";

import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";

import { ConnectionCard, ReposCard, MappingCard, useGitHubPage } from "./components";

export default function GitHubPage() {
  const { t } = useI18n();
  const gt = t.githubPage;

  const {
    loading,
    connection,
    repos,
    mappings,
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
  } = useGitHubPage();

  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center py-12">
          <LoadingSpinner />
        </div>
      </Layout>
    );
  }

  const hasEnabledRepos = Object.keys(repoDirs).length > 0;

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={gt.pageTitle} description={gt.pageDescription} />

        {/* GitHub 连接状态 */}
        <ConnectionCard
          connection={connection}
          onConnect={handleConnect}
          onDisconnect={handleDisconnect}
        />

        {/* 已授权仓库 + 映射开关 */}
        <ReposCard
          repos={repos}
          connected={connection?.connected ?? false}
          onToggleMapping={handleToggleMapping}
        />

        {/* 仓库 ↔ Deployment 映射 */}
        {hasEnabledRepos && (
          <MappingCard
            mappings={mappings}
            connected={connection?.connected ?? false}
            namespaces={namespaces}
            deployments={deployments}
            repoDirs={repoDirs}
            repoNamespaces={repoNamespaces}
            onUpdateMapping={handleUpdateMapping}
            onConfirmMapping={handleConfirmMapping}
            onConfirmAll={handleConfirmAll}
            onAddMapping={handleAddMapping}
            onDeleteMapping={handleDeleteMapping}
            onAddRepoNamespace={handleAddRepoNamespace}
            onRemoveRepoNamespace={handleRemoveRepoNamespace}
          />
        )}
      </div>
    </Layout>
  );
}
