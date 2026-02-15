"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Info,
  Bot,
  Box,
  Server,
  Layers,
  Network,
  FolderTree,
  Globe,
  AlertTriangle,
  Activity,
  Gauge,
  Palette,
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
  User,
  LogIn,
  LogOut,
  Languages,
  Sun,
  Moon,
  Monitor,
  Check,
  Github,
  Copy,
  Database,
  Play,
  Clock,
  HardDrive,
  HardDriveDownload,
  Shield,
  SlidersHorizontal,
  UserCheck,
} from "lucide-react";
import { useState, useEffect } from "react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { useClusterStore } from "@/store/clusterStore";
import { useTheme } from "@/theme/context";
import type { Language, Theme } from "@/types/common";
import { UserRole } from "@/types/auth";

// 语言选项
const languages: { code: Language; label: string }[] = [
  { code: "zh", label: "中文" },
  { code: "ja", label: "日本語" },
];

// 主题选项（label 会在组件中使用 i18n）
const themeOptions: { value: Theme; icon: typeof Sun }[] = [
  { value: "light", icon: Sun },
  { value: "dark", icon: Moon },
  { value: "system", icon: Monitor },
];

// 角色显示名称
const getRoleName = (role: number): string => {
  switch (role) {
    case UserRole.ADMIN:
      return "Admin";
    case UserRole.OPERATOR:
      return "Operator";
    case UserRole.VIEWER:
      return "Viewer";
    default:
      return "User";
  }
};

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
  authOnly?: boolean;
}

const navGroups: NavGroup[] = [
  { key: "about", href: "/about", icon: Info },
  { key: "overview", href: "/overview", icon: LayoutDashboard },
  {
    key: "workbench",
    icon: Bot,
    children: [
      { key: "workbenchHome", href: "/workbench", icon: LayoutDashboard },
      { key: "slo", href: "/workbench/slo", icon: Activity },
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
      { key: "daemonset", href: "/cluster/daemonset", icon: Copy },
      { key: "statefulset", href: "/cluster/statefulset", icon: Database },
      { key: "alert", href: "/cluster/alert", icon: AlertTriangle },
    ],
  },
  {
    key: "workload",
    icon: Play,
    children: [
      { key: "job", href: "/cluster/job", icon: Play },
      { key: "cronjob", href: "/cluster/cronjob", icon: Clock },
    ],
  },
  {
    key: "storage",
    icon: HardDrive,
    children: [
      { key: "pv", href: "/cluster/pv", icon: HardDrive },
      { key: "pvc", href: "/cluster/pvc", icon: HardDriveDownload },
    ],
  },
  {
    key: "policy",
    icon: Shield,
    children: [
      { key: "networkPolicy", href: "/cluster/network-policy", icon: Shield },
      { key: "resourceQuota", href: "/cluster/resource-quota", icon: Gauge },
      { key: "limitRange", href: "/cluster/limit-range", icon: SlidersHorizontal },
      { key: "serviceAccount", href: "/cluster/service-account", icon: UserCheck },
    ],
  },
  {
    key: "monitoring",
    icon: Activity,
    children: [
      { key: "metrics", href: "/monitoring/metrics", icon: Activity },
      { key: "logs", href: "/monitoring/logs", icon: FileText },
    ],
  },
  {
    key: "settings",
    icon: Settings,
    children: [
      { key: "aiSettings", href: "/settings/ai", icon: Bot },
      { key: "notifications", href: "/settings/notifications", icon: Bell },
    ],
  },
  {
    key: "admin",
    icon: UserCog,
    children: [
      { key: "users", href: "/admin/users", icon: Users, adminOnly: true },
      { key: "roles", href: "/admin/roles", icon: ShieldCheck },
      { key: "audit", href: "/admin/audit", icon: ClipboardList },
    ],
  },
  {
    key: "stylePreview",
    icon: Palette,
    authOnly: true,
    children: [
      { key: "stylePreviewSLO", href: "/style-preview", icon: Activity },
      { key: "stylePreviewMetrics", href: "/style-preview/metrics", icon: Gauge },
    ],
  },
];

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
}

// 根据路径计算应该展开哪些组
function getActiveGroups(pathname: string): string[] {
  const active: string[] = [];
  for (const group of navGroups) {
    if (group.children?.some((child) => pathname === child.href || pathname.startsWith(child.href + "/"))) {
      active.push(group.key);
    }
  }
  return active;
}

