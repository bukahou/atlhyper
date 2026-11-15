"use client";

import { useState } from "react";
import { Layout } from "@/components/layout/Layout";
import { DemoBanner, LoadingSpinner, PageHeader } from "@/components/common";
import { useCan } from "@/hooks/useCan";
import { useManualRefetch } from "@/hooks/useManualRefetch";
import { useSessionReady } from "@/hooks/useSessionReady";
import { useI18n } from "@/i18n/context";
import { notifyKeys } from "@/queries/keys";
import { AlertsTab, DeliveriesTab, PolicyTab, ReceiversTab, RulesTab, SilencesTab } from "./components";
import { NotifyTabBar, type NotifyTabKey } from "./components/NotifyTabBar";

export default function NotificationsPage() {
  const { t } = useI18n();
  // 通知设置是纯管理功能页：权限加载完才决定显示真实数据还是演示数据；无 notify.manage 时整页演示数据
  const sessionReady = useSessionReady();
  const canManageNotifications = useCan()("notify.manage");
  const isDemo = sessionReady && !canManageNotifications;

  const [activeTab, setActiveTab] = useState<NotifyTabKey>("alerts");
  const { refetch: refreshNotifications } = useManualRefetch(notifyKeys.all);

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title={t.nav.notifications}
          description={t.notifications.pageDescription}
          // 演示数据没有可刷新的东西
          onRefresh={canManageNotifications ? refreshNotifications : undefined}
        />

        {isDemo && <DemoBanner hint={t.notifications.demoHint} />}

        <NotifyTabBar activeTab={activeTab} onSelect={setActiveTab} labels={t.notifications.tabs} />

        {!sessionReady ? (
          <div className="flex items-center justify-center py-12">
            <LoadingSpinner />
          </div>
        ) : (
          // 一次只挂载当前页签：没在看的页签不取数、不轮询
          <div role="tabpanel">
            {activeTab === "alerts" && <AlertsTab isDemo={isDemo} />}
            {activeTab === "deliveries" && <DeliveriesTab isDemo={isDemo} />}
            {activeTab === "receivers" && <ReceiversTab isDemo={isDemo} />}
            {activeTab === "rules" && <RulesTab isDemo={isDemo} />}
            {activeTab === "silences" && <SilencesTab isDemo={isDemo} />}
            {activeTab === "policy" && <PolicyTab isDemo={isDemo} />}
          </div>
        )}
      </div>
    </Layout>
  );
}
