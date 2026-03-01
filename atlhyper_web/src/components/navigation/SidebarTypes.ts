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
  Gauge,
  GitGraph,
  Palette,
  FileText,
  Users,
  ClipboardList,
  ShieldCheck,
  Bell,
  UserCog,
  Settings,
  Sun,
  Moon,
  Monitor,
  Copy,
  Database,
  Play,
  Clock,
  HardDrive,
  HardDriveDownload,
  Shield,
  SlidersHorizontal,
  UserCheck,
  Waypoints,
  ToggleLeft,
} from "lucide-react";
import type { Language, Theme } from "@/types/common";
import { UserRole } from "@/types/auth";

// --- Types ---

export interface NavChild {
  key: string;
  href: string;
  icon: typeof Box;
  adminOnly?: boolean;
  section?: string; // 组内分隔线标签（i18n nav key）
}

export interface NavGroup {
  key: string;
  icon: typeof LayoutDashboard;
  href?: string;
  children?: NavChild[];
  authOnly?: boolean;
}

// --- Constants ---

export const languages: { code: Language; label: string }[] = [
  { code: "zh", label: "中文" },
  { code: "ja", label: "日本語" },
];

// 主题选项（label 会在组件中使用 i18n）
export const themeOptions: { value: Theme; icon: typeof Sun }[] = [
  { value: "light", icon: Sun },
  { value: "dark", icon: Moon },
  { value: "system", icon: Monitor },
];

export const navGroups: NavGroup[] = [
  { key: "overview", href: "/overview", icon: LayoutDashboard },
  {
    key: "observe",
    icon: Activity,
    children: [
      { key: "observeLanding", href: "/observe", icon: LayoutDashboard },
      { key: "apm", href: "/observe/apm", icon: Waypoints },
      { key: "logs", href: "/observe/logs", icon: FileText },
      { key: "metrics", href: "/observe/metrics", icon: Activity },
      { key: "slo", href: "/observe/slo", icon: Gauge },
    ],
  },
  {
    key: "aiops",
    icon: Bot,
    children: [
      { key: "riskDashboard", href: "/aiops/risk", icon: Gauge },
      { key: "incidentsNav", href: "/aiops/incidents", icon: AlertTriangle },
      { key: "topologyNav", href: "/aiops/topology", icon: GitGraph },
      { key: "ai", href: "/aiops/chat", icon: Bot },
    ],
  },
  {
    key: "cluster",
    icon: Server,
    children: [
      // 核心
      { key: "pod", href: "/cluster/pod", icon: Box, section: "core" },
      { key: "node", href: "/cluster/node", icon: Server },
      { key: "deployment", href: "/cluster/deployment", icon: Layers },
      { key: "service", href: "/cluster/service", icon: Network },
      { key: "namespace", href: "/cluster/namespace", icon: FolderTree },
      { key: "ingress", href: "/cluster/ingress", icon: Globe },
      { key: "event", href: "/cluster/event", icon: AlertTriangle },
      // 工作负载
      { key: "daemonset", href: "/cluster/daemonset", icon: Copy, section: "workload" },
      { key: "statefulset", href: "/cluster/statefulset", icon: Database },
      { key: "job", href: "/cluster/job", icon: Play },
      { key: "cronjob", href: "/cluster/cronjob", icon: Clock },
      // 存储
      { key: "pv", href: "/cluster/pv", icon: HardDrive, section: "storage" },
      { key: "pvc", href: "/cluster/pvc", icon: HardDriveDownload },
      // 策略
      { key: "networkPolicy", href: "/cluster/netpol", icon: Shield, section: "policy" },
      { key: "resourceQuota", href: "/cluster/quota", icon: Gauge },
      { key: "limitRange", href: "/cluster/limit", icon: SlidersHorizontal },
      { key: "serviceAccount", href: "/cluster/sa", icon: UserCheck },
    ],
  },
  {
    key: "settings",
    icon: Settings,
    children: [
      { key: "aiSettings", href: "/settings/ai", icon: Bot },
      { key: "notifications", href: "/settings/notif", icon: Bell },
    ],
  },
  {
    key: "admin",
    icon: UserCog,
    children: [
      { key: "users", href: "/admin/users", icon: Users, adminOnly: true },
      { key: "roles", href: "/admin/roles", icon: ShieldCheck },
      { key: "audit", href: "/admin/audit", icon: ClipboardList },
      { key: "commands", href: "/admin/commands", icon: ClipboardList },
      { key: "datasource", href: "/admin/datasource", icon: ToggleLeft },
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

// --- Helper Functions ---

// 角色显示名称
export const getRoleName = (role: number): string => {
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

// 根据路径计算应该展开哪些组
export function getActiveGroups(pathname: string): string[] {
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
export function loadExpandedGroups(pathname: string): string[] {
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
export function saveExpandedGroups(groups: string[]) {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(EXPANDED_GROUPS_KEY, JSON.stringify(groups));
  } catch {
    // ignore
  }
}
