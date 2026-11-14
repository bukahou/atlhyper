"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner, DemoBanner, QueryView } from "@/components/common";
import { toast } from "@/components/common/Toast";

import { SlackCard, EmailCard } from "./components";
import type { ChannelType, EmailConfig, EmailUpdateData, NotifyChannel } from "@/api/notify";
import { useCan } from "@/hooks/useCan";
import { useSessionReady } from "@/hooks/useSessionReady";
import { mockChannels } from "@/mock/notify";
import {
  notifyChannelsQuery,
  testNotifyChannelMutation,
  updateEmailChannelMutation,
  updateSlackChannelMutation,
} from "@/queries/notify";

/** 还没配置过邮件渠道时表单用的空配置 */
const EMPTY_EMAIL_CONFIG: EmailConfig = {
  smtpHost: "",
  smtpPort: 587,
  smtpUser: "",
  fromAddress: "",
  toAddresses: [],
};

export default function NotificationsPage() {
  const { t } = useI18n();
  // 通知设置是纯管理功能页：权限加载完才决定显示真实配置还是演示数据（Operator 即可查看和修改）
  const sessionReady = useSessionReady();
  const canManageNotifications = useCan()("notify.manage");
  const isDemo = sessionReady && !canManageNotifications;

  const channelsQuery = useQuery(notifyChannelsQuery(canManageNotifications));
  // 保存成功后工厂把接口带回的渠道换进列表，这里只管提示
  const saveSlack = useMutation(updateSlackChannelMutation());
  const saveEmail = useMutation(updateEmailChannelMutation());
  const testChannel = useMutation(testNotifyChannelMutation());

  // 保存 Slack 配置（失败时抛给卡片，卡片据此保留用户的输入）
  const handleSaveSlack = async (data: { enabled?: boolean; webhookUrl?: string }) => {
    try {
      await saveSlack.mutateAsync(data);
      toast.success(t.notifications.slackSaved);
    } catch (err) {
      console.error("Failed to save Slack config:", err);
      toast.error(t.notifications.saveFailed);
      throw err;
    }
  };

  // 保存 Email 配置
  const handleSaveEmail = async (data: EmailUpdateData) => {
    try {
      await saveEmail.mutateAsync(data);
      toast.success(t.notifications.emailSaved);
    } catch (err) {
      console.error("Failed to save Email config:", err);
      toast.error(t.notifications.saveFailed);
      throw err;
    }
  };

  // 测试渠道：按已保存的配置发一条测试消息
  const handleTestChannel = async (channelType: ChannelType) => {
    const result = await testChannel.mutateAsync(channelType);
    if (result.success) {
      toast.success(result.message);
    } else {
      toast.error(result.message);
    }
    return result;
  };

  const renderChannelCards = (channels: NotifyChannel[]) => {
    const slackChannel = channels.find((channel) => channel.type === "slack");
    const emailChannel = channels.find((channel) => channel.type === "email");
    return (
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Slack 卡片 */}
        <SlackCard
          webhookConfigured={slackChannel?.configuredSecrets?.includes("webhookUrl") || false}
          enabled={slackChannel?.enabled || false}
          effectiveEnabled={slackChannel?.effectiveEnabled || false}
          validationErrors={slackChannel?.validationErrors || []}
          readOnly={isDemo}
          onSave={handleSaveSlack}
          onTest={() => handleTestChannel("slack")}
        />

        {/* Email 卡片 */}
        <EmailCard
          config={(emailChannel?.config as EmailConfig) || EMPTY_EMAIL_CONFIG}
          enabled={emailChannel?.enabled || false}
          effectiveEnabled={emailChannel?.effectiveEnabled || false}
          validationErrors={emailChannel?.validationErrors || []}
          readOnly={isDemo}
          onSave={handleSaveEmail}
          onTest={() => handleTestChannel("email")}
        />
      </div>
    );
  };

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.notifications} description={t.notifications.pageDescription} />

        {isDemo && <DemoBanner hint={t.common.demoModeHint} />}

        {!sessionReady ? (
          <div className="flex items-center justify-center py-12">
            <LoadingSpinner />
          </div>
        ) : isDemo ? (
          renderChannelCards(mockChannels)
        ) : (
          <QueryView query={channelsQuery}>{renderChannelCards}</QueryView>
        )}
      </div>
    </Layout>
  );
}
