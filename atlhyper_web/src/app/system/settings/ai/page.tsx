"use client";

import { useEffect, useState, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { toast } from "@/components/common/Toast";
import { useAuthStore } from "@/store/authStore";
import { AlertTriangle, Bot, Plus, Eye } from "lucide-react";
import { UserRole } from "@/types/auth";

import {
  listProviders,
  createProvider,
  updateProvider,
  deleteProvider,
  updateActiveConfig,
  mockProviderList,
  type AIProvider,
  type ProviderListResponse,
} from "@/api/ai-provider";

import { GlobalSettingsCard, ProviderCard, ProviderModal } from "./components";

export default function AISettingsPage() {
  const { t } = useI18n();
  const { user, isAuthenticated } = useAuthStore();
  const aiT = t.aiSettingsPage;

  const hasViewPermission = isAuthenticated && user && user.role >= UserRole.OPERATOR;
  const isAdmin = user?.role === UserRole.ADMIN;
  const isDemo = !hasViewPermission;

  // State
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<ProviderListResponse | null>(null);

  // Modal state
  const [showModal, setShowModal] = useState(false);
  const [editingProvider, setEditingProvider] = useState<AIProvider | null>(null);
  const [saving, setSaving] = useState(false);

  // Global settings state
  const [globalEnabled, setGlobalEnabled] = useState(false);
  const [globalTimeout, setGlobalTimeout] = useState(30);
  const [savingGlobal, setSavingGlobal] = useState(false);

  // Load data
  const loadData = useCallback(async () => {
    if (isDemo) {
      setData(mockProviderList);
      setGlobalEnabled(mockProviderList.active_config.enabled);
      setGlobalTimeout(mockProviderList.active_config.tool_timeout);
      setLoading(false);
      return;
    }

    try {
      const res = await listProviders();
      setData(res.data);
      setGlobalEnabled(res.data.active_config.enabled);
      setGlobalTimeout(res.data.active_config.tool_timeout);
    } catch (err) {
      console.error("Failed to load providers:", err);
      toast.error(aiT.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [isDemo]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Open modal for new provider
  const handleAddProvider = () => {
    setEditingProvider(null);
    setShowModal(true);
  };

  // Open modal for editing
  const handleEditProvider = (provider: AIProvider) => {
    setEditingProvider(provider);
    setShowModal(true);
  };

  // Save provider
  const handleSaveProvider = async (formData: {
    name: string;
    provider: string;
    apiKey: string;
    model: string;
    description: string;
  }) => {
    if (!formData.name || !formData.provider || !formData.model) {
      toast.error(aiT.requiredFields);
      return;
    }
    if (!editingProvider && !formData.apiKey) {
      toast.error(aiT.apiKeyRequired);
      return;
    }

    setSaving(true);
    try {
      if (editingProvider) {
        await updateProvider(editingProvider.id, {
          name: formData.name,
          provider: formData.provider,
          model: formData.model,
          description: formData.description,
          ...(formData.apiKey ? { api_key: formData.apiKey } : {}),
        });
        toast.success(aiT.providerUpdated);
      } else {
        await createProvider({
          name: formData.name,
          provider: formData.provider,
          api_key: formData.apiKey,
          model: formData.model,
          description: formData.description,
        });
        toast.success(aiT.providerAdded);
      }
      setShowModal(false);
      loadData();
    } catch (err) {
      console.error("Failed to save provider:", err);
      toast.error(aiT.saveFailed);
    } finally {
      setSaving(false);
    }
  };

  // Delete provider
  const handleDeleteProvider = async (provider: AIProvider) => {
    if (provider.is_active) {
      toast.error(aiT.cannotDeleteActive);
      return;
    }
    if (!confirm(aiT.confirmDelete.replace("{name}", provider.name))) return;

    try {
      await deleteProvider(provider.id);
      toast.success(aiT.providerDeleted);
      loadData();
    } catch (err) {
      console.error("Failed to delete provider:", err);
      toast.error(aiT.deleteFailed);
    }
  };

  // Activate provider
  const handleActivateProvider = async (provider: AIProvider) => {
    try {
      await updateActiveConfig({ provider_id: provider.id });
      toast.success(aiT.providerActivated.replace("{name}", provider.name));
      loadData();
    } catch (err) {
      console.error("Failed to activate provider:", err);
      toast.error(aiT.activateFailed);
    }
  };

  // Toggle global enabled
  const handleToggleEnabled = async () => {
    if (!isAdmin) return;
    setSavingGlobal(true);
    try {
      const newEnabled = !globalEnabled;
      await updateActiveConfig({ enabled: newEnabled });
      setGlobalEnabled(newEnabled);
      toast.success(newEnabled ? aiT.aiEnabled : aiT.aiDisabled);
    } catch (err) {
      console.error("Failed to toggle enabled:", err);
      toast.error(aiT.settingChangeFailed);
    } finally {
      setSavingGlobal(false);
    }
  };

  // Save global timeout
  const handleSaveGlobalTimeout = async () => {
    if (!isAdmin) return;
    setSavingGlobal(true);
    try {
      await updateActiveConfig({ tool_timeout: globalTimeout });
      toast.success(aiT.timeoutSaved);
    } catch (err) {
      console.error("Failed to save timeout:", err);
      toast.error(aiT.saveFailed);
    } finally {
      setSavingGlobal(false);
    }
  };

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
                {t.locale === "zh" ? "演示模式" : "デモモード"}
              </p>
              <p className="text-xs text-amber-600 dark:text-amber-400">
                {t.locale === "zh"
                  ? "当前展示的是示例数据。登录并获得 Operator 权限后可查看真实配置。"
                  : "サンプルデータを表示中です。Operator 権限でログインすると実際の設定を確認できます。"}
              </p>
            </div>
          </div>
        )}

        {/* 非 Admin 提示（有查看权限但无修改权限） */}
        {!isDemo && !isAdmin && (
          <div className="flex items-center gap-3 p-4 rounded-xl bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
            <AlertTriangle className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0" />
            <p className="text-sm text-blue-800 dark:text-blue-300">
              {t.locale === "zh"
                ? "您只有查看权限。如需修改配置，请联系管理员。"
                : "閲覧のみ可能です。設定を変更するには管理者にお問い合わせください。"}
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
