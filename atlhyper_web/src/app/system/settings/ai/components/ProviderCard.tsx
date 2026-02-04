"use client";

import { Bot, Check, Settings, Trash2, Zap, MessageSquare, Coins } from "lucide-react";
import type { AIProvider } from "@/api/ai-provider";

const providerColors: Record<string, string> = {
  gemini: "bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400",
  openai: "bg-green-100 dark:bg-green-900/40 text-green-600 dark:text-green-400",
  anthropic: "bg-orange-100 dark:bg-orange-900/40 text-orange-600 dark:text-orange-400",
};

const providerNames: Record<string, string> = {
  gemini: "Google Gemini",
  openai: "OpenAI",
  anthropic: "Anthropic Claude",
};

interface ProviderCardProps {
  provider: AIProvider;
  isAdmin: boolean;
  onActivate: (provider: AIProvider) => void;
  onEdit: (provider: AIProvider) => void;
  onDelete: (provider: AIProvider) => void;
}

export function ProviderCard({
  provider,
  isAdmin,
  onActivate,
  onEdit,
  onDelete,
}: ProviderCardProps) {
  return (
    <div
      className={`relative rounded-xl border p-4 transition-all ${
        provider.is_active
          ? "border-green-500 bg-green-50/50 dark:bg-green-900/10"
          : "border-[var(--border-color)] hover:border-violet-300"
      }`}
    >
      {/* Active Badge */}
      {provider.is_active && (
        <div className="absolute -top-2 -right-2 px-2 py-0.5 rounded-full bg-green-500 text-white text-xs flex items-center gap-1">
          <Check className="w-3 h-3" />
          有効
        </div>
      )}

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
      <div className="space-y-1 text-sm mb-4">
        <div className="flex items-center gap-2 text-muted">
          <Zap className="w-3.5 h-3.5" />
          <span className="truncate">{provider.model}</span>
        </div>
        {provider.description && (
          <p className="text-muted truncate">{provider.description}</p>
        )}
      </div>

      {/* Stats */}
      <div className="flex items-center gap-4 text-xs text-muted mb-4">
        <div className="flex items-center gap-1">
          <MessageSquare className="w-3 h-3" />
          {provider.total_requests.toLocaleString()}
        </div>
        <div className="flex items-center gap-1">
          <Coins className="w-3 h-3" />
          {provider.total_tokens.toLocaleString()}
        </div>
      </div>

      {/* Actions */}
      {isAdmin && (
        <div className="flex items-center gap-2">
          {!provider.is_active && (
            <button
              onClick={() => onActivate(provider)}
              className="flex-1 px-3 py-1.5 text-sm rounded-lg border border-green-500 text-green-600 hover:bg-green-50 dark:hover:bg-green-900/20 transition-colors"
            >
              有効化
            </button>
          )}
          <button
            onClick={() => onEdit(provider)}
            className="p-1.5 rounded-lg border border-[var(--border-color)] hover:bg-[var(--bg-secondary)] transition-colors"
            title="設定"
          >
            <Settings className="w-4 h-4 text-muted" />
          </button>
          {!provider.is_active && (
            <button
              onClick={() => onDelete(provider)}
              className="p-1.5 rounded-lg border border-red-200 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
              title="削除"
            >
              <Trash2 className="w-4 h-4 text-red-500" />
            </button>
          )}
        </div>
      )}
    </div>
  );
}
