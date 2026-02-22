/**
 * 数据源集中配置
 *
 * 模块注册表 + localStorage 持久化读写
 */

export type DataSourceMode = "mock" | "api";

export interface ModuleDataSource {
  key: string;
  category: string;
  labelKey: string;
  hasMock: boolean;
  defaultMode: DataSourceMode;
}

export const MODULE_REGISTRY: ModuleDataSource[] = [
  // Observe
  { key: "metrics", category: "observe", labelKey: "metrics", hasMock: true, defaultMode: "mock" },
  { key: "logs", category: "observe", labelKey: "logs", hasMock: true, defaultMode: "mock" },
  { key: "apm", category: "observe", labelKey: "apm", hasMock: true, defaultMode: "mock" },
  { key: "slo", category: "observe", labelKey: "slo", hasMock: true, defaultMode: "mock" },
  { key: "overview", category: "observe", labelKey: "overview", hasMock: true, defaultMode: "mock" },
  // Cluster
  { key: "pod", category: "cluster", labelKey: "pod", hasMock: true, defaultMode: "mock" },
  { key: "node", category: "cluster", labelKey: "node", hasMock: true, defaultMode: "mock" },
  { key: "deployment", category: "cluster", labelKey: "deployment", hasMock: true, defaultMode: "mock" },
  { key: "service", category: "cluster", labelKey: "service", hasMock: true, defaultMode: "mock" },
  { key: "namespace", category: "cluster", labelKey: "namespace", hasMock: true, defaultMode: "mock" },
  { key: "ingress", category: "cluster", labelKey: "ingress", hasMock: true, defaultMode: "mock" },
  { key: "event", category: "cluster", labelKey: "event", hasMock: true, defaultMode: "mock" },
  { key: "daemonset", category: "cluster", labelKey: "daemonset", hasMock: true, defaultMode: "mock" },
  { key: "statefulset", category: "cluster", labelKey: "statefulset", hasMock: true, defaultMode: "mock" },
  { key: "job", category: "cluster", labelKey: "job", hasMock: true, defaultMode: "mock" },
  { key: "cronjob", category: "cluster", labelKey: "cronjob", hasMock: true, defaultMode: "mock" },
  { key: "pv", category: "cluster", labelKey: "pv", hasMock: true, defaultMode: "mock" },
  { key: "pvc", category: "cluster", labelKey: "pvc", hasMock: true, defaultMode: "mock" },
  { key: "netpol", category: "cluster", labelKey: "networkPolicy", hasMock: true, defaultMode: "mock" },
  { key: "quota", category: "cluster", labelKey: "resourceQuota", hasMock: true, defaultMode: "mock" },
  { key: "limit", category: "cluster", labelKey: "limitRange", hasMock: true, defaultMode: "mock" },
  { key: "sa", category: "cluster", labelKey: "serviceAccount", hasMock: true, defaultMode: "mock" },
  // Admin
  { key: "users", category: "admin", labelKey: "users", hasMock: false, defaultMode: "api" },
  { key: "roles", category: "admin", labelKey: "roles", hasMock: false, defaultMode: "api" },
  { key: "audit", category: "admin", labelKey: "audit", hasMock: false, defaultMode: "api" },
  { key: "commands", category: "admin", labelKey: "commands", hasMock: false, defaultMode: "api" },
  // Settings
  { key: "aiSettings", category: "settings", labelKey: "aiSettings", hasMock: false, defaultMode: "api" },
  { key: "notifications", category: "settings", labelKey: "notifications", hasMock: false, defaultMode: "api" },
  // AIOps
  { key: "risk", category: "aiops", labelKey: "riskDashboard", hasMock: false, defaultMode: "api" },
  { key: "incidents", category: "aiops", labelKey: "incidentsNav", hasMock: false, defaultMode: "api" },
  { key: "topology", category: "aiops", labelKey: "topologyNav", hasMock: false, defaultMode: "api" },
  { key: "chat", category: "aiops", labelKey: "ai", hasMock: false, defaultMode: "api" },
];

const STORAGE_KEY = "atlhyper-datasource";

/** 从 localStorage 读取所有数据源配置 */
function loadConfig(): Record<string, DataSourceMode> {
  if (typeof window === "undefined") return {};
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

/** 保存数据源配置到 localStorage */
function saveConfig(config: Record<string, DataSourceMode>): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
  } catch {
    // ignore
  }
}

/** 获取指定模块的数据源模式 */
export function getDataSourceMode(key: string): DataSourceMode {
  const config = loadConfig();
  if (config[key]) return config[key];
  const mod = MODULE_REGISTRY.find((m) => m.key === key);
  return mod?.defaultMode ?? "api";
}

/** 设置指定模块的数据源模式 */
export function setDataSourceMode(key: string, mode: DataSourceMode): void {
  const config = loadConfig();
  config[key] = mode;
  saveConfig(config);
}
