"use client";

import { useEffect, useState, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { toast } from "@/components/common/Toast";
import { useAuthStore } from "@/store/authStore";
import { AlertTriangle, Eye } from "lucide-react";
import { UserRole } from "@/types/auth";

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

  // 权限判断：Operator 即可查看和修改
  const hasPermission = isAuthenticated && user && user.role >= UserRole.OPERATOR;
  const isDemo = !hasPermission;

  // 状态
  const [loading, setLoading] = useState(true);
  const [channels, setChannels] = useState<NotifyChannel[]>([]);

  // 加载数据
  useEffect(() => {
    if (isDemo) {
      // 无权限时使用 mock 数据
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
        toast.error(t.notifications.loadFailed);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [isDemo]);

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
        toast.success(t.notifications.slackSaved);
      } catch (err) {
        console.error("Failed to save Slack config:", err);
        toast.error(t.notifications.saveFailed);
        throw err;
      }
    },
    [t]
  );

  // 保存 Email 配置
  const handleSaveEmail = useCallback(async (data: EmailUpdateData) => {
    try {
      const res = await updateEmail(data);
      // 更新本地状态
      setChannels((prev) =>
        prev.map((ch) => (ch.type === "email" ? res.data : ch))
      );
      toast.success(t.notifications.emailSaved);
    } catch (err) {
      console.error("Failed to save Email config:", err);
      toast.error(t.notifications.saveFailed);
      throw err;
    }
  }, [t]);

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
          description={t.notifications.pageDescription}
        />

        {/* 演示模式提示 */}
        {isDemo && (
          <div className="flex items-center gap-3 p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-xl">
            <Eye className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0" />
            <div>
              <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
                {t.locale === "zh" ? "演示模式" : "デモモード"}
              </p>
              <p className="text-xs text-amber-600 dark:text-amber-400">
                {t.locale === "zh"
                  ? "当前展示的是示例数据。登录并获得 Operator 权限后可查看真实配置。"
                  : "サンプルデータを表示中です。Operator 権限でログインすると実際の設定を確認できます。"}
              </p>
            </div>
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
              readOnly={isDemo}
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
              readOnly={isDemo}
              onSave={handleSaveEmail}
              onTest={handleTestEmail}
            />
          </div>
        )}
      </div>
    </Layout>
  );
}