// localStorage key for persisting expanded groups
const EXPANDED_GROUPS_KEY = "sidebar-expanded-groups";

// 从 localStorage 读取展开状态
function loadExpandedGroups(pathname: string): string[] {
  if (typeof window === "undefined") {
    return getActiveGroups(pathname);
  }
  try {
    const stored = localStorage.getItem(EXPANDED_GROUPS_KEY);
    if (stored) {
      const parsed = JSON.parse(stored) as string[];
      // 确保当前路径对应的组也被展开
      const activeGroups = getActiveGroups(pathname);
      const merged = [...new Set([...parsed, ...activeGroups])];
      return merged;
    }
  } catch {
    // ignore
  }
  return getActiveGroups(pathname);
}

// 保存展开状态到 localStorage
function saveExpandedGroups(groups: string[]) {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(EXPANDED_GROUPS_KEY, JSON.stringify(groups));
  } catch {
    // ignore
  }
}

export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const pathname = usePathname();
  const { t, language, setLanguage } = useI18n();
  const { isAuthenticated, user, openLoginDialog, logout } = useAuthStore();
  const { clusterIds, currentClusterId, setCurrentCluster } = useClusterStore();
  const { theme, setTheme } = useTheme();
  const isAdmin = user?.role === 3;
  const [expandedGroups, setExpandedGroups] = useState<string[]>(() => loadExpandedGroups(pathname));
  const [hoveredGroup, setHoveredGroup] = useState<string | null>(null);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [clusterMenuOpen, setClusterMenuOpen] = useState(false);
  const [settingsMenuOpen, setSettingsMenuOpen] = useState(false);

  // 路由变化时，确保当前激活的组被展开（但不收起其他组）
  useEffect(() => {
    const activeGroups = getActiveGroups(pathname);
    setExpandedGroups((prev) => {
      const newGroups = [...prev];
      for (const group of activeGroups) {
        if (!newGroups.includes(group)) {
          newGroups.push(group);
        }
      }
      // 保存到 localStorage
      saveExpandedGroups(newGroups);
      return newGroups;
    });
  }, [pathname]);

  // 展开/收起状态变化时保存到 localStorage
  useEffect(() => {
    saveExpandedGroups(expandedGroups);
  }, [expandedGroups]);

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
      className={`h-full flex flex-col relative z-40 overflow-visible ${
        collapsed ? "w-14" : "w-56"
      } ml-4 mr-6 my-6 rounded-2xl bg-[var(--sidebar-bg)] border border-[var(--border-color)]/50 shadow-[0_10px_40px_rgb(0,0,0,0.15),0_0_20px_rgb(0,0,0,0.1)] dark:shadow-[0_10px_40px_rgb(0,0,0,0.5),0_0_20px_rgb(0,0,0,0.3)] ring-1 ring-black/5 dark:ring-white/5`}
      style={{ transition: "width 200ms ease", height: "calc(100% - 48px)" }}
    >
      {/* Logo */}
      <div className={`h-14 flex items-center border-b border-[var(--border-color)]/20 ${collapsed ? "justify-center" : "px-3"}`}>
        <Link href="/" className="flex-shrink-0">
          <img src="/icon.png" alt="AtlHyper" className="w-8 h-8" />
        </Link>
        {!collapsed && (
          <>
            <span className="flex-1 text-center text-lg font-bold text-primary tracking-tight">AtlHyper</span>
            <a
              href="https://github.com/bukahou/atlhyper"
              target="_blank"
              rel="noopener noreferrer"
              className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-white/10 transition-colors"
              title="GitHub"
            >
              <Github className="w-5 h-5 text-muted hover:text-default" />
            </a>
          </>
        )}
      </div>

      {/* Cluster Selector */}
      {clusterIds.length > 0 && (
        <div className={`relative border-b border-[var(--border-color)]/20 ${collapsed ? "py-2 flex justify-center" : "px-3 py-2"}`}>
          {collapsed ? (
            <div
              className="relative"
              onMouseEnter={() => clusterIds.length > 1 && setClusterMenuOpen(true)}
              onMouseLeave={() => setClusterMenuOpen(false)}
            >
              <button className="p-2 rounded-xl hover:bg-white/5 transition-colors" title={currentClusterId}>
                <Server className="w-5 h-5 text-primary" />
              </button>
              {clusterIds.length > 1 && clusterMenuOpen && (
                <div className="absolute left-full top-0 ml-2 z-50">
                  <div className="py-2 px-1 min-w-[180px] rounded-2xl border border-white/10 dark:border-white/5 bg-card/95 backdrop-blur-xl shadow-[0_8px_30px_rgb(0,0,0,0.12)] dark:shadow-[0_8px_30px_rgb(0,0,0,0.4)] ring-1 ring-black/5 dark:ring-white/10">
                    <div className="px-3 py-2 text-[11px] font-semibold text-muted uppercase tracking-wider border-b border-[var(--border-color)]/30 mb-1">
                      Select Cluster
                    </div>
                    {clusterIds.map((id) => (
                      <button
                        key={id}
                        onClick={() => { setCurrentCluster(id); window.location.reload(); }}
                        className={`w-full flex items-center justify-between gap-2 px-3 py-2 rounded-xl text-sm transition-all ${
                          id === currentClusterId ? "text-primary bg-primary/10" : "text-secondary hover:bg-white/5"
                        }`}
                      >
                        <span className="font-mono truncate">{id}</span>
                        {id === currentClusterId && <Check className="w-4 h-4 flex-shrink-0" />}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="relative">
              <button
                onClick={() => clusterIds.length > 1 && setClusterMenuOpen(!clusterMenuOpen)}
                className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-sm transition-all hover:bg-white/5 ${
                  clusterIds.length <= 1 ? "cursor-default" : ""
                }`}
              >
                <Server className="w-4 h-4 text-primary flex-shrink-0" />
                <span className="font-mono text-secondary truncate flex-1 text-left">{currentClusterId}</span>
                {clusterIds.length > 1 && (
                  <ChevronDown className={`w-4 h-4 text-muted transition-transform ${clusterMenuOpen ? "rotate-180" : ""}`} />
                )}
              </button>
              {clusterIds.length > 1 && clusterMenuOpen && (
                <div className="absolute left-0 right-0 top-full mt-1 z-50 py-1 rounded-xl border border-white/10 dark:border-white/5 bg-card/95 backdrop-blur-xl shadow-lg">
                  {clusterIds.map((id) => (
                    <button
                      key={id}
                      onClick={() => { setCurrentCluster(id); setClusterMenuOpen(false); window.location.reload(); }}
                      className={`w-full flex items-center justify-between gap-2 px-3 py-2 text-sm transition-all ${
                        id === currentClusterId ? "text-primary" : "text-secondary hover:bg-white/5"
                      }`}
                    >
                      <span className="font-mono truncate">{id}</span>
                      {id === currentClusterId && <Check className="w-4 h-4 flex-shrink-0" />}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Navigation - 添加 min-h-0 让 flex-1 子元素可以滚动 */}
      <nav
        className={`flex-1 min-h-0 py-3 ${collapsed ? "px-2 overflow-visible" : "px-3 overflow-y-auto"}`}
        style={{ scrollbarWidth: 'thin', scrollbarColor: 'var(--border-color) transparent' }}
      >
        {navGroups.filter((g) => !g.authOnly || isAuthenticated).map((group) => {
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
                    className={`flex items-center justify-center w-10 h-10 mx-auto rounded-xl transition-all duration-150 ${
                      active ? "bg-primary/15 text-primary shadow-sm" : "text-muted hover:bg-white/5 dark:hover:bg-white/5 hover:text-default"
                    }`}
                    title={t.nav[group.key as keyof typeof t.nav]}
                  >
                    <Icon className="w-5 h-5" />
                  </Link>
                ) : (
                  <button
                    className={`flex items-center justify-center w-10 h-10 mx-auto rounded-xl transition-all duration-150 ${
                      active ? "bg-primary/15 text-primary shadow-sm" : "text-muted hover:bg-white/5 dark:hover:bg-white/5 hover:text-default"
                    }`}
                    title={t.nav[group.key as keyof typeof t.nav]}
                  >
                    <Icon className="w-5 h-5" />
                  </button>
                )}

                {/* Flyout (collapsed mode) - Horizon style */}
                {hasChildren && hoveredGroup === group.key && (
                  <div className="absolute left-full top-0 pl-2 z-50">
                    <div className="py-3 px-2 min-w-[180px] rounded-2xl border border-white/10 dark:border-white/5 bg-card/95 backdrop-blur-xl shadow-[0_8px_30px_rgb(0,0,0,0.12)] dark:shadow-[0_8px_30px_rgb(0,0,0,0.4)] ring-1 ring-black/5 dark:ring-white/10 animate-in fade-in slide-in-from-left-2 duration-200">
                      <div className="px-3 py-2 text-[11px] font-semibold text-muted uppercase tracking-wider border-b border-[var(--border-color)]/30 mb-1">
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
                              className={`flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm whitespace-nowrap transition-all duration-150 ${
                                isActive(child.href)
                                  ? "bg-primary/15 text-primary font-medium shadow-sm"
                                  : "text-secondary hover:bg-[var(--hover-bg)] hover:text-default hover:translate-x-0.5"
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
              <div key={group.key} className="mb-1.5">
                <button
                  onClick={() => toggleGroup(group.key)}
                  className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-150 text-sm ${
                    active ? "text-primary bg-primary/5" : "text-secondary hover:bg-white/5 dark:hover:bg-white/5"
                  }`}
                >
                  <Icon className="w-[18px] h-[18px] flex-shrink-0" />
                  <span className="flex-1 text-left font-medium">{t.nav[group.key as keyof typeof t.nav]}</span>
                  {isExpanded ? <ChevronDown className="w-3.5 h-3.5 text-muted transition-transform" /> : <ChevronRight className="w-3.5 h-3.5 text-muted transition-transform" />}
                </button>
                {isExpanded && (
                  <div className="mt-1 ml-2 space-y-0.5 pl-4 border-l-2 border-primary/20">
                    {group.children!
                      .filter((child) => !child.adminOnly || isAdmin)
                      .map((child) => {
                        const ChildIcon = child.icon;
                        return (
                          <Link
                            key={child.key}
                            href={child.href}
                            className={`flex items-center gap-2.5 px-3 py-2 rounded-xl text-sm transition-all duration-150 ${
                              isActive(child.href)
                                ? "bg-primary/15 text-primary font-medium shadow-sm"
                                : "text-muted hover:bg-white/5 dark:hover:bg-white/5 hover:text-default hover:translate-x-0.5"
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
              className={`flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm transition-all duration-150 mb-1.5 ${
                active ? "bg-primary/15 text-primary font-medium shadow-sm" : "text-secondary hover:bg-white/5 dark:hover:bg-white/5"
              }`}
            >
              <Icon className="w-[18px] h-[18px] flex-shrink-0" />
              <span className="font-medium">{t.nav[group.key as keyof typeof t.nav]}</span>
            </Link>
          );
        })}
      </nav>

      {/* 底部: 用户区域 + 设置/折叠 */}
      <div className={`border-t border-[var(--border-color)]/20 ${collapsed ? "py-2" : "p-3"}`}>
        {/* 用户区域（独立一行） */}
        <div
          className="relative"
          onMouseEnter={() => setUserMenuOpen(true)}
          onMouseLeave={() => setUserMenuOpen(false)}
        >
          {isAuthenticated ? (
            <>
              <button
                className={`flex items-center gap-3 rounded-xl transition-all duration-150 hover:bg-white/5 ${
                  collapsed ? "p-2 mx-auto" : "w-full px-3 py-2.5"
                }`}
              >
                <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center flex-shrink-0">
                  <User className="w-4 h-4 text-primary" />
                </div>
                {!collapsed && (
                  <div className="flex-1 text-left min-w-0">
                    <p className="text-sm font-medium text-default truncate">
                      {user?.displayName || user?.username || "User"}
                    </p>
                    <p className="text-xs text-muted">
                      {user ? getRoleName(user.role) : ""}
                    </p>
                  </div>
                )}
              </button>

              {/* 用户悬浮卡片 */}
              {userMenuOpen && (
                <div className={`absolute z-50 ${collapsed ? "left-full bottom-0 pl-2" : "left-0 right-0 bottom-full pb-2"}`}>
                  <div className="py-2 rounded-2xl border border-[var(--border-color)] bg-card shadow-[0_8px_30px_rgb(0,0,0,0.15)] dark:shadow-[0_8px_30px_rgb(0,0,0,0.5)] min-w-[180px]">
                    {/* 用户信息 */}
                    <div className="px-4 py-3 border-b border-[var(--border-color)]/30">
                      <p className="text-sm font-medium text-default">
                        {user?.displayName || user?.username}
                      </p>
                      {user?.email && (
                        <p className="text-xs text-secondary truncate">
                          {user.email}
                        </p>
                      )}
                      <p className="text-xs text-muted">
                        {user ? getRoleName(user.role) : ""}
                      </p>
                    </div>

                    {/* 退出登录 */}
                    <div className="px-2 py-2">
                      <button
                        onClick={logout}
                        className="w-full flex items-center gap-2 px-3 py-2 rounded-xl text-sm text-red-500 hover:bg-red-500/10 transition-all"
                      >
                        <LogOut className="w-4 h-4" />
                        {t.common.logout}
                      </button>
                    </div>
                  </div>
                </div>
              )}
            </>
          ) : (
            <button
              onClick={() => openLoginDialog()}
              className={`flex items-center gap-2 rounded-xl text-primary hover:bg-primary/10 transition-all ${
                collapsed ? "p-2 mx-auto" : "w-full px-3 py-2.5"
              }`}
            >
              <LogIn className="w-5 h-5" />
              {!collapsed && <span className="text-sm font-medium">{t.common.login}</span>}
            </button>
          )}
        </div>

        {/* 设置 + 折叠 并列 */}
        <div className={`flex items-center gap-1 mt-2 ${collapsed ? "flex-col" : ""}`}>
          {/* 设置按钮（语言+主题） */}
          <div
            className="relative flex-1"
            onMouseEnter={() => setSettingsMenuOpen(true)}
            onMouseLeave={() => setSettingsMenuOpen(false)}
          >
            <button
              className={`flex items-center gap-2 rounded-xl hover:bg-white/5 transition-all ${
                collapsed ? "p-2 mx-auto" : "w-full px-3 py-2"
              }`}
              title="Settings"
            >
              <Settings className="w-4 h-4 text-muted" />
              {!collapsed && <span className="text-xs text-muted font-medium">{t.common.settings}</span>}
            </button>

            {/* 设置弹出面板 */}
            {settingsMenuOpen && (
              <div className={`absolute z-50 ${collapsed ? "left-full bottom-0 pl-2" : "left-0 bottom-full pb-2"}`}>
                <div className="py-2 rounded-2xl border border-[var(--border-color)] bg-card shadow-[0_8px_30px_rgb(0,0,0,0.15)] dark:shadow-[0_8px_30px_rgb(0,0,0,0.5)] min-w-[180px]">
                  {/* 语言切换 */}
                  <div className="px-2 py-2 border-b border-[var(--border-color)]/30">
                    <div className="px-2 py-1 text-[11px] font-semibold text-muted uppercase tracking-wider flex items-center gap-2">
                      <Languages className="w-3.5 h-3.5" />
                      Language
                    </div>
                    <div className="flex gap-1 mt-1">
                      {languages.map((lang) => (
                        <button
                          key={lang.code}
                          onClick={() => setLanguage(lang.code)}
                          className={`flex-1 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                            language === lang.code
                              ? "bg-primary/15 text-primary"
                              : "text-secondary hover:bg-white/5"
                          }`}
                        >
                          {lang.label}
                        </button>
                      ))}
                    </div>
                  </div>

                  {/* 主题切换 */}
                  <div className="px-2 py-2">
                    <div className="px-2 py-1 text-[11px] font-semibold text-muted uppercase tracking-wider flex items-center gap-2">
                      <Sun className="w-3.5 h-3.5" />
                      Theme
                    </div>
                    <div className="flex gap-1 mt-1">
                      {themeOptions.map((th) => {
                        const ThemeIcon = th.icon;
                        const themeLabel = th.value === "light" ? t.common.themeLight : th.value === "dark" ? t.common.themeDark : t.common.themeSystem;
                        return (
                          <button
                            key={th.value}
                            onClick={() => setTheme(th.value)}
                            className={`flex-1 px-2 py-1.5 rounded-lg text-xs font-medium transition-all flex items-center justify-center gap-1 ${
                              theme === th.value
                                ? "bg-primary/15 text-primary"
                                : "text-secondary hover:bg-white/5"
                            }`}
                          >
                            <ThemeIcon className="w-3.5 h-3.5" />
                            {themeLabel}
                          </button>
                        );
                      })}
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* 折叠按钮 */}
          <button
            onClick={onToggle}
            className={`flex items-center gap-2 rounded-xl hover:bg-white/5 transition-all duration-150 ${
              collapsed ? "p-2" : "flex-1 px-3 py-2"
            }`}
            title={collapsed ? t.common.expandSidebar : t.common.collapseSidebar}
          >
            {collapsed ? (
              <ChevronsRight className="w-4 h-4 text-muted" />
            ) : (
              <>
                <ChevronsLeft className="w-4 h-4 text-muted" />
                <span className="text-xs text-muted font-medium">{t.common.collapseSidebar}</span>
              </>
            )}
          </button>
        </div>
      </div>
    </aside>
  );
}
