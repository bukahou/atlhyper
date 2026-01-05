"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Wrench,
  Box,
  Server,
  Layers,
  Network,
  FolderTree,
  Globe,
  AlertTriangle,
  Activity,
  FileText,
  Users,
  ClipboardList,
  ChevronDown,
  ChevronRight,
  FlaskConical,
  ShieldAlert,
  ShieldX,
  ShieldCheck,
  Bell,
  UserCog,
} from "lucide-react";
import { useState } from "react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { testPublicApi, testOperatorApi, testAdminApi } from "@/api/test";

interface NavChild {
  key: string;
  href: string;
  adminOnly?: boolean; // 仅 Admin 可见
}

interface NavItem {
  key: string;
  href?: string;
  icon: typeof LayoutDashboard;
  children?: NavChild[];
}

const navItems: NavItem[] = [
  { key: "overview", href: "/overview", icon: LayoutDashboard },
  {
    key: "workbench",
    icon: Wrench,
    children: [
      { key: "ai", href: "/workbench" },
      { key: "clusters", href: "/system/clusters" },
      { key: "agents", href: "/system/agents" },
      { key: "notifications", href: "/system/notifications" },
    ],
  },
  {
    key: "cluster",
    icon: Server,
    children: [
      { key: "pod", href: "/cluster/pod" },
      { key: "node", href: "/cluster/node" },
      { key: "deployment", href: "/cluster/deployment" },
      { key: "service", href: "/cluster/service" },
      { key: "namespace", href: "/cluster/namespace" },
      { key: "ingress", href: "/cluster/ingress" },
      { key: "alert", href: "/cluster/alert" },
    ],
  },
  {
    key: "system",
    icon: Activity,
    children: [
      { key: "metrics", href: "/system/metrics" },
      { key: "logs", href: "/system/logs" },
      { key: "alerts", href: "/system/alerts" },
    ],
  },
  {
    key: "account",
    icon: UserCog,
    children: [
      { key: "users", href: "/system/users", adminOnly: true },
      { key: "roles", href: "/system/roles" },
      { key: "audit", href: "/system/audit" },
    ],
  },
];

const iconMap: Record<string, typeof Box> = {
  // workbench
  ai: Wrench,
  clusters: Server,
  agents: Server,
  notifications: Bell,
  // cluster
  pod: Box,
  node: Server,
  deployment: Layers,
  service: Network,
  namespace: FolderTree,
  ingress: Globe,
  alert: AlertTriangle,
  // system
  metrics: Activity,
  logs: FileText,
  alerts: AlertTriangle,
  // account
  users: Users,
  roles: ShieldCheck,
  audit: ClipboardList,
};

interface SidebarProps {
  collapsed?: boolean;
}

