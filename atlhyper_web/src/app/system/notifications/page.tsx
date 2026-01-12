"use client";

import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader } from "@/components/common";
import { Construction } from "lucide-react";

export default function NotificationsPage() {
  const { t } = useI18n();

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title="通知配置" description="配置告警通知渠道" />

        <div className="bg-card rounded-xl border border-[var(--border-color)] p-12">
          <div className="flex flex-col items-center justify-center text-center">
            <Construction className="w-16 h-16 text-yellow-500 mb-4" />
            <h2 className="text-xl font-semibold text-default mb-2">
              功能设计开发中
            </h2>
            <p className="text-muted max-w-md">
              通知配置功能正在设计开发中，敬请期待。
            </p>
            <div className="mt-6 text-sm text-muted">
              <p>计划功能：</p>
              <ul className="mt-2 space-y-1 text-left">
                <li>• Slack 通知集成</li>
                <li>• 邮件告警通知</li>
                <li>• Webhook 自定义通知</li>
                <li>• 企业微信/钉钉集成</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}
