"use client";

import { useState, useEffect } from "react";
import { MessageSquare, Eye, EyeOff, Loader2, AlertCircle } from "lucide-react";
import type { SlackConfig } from "@/api/notify";

interface SlackCardProps {
  config: SlackConfig;
  enabled: boolean;
  effectiveEnabled: boolean;
  validationErrors: string[];
  readOnly: boolean;
  onSave: (data: { enabled?: boolean; webhook_url?: string }) => Promise<void>;
  onTest: () => Promise<{ success: boolean; message: string }>;
}

export function SlackCard({
  config,
  enabled,
  effectiveEnabled,
  validationErrors,
  readOnly,
  onSave,
  onTest,
}: SlackCardProps) {
  const [localEnabled, setLocalEnabled] = useState(enabled);
  const [webhookUrl, setWebhookUrl] = useState(config.webhook_url || "");
  const [showWebhook, setShowWebhook] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);

  // 同步外部状态
  useEffect(() => {
    setLocalEnabled(enabled);
    setWebhookUrl(config.webhook_url || "");
  }, [enabled, config.webhook_url]);

  // 检测变化
  useEffect(() => {
    const changed = localEnabled !== enabled || webhookUrl !== (config.webhook_url || "");
    setHasChanges(changed);
  }, [localEnabled, webhookUrl, enabled, config.webhook_url]);

  const handleToggle = async () => {
    if (readOnly) return;
    const newEnabled = !localEnabled;
    setLocalEnabled(newEnabled);
    // 立即保存开关状态
    setSaving(true);
    try {
      await onSave({ enabled: newEnabled });
    } finally {
      setSaving(false);
    }
  };

  const handleSave = async () => {
    if (readOnly || !hasChanges) return;
    setSaving(true);
    try {
      await onSave({ enabled: localEnabled, webhook_url: webhookUrl });
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    setTesting(true);
    try {
      await onTest();
    } finally {
      setTesting(false);
    }
  };

  // 隐藏 webhook URL 中间部分
  const maskWebhook = (url: string) => {
    if (!url || url.length < 40) return url;
    return url.slice(0, 35) + "****" + url.slice(-10);
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      {/* 头部 */}
      <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)]">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-purple-100 dark:bg-purple-900/40 flex items-center justify-center">
            <MessageSquare className="w-5 h-5 text-purple-600 dark:text-purple-400" />
          </div>
          <div>
            <h3 className="font-medium text-default">Slack 通知</h3>
            <p className="text-sm text-muted">
              {effectiveEnabled ? (
                <span className="text-green-600">已启用</span>
              ) : localEnabled ? (
                <span className="text-yellow-600">配置不完整</span>
              ) : (
                "已禁用"
              )}
            </p>
          </div>
        </div>

        {/* 开关 */}
        <button
          onClick={handleToggle}
          disabled={readOnly || saving}
          className={`relative w-12 h-6 rounded-full transition-colors ${
            localEnabled
              ? "bg-green-500"
              : "bg-gray-300 dark:bg-gray-600"
          } ${readOnly ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}`}
        >
          <span
            className={`absolute top-1 left-1 w-4 h-4 rounded-full bg-white transition-transform ${
              localEnabled ? "translate-x-6" : "translate-x-0"
            }`}
          />
        </button>
      </div>

      {/* 校验错误提示 */}
      {validationErrors.length > 0 && (
        <div className="mx-6 mt-4 p-3 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800">
          <div className="flex items-start gap-2">
            <AlertCircle className="w-4 h-4 text-yellow-600 mt-0.5 flex-shrink-0" />
            <div className="text-sm text-yellow-700 dark:text-yellow-400">
              <p className="font-medium mb-1">配置不完整</p>
              <ul className="list-disc list-inside space-y-0.5">
                {validationErrors.map((err, i) => (
                  <li key={i}>{err}</li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      )}

      {/* 内容 */}
      <div className="px-6 py-4 space-y-4">
        {/* Webhook URL */}
        <div>
          <label className="block text-sm font-medium text-default mb-2">
            Webhook URL <span className="text-red-500">*</span>
          </label>
          <div className="relative">
            <input
              type={showWebhook ? "text" : "password"}
              value={webhookUrl}
              onChange={(e) => setWebhookUrl(e.target.value)}
              placeholder="https://hooks.slack.com/services/..."
              disabled={readOnly}
              className={`w-full px-3 py-2 pr-10 rounded-lg border text-sm font-mono
                bg-[var(--bg-primary)] text-default
                border-[var(--border-color)]
                focus:outline-none focus:ring-2 focus:ring-purple-500/50
                ${readOnly ? "opacity-60 cursor-not-allowed" : ""}`}
            />
            <button
              type="button"
              onClick={() => setShowWebhook(!showWebhook)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted hover:text-default"
            >
              {showWebhook ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            </button>
          </div>
          {readOnly && webhookUrl && (
            <p className="mt-1 text-xs text-muted font-mono">
              {maskWebhook(webhookUrl)}
            </p>
          )}
        </div>

        {/* 提示 */}
        {!readOnly && (
          <p className="text-xs text-muted">
            在 Slack 中创建 Incoming Webhook 应用，并将 Webhook URL 粘贴到此处。
          </p>
        )}
      </div>

      {/* 操作按钮 */}
      {!readOnly && (
        <div className="flex justify-end gap-3 px-6 py-4 border-t border-[var(--border-color)] bg-[var(--bg-secondary)]">
          <button
            onClick={handleTest}
            disabled={testing || !effectiveEnabled}
            className="px-4 py-2 text-sm rounded-lg border border-[var(--border-color)]
              bg-[var(--bg-primary)] text-default
              hover:bg-[var(--bg-secondary)]
              disabled:opacity-50 disabled:cursor-not-allowed
              transition-colors flex items-center gap-2"
          >
            {testing && <Loader2 className="w-4 h-4 animate-spin" />}
            测试
          </button>
          <button
            onClick={handleSave}
            disabled={saving || !hasChanges}
            className="px-4 py-2 text-sm rounded-lg
              bg-purple-600 text-white
              hover:bg-purple-700
              disabled:opacity-50 disabled:cursor-not-allowed
              transition-colors flex items-center gap-2"
          >
            {saving && <Loader2 className="w-4 h-4 animate-spin" />}
            保存
          </button>
        </div>
      )}
    </div>
  );
}
