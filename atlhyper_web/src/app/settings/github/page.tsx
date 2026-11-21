"use client";

import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner, DemoBanner } from "@/components/common";

import { ConnectionCard, ReposCard, useGitHubPage } from "./components";

export default function GitHubPage() {
  const { t } = useI18n();
  const gt = t.githubPage;

  const { isDemo, loading, connection, repos, handleConnect, handleDisconnect } = useGitHubPage();

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
        <PageHeader title={gt.pageTitle} description={gt.pageDescription} />

        {isDemo && <DemoBanner hint={t.common.demoModeHintGitHub} />}

        <ConnectionCard connection={connection} onConnect={handleConnect} onDisconnect={handleDisconnect} />
        <ReposCard repos={repos} connected={connection?.connected ?? false} />
      </div>
    </Layout>
  );
}
