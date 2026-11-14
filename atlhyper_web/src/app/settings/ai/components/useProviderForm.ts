import { useState } from "react";
import type { AIProvider, ProviderModelInfo } from "@/api/ai-provider";
import {
  initialProviderForm,
  presetModelsOf,
  withCustomModelToggled,
  withProviderVendor,
  type ProviderFormDraft,
} from "./provider-form";

interface UseProviderFormOptions {
  editingProvider: AIProvider | null;
  models: ProviderModelInfo[];
}

/**
 * Provider 弹窗的表单状态。
 *
 * 表单只在弹窗打开期间存在（ProviderModal 关着时不挂载它），所以初始值在挂载时算一次就够，
 * 不需要「弹窗一打开就用 effect 把各字段重置一遍」。
 */
export function useProviderForm({ editingProvider, models }: UseProviderFormOptions) {
  const [draft, setDraft] = useState<ProviderFormDraft>(() => initialProviderForm(editingProvider, models));
  const [showApiKey, setShowApiKey] = useState(false);

  const updateDraft = (changes: Partial<ProviderFormDraft>) => setDraft((current) => ({ ...current, ...changes }));

  return {
    formName: draft.name,
    setFormName: (name: string) => updateDraft({ name }),
    formProvider: draft.provider,
    // 换厂商时顺带把预设模型换成新厂商的（见 withProviderVendor）
    setFormProvider: (provider: string) => setDraft((current) => withProviderVendor(current, provider, models)),
    formApiKey: draft.apiKey,
    setFormApiKey: (apiKey: string) => updateDraft({ apiKey }),
    formModel: draft.model,
    setFormModel: (model: string) => updateDraft({ model }),
    formCustomModel: draft.customModel,
    setFormCustomModel: (customModel: string) => updateDraft({ customModel }),
    formUseCustomModel: draft.useCustomModel,
    handleToggleCustomModel: (checked: boolean) =>
      setDraft((current) => withCustomModelToggled(current, checked, models)),
    formBaseUrl: draft.baseUrl,
    setFormBaseUrl: (baseUrl: string) => updateDraft({ baseUrl }),
    formDescription: draft.description,
    setFormDescription: (description: string) => updateDraft({ description }),
    showApiKey,
    setShowApiKey,
    getModelsForProvider: (provider: string) => presetModelsOf(models, provider),
    getFormData: () => ({
      name: draft.name,
      provider: draft.provider,
      apiKey: draft.apiKey,
      model: draft.useCustomModel ? draft.customModel : draft.model,
      baseUrl: draft.baseUrl,
      description: draft.description,
    }),
  };
}
