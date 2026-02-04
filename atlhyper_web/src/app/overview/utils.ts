import { safeNumber, safeString, safePercent, safeTimestamp } from "@/utils/safeData";
import type { ClusterOverview, TransformedOverview } from "@/types/overview";

// 默认空数据结构
export const emptyData: TransformedOverview = {
  clusterId: "",
  healthCard: { status: "Unknown", reason: "", nodeReadyPct: 0, podHealthyPct: 0 },
  nodesCard: { totalNodes: 0, readyNodes: 0, nodeReadyPct: 0 },
  cpuCard: { percent: 0 },
  memCard: { percent: 0 },
  alertsTotal: 0,
  workloads: {
    deployments: { total: 0, ready: 0 },
    daemonsets: { total: 0, ready: 0 },
    statefulsets: { total: 0, ready: 0 },
    jobs: { total: 0, running: 0, succeeded: 0, failed: 0 },
  },
  podStatus: {
    total: 0,
    running: 0,
    pending: 0,
    failed: 0,
    succeeded: 0,
    runningPercent: 0,
    pendingPercent: 0,
    failedPercent: 0,
    succeededPercent: 0,
  },
  peakStats: {
    peakCpu: 0,
    peakCpuNode: "",
    peakMem: 0,
    peakMemNode: "",
    hasData: false,
  },
  alertTrends: [],
  recentAlerts: [],
  nodeUsages: [],
};

// 安全转换 API 数据
export function transformOverview(data: ClusterOverview | null | undefined): TransformedOverview {
  if (!data) return emptyData;

  const cards = data.cards || {};
  const workloads = data.workloads || {};
  const alerts = data.alerts || {};
  const nodes = data.nodes || {};

  return {
    clusterId: safeString(data.clusterId),
    healthCard: {
      status: safeString(cards.clusterHealth?.status, "Unknown"),
      reason: safeString(cards.clusterHealth?.reason),
      nodeReadyPct: safePercent(cards.clusterHealth?.nodeReadyPercent),
      podHealthyPct: safePercent(cards.clusterHealth?.podReadyPercent),
    },
    nodesCard: {
      totalNodes: safeNumber(cards.nodeReady?.total),
      readyNodes: safeNumber(cards.nodeReady?.ready),
      nodeReadyPct: safePercent(cards.nodeReady?.percent),
    },
    cpuCard: { percent: safePercent(cards.cpuUsage?.percent) },
    memCard: { percent: safePercent(cards.memUsage?.percent) },
    alertsTotal: safeNumber(cards.events24h),
    workloads: {
      deployments: {
        total: safeNumber(workloads.summary?.deployments?.total),
        ready: safeNumber(workloads.summary?.deployments?.ready),
      },
      daemonsets: {
        total: safeNumber(workloads.summary?.daemonsets?.total),
        ready: safeNumber(workloads.summary?.daemonsets?.ready),
      },
      statefulsets: {
        total: safeNumber(workloads.summary?.statefulsets?.total),
        ready: safeNumber(workloads.summary?.statefulsets?.ready),
      },
      jobs: {
        total: safeNumber(workloads.summary?.jobs?.total),
        running: safeNumber(workloads.summary?.jobs?.running),
        succeeded: safeNumber(workloads.summary?.jobs?.succeeded),
        failed: safeNumber(workloads.summary?.jobs?.failed),
      },
    },
    podStatus: {
      total: safeNumber(workloads.podStatus?.total),
      running: safeNumber(workloads.podStatus?.running),
      pending: safeNumber(workloads.podStatus?.pending),
      failed: safeNumber(workloads.podStatus?.failed),
      succeeded: safeNumber(workloads.podStatus?.succeeded),
      runningPercent: safePercent(workloads.podStatus?.runningPercent),
      pendingPercent: safePercent(workloads.podStatus?.pendingPercent),
      failedPercent: safePercent(workloads.podStatus?.failedPercent),
      succeededPercent: safePercent(workloads.podStatus?.succeededPercent),
    },
    peakStats: {
      peakCpu: safeNumber(workloads.peakStats?.peakCpu),
      peakCpuNode: safeString(workloads.peakStats?.peakCpuNode),
      peakMem: safeNumber(workloads.peakStats?.peakMem),
      peakMemNode: safeString(workloads.peakStats?.peakMemNode),
      hasData: workloads.peakStats?.hasData ?? false,
    },
    alertTrends: (alerts.trend || [])
      .map((p) => ({
        ts: safeTimestamp(p.at),
        kinds: (p.kinds || {}) as Record<string, number>,
      }))
      .filter((it) => it.ts > 0),
    recentAlerts: (alerts.recent || []).map((x) => ({
      time: safeString(x.timestamp),
      severity: safeString(x.severity),
      kind: safeString(x.kind),
      namespace: safeString(x.namespace),
      message: safeString(x.message),
      reason: safeString(x.reason),
      name: safeString(x.name),
    })),
    nodeUsages: (nodes.usage || []).map((it) => ({
      nodeName: safeString(it.node),
      cpuPercent: safePercent(it.cpuUsage),
      memoryPercent: safePercent(it.memUsage),
    })),
  };
}
