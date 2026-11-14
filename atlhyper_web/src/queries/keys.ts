/**
 * 查询键工厂
 *
 * 键的形状固定为 [域, 资源, 范围]。页面不手写键，一律用这里的工厂：
 * - clusterId 进键：切换集群即切换缓存，旧集群的数据不会显示在新集群下
 * - mode 进键：真实数据与演示数据永不共用缓存
 */
import type { DataSourceMode } from "@/config/data-source";

export const sessionKeys = {
  clusterList: () => ["session", "cluster-list"] as const,
};

export const adminKeys = {
  permissionTable: () => ["admin", "permission-table"] as const,
};

export const clusterKeys = {
  events: (clusterId: string, mode: DataSourceMode) => ["cluster", "events", { clusterId, mode }] as const,
};
