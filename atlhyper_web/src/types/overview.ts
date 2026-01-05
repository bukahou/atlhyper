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

// 资源趋势点
export interface ResourceTrendPoint {
  at: string;
  cpuPeak: number;
  memPeak: number;
  tempPeak?: number;
}

// 趋势峰值统计（底部状态卡片）
export interface TrendPeakStats {
  peakCpu: number;       // 当前最高 CPU 使用率 %
  peakCpuNode: string;   // 最高 CPU 节点名
  peakMem: number;       // 当前最高内存使用率 %
  peakMemNode: string;   // 最高内存节点名
  peakTemp: number;      // 当前最高温度
  peakTempNode: string;  // 最高温度节点名
  netRxKBps: number;     // 集群总入流量 KB/s
  netTxKBps: number;     // 集群总出流量 KB/s
  hasData: boolean;      // 是否有 metrics 插件数据
}

// 资源趋势
export interface ResourceTrends {
  resourceUsage: ResourceTrendPoint[];
  peakStats?: TrendPeakStats;
}

// 告警趋势点
export interface AlertTrendPoint {
  at: string;
  critical: number;
  warning: number;
  info: number;
}

// 告警统计
export interface AlertTotals {
  critical: number;
  warning: number;
  info: number;
}

// 最近告警
export interface RecentAlert {
  Timestamp: string;
  Severity: "critical" | "warning" | "info";
  Kind: string;
  Namespace: string;
  Name: string;
  Message: string;
  ReasonCode: string;
  Node?: string;
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
  trends: ResourceTrends;
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
  cpuSeries: [number, number][];
  memSeries: [number, number][];
  tempSeries: [number, number][];
  alertTrends: {
    ts: number;
    critical: number;
    warning: number;
    info: number;
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
  peakStats: {
    peakCpu: number;
    peakCpuNode: string;
    peakMem: number;
    peakMemNode: string;
    peakTemp: number;
    peakTempNode: string;
    netRxKBps: number;
    netTxKBps: number;
    hasData: boolean;
  };
}
