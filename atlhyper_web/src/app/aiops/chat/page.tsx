"use client";

import { Eye } from "lucide-react";
import { Layout } from "@/components/layout/Layout";
import { ChatPanel } from "@/components/ai/ChatPanel";
import { useI18n } from "@/i18n/context";
import { StatusViews } from "./StatusViews";
import { useAIChat } from "./hooks";

export default function AIChatPage() {
  const { t } = useI18n();
  const aiChatT = t.aiChatPage;

  const {
    aiStatus,
    isDemo,
    conversations,
    currentConvId,
    messages,
    streaming,
    streamSegments,
    currentStats,
    currentClusterId,
    handleSelect,
    handleNew,
    handleDelete,
    handleSend,
    handleStop,
    goToSettings,
    handleDemoAction,
    openLoginDialog,
  } = useAIChat();

  // 非 ready 状态：显示加载中 / 未启用 / 未配置
  if (aiStatus !== "ready") {
    return <StatusViews aiStatus={aiStatus} goToSettings={goToSettings} />;
  }

  // AI 配置就绪，显示聊天界面
  return (
    <Layout>
      {/* 负边距抵消 Layout 的 p-6，全屏聊天布局 */}
      <div className="-m-6 h-[calc(100vh-3.5rem)] flex flex-col relative">
        {/* 演示模式横幅 */}
        {isDemo && (
          <div className="flex items-center justify-between gap-3 px-4 py-2 bg-amber-50 dark:bg-amber-900/20 border-b border-amber-200 dark:border-amber-800">
            <div className="flex items-center gap-2">
              <Eye className="w-4 h-4 text-amber-600 dark:text-amber-400" />
              <span className="text-sm text-amber-800 dark:text-amber-300">
                {aiChatT.demoMode} - {aiChatT.demoModeDesc}
              </span>
            </div>
            <button
              onClick={() => openLoginDialog()}
              className="px-3 py-1 text-xs font-medium rounded-lg bg-amber-600 text-white hover:bg-amber-700 transition-colors"
            >
              {aiChatT.login}
            </button>
          </div>
        )}
        <div className="flex-1 flex">
          <ChatPanel
            messages={messages}
            streaming={streaming}
            streamSegments={streamSegments}
            conversations={conversations}
            currentConvId={currentConvId}
            clusterId={currentClusterId}
            currentStats={currentStats}
            onSelectConv={isDemo ? () => {} : handleSelect}
            onNewConv={isDemo ? handleDemoAction : handleNew}
            onDeleteConv={isDemo ? handleDemoAction : handleDelete}
            onSend={isDemo ? handleDemoAction : handleSend}
            onStop={handleStop}
            readOnly={isDemo}
          />
        </div>
      </div>
    </Layout>
  );
}
