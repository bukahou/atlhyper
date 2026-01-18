/**
 * 集群概览相关类型定义
 */

// 集群健康状态
export interface ClusterHealth {
  status: "Healthy" | "Degraded" | "Unhealthy" | "Unknown";
  reason?: string;
  nodeReadyPercent: number;
  podReadyPercent: number;
}

// 节点就绪统计
export interface NodeReady {
  total: number;
  ready: number;
  percent: number;
}

// 资源使用率
export interface ResourceUsage {
  percent: number;
}

// 概览卡片数据
export interface OverviewCards {
  clusterHealth: ClusterHealth;
  nodeReady: NodeReady;
  cpuUsage: ResourceUsage;
  memUsage: ResourceUsage;
  events24h: number;
}

// 工作负载状态
export interface WorkloadStatus {
  total: number;
  ready: number;
}

// Job 状态
export interface JobStatus {
  total: number;
  running: number;
  succeeded: number;
  failed: number;
}

// 工作负载汇总
export interface WorkloadSummary {
  deployments: WorkloadStatus;
  daemonsets: WorkloadStatus;
  statefulsets: WorkloadStatus;
  jobs: JobStatus;
}

// Pod 状态分布
export interface PodStatusDistribution {
  total: number;
  running: number;
  pending: number;
  failed: number;
  succeeded: number;
  unknown: number;
  runningPercent: number;
  pendingPercent: number;
  failedPercent: number;
  succeededPercent: number;
}

// 峰值统计
export interface PeakStats {
  peakCpu: number;
  peakCpuNode: string;
  peakMem: number;
  peakMemNode: string;
  hasData: boolean;
}

// 工作负载数据
export interface WorkloadsData {
  summary: WorkloadSummary;
  podStatus: PodStatusDistribution;
  peakStats?: PeakStats;
}

// 告警趋势点（按资源类型统计）
export interface AlertTrendPoint {
  at: string;
  kinds: Record<string, number>; // 每种资源类型的告警数量: {"Pod": 5, "Node": 2}
}

// 告警统计
export interface AlertTotals {
  critical: number;
  warning: number;
  info: number;
}

// 最近告警（后端返回小写字段名）
export interface RecentAlert {
  timestamp: string;
  severity: "critical" | "warning" | "info";
  kind: string;
  namespace: string;
  name: string;
  message: string;
  reason: string;
}

// 告警数据
export interface AlertsData {
  trend: AlertTrendPoint[];
  totals: AlertTotals;
  recent: RecentAlert[];
}

// 节点资源使用
export interface NodeUsage {
  node: string;
  cpuUsage: number;
  memUsage: number;
}

// 节点数据
export interface NodesData {
  usage: NodeUsage[];
}

// 集群概览完整数据
export interface ClusterOverview {
  clusterId: string;
  cards: OverviewCards;
  workloads: WorkloadsData;
  alerts: AlertsData;
  nodes: NodesData;
}

// 转换后的概览数据（用于页面展示）
export interface TransformedOverview {
  clusterId: string;
  healthCard: {
    status: string;
    reason: string;
    nodeReadyPct: number;
    podHealthyPct: number;
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
  }[];
}
