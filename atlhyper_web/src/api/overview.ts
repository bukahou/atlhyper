/**
 * 集群概览 API
 *
 * 适配 Master V2 API
 */

import { get } from "./request";
import type { ClusterOverview } from "@/types/overview";

// ============================================================
// 查询参数类型
// ============================================================

interface OverviewParams {
  cluster_id: string;
}

// ============================================================
// 响应类型（Master V2 直接返回数据）
// ============================================================

// Master V2 返回的数据结构与前端 ClusterOverview 略有不同
// 需要适配字段名称（snake_case → camelCase）
interface OverviewApiResponse {
  cluster_id: string;
  cards: {
    cluster_health: {
      status: string;
      reason?: string;
      node_ready_percent: number;
      pod_ready_percent: number;
    };
    node_ready: {
      total: number;
      ready: number;
      percent: number;
    };
    cpu_usage: {
      percent: number;
    };
    mem_usage: {
      percent: number;
    };
    events_24h: number;
  };
  workloads: {
    summary: {
      deployments: { total: number; ready: number };
      daemonsets: { total: number; ready: number };
      statefulsets: { total: number; ready: number };
      jobs: { total: number; running: number; succeeded: number; failed: number };
    };
    pod_status: {
      total: number;
      running: number;
      pending: number;
      failed: number;
      succeeded: number;
      unknown: number;
      running_percent: number;
      pending_percent: number;
      failed_percent: number;
      succeeded_percent: number;
    };
    peak_stats?: {
      peak_cpu: number;
      peak_cpu_node: string;
      peak_mem: number;
      peak_mem_node: string;
      has_data: boolean;
    };
  };
  alerts: {
    trend: Array<{
      at: string;
      kinds: Record<string, number>; // 按资源类型统计
    }>;
    totals: {
      critical: number;
      warning: number;
      info: number;
    };
    recent: Array<{
      timestamp: string;
      severity: string;
      kind: string;
      namespace: string;
      name: string;
      message: string;
      reason: string;
    }>;
  };
  nodes: {
    usage: Array<{
      node: string;
      cpu_usage: number;
      mem_usage: number;
    }>;
  };
}

// ============================================================
// 数据转换函数
// ============================================================

/**
 * 将 API 响应转换为前端类型
 * snake_case → camelCase
 */
function transformResponse(data: OverviewApiResponse): ClusterOverview {
  return {
    clusterId: data.cluster_id,
    cards: {
      clusterHealth: {
        status: data.cards.cluster_health.status as "Healthy" | "Degraded" | "Unhealthy" | "Unknown",
        reason: data.cards.cluster_health.reason,
        nodeReadyPercent: data.cards.cluster_health.node_ready_percent,
        podReadyPercent: data.cards.cluster_health.pod_ready_percent,
      },
      nodeReady: {
        total: data.cards.node_ready.total,
        ready: data.cards.node_ready.ready,
        percent: data.cards.node_ready.percent,
      },
      cpuUsage: {
        percent: data.cards.cpu_usage.percent,
      },
      memUsage: {
        percent: data.cards.mem_usage.percent,
      },
      events24h: data.cards.events_24h,
    },
    workloads: {
      summary: {
        deployments: data.workloads.summary.deployments,
        daemonsets: data.workloads.summary.daemonsets,
        statefulsets: data.workloads.summary.statefulsets,
        jobs: data.workloads.summary.jobs,
      },
      podStatus: {
        total: data.workloads.pod_status.total,
        running: data.workloads.pod_status.running,
        pending: data.workloads.pod_status.pending,
        failed: data.workloads.pod_status.failed,
        succeeded: data.workloads.pod_status.succeeded,
        unknown: data.workloads.pod_status.unknown,
        runningPercent: data.workloads.pod_status.running_percent,
        pendingPercent: data.workloads.pod_status.pending_percent,
        failedPercent: data.workloads.pod_status.failed_percent,
        succeededPercent: data.workloads.pod_status.succeeded_percent,
      },
      peakStats: data.workloads.peak_stats
        ? {
            peakCpu: data.workloads.peak_stats.peak_cpu,
            peakCpuNode: data.workloads.peak_stats.peak_cpu_node,
            peakMem: data.workloads.peak_stats.peak_mem,
            peakMemNode: data.workloads.peak_stats.peak_mem_node,
            hasData: data.workloads.peak_stats.has_data,
          }
        : undefined,
    },
    alerts: {
      trend: data.alerts.trend.map((item) => ({
        at: item.at,
        kinds: item.kinds || {},
      })),
      totals: {
        critical: data.alerts.totals.critical,
        warning: data.alerts.totals.warning,
        info: data.alerts.totals.info,
      },
      recent: data.alerts.recent.map((item) => ({
        timestamp: item.timestamp,
        severity: item.severity as "critical" | "warning" | "info",
        kind: item.kind,
        namespace: item.namespace,
        name: item.name,
        message: item.message,
        reason: item.reason,
      })),
    },
    nodes: {
      usage: data.nodes.usage.map((item) => ({
        node: item.node,
        cpuUsage: item.cpu_usage,
        memUsage: item.mem_usage,
      })),
    },
  };
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取集群概览
 * GET /api/v2/overview?cluster_id=xxx
 */
export async function getClusterOverview(params: OverviewParams) {
  const response = await get<OverviewApiResponse>("/api/v2/overview", params);
  return {
    ...response,
    data: {
      data: transformResponse(response.data),
    },
  };
}

// ============================================================
// 兼容旧接口（后续移除）
// ============================================================

/** @deprecated 使用新参数格式 */
export function getClusterOverviewLegacy(data: { ClusterID: string }) {
  return getClusterOverview({ cluster_id: data.ClusterID });
}
