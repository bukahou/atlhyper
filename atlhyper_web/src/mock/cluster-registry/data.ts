/**
 * 集群管理页的演示数据（没有 clusters.manage 权限时整页展示，均为虚构）
 *
 * 四行覆盖页面上的每一种状态：在线且 Agent 已带令牌、在线但 Agent 还没带令牌、
 * 登记了但从未上报、正在上报但尚未登记。令牌前缀是占位符，不对应任何真实令牌。
 */
import type { ClusterRegistryItem } from "@/api/cluster-registry";

/** 相对现在往前若干分钟的时刻：让「最近上报」在任何时候打开都像刚发生 */
function minutesAgo(minutes: number): string {
  return new Date(Date.now() - minutes * 60_000).toISOString();
}

export const MOCK_CLUSTER_REGISTRY: ClusterRegistryItem[] = [
  {
    clusterId: "demo-prod",
    name: "Demo Production",
    description: "Example production cluster",
    registered: true,
    tokenIssued: true,
    tokenPrefix: "atlk_Dem",
    tokenIssuedAt: "2026-10-01T09:00:00Z",
    status: "online",
    lastSeen: minutesAgo(0),
    authenticated: true,
  },
  {
    clusterId: "demo-staging",
    name: "Demo Staging",
    description: "Token issued, agent not yet configured",
    registered: true,
    tokenIssued: true,
    tokenPrefix: "atlk_Dem",
    tokenIssuedAt: "2026-11-01T03:30:00Z",
    status: "online",
    lastSeen: minutesAgo(1),
    authenticated: false,
  },
  {
    clusterId: "demo-edge.site-01",
    name: "Demo Edge",
    description: "",
    registered: true,
    tokenIssued: false,
    tokenPrefix: "",
    tokenIssuedAt: "",
    status: "never",
    lastSeen: "",
    authenticated: false,
  },
  {
    clusterId: "demo-unregistered",
    name: "",
    description: "",
    registered: false,
    tokenIssued: false,
    tokenPrefix: "",
    tokenIssuedAt: "",
    status: "online",
    lastSeen: minutesAgo(0),
    authenticated: false,
  },
];
