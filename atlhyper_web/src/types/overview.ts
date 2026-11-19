// types/overview.ts
// 集群概览（getClusterOverview）：接口类型由 api/openapi.yaml 生成（types/api.gen.ts），这里只起常用的名字；
// 页面用的转换结果（TransformedOverview）在下方手写。

import type { components } from "./api.gen";
import type { HardwareStatus } from "./hardware";

type Schemas = components["schemas"];

/** 集群健康结论（后端判定） */
export type ClusterHealthStatus = Schemas["OverviewClusterHealth"]["status"];

/** 健康原因代码（后端判定，文案在 i18n overview.healthReasons） */
export type ClusterHealthReason = NonNullable<Schemas["OverviewClusterHealth"]["reason"]>;

export type ClusterHealth = Schemas["OverviewClusterHealth"];
export type NodeReady = Schemas["OverviewResourceReady"];
export type ResourceUsage = Schemas["OverviewPercent"];
export type OverviewCards = Schemas["OverviewCards"];
export type WorkloadStatus = Schemas["OverviewWorkloadStatus"];
export type JobStatus = Schemas["OverviewJobStatus"];
export type WorkloadSummary = Schemas["OverviewWorkloadSummary"];
export type PodStatusDistribution = Schemas["OverviewPodStatus"];
export type PeakStats = Schemas["OverviewPeakStats"];
export type WorkloadsData = Schemas["OverviewWorkloads"];
export type AlertTrendPoint = Schemas["OverviewAlertTrend"];
export type AlertTotals = Schemas["OverviewAlertTotals"];
export type RecentAlert = Schemas["OverviewRecentAlert"];
export type AlertsData = Schemas["OverviewAlerts"];
export type NodeUsage = Schemas["OverviewNodeUsage"];
export type NodesData = Schemas["OverviewNodes"];
export type ClusterOverview = Schemas["ClusterOverview"];

/** 页面显示用：数据还没到时为 Unknown（后端不返回这个值） */
export type HealthDisplayStatus = ClusterHealthStatus | "Unknown";

// 转换后的概览数据（用于页面展示）
export interface TransformedOverview {
  clusterId: string;
  healthCard: {
    status: HealthDisplayStatus;
    reason: ClusterHealthReason | "";
    nodeReadyPct: number;
    podHealthyPct: number;
    nodeReadyStatus: HealthDisplayStatus;
    podReadyStatus: HealthDisplayStatus;
  };
  nodesCard: {
    totalNodes: number;
    readyNodes: number;
    nodeReadyPct: number;
  };
  cpuCard: {
    percent: number;
  };
  memCard: {
    percent: number;
  };
  alertsTotal: number;
  // 工作负载统计
  workloads: {
    deployments: { total: number; ready: number };
    daemonsets: { total: number; ready: number };
    statefulsets: { total: number; ready: number };
    jobs: { total: number; running: number; succeeded: number; failed: number };
  };
  podStatus: {
    total: number;
    running: number;
    pending: number;
    failed: number;
    succeeded: number;
    runningPercent: number;
    pendingPercent: number;
    failedPercent: number;
    succeededPercent: number;
  };
  peakStats: {
    peakCpu: number;
    peakCpuNode: string;
    peakMem: number;
    peakMemNode: string;
    hasData: boolean;
  };
  alertTrends: {
    ts: number;
    kinds: Record<string, number>; // 按资源类型统计
  }[];
  recentAlerts: {
    time: string;
    severity: string;
    kind: string;
    namespace: string;
    message: string;
    reason: string;
    name: string;
  }[];
  nodeUsages: {
    nodeName: string;
    cpuPercent: number;
    memoryPercent: number;
    cpuLevel: HardwareStatus;
    memoryLevel: HardwareStatus;
  }[];
}