// ============================================================
// [TEST] 测试区域状态和处理函数
// 功能开发完成后需要删除这部分代码
// ============================================================
function TestSection({ collapsed }: { collapsed: boolean }) {
  const [testResult, setTestResult] = useState<{ type: "success" | "error"; message: string } | null>(null);
  const [showTest, setShowTest] = useState(false);

  const runTest = async (name: string, testFn: () => Promise<unknown>) => {
    setTestResult(null);
    try {
      await testFn();
      setTestResult({ type: "success", message: `${name}: 成功` });
    } catch (err) {
      const msg = err instanceof Error ? err.message : "失败";
      setTestResult({ type: "error", message: `${name}: ${msg}` });
    }
  };

  if (collapsed) return null;

  return (
    <div className="border-t border-[var(--border-color)] mt-4 pt-4">
      <button
        onClick={() => setShowTest(!showTest)}
        className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-yellow-600 dark:text-yellow-400 hover-bg"
      >
        <FlaskConical className="w-5 h-5" />
        <span className="flex-1 text-left text-sm">权限测试</span>
        {showTest ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
      </button>

      {showTest && (
        <div className="ml-4 mt-2 space-y-2">
          <p className="text-xs text-muted px-3 mb-2">无需登录:</p>
          <button
            onClick={() => runTest("公开", testPublicApi)}
            className="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-lg hover-bg text-muted"
          >
            <ShieldCheck className="w-4 h-4 text-green-500" />
            查看接口
          </button>

          <p className="text-xs text-muted px-3 mt-3 mb-2">需要登录:</p>
          <button
            onClick={() => runTest("操作", testOperatorApi)}
            className="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-lg hover-bg text-muted"
          >
            <ShieldAlert className="w-4 h-4 text-orange-500" />
            操作接口
          </button>
          <button
            onClick={() => runTest("管理", testAdminApi)}
            className="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-lg hover-bg text-muted"
          >
            <ShieldX className="w-4 h-4 text-red-500" />
            管理接口
          </button>

          {testResult && (
            <div
              className={`px-3 py-2 text-xs rounded-lg ${
                testResult.type === "success"
                  ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                  : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
              }`}
            >
              {testResult.message}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
// ============================================================
// [TEST] 测试区域结束
// ============================================================

export function Sidebar({ collapsed = false }: SidebarProps) {
  const pathname = usePathname();
  const { t } = useI18n();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 3; // Admin role
  const [expandedItems, setExpandedItems] = useState<string[]>(["workbench", "cluster", "system", "account"]);

  const toggleExpand = (key: string) => {
    setExpandedItems((prev) =>
      prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]
    );
  };

  const isActive = (href: string) => pathname === href;
  const isParentActive = (children: { href: string }[]) =>
    children.some((child) => pathname === child.href);

  return (
    <aside
      className={`bg-[var(--sidebar-bg)] border-r border-[var(--border-color)] h-full transition-all duration-300 ${
        collapsed ? "w-16" : "w-64"
      }`}
    >
      {/* Logo */}
      <div className="h-16 flex items-center justify-center border-b border-[var(--border-color)]">
        <Link href="/" className="flex items-center gap-2">
          <img src="/icon.svg" alt="AtlHyper" className="w-8 h-8" />
          {!collapsed && (
            <span className="text-xl font-bold text-primary">AtlHyper</span>
          )}
        </Link>
      </div>

      {/* Navigation */}
      <nav className="p-2 space-y-1">
        {navItems.map((item) => {
          const Icon = item.icon;
          const hasChildren = item.children && item.children.length > 0;
          const isExpanded = expandedItems.includes(item.key);
          const parentActive = hasChildren && isParentActive(item.children!);

          if (hasChildren) {
            return (
              <div key={item.key}>
                <button
                  onClick={() => toggleExpand(item.key)}
                  className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                    parentActive
                      ? "bg-primary/10 text-primary"
                      : "text-secondary hover-bg"
                  }`}
                >
                  <Icon className="w-5 h-5 flex-shrink-0" />
                  {!collapsed && (
                    <>
                      <span className="flex-1 text-left text-sm">
                        {t.nav[item.key as keyof typeof t.nav]}
                      </span>
                      {isExpanded ? (
                        <ChevronDown className="w-4 h-4" />
                      ) : (
                        <ChevronRight className="w-4 h-4" />
                      )}
                    </>
                  )}
                </button>
                {!collapsed && isExpanded && (
                  <div className="ml-4 mt-1 space-y-1">
                    {item.children!
                      .filter((child) => !child.adminOnly || isAdmin)
                      .map((child) => {
                        const ChildIcon = iconMap[child.key] || Box;
                        return (
                          <Link
                            key={child.key}
                            href={child.href}
                            className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                              isActive(child.href)
                                ? "bg-primary text-white"
                                : "text-muted hover-bg"
                            }`}
                          >
                            <ChildIcon className="w-4 h-4" />
                            {t.nav[child.key as keyof typeof t.nav]}
                          </Link>
                        );
                      })}
                  </div>
                )}
              </div>
            );
          }

          return (
            <Link
              key={item.key}
              href={item.href!}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                isActive(item.href!)
                  ? "bg-primary text-white"
                  : "text-secondary hover-bg"
              }`}
            >
              <Icon className="w-5 h-5 flex-shrink-0" />
              {!collapsed && (
                <span className="text-sm">
                  {t.nav[item.key as keyof typeof t.nav]}
                </span>
              )}
            </Link>
          );
        })}

        {/* [TEST] 权限测试区域 - 功能开发完成后删除 */}
        <TestSection collapsed={collapsed} />
      </nav>
    </aside>
  );
}
