"use client";

import { Bot, Settings, Trash2, Zap, MessageSquare, Coins, AlertTriangle } from "lucide-react";
import { useI18n } from "@/i18n/context";
import type { AIProvider } from "@/api/ai-provider";

const providerColors: Record<string, string> = {
  gemini: "bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400",
  openai: "bg-green-100 dark:bg-green-900/40 text-green-600 dark:text-green-400",
  anthropic: "bg-orange-100 dark:bg-orange-900/40 text-orange-600 dark:text-orange-400",
  ollama: "bg-purple-100 dark:bg-purple-900/40 text-purple-600 dark:text-purple-400",
};

const providerNames: Record<string, string> = {
  gemini: "Google Gemini",
  openai: "OpenAI",
  anthropic: "Anthropic Claude",
  ollama: "Ollama",
};

const roleColors: Record<string, string> = {
  background: "bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300",
  chat: "bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300",
  analysis: "bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300",
};

interface ProviderCardProps {
  provider: AIProvider;
  isAdmin: boolean;
  onEdit: (provider: AIProvider) => void;
  onDelete: (provider: AIProvider) => void;
}

export function ProviderCard({
  provider,
  isAdmin,
  onEdit,
  onDelete,
}: ProviderCardProps) {
  const { t } = useI18n();
  const aiT = t.aiSettingsPage;

  return (
    <div
      className="relative rounded-xl border p-4 transition-all border-[var(--border-color)] hover:border-violet-300"
    >

      {/* Header */}
      <div className="flex items-start gap-3 mb-3">
        <div
          className={`w-10 h-10 rounded-lg flex items-center justify-center ${
            providerColors[provider.provider] || "bg-gray-100 text-gray-600"
          }`}
        >
          <Bot className="w-5 h-5" />
        </div>
        <div className="flex-1 min-w-0">
          <h4 className="font-medium text-default truncate">{provider.name}</h4>
          <p className="text-sm text-muted">
            {providerNames[provider.provider] || provider.provider}
          </p>
        </div>
      </div>

      {/* Info */}
      <div className="space-y-1 text-sm mb-3">
        <div className="flex items-center gap-2 text-muted">
          <Zap className="w-3.5 h-3.5" />
          <span className="truncate">{provider.model}</span>
        </div>
        {provider.description && (
          <p className="text-muted truncate">{provider.description}</p>
        )}
      </div>

      {/* Role Tags */}
      <div className="flex flex-wrap gap-1.5 mb-4">
        {provider.roles && provider.roles.length > 0 ? (
          provider.roles.map((role) => (
            <span
              key={role}
              className={`px-2 py-0.5 text-xs rounded-full font-medium ${roleColors[role] || "bg-gray-100 text-gray-600"}`}
            >
              {role === "background" ? aiT.roleBackground :
               role === "chat" ? aiT.roleChat :
               role === "analysis" ? aiT.roleAnalysis : role}
            </span>
          ))
        ) : (
          <span className="px-2 py-0.5 text-xs rounded-full bg-gray-100 dark:bg-gray-800 text-gray-400">
            {aiT.roleUnassigned}
          </span>
        )}
      </div>

      {/* Stats */}
      <div className="flex items-center gap-4 text-xs text-muted mb-4">
        <div className="flex items-center gap-1">
          <MessageSquare className="w-3 h-3" />
          {provider.totalRequests.toLocaleString()}
        </div>
        <div className="flex items-center gap-1">
          <Coins className="w-3 h-3" />
          {provider.totalTokens.toLocaleString()}
        </div>
      </div>

      {/* Last Error */}
      {provider.lastError && (
        <div className="flex items-start gap-2 mb-4 p-2 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
          <AlertTriangle className="w-3.5 h-3.5 text-red-500 flex-shrink-0 mt-0.5" />
          <span className="text-xs text-red-600 dark:text-red-400 break-all">{provider.lastError}</span>
        </div>
      )}

      {/* Actions */}
      {isAdmin && (
        <div className="flex items-center gap-2">
          <button
            onClick={() => onEdit(provider)}
            className="p-1.5 rounded-lg border border-[var(--border-color)] hover:bg-[var(--bg-secondary)] transition-colors"
            title={aiT.settings}
          >
            <Settings className="w-4 h-4 text-muted" />
          </button>
          <button
            onClick={() => onDelete(provider)}
            className="p-1.5 rounded-lg border border-red-200 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
            title={aiT.delete}
          >
            <Trash2 className="w-4 h-4 text-red-500" />
          </button>
        </div>
      )}
    </div>
  );
}
