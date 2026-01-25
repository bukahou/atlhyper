"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Bot,
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
  ShieldCheck,
  Bell,
  UserCog,
  ChevronsLeft,
  ChevronsRight,
  ChevronDown,
  ChevronRight,
  Settings,
} from "lucide-react";
import { useState } from "react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";

interface NavChild {
  key: string;
  href: string;
  icon: typeof Box;
  adminOnly?: boolean;
}

interface NavGroup {
  key: string;
  icon: typeof LayoutDashboard;
  href?: string;
  children?: NavChild[];
}

const navGroups: NavGroup[] = [
  { key: "overview", href: "/overview", icon: LayoutDashboard },
  {
    key: "workbench",
    icon: Bot,
    children: [
      { key: "workbenchHome", href: "/workbench", icon: LayoutDashboard },
      { key: "ai", href: "/workbench/ai", icon: Bot },
      { key: "commands", href: "/workbench/commands", icon: ClipboardList },
    ],
  },
  {
    key: "cluster",
    icon: Server,
    children: [
      { key: "pod", href: "/cluster/pod", icon: Box },
      { key: "node", href: "/cluster/node", icon: Server },
      { key: "deployment", href: "/cluster/deployment", icon: Layers },
      { key: "service", href: "/cluster/service", icon: Network },
      { key: "namespace", href: "/cluster/namespace", icon: FolderTree },
      { key: "ingress", href: "/cluster/ingress", icon: Globe },
      { key: "alert", href: "/cluster/alert", icon: AlertTriangle },
    ],
  },
  {
    key: "system",
    icon: Activity,
    children: [
      { key: "metrics", href: "/system/metrics", icon: Activity },
      { key: "logs", href: "/system/logs", icon: FileText },
    ],
  },
  {
    key: "settings",
    icon: Settings,
    children: [
      { key: "aiSettings", href: "/system/settings/ai", icon: Bot, adminOnly: true },
      { key: "notifications", href: "/system/notifications", icon: Bell, adminOnly: true },
    ],
  },
  {
    key: "account",
    icon: UserCog,
    children: [
      { key: "users", href: "/system/users", icon: Users, adminOnly: true },
      { key: "roles", href: "/system/roles", icon: ShieldCheck },
      { key: "audit", href: "/system/audit", icon: ClipboardList },
    ],
  },
];

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
}

