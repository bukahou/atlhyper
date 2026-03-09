import { useState, useEffect, useCallback } from "react";
import type { AIProvider, ProviderModelInfo } from "@/api/ai-provider";

interface UseProviderFormOptions {
  isOpen: boolean;
  editingProvider: AIProvider | null;
  models: ProviderModelInfo[];
}

export function useProviderForm({ isOpen, editingProvider, models }: UseProviderFormOptions) {
  const [formName, setFormName] = useState("");
  const [formProvider, setFormProvider] = useState("gemini");
  const [formApiKey, setFormApiKey] = useState("");
  const [formModel, setFormModel] = useState("");
  const [formCustomModel, setFormCustomModel] = useState("");
  const [formUseCustomModel, setFormUseCustomModel] = useState(false);
  const [formBaseUrl, setFormBaseUrl] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [showApiKey, setShowApiKey] = useState(false);

  const getModelsForProvider = useCallback(
    (provider: string): string[] => {
      const info = models.find((m) => m.provider === provider);
      return info?.models || [];
    },
    [models]
  );

  // Initialize form when modal opens
  useEffect(() => {
    if (isOpen) {
      if (editingProvider) {
        setFormName(editingProvider.name);
        setFormProvider(editingProvider.provider);
        setFormApiKey("");
        setFormBaseUrl(editingProvider.baseUrl || "");
        setFormDescription(editingProvider.description);
        setShowApiKey(false);

        const presetModels = getModelsForProvider(editingProvider.provider);
        if (presetModels.includes(editingProvider.model)) {
          setFormModel(editingProvider.model);
          setFormUseCustomModel(false);
          setFormCustomModel("");
        } else {
          setFormModel("");
          setFormUseCustomModel(true);
          setFormCustomModel(editingProvider.model);
        }
      } else {
        setFormName("");
        setFormProvider("gemini");
        setFormApiKey("");
        setFormModel(getModelsForProvider("gemini")[0] || "");
        setFormCustomModel("");
        setFormUseCustomModel(false);
        setFormBaseUrl("");
        setFormDescription("");
        setShowApiKey(false);
      }
    }
  }, [isOpen, editingProvider, getModelsForProvider]);

  // Update model when provider changes
  useEffect(() => {
    if (isOpen && !formUseCustomModel) {
      const providerModels = getModelsForProvider(formProvider);
      if (providerModels.length > 0 && !providerModels.includes(formModel)) {
        setFormModel(providerModels[0]);
      }
    }
  }, [formProvider, isOpen, formUseCustomModel, getModelsForProvider, formModel]);

  const handleToggleCustomModel = (checked: boolean) => {
    setFormUseCustomModel(checked);
    if (!checked) {
      const providerModels = getModelsForProvider(formProvider);
      if (providerModels.length > 0) setFormModel(providerModels[0]);
    }
  };

  const getFormData = () => ({
    name: formName,
    provider: formProvider,
    apiKey: formApiKey,
    model: formUseCustomModel ? formCustomModel : formModel,
    baseUrl: formBaseUrl,
    description: formDescription,
  });

  return {
    formName,
    setFormName,
    formProvider,
    setFormProvider,
    formApiKey,
    setFormApiKey,
    formModel,
    setFormModel,
    formCustomModel,
    setFormCustomModel,
    formUseCustomModel,
    handleToggleCustomModel,
    formBaseUrl,
    setFormBaseUrl,
    formDescription,
    setFormDescription,
    showApiKey,
    setShowApiKey,
    getModelsForProvider,
    getFormData,
  };
}
