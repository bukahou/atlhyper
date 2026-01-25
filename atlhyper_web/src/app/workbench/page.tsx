"use client";

import Link from "next/link";
import { Bot, Server, Bell, Settings, ArrowRight } from "lucide-react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";

interface QuickActionCardProps {
  icon: typeof Bot;
  title: string;
  description: string;
  href: string;
  color: string;
}

function QuickActionCard({ icon: Icon, title, description, href, color }: QuickActionCardProps) {
  return (
    <Link
      href={href}
      className="group flex flex-col p-5 rounded-xl border border-[var(--border-color)] bg-card hover:border-primary/50 hover:shadow-lg transition-all duration-200"
    >
      <div className={`w-10 h-10 rounded-lg flex items-center justify-center mb-4 ${color}`}>
        <Icon className="w-5 h-5" />
      </div>
      <h3 className="text-base font-medium text-default mb-1 group-hover:text-primary transition-colors">
        {title}
      </h3>
      <p className="text-sm text-muted flex-1">{description}</p>
      <div className="flex items-center gap-1 mt-4 text-sm text-primary opacity-0 group-hover:opacity-100 transition-opacity">
        <span>进入</span>
        <ArrowRight className="w-4 h-4" />
      </div>
    </Link>
  );
}

export default function WorkbenchPage() {
  const { t } = useI18n();

  const quickActions: QuickActionCardProps[] = [
    {
      icon: Bot,
      title: t.nav.ai,
      description: "使用 AI 助手进行集群诊断、资源查询和问题排查",
      href: "/workbench/ai",
      color: "bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400",
    },
    {
      icon: Server,
      title: t.nav.clusters,
      description: "管理集群连接配置，查看集群状态和资源概览",
      href: "/system/clusters",
      color: "bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400",
    },
    {
      icon: Server,
      title: t.nav.agents,
      description: "查看 Agent 连接状态、版本信息和心跳监控",
      href: "/system/agents",
      color: "bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400",
    },
    {
      icon: Bell,
      title: t.nav.notifications,
      description: "配置 Slack、邮件等通知渠道和告警规则",
      href: "/system/notifications",
      color: "bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400",
    },
  ];

  return (
    <Layout>
      <div className="max-w-5xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-2xl font-semibold text-default mb-2">{t.nav.workbench}</h1>
          <p className="text-muted">{t.workbench.pageDescription}</p>
        </div>

        {/* Quick Actions Grid */}
        <div className="mb-8">
          <h2 className="text-sm font-medium text-secondary uppercase tracking-wide mb-4">
            {t.workbench.quickActions}
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            {quickActions.map((action) => (
              <QuickActionCard key={action.href} {...action} />
            ))}
          </div>
        </div>

        {/* Coming Soon Section */}
        <div className="rounded-xl border border-dashed border-[var(--border-color)] bg-[var(--hover-bg)] p-8 text-center">
          <Settings className="w-10 h-10 text-muted mx-auto mb-3" />
          <h3 className="text-base font-medium text-default mb-1">更多功能开发中</h3>
          <p className="text-sm text-muted">
            批量运维、脚本执行、定时任务等功能正在规划中...
          </p>
        </div>
      </div>
    </Layout>
  );
}
