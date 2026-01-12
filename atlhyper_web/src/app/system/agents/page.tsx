"use client";

import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader } from "@/components/common";
import { Construction } from "lucide-react";

export default function AgentsPage() {
  const { t } = useI18n();

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title="Agents"
          description="监控各节点 Agent 连接状态"
        />

        <div className="bg-card rounded-xl border border-[var(--border-color)] p-12">
          <div className="flex flex-col items-center justify-center text-center">
            <Construction className="w-16 h-16 text-yellow-500 mb-4" />
            <h2 className="text-xl font-semibold text-default mb-2">
              功能设计开发中
            </h2>
            <p className="text-muted max-w-md">
              Agent 管理功能正在设计开发中，敬请期待。
            </p>
            <div className="mt-6 text-sm text-muted">
              <p>计划功能：</p>
              <ul className="mt-2 space-y-1 text-left">
                <li>• Agent 连接状态监控</li>
                <li>• Agent 版本管理</li>
                <li>• Agent 配置下发</li>
                <li>• Agent 健康检查</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}
