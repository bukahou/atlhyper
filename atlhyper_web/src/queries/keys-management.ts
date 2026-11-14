/**
 * 查询键工厂 —— 管理与设置（用户 / 审计 / 指令历史 / 集群注册表 / AI 设置 / 通知 / GitHub / 部署）
 *
 * 约定同 keys.ts。这些页面在没有权限时整页显示演示数据，演示数据由页面直接取，不经过查询，
 * 所以大多数键里不带 mode；只有带数据源开关的 GitHub 与部署带 mode。
 */
import type { DataSourceMode } from "@/config/data-source";

/** 指令历史的筛选条件 + 分页 */
export interface CommandHistoryFilter {
  source: string;
  status: string;
  action: string;
  search: string;
  limit: number;
  offset: number;
}

export const adminKeys = {
  permissionTable: () => ["admin", "permission-table"] as const,
  users: () => ["admin", "users"] as const,
  auditLogs: () => ["admin", "audit-logs"] as const,
  commandHistory: (clusterId: string, filter: CommandHistoryFilter) =>
    ["admin", "command-history", { clusterId, ...filter }] as const,
  /** 集群注册表：已登记的集群 + 正在上报但未登记的集群 */
  clusterRegistry: () => ["admin", "cluster-registry"] as const,
};

export const aiSettingsKeys = {
  /** Provider 列表（连同可选模型与全局设置，接口一次返回） */
  providers: () => ["ai-settings", "providers"] as const,
  roles: () => ["ai-settings", "roles"] as const,
  budgets: () => ["ai-settings", "budgets"] as const,
  /** AI 对话是否可用（有没有给 chat 角色分配 Provider） */
  chatReadiness: () => ["ai-settings", "chat-readiness"] as const,
  /** 某个角色的全部调用记录（不分页数）。只用来按前缀匹配 */
  usageReports: (role: string) => ["ai-settings", "usage-reports", { role }] as const,
};

export const notifyKeys = {
  channels: () => ["notify", "channels"] as const,
};

export const githubKeys = {
  all: ["github"] as const,
  connection: (mode: DataSourceMode) => ["github", "connection", { mode }] as const,
  repos: (mode: DataSourceMode) => ["github", "repos", { mode }] as const,
};

export const deployKeys = {
  all: ["deploy"] as const,
  config: (clusterId: string, mode: DataSourceMode) => ["deploy", "config", { clusterId, mode }] as const,
  /** 同步状态由后端按它自己的部署配置返回，接口不分集群 */
  status: (mode: DataSourceMode) => ["deploy", "status", { mode }] as const,
  history: (clusterId: string, mode: DataSourceMode) => ["deploy", "history", { clusterId, mode }] as const,
  kustomizePaths: (mode: DataSourceMode, repo: string) => ["deploy", "kustomize-paths", { mode, repo }] as const,
};
