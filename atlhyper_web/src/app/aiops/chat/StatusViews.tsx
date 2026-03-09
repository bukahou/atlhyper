"use client";

import { Loader2, Settings } from "lucide-react";
import { Layout } from "@/components/layout/Layout";
import { StatusPage } from "@/components/common";
import { useI18n } from "@/i18n/context";
import type { AIConfigStatus } from "./types";

interface StatusViewsProps {
  aiStatus: AIConfigStatus;
  goToSettings: () => void;
}

/**
 * AI 聊天页面的状态视图（加载中 / 未启用 / 未配置）。
 * 当 aiStatus 为 "ready" 时返回 null，由调用方渲染正常聊天界面。
 */
export function StatusViews({ aiStatus, goToSettings }: StatusViewsProps) {
  const { t } = useI18n();
  const aiChatT = t.aiChatPage;

  // 加载中（仅登录后）
  if (aiStatus === "loading") {
    return (
      <Layout>
        <div className="-m-6 h-[calc(100vh-3.5rem)]">
          <StatusPage
            icon={Loader2}
            title={aiChatT.loading}
            description={aiChatT.checkingConfig}
          />
        </div>
      </Layout>
    );
  }

  // 未配置 AI 提供商
  if (aiStatus === "not_configured") {
    return (
      <Layout>
        <div className="-m-6 h-[calc(100vh-3.5rem)]">
          <StatusPage
            icon={Settings}
            title={aiChatT.notConfigured}
            description={aiChatT.notConfiguredDesc}
            action={{ label: aiChatT.goToSettings, onClick: goToSettings }}
          />
        </div>
      </Layout>
    );
  }

  // Chat 角色未分配 Provider
  if (aiStatus === "chat_not_assigned") {
    return (
      <Layout>
        <div className="-m-6 h-[calc(100vh-3.5rem)]">
          <StatusPage
            icon={Settings}
            title={aiChatT.chatNotAssigned}
            description={aiChatT.chatNotAssignedDesc}
            action={{ label: aiChatT.goToSettings, onClick: goToSettings }}
          />
        </div>
      </Layout>
    );
  }

  // aiStatus === "ready" 时不渲染
  return null;
}
