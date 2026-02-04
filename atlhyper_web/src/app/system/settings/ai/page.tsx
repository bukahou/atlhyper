"use client";

import { useEffect, useState, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { toast } from "@/components/common/Toast";
import { useAuthStore } from "@/store/authStore";
import { AlertTriangle, Bot, Plus } from "lucide-react";

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

  const isGuest = !isAuthenticated;
  const isAdmin = user?.role === 3;

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
    if (isGuest) {
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
      toast.error("AI設定の読み込みに失敗しました");
    } finally {
      setLoading(false);
    }
  }, [isGuest]);

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
      toast.error("必須項目を入力してください");
      return;
    }
    if (!editingProvider && !formData.apiKey) {
      toast.error("API Keyを入力してください");
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
        toast.success("プロバイダーを更新しました");
      } else {
        await createProvider({
          name: formData.name,
          provider: formData.provider,
          api_key: formData.apiKey,
          model: formData.model,
          description: formData.description,
        });
        toast.success("プロバイダーを追加しました");
      }
      setShowModal(false);
      loadData();
    } catch (err) {
      console.error("Failed to save provider:", err);
      toast.error("保存に失敗しました");
    } finally {
      setSaving(false);
    }
  };

  // Delete provider
  const handleDeleteProvider = async (provider: AIProvider) => {
    if (provider.is_active) {
      toast.error("有効なプロバイダーは削除できません");
      return;
    }
    if (!confirm(`「${provider.name}」を削除しますか？`)) return;

    try {
      await deleteProvider(provider.id);
      toast.success("プロバイダーを削除しました");
      loadData();
    } catch (err) {
      console.error("Failed to delete provider:", err);
      toast.error("削除に失敗しました");
    }
  };

  // Activate provider
  const handleActivateProvider = async (provider: AIProvider) => {
    try {
      await updateActiveConfig({ provider_id: provider.id });
      toast.success(`「${provider.name}」を有効化しました`);
      loadData();
    } catch (err) {
      console.error("Failed to activate provider:", err);
      toast.error("有効化に失敗しました");
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
      toast.success(newEnabled ? "AI機能を有効化しました" : "AI機能を無効化しました");
    } catch (err) {
      console.error("Failed to toggle enabled:", err);
      toast.error("設定の変更に失敗しました");
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
      toast.success("タイムアウト設定を保存しました");
    } catch (err) {
      console.error("Failed to save timeout:", err);
      toast.error("保存に失敗しました");
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
          title="AI プロバイダー設定"
          description="複数のAIプロバイダーを管理し、切り替えることができます"
        />

        {/* Guest 提示 */}
        {isGuest && (
          <div className="flex items-center gap-3 p-4 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800">
            <AlertTriangle className="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0" />
            <p className="text-sm text-yellow-800 dark:text-yellow-300">
              デモモード - サンプルデータを表示しています。ログインして実際の設定を確認してください。
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
            <h3 className="text-lg font-medium text-default">プロバイダー一覧</h3>
            {isAdmin && (
              <button
                onClick={handleAddProvider}
                className="flex items-center gap-2 px-4 py-2 text-sm rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors"
              >
                <Plus className="w-4 h-4" />
                新規追加
              </button>
            )}
          </div>

          <div className="p-6">
            {data?.providers.length === 0 ? (
              <div className="text-center py-12 text-muted">
                <Bot className="w-12 h-12 mx-auto mb-3 opacity-30" />
                <p>プロバイダーが登録されていません</p>
                {isAdmin && (
                  <button
                    onClick={handleAddProvider}
                    className="mt-4 text-violet-600 hover:underline"
                  >
                    最初のプロバイダーを追加
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
                    <span>新規追加</span>
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
