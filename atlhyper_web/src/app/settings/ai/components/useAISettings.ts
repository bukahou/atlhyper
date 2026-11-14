import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "@/components/common/Toast";
import { useI18n } from "@/i18n/context";
import type { AIProvider } from "@/api/ai-provider";
import { useCan } from "@/hooks/useCan";
import { useSessionReady } from "@/hooks/useSessionReady";
import { mockProviderList } from "@/mock/ai-settings";
import {
  aiProvidersQuery,
  createAIProviderMutation,
  deleteAIProviderMutation,
  updateAIGlobalSettingsMutation,
  updateAIProviderMutation,
} from "@/queries/ai-settings";

/** 全局设置还没取到时输入框里显示的超时（秒） */
const DEFAULT_TOOL_TIMEOUT_SECONDS = 30;

export function useAISettings() {
  const { t } = useI18n();
  const can = useCan();
  const aiT = t.aiSettingsPage;

  // AI 设置是纯管理功能页：权限加载完才决定显示真实配置还是演示数据
  const sessionReady = useSessionReady();
  const hasViewPermission = can("ai.settings.read");
  const isAdmin = can("ai.settings.write");
  const isDemo = sessionReady && !hasViewPermission;

  // Provider 列表（连同可选模型与全局设置）：全页共用这一份查询
  const providersQuery = useQuery(aiProvidersQuery(hasViewPermission));
  const data = isDemo ? mockProviderList : providersQuery.data;

  // 写操作成功后工厂让 Provider 列表与角色总览重新取数
  const createProvider = useMutation(createAIProviderMutation());
  const updateProvider = useMutation(updateAIProviderMutation());
  const deleteProvider = useMutation(deleteAIProviderMutation());
  const saveGlobalSettings = useMutation(updateAIGlobalSettingsMutation());

  // Modal state
  const [showModal, setShowModal] = useState(false);
  const [editingProvider, setEditingProvider] = useState<AIProvider | null>(null);

  // 全局超时：用户没改过（null）时显示已保存的值
  const [timeoutDraft, setTimeoutDraft] = useState<number | null>(null);
  const globalTimeout = timeoutDraft ?? data?.settings.toolTimeout ?? DEFAULT_TOOL_TIMEOUT_SECONDS;

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
  const handleSaveProvider = (formData: {
    name: string;
    provider: string;
    apiKey: string;
    model: string;
    baseUrl: string;
    description: string;
  }) => {
    if (!formData.name || !formData.provider || !formData.model) {
      toast.error(aiT.requiredFields);
      return;
    }
    // Ollama（自部署）可以不填 key，与后端校验一致
    if (!editingProvider && !formData.apiKey && formData.provider !== "ollama") {
      toast.error(aiT.apiKeyRequired);
      return;
    }

    const onError = (err: unknown) => {
      console.error("Failed to save provider:", err);
      toast.error(aiT.saveFailed);
    };

    if (editingProvider) {
      updateProvider.mutate(
        {
          providerId: editingProvider.id,
          changes: {
            name: formData.name,
            provider: formData.provider,
            model: formData.model,
            baseUrl: formData.baseUrl.trim(), // 空字符串 = 清空（恢复默认地址），不能转成「不传」
            description: formData.description,
            ...(formData.apiKey ? { apiKey: formData.apiKey } : {}),
          },
        },
        {
          onSuccess: () => {
            toast.success(aiT.providerUpdated);
            setShowModal(false);
          },
          onError,
        },
      );
    } else {
      createProvider.mutate(
        {
          name: formData.name,
          provider: formData.provider,
          apiKey: formData.apiKey,
          model: formData.model,
          baseUrl: formData.baseUrl || undefined,
          description: formData.description,
        },
        {
          onSuccess: () => {
            toast.success(aiT.providerAdded);
            setShowModal(false);
          },
          onError,
        },
      );
    }
  };

  // Delete provider
  const handleDeleteProvider = (provider: AIProvider) => {
    if (!confirm(aiT.confirmDelete.replace("{name}", provider.name))) return;

    deleteProvider.mutate(provider.id, {
      onSuccess: () => toast.success(aiT.providerDeleted),
      onError: (err) => {
        console.error("Failed to delete provider:", err);
        toast.error(aiT.deleteFailed);
      },
    });
  };

  // Save global timeout
  const handleSaveGlobalTimeout = () => {
    if (!isAdmin) return;
    saveGlobalSettings.mutate(
      { toolTimeout: globalTimeout },
      {
        onSuccess: () => toast.success(aiT.timeoutSaved),
        onError: (err) => {
          console.error("Failed to save timeout:", err);
          toast.error(aiT.saveFailed);
        },
      },
    );
  };

  return {
    // Permissions
    sessionReady,
    isAdmin,
    isDemo,
    // Data
    providersQuery,
    data,
    // Global settings
    globalTimeout,
    savingGlobal: saveGlobalSettings.isPending,
    setGlobalTimeout: setTimeoutDraft,
    handleSaveGlobalTimeout,
    // Modal
    showModal,
    setShowModal,
    editingProvider,
    saving: createProvider.isPending || updateProvider.isPending,
    // Provider actions
    handleAddProvider,
    handleEditProvider,
    handleSaveProvider,
    handleDeleteProvider,
  };
}
