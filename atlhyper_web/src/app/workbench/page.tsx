"use client";

import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { PageHeader } from "@/components/common";
import { Terminal, Settings, Bell, Database, type LucideIcon } from "lucide-react";

interface Tool {
  icon: LucideIcon;
  title: string;
  description: string;
  requiresAuth: boolean;
}

const tools: Tool[] = [
  { icon: Terminal, title: "kubectl 终端", description: "在线执行 kubectl 命令", requiresAuth: true },
  { icon: Settings, title: "Slack 配置", description: "配置告警通知 Webhook", requiresAuth: true },
  { icon: Bell, title: "告警规则", description: "管理告警触发规则", requiresAuth: true },
  { icon: Database, title: "数据导出", description: "导出监控数据", requiresAuth: true },
];

function ToolCard({ tool, onClick }: { tool: Tool; onClick: () => void }) {
  const Icon = tool.icon;
  return (
    <button onClick={onClick} className="bg-card rounded-xl border border-[var(--border-color)] p-6 text-left hover:border-primary hover:shadow-lg transition-all group">
      <div className="p-3 bg-primary/10 rounded-lg w-fit mb-4 group-hover:bg-primary/20 transition-colors">
        <Icon className="w-6 h-6 text-primary" />
      </div>
      <h3 className="font-semibold text-default mb-1">{tool.title}</h3>
      <p className="text-sm text-muted">{tool.description}</p>
      {tool.requiresAuth && <p className="text-xs text-primary mt-2">需要登录</p>}
    </button>
  );
}

export default function WorkbenchPage() {
  const { t } = useI18n();
  const { isAuthenticated, openLoginDialog } = useAuthStore();

  const handleToolClick = (tool: Tool) => {
    const action = () => console.log("Open tool:", tool.title);
    if (tool.requiresAuth && !isAuthenticated) {
      openLoginDialog(action);
    } else {
      action();
    }
  };

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.workbench} description="运维工具集合" />

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {tools.map((tool) => (
            <ToolCard key={tool.title} tool={tool} onClick={() => handleToolClick(tool)} />
          ))}
        </div>

        <div className="bg-card rounded-xl border border-[var(--border-color)] p-6">
          <h3 className="text-lg font-semibold text-default mb-4">快速操作</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {["刷新集群信息", "清理已完成 Pod", "同步 ConfigMap", "集群健康检查"].map((label) => (
              <button
                key={label}
                onClick={() => handleToolClick({ title: label, requiresAuth: true } as Tool)}
                className="px-4 py-2 text-sm bg-[var(--background)] rounded-lg hover-bg transition-colors"
              >
                {label}
              </button>
            ))}
          </div>
        </div>
      </div>
    </Layout>
  );
}
