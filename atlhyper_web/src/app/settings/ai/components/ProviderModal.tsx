"use client";

import { X, Eye, EyeOff, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import type { AIProvider, ProviderModelInfo } from "@/api/ai-provider";
import { useProviderForm } from "./useProviderForm";

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
    baseUrl: string;
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

  const form = useProviderForm({ isOpen, editingProvider, models });

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
              value={form.formName}
              onChange={(e) => form.setFormName(e.target.value)}
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
              value={form.formProvider}
              onChange={(e) => form.setFormProvider(e.target.value)}
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
                type={form.showApiKey ? "text" : "password"}
                value={form.formApiKey}
                onChange={(e) => form.setFormApiKey(e.target.value)}
                placeholder={editingProvider ? aiT.apiKeyUpdatePlaceholder : aiT.apiKeyPlaceholder}
                className="w-full px-3 py-2 pr-10 rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 placeholder:text-gray-400 font-mono"
              />
              <button
                type="button"
                onClick={() => form.setShowApiKey(!form.showApiKey)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted hover:text-default"
              >
                {form.showApiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
            {editingProvider?.apiKeySet && (
              <p className="mt-1 text-xs text-muted">
                {aiT.current}: {editingProvider.apiKeyMasked}
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
                checked={form.formUseCustomModel}
                onChange={(e) => form.handleToggleCustomModel(e.target.checked)}
                className="w-4 h-4 rounded"
              />
              <label htmlFor="useCustomModel" className="text-sm text-muted">
                {aiT.customModel}
              </label>
            </div>
            {form.formUseCustomModel ? (
              <input
                type="text"
                value={form.formCustomModel}
                onChange={(e) => form.setFormCustomModel(e.target.value)}
                placeholder={aiT.customModelPlaceholder}
                className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 placeholder:text-gray-400 font-mono"
              />
            ) : (
              <select
                value={form.formModel}
                onChange={(e) => form.setFormModel(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
              >
                {form.getModelsForProvider(form.formProvider).map((m) => (
                  <option key={m} value={m} className="bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100">
                    {m}
                  </option>
                ))}
              </select>
            )}
          </div>

          {/* Base URL (Ollama等) */}
          {form.formProvider === "ollama" && (
            <div>
              <label className="block text-sm font-medium text-default mb-1">
                {aiT.baseUrl}
              </label>
              <input
                type="text"
                value={form.formBaseUrl}
                onChange={(e) => form.setFormBaseUrl(e.target.value)}
                placeholder={aiT.baseUrlPlaceholder}
                className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 placeholder:text-gray-400 font-mono"
              />
              <p className="mt-1 text-xs text-muted">{aiT.baseUrlHint}</p>
            </div>
          )}

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-default mb-1">{aiT.description}</label>
            <textarea
              value={form.formDescription}
              onChange={(e) => form.setFormDescription(e.target.value)}
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
            onClick={() => onSave(form.getFormData())}
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
