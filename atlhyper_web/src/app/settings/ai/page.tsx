"use client";

import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { AlertTriangle, Bot, Plus, Eye } from "lucide-react";

import { GlobalSettingsCard, ProviderCard, ProviderModal, RoleOverviewCard, BudgetConfigCard, UsageHistoryCard } from "./components";
import { useAISettings } from "./components/useAISettings";

export default function AISettingsPage() {
  const { t } = useI18n();
  const aiT = t.aiSettingsPage;

  const {
    isAdmin,
    isDemo,
    loading,
    data,
    globalEnabled,
    globalTimeout,
    savingGlobal,
    setGlobalTimeout,
    handleToggleEnabled,
    handleSaveGlobalTimeout,
    showModal,
    setShowModal,
    editingProvider,
    saving,
    handleAddProvider,
    handleEditProvider,
    handleSaveProvider,
    handleDeleteProvider,
    handleActivateProvider,
  } = useAISettings();

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
        <PageHeader
          title={aiT.pageTitle}
          description={aiT.pageDescription}
        />

        {/* 演示模式提示 */}
        {isDemo && (
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

        {/* 非 Admin 提示（有查看权限但无修改权限） */}
        {!isDemo && !isAdmin && (
          <div className="flex items-center gap-3 p-4 rounded-xl bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
            <AlertTriangle className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0" />
            <p className="text-sm text-blue-800 dark:text-blue-300">
              {t.common.viewOnlyHint}
            </p>
          </div>
        )}

        {/* Global Settings Card */}
        <GlobalSettingsCard
          enabled={globalEnabled}
          timeout={globalTimeout}
          isAdmin={isAdmin}
          saving={savingGlobal}
          onToggleEnabled={handleToggleEnabled}
          onTimeoutChange={setGlobalTimeout}
          onSaveTimeout={handleSaveGlobalTimeout}
        />

        {/* Role Overview */}
        <RoleOverviewCard />

        {/* Provider List */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)]">
            <h3 className="text-lg font-medium text-default">{aiT.providerList}</h3>
            {isAdmin && (
              <button
                onClick={handleAddProvider}
                className="flex items-center gap-2 px-4 py-2 text-sm rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors"
              >
                <Plus className="w-4 h-4" />
                {aiT.addProvider}
              </button>
            )}
          </div>

          <div className="p-6">
            {data?.providers.length === 0 ? (
              <div className="text-center py-12 text-muted">
                <Bot className="w-12 h-12 mx-auto mb-3 opacity-30" />
                <p>{aiT.noProviders}</p>
                {isAdmin && (
                  <button
                    onClick={handleAddProvider}
                    className="mt-4 text-violet-600 hover:underline"
                  >
                    {aiT.addFirstProvider}
                  </button>
                )}
              </div>
            ) : (
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {data?.providers.map((provider) => (
                  <ProviderCard
                    key={provider.id}
                    provider={provider}
                    isAdmin={isAdmin}
                    onActivate={handleActivateProvider}
                    onEdit={handleEditProvider}
                    onDelete={handleDeleteProvider}
                  />
                ))}

                {/* Add Card */}
                {isAdmin && (
                  <button
                    onClick={handleAddProvider}
                    className="flex flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-[var(--border-color)] p-6 text-muted hover:border-violet-300 hover:text-violet-600 transition-colors min-h-[200px]"
                  >
                    <Plus className="w-8 h-8" />
                    <span>{aiT.addProvider}</span>
                  </button>
                )}
              </div>
            )}
          </div>
        </div>
        {/* Budget Config */}
        <BudgetConfigCard />

        {/* Usage History */}
        <UsageHistoryCard />
      </div>

      {/* Modal */}
      <ProviderModal
        isOpen={showModal}
        editingProvider={editingProvider}
        models={data?.models || []}
        saving={saving}
        onClose={() => setShowModal(false)}
        onSave={handleSaveProvider}
      />
    </Layout>
  );
}
