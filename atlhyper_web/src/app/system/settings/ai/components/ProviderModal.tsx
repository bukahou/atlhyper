"use client";

import { useState, useEffect } from "react";
import { X, Eye, EyeOff, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import type { AIProvider, ProviderModelInfo } from "@/api/ai-provider";

interface ProviderModalProps {
  isOpen: boolean;
  editingProvider: AIProvider | null;
  models: ProviderModelInfo[];
  saving: boolean;
  onClose: () => void;
  onSave: (data: {
    name: string;
    provider: string;
    apiKey: string;
    model: string;
    description: string;
  }) => void;
}

export function ProviderModal({
  isOpen,
  editingProvider,
  models,
  saving,
  onClose,
  onSave,
}: ProviderModalProps) {
  const { t } = useI18n();
  const aiT = t.aiSettingsPage;

  const [formName, setFormName] = useState("");
  const [formProvider, setFormProvider] = useState("gemini");
  const [formApiKey, setFormApiKey] = useState("");
  const [formModel, setFormModel] = useState("");
  const [formCustomModel, setFormCustomModel] = useState("");
  const [formUseCustomModel, setFormUseCustomModel] = useState(false);
  const [formDescription, setFormDescription] = useState("");
  const [showApiKey, setShowApiKey] = useState(false);

  const getModelsForProvider = (provider: string): string[] => {
    const info = models.find((m) => m.provider === provider);
    return info?.models || [];
  };

  // Initialize form when modal opens
  useEffect(() => {
    if (isOpen) {
      if (editingProvider) {
        setFormName(editingProvider.name);
        setFormProvider(editingProvider.provider);
        setFormApiKey("");
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
        setFormDescription("");
        setShowApiKey(false);
      }
    }
  }, [isOpen, editingProvider]);

  // Update model when provider changes
  useEffect(() => {
    if (isOpen && !formUseCustomModel) {
      const providerModels = getModelsForProvider(formProvider);
      if (providerModels.length > 0 && !providerModels.includes(formModel)) {
        setFormModel(providerModels[0]);
      }
    }
  }, [formProvider, isOpen, formUseCustomModel]);

  const handleSave = () => {
    const currentModel = formUseCustomModel ? formCustomModel : formModel;
    onSave({
      name: formName,
      provider: formProvider,
      apiKey: formApiKey,
      model: currentModel,
      description: formDescription,
    });
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div className="bg-card rounded-xl border border-[var(--border-color)] w-full max-w-lg max-h-[90vh] overflow-y-auto">
        {/* Modal Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)]">
          <h3 className="text-lg font-medium text-default">
            {editingProvider ? aiT.editProvider : aiT.newProvider}
          </h3>
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-[var(--bg-secondary)]"
          >
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        {/* Modal Body */}
        <div className="px-6 py-4 space-y-4">
          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-default mb-1">
              {aiT.name} <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={formName}
              onChange={(e) => setFormName(e.target.value)}
              placeholder={aiT.namePlaceholder}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 placeholder:text-gray-400"
            />
          </div>

          {/* Provider */}
          <div>
            <label className="block text-sm font-medium text-default mb-1">
              {aiT.provider} <span className="text-red-500">*</span>
            </label>
            <select
              value={formProvider}
              onChange={(e) => setFormProvider(e.target.value)}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
            >
              {models.map((m) => (
                <option key={m.provider} value={m.provider} className="bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100">
                  {m.name}
                </option>
              ))}
            </select>
          </div>

          {/* API Key */}
          <div>
            <label className="block text-sm font-medium text-default mb-1">
              {aiT.apiKey} {!editingProvider && <span className="text-red-500">*</span>}
            </label>
            <div className="relative">
              <input
                type={showApiKey ? "text" : "password"}
                value={formApiKey}
                onChange={(e) => setFormApiKey(e.target.value)}
                placeholder={editingProvider ? aiT.apiKeyUpdatePlaceholder : aiT.apiKeyPlaceholder}
                className="w-full px-3 py-2 pr-10 rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 placeholder:text-gray-400 font-mono"
              />
              <button
                type="button"
                onClick={() => setShowApiKey(!showApiKey)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted hover:text-default"
              >
                {showApiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
            {editingProvider?.api_key_set && (
              <p className="mt-1 text-xs text-muted">
                {aiT.current}: {editingProvider.api_key_masked}
              </p>
            )}
          </div>

          {/* Model */}
          <div>
            <label className="block text-sm font-medium text-default mb-1">
              {aiT.model} <span className="text-red-500">*</span>
            </label>
            <div className="flex items-center gap-2 mb-2">
              <input
                type="checkbox"
                id="useCustomModel"
                checked={formUseCustomModel}
                onChange={(e) => {
                  setFormUseCustomModel(e.target.checked);
                  if (!e.target.checked) {
                    const providerModels = getModelsForProvider(formProvider);
                    if (providerModels.length > 0) setFormModel(providerModels[0]);
                  }
                }}
                className="w-4 h-4 rounded"
              />
              <label htmlFor="useCustomModel" className="text-sm text-muted">
                {aiT.customModel}
              </label>
            </div>
            {formUseCustomModel ? (
              <input
                type="text"
                value={formCustomModel}
                onChange={(e) => setFormCustomModel(e.target.value)}
                placeholder={aiT.customModelPlaceholder}
                className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 placeholder:text-gray-400 font-mono"
              />
            ) : (
              <select
                value={formModel}
                onChange={(e) => setFormModel(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
              >
                {getModelsForProvider(formProvider).map((m) => (
                  <option key={m} value={m} className="bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100">
                    {m}
                  </option>
                ))}
              </select>
            )}
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-default mb-1">{aiT.description}</label>
            <textarea
              value={formDescription}
              onChange={(e) => setFormDescription(e.target.value)}
              placeholder={aiT.descriptionPlaceholder}
              rows={2}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 placeholder:text-gray-400 resize-none"
            />
          </div>
        </div>

        {/* Modal Footer */}
        <div className="flex justify-end gap-3 px-6 py-4 border-t border-[var(--border-color)] bg-[var(--bg-secondary)]">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm rounded-lg border border-[var(--border-color)] text-default hover:bg-[var(--bg-primary)]"
          >
            {aiT.cancel}
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-2 text-sm rounded-lg bg-violet-600 text-white hover:bg-violet-700 disabled:opacity-50 flex items-center gap-2"
          >
            {saving && <Loader2 className="w-4 h-4 animate-spin" />}
            {aiT.save}
          </button>
        </div>
      </div>
    </div>
  );
}
