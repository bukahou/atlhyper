"use client";

import { useState, useEffect, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { getSlackConfig, updateSlackConfig, type SlackConfigResponse } from "@/api/config";
import { Bell, MessageSquare, CheckCircle, XCircle, Send, Save, Clock } from "lucide-react";
import { UserRole } from "@/types/auth";

export default function NotificationsPage() {
  const { t } = useI18n();
  const { user: currentUser, isAuthenticated, openLoginDialog } = useAuthStore();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  // 配置状态
  const [config, setConfig] = useState<SlackConfigResponse | null>(null);
  const [formData, setFormData] = useState({
    enable: false,
    webhook: "",
    intervalSec: 5,
  });

  const isAdmin = currentUser?.role === UserRole.ADMIN;

  const fetchConfig = useCallback(async () => {
    try {
      const res = await getSlackConfig();
      const cfg = res.data.data;
      setConfig(cfg);
      setFormData({
        enable: cfg.Enable === 1,
        webhook: cfg.Webhook || "",
        intervalSec: cfg.IntervalSec || 5,
      });
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载配置失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  const handleSave = async () => {
    if (!isAuthenticated) {
      openLoginDialog(() => handleSave());
      return;
    }
    if (!isAdmin) return;

    setSaving(true);
    setError("");
    setSuccess("");

    try {
      await updateSlackConfig({
        enable: formData.enable ? 1 : 0,
        webhook: formData.webhook,
        intervalSec: formData.intervalSec,
      });
      setSuccess("配置已保存");
      fetchConfig();
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存失败");
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    if (!formData.webhook) {
      setError("请先填写 Webhook URL");
      return;
    }

    setTesting(true);
    setError("");
    setSuccess("");

    try {
      // 调用预览接口测试发送
      const res = await fetch(`/uiapi/alert/slack/preview?webhook=${encodeURIComponent(formData.webhook)}`);
      if (!res.ok) {
        throw new Error("测试发送失败");
      }
      setSuccess("测试消息已发送，请检查 Slack 频道");
    } catch (err) {
      setError(err instanceof Error ? err.message : "测试发送失败");
    } finally {
      setTesting(false);
    }
  };

  const formatDate = (dateStr: string) => {
    if (!dateStr || dateStr === "0001-01-01T00:00:00Z") return "-";
    return new Date(dateStr).toLocaleString();
  };

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title="通知配置" description="配置告警通知渠道" />

        {loading ? (
          <div className="py-12">
            <LoadingSpinner />
          </div>
        ) : (
          <div className="max-w-2xl space-y-6">
            {/* Slack 配置卡片 */}
            <div className="bg-card rounded-xl border border-[var(--border-color)] p-6">
              <div className="flex items-center gap-3 mb-6">
                <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
                  <MessageSquare className="w-5 h-5 text-purple-600 dark:text-purple-400" />
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-default">Slack 通知</h3>
                  <p className="text-sm text-muted">将告警消息发送到 Slack 频道</p>
                </div>
                <div className="ml-auto">
                  {config?.Enable === 1 ? (
                    <span className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
                      <CheckCircle className="w-3 h-3" />
                      已启用
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium rounded-full bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400">
                      <XCircle className="w-3 h-3" />
                      未启用
                    </span>
                  )}
                </div>
              </div>

              <div className="space-y-4">
                {/* 启用开关 */}
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium text-default">启用 Slack 通知</label>
                  <button
                    onClick={() => setFormData({ ...formData, enable: !formData.enable })}
                    disabled={!isAdmin}
                    className={`relative w-12 h-6 rounded-full transition-colors ${
                      formData.enable ? "bg-primary" : "bg-gray-300 dark:bg-gray-600"
                    } ${!isAdmin ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}`}
                  >
                    <div
                      className={`absolute top-1 w-4 h-4 bg-white rounded-full transition-transform ${
                        formData.enable ? "translate-x-7" : "translate-x-1"
                      }`}
                    />
                  </button>
                </div>

                {/* Webhook URL */}
                <div>
                  <label className="block text-sm font-medium text-muted mb-1">
                    Webhook URL
                  </label>
                  <input
                    type="url"
                    placeholder="https://hooks.slack.com/services/..."
                    value={formData.webhook}
                    onChange={(e) => setFormData({ ...formData, webhook: e.target.value })}
                    disabled={!isAdmin}
                    className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none disabled:opacity-50 disabled:cursor-not-allowed font-mono text-sm"
                  />
                  <p className="text-xs text-muted mt-1">
                    在 Slack 中创建 Incoming Webhook 并粘贴 URL
                  </p>
                </div>

                {/* 发送间隔 */}
                <div>
                  <label className="block text-sm font-medium text-muted mb-1">
                    发送间隔（秒）
                  </label>
                  <div className="flex items-center gap-2">
                    <input
                      type="number"
                      min={1}
                      max={3600}
                      value={formData.intervalSec}
                      onChange={(e) => setFormData({ ...formData, intervalSec: Number(e.target.value) || 5 })}
                      disabled={!isAdmin}
                      className="w-24 px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none disabled:opacity-50 disabled:cursor-not-allowed"
                    />
                    <span className="text-sm text-muted">秒</span>
                  </div>
                  <p className="text-xs text-muted mt-1">
                    相同类型的告警在此间隔内不会重复发送
                  </p>
                </div>

                {/* 最后更新时间 */}
                {config && (
                  <div className="flex items-center gap-2 text-sm text-muted pt-2 border-t border-[var(--border-color)]">
                    <Clock className="w-4 h-4" />
                    <span>最后更新: {formatDate(config.UpdatedAt)}</span>
                  </div>
                )}

                {/* 错误/成功提示 */}
                {error && (
                  <div className="p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg text-sm">
                    {error}
                  </div>
                )}
                {success && (
                  <div className="p-3 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 rounded-lg text-sm">
                    {success}
                  </div>
                )}

                {/* 操作按钮 */}
                {isAdmin && (
                  <div className="flex gap-3 pt-2">
                    <button
                      onClick={handleTest}
                      disabled={testing || !formData.webhook}
                      className="flex items-center gap-2 px-4 py-2 border border-[var(--border-color)] rounded-lg hover-bg disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <Send className="w-4 h-4" />
                      {testing ? "发送中..." : "测试发送"}
                    </button>
                    <button
                      onClick={handleSave}
                      disabled={saving}
                      className="flex items-center gap-2 px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg disabled:opacity-50"
                    >
                      <Save className="w-4 h-4" />
                      {saving ? "保存中..." : "保存配置"}
                    </button>
                  </div>
                )}

                {!isAdmin && (
                  <div className="p-3 bg-yellow-50 dark:bg-yellow-900/20 text-yellow-700 dark:text-yellow-400 rounded-lg text-sm">
                    需要管理员权限才能修改配置
                  </div>
                )}
              </div>
            </div>

            {/* 其他通知渠道（占位） */}
            <div className="bg-card rounded-xl border border-[var(--border-color)] p-6 opacity-60">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-gray-100 dark:bg-gray-800 rounded-lg">
                  <Bell className="w-5 h-5 text-gray-500" />
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-default">更多通知渠道</h3>
                  <p className="text-sm text-muted">邮件、Webhook、企业微信等 - 即将推出</p>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}
