import { useEffect, useState, useCallback } from "react";
import { toast } from "@/components/common/Toast";
import { useAuthStore } from "@/store/authStore";
import { useI18n } from "@/i18n/context";
import { UserRole } from "@/types/auth";

import {
  listProviders,
  createProvider,
  updateProvider,
  deleteProvider,
  updateActiveConfig,
  updateProviderRoles,
  mockProviderList,
  type AIProvider,
  type ProviderListResponse,
} from "@/api/ai-provider";

export function useAISettings() {
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
      setGlobalEnabled(mockProviderList.activeConfig.enabled);
      setGlobalTimeout(mockProviderList.activeConfig.toolTimeout);
      setLoading(false);
      return;
    }

    try {
      const res = await listProviders();
      setData(res.data);
      setGlobalEnabled(res.data.activeConfig.enabled);
      setGlobalTimeout(res.data.activeConfig.toolTimeout);
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
    baseUrl: string;
    description: string;
    roles: string[];
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
          baseUrl: formData.baseUrl || undefined,
          description: formData.description,
          ...(formData.apiKey ? { apiKey: formData.apiKey } : {}),
        });
        // 角色分配（编辑时才有）
        try {
          await updateProviderRoles(editingProvider.id, formData.roles);
        } catch (roleErr: unknown) {
          const status = (roleErr as { response?: { status?: number } })?.response?.status;
          if (status === 409) {
            toast.error(aiT.roleAssignConflict);
          } else {
            toast.error(aiT.saveFailed);
          }
          setSaving(false);
          loadData();
          return;
        }
        toast.success(aiT.providerUpdated);
      } else {
        await createProvider({
          name: formData.name,
          provider: formData.provider,
          apiKey: formData.apiKey,
          model: formData.model,
          baseUrl: formData.baseUrl || undefined,
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
    if (provider.isActive) {
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
      await updateActiveConfig({ providerId: provider.id });
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
      await updateActiveConfig({ toolTimeout: globalTimeout });
      toast.success(aiT.timeoutSaved);
    } catch (err) {
      console.error("Failed to save timeout:", err);
      toast.error(aiT.saveFailed);
    } finally {
      setSavingGlobal(false);
    }
  };

  return {
    // Permissions
    isAdmin,
    isDemo,
    // Data
    loading,
    data,
    // Global settings
    globalEnabled,
    globalTimeout,
    savingGlobal,
    setGlobalTimeout,
    handleToggleEnabled,
    handleSaveGlobalTimeout,
    // Modal
    showModal,
    setShowModal,
    editingProvider,
    saving,
    // Provider actions
    handleAddProvider,
    handleEditProvider,
    handleSaveProvider,
    handleDeleteProvider,
    handleActivateProvider,
  };
}
