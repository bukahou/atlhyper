"use client";

import { useEffect, useState, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { toast } from "@/components/common/Toast";
import { useAuthStore } from "@/store/authStore";
import { AlertTriangle } from "lucide-react";

import { SlackCard, EmailCard } from "./components";
import {
  listChannels,
  updateSlack,
  updateEmail,
  testChannel,
  mockChannels,
  type NotifyChannel,
  type SlackConfig,
  type EmailConfig,
  type EmailUpdateData,
} from "@/api/notify";

export default function NotificationsPage() {
  const { t } = useI18n();
  const { user, isAuthenticated } = useAuthStore();

  // 权限判断
  const isGuest = !isAuthenticated;
  const isAdmin = user?.role === 3;

  // 状态
  const [loading, setLoading] = useState(true);
  const [channels, setChannels] = useState<NotifyChannel[]>([]);

  // 加载数据
  useEffect(() => {
    if (isGuest) {
      // Guest 使用 mock 数据
      setChannels(mockChannels);
      setLoading(false);
      return;
    }

    // 加载真实数据
    listChannels()
      .then((res) => {
        setChannels(res.data.channels || []);
      })
      .catch((err) => {
        console.error("Failed to load channels:", err);
        toast.error("加载通知配置失败");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [isGuest]);

  // 获取渠道配置
  const getSlackChannel = useCallback(() => {
    return channels.find((ch) => ch.type === "slack");
  }, [channels]);

  const getEmailChannel = useCallback(() => {
    return channels.find((ch) => ch.type === "email");
  }, [channels]);

  // 保存 Slack 配置
  const handleSaveSlack = useCallback(
    async (data: { enabled?: boolean; webhook_url?: string }) => {
      try {
        const res = await updateSlack(data);
        // 更新本地状态
        setChannels((prev) =>
          prev.map((ch) => (ch.type === "slack" ? res.data : ch))
        );
        toast.success("Slack 配置已保存");
      } catch (err) {
        console.error("Failed to save Slack config:", err);
        toast.error("保存失败");
        throw err;
      }
    },
    []
  );

  // 保存 Email 配置
  const handleSaveEmail = useCallback(async (data: EmailUpdateData) => {
    try {
      const res = await updateEmail(data);
      // 更新本地状态
      setChannels((prev) =>
        prev.map((ch) => (ch.type === "email" ? res.data : ch))
      );
      toast.success("邮件配置已保存");
    } catch (err) {
      console.error("Failed to save Email config:", err);
      toast.error("保存失败");
      throw err;
    }
  }, []);

  // 测试 Slack
  const handleTestSlack = useCallback(async () => {
    const result = await testChannel("slack");
    if (result.success) {
      toast.success(result.message);
    } else {
      toast.error(result.message);
    }
    return result;
  }, []);

  // 测试 Email
  const handleTestEmail = useCallback(async () => {
    const result = await testChannel("email");
    if (result.success) {
      toast.success(result.message);
    } else {
      toast.error(result.message);
    }
    return result;
  }, []);

  // 渲染
  const slackChannel = getSlackChannel();
  const emailChannel = getEmailChannel();

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title={t.nav.notifications}
          description={t.notifications?.pageDescription || "配置告警通知渠道"}
        />

        {/* Guest 提示 */}
        {isGuest && (
          <div className="flex items-center gap-3 p-4 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800">
            <AlertTriangle className="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0" />
            <p className="text-sm text-yellow-800 dark:text-yellow-300">
              演示模式 - 显示的是示例数据。请登录后查看真实配置。
            </p>
          </div>
        )}

        {/* 非 Admin 提示 */}
        {!isGuest && !isAdmin && (
          <div className="flex items-center gap-3 p-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
            <AlertTriangle className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0" />
            <p className="text-sm text-blue-800 dark:text-blue-300">
              您只有查看权限。如需修改配置，请联系管理员。
            </p>
          </div>
        )}

        {/* 加载状态 */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <LoadingSpinner />
          </div>
        ) : (
          <div className="grid gap-6 lg:grid-cols-2">
            {/* Slack 卡片 */}
            <SlackCard
              config={(slackChannel?.config as SlackConfig) || { webhook_url: "" }}
              enabled={slackChannel?.enabled || false}
              effectiveEnabled={slackChannel?.effective_enabled || false}
              validationErrors={slackChannel?.validation_errors || []}
              readOnly={!isAdmin}
              onSave={handleSaveSlack}
              onTest={handleTestSlack}
            />

            {/* Email 卡片 */}
            <EmailCard
              config={
                (emailChannel?.config as EmailConfig) || {
                  smtp_host: "",
                  smtp_port: 587,
                  smtp_user: "",
                  from_address: "",
                  to_addresses: [],
                }
              }
              enabled={emailChannel?.enabled || false}
              effectiveEnabled={emailChannel?.effective_enabled || false}
              validationErrors={emailChannel?.validation_errors || []}
              readOnly={!isAdmin}
              onSave={handleSaveEmail}
              onTest={handleTestEmail}
            />
          </div>
        )}
      </div>
    </Layout>
  );
}