export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const pathname = usePathname();
  const { t } = useI18n();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 3;
  const [expandedGroups, setExpandedGroups] = useState<string[]>(["workbench", "cluster", "system", "settings", "account"]);
  const [hoveredGroup, setHoveredGroup] = useState<string | null>(null);

  const isActive = (href: string) => pathname === href;
  const isGroupActive = (group: NavGroup) => {
    if (group.href) return pathname === group.href;
    return group.children?.some((child) => pathname === child.href) ?? false;
  };

  const toggleGroup = (key: string) => {
    setExpandedGroups((prev) =>
      prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]
    );
  };

  return (
    <aside
      className={`h-full flex flex-col bg-[var(--sidebar-bg)] relative z-40 overflow-visible ${
        collapsed ? "w-14" : "w-56"
      }`}
      style={{ transition: "width 200ms ease" }}
    >
      {/* Logo */}
      <div className={`h-14 flex items-center border-b border-[var(--border-color)]/30 ${collapsed ? "justify-center" : "px-3"}`}>
        <Link href="/" className="flex items-center gap-2">
          <img src="/icon.svg" alt="AtlHyper" className="w-7 h-7" />
          {!collapsed && <span className="text-base font-bold text-primary">AtlHyper</span>}
        </Link>
      </div>

      {/* Navigation */}
      <nav className={`flex-1 py-2 ${collapsed ? "px-1.5 overflow-visible" : "px-2 overflow-y-auto"}`}>
        {navGroups.map((group) => {
          const Icon = group.icon;
          const active = isGroupActive(group);
          const hasChildren = !!group.children;
          const isExpanded = expandedGroups.includes(group.key);

          // 折叠模式: icon-only + hover flyout
          if (collapsed) {
            return (
              <div
                key={group.key}
                className="relative mb-0.5"
                onMouseEnter={() => hasChildren && setHoveredGroup(group.key)}
                onMouseLeave={() => setHoveredGroup(null)}
              >
                {group.href ? (
                  <Link
                    href={group.href}
                    className={`flex items-center justify-center w-full h-10 rounded-lg transition-colors ${
                      active ? "bg-primary/10 text-primary" : "text-muted hover:bg-[var(--hover-bg)] hover:text-default"
                    }`}
                    title={t.nav[group.key as keyof typeof t.nav]}
                  >
                    {active && <span className="absolute left-0 top-2 bottom-2 w-[3px] rounded-r-full bg-primary" />}
                    <Icon className="w-5 h-5" />
                  </Link>
                ) : (
                  <button
                    className={`flex items-center justify-center w-full h-10 rounded-lg transition-colors ${
                      active ? "bg-primary/10 text-primary" : "text-muted hover:bg-[var(--hover-bg)] hover:text-default"
                    }`}
                    title={t.nav[group.key as keyof typeof t.nav]}
                  >
                    {active && <span className="absolute left-0 top-2 bottom-2 w-[3px] rounded-r-full bg-primary" />}
                    <Icon className="w-5 h-5" />
                  </button>
                )}

                {/* Flyout (collapsed mode) */}
                {hasChildren && hoveredGroup === group.key && (
                  <div className="absolute left-full top-0 pl-1 z-50">
                    <div className="py-2 px-1 min-w-[160px] rounded-lg border border-[var(--border-color)] bg-card shadow-lg">
                      <div className="px-3 py-1.5 text-[11px] font-medium text-muted uppercase tracking-wide">
                        {t.nav[group.key as keyof typeof t.nav]}
                      </div>
                      {group.children!
                        .filter((child) => !child.adminOnly || isAdmin)
                        .map((child) => {
                          const ChildIcon = child.icon;
                          return (
                            <Link
                              key={child.key}
                              href={child.href}
                              className={`flex items-center gap-2.5 px-3 py-2 rounded-md text-sm whitespace-nowrap transition-colors ${
                                isActive(child.href)
                                  ? "bg-primary/10 text-primary font-medium"
                                  : "text-secondary hover:bg-[var(--hover-bg)] hover:text-default"
                              }`}
                            >
                              <ChildIcon className="w-4 h-4" />
                              {t.nav[child.key as keyof typeof t.nav]}
                            </Link>
                          );
                        })}
                    </div>
                  </div>
                )}
              </div>
            );
          }

          // 展开模式: 完整导航
          if (hasChildren) {
            return (
              <div key={group.key} className="mb-1">
                <button
                  onClick={() => toggleGroup(group.key)}
                  className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg transition-colors text-sm ${
                    active ? "text-primary" : "text-secondary hover:bg-[var(--hover-bg)]"
                  }`}
                >
                  <Icon className="w-[18px] h-[18px] flex-shrink-0" />
                  <span className="flex-1 text-left">{t.nav[group.key as keyof typeof t.nav]}</span>
                  {isExpanded ? <ChevronDown className="w-3.5 h-3.5 text-muted" /> : <ChevronRight className="w-3.5 h-3.5 text-muted" />}
                </button>
                {isExpanded && (
                  <div className="ml-3 mt-0.5 space-y-0.5 border-l border-[var(--border-color)]/40 pl-3">
                    {group.children!
                      .filter((child) => !child.adminOnly || isAdmin)
                      .map((child) => {
                        const ChildIcon = child.icon;
                        return (
                          <Link
                            key={child.key}
                            href={child.href}
                            className={`flex items-center gap-2.5 px-2.5 py-1.5 rounded-md text-sm transition-colors ${
                              isActive(child.href)
                                ? "bg-primary/10 text-primary font-medium"
                                : "text-muted hover:bg-[var(--hover-bg)] hover:text-default"
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
              key={group.key}
              href={group.href!}
              className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-colors mb-1 ${
                active ? "bg-primary/10 text-primary font-medium" : "text-secondary hover:bg-[var(--hover-bg)]"
              }`}
            >
              <Icon className="w-[18px] h-[18px] flex-shrink-0" />
              <span>{t.nav[group.key as keyof typeof t.nav]}</span>
            </Link>
          );
        })}
      </nav>

      {/* 底部: 折叠/展开按钮 (固定位置) */}
      <div className={`py-2 border-t border-[var(--border-color)]/30 ${collapsed ? "flex justify-center" : "px-2"}`}>
        <button
          onClick={onToggle}
          className={`flex items-center gap-2 rounded-lg hover:bg-[var(--hover-bg)] transition-colors ${
            collapsed ? "p-2" : "w-full px-3 py-2"
          }`}
          title={collapsed ? "展开侧栏" : "收起侧栏"}
        >
          {collapsed ? (
            <ChevronsRight className="w-4 h-4 text-muted" />
          ) : (
            <>
              <ChevronsLeft className="w-4 h-4 text-muted" />
              <span className="text-xs text-muted">收起</span>
            </>
          )}
        </button>
      </div>
    </aside>
  );
}
