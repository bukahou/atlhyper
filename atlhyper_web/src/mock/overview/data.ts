/**
 * Overview Mock 数据 — 从现有 cluster mock 聚合生成 ClusterOverview
 *
 * 不创建新数据，完全基于 mock/cluster/ 中已有的
 * MOCK_PODS, MOCK_NODES, MOCK_DEPLOYMENTS, MOCK_EVENTS 等聚合计算。
 */

import type { ClusterOverview } from "@/types/overview";
import {
  MOCK_PODS, MOCK_NODES, MOCK_DEPLOYMENTS, MOCK_EVENTS,
} from "../cluster/data";
import {
  MOCK_STATEFULSETS, MOCK_DAEMONSETS, MOCK_JOBS,
} from "../cluster/data-extra";

// ============================================================
// 聚合 helpers
// ============================================================

function parseReplicas(replicas: string): { ready: number; desired: number } {
  const parts = replicas.split("/");
  return {
    ready: parseInt(parts[0], 10) || 0,
    desired: parseInt(parts[1] || parts[0], 10) || 0,
  };
}

// ============================================================
// 生成 ClusterOverview
// ============================================================

function buildClusterOverview(): ClusterOverview {
  const clusterId = "zgmf-x10a";

  // --- Cards ---
  const totalNodes = MOCK_NODES.length;
  const readyNodes = MOCK_NODES.filter((n) => n.ready).length;
  const nodeReadyPct = totalNodes > 0 ? (readyNodes / totalNodes) * 100 : 0;

  const totalPods = MOCK_PODS.length;
  const runningPods = MOCK_PODS.filter((p) => p.phase === "Running").length;
  const podReadyPct = totalPods > 0 ? (runningPods / totalPods) * 100 : 0;

  // CPU/Mem: 基于 MOCK_NODES 模拟使用率
  // raspi-nfs (control-plane): CPU 35%, Mem 45%
  // jegan-worker-01: CPU 42%, Mem 62%
  // jegan-worker-02: CPU 28%, Mem 55%
  const nodeUsages = [
    { node: "raspi-nfs", cpuUsage: 35, memUsage: 45 },
    { node: "jegan-worker-01", cpuUsage: 42, memUsage: 62 },
    { node: "jegan-worker-02", cpuUsage: 28, memUsage: 55 },
  ];
  const avgCpu = nodeUsages.reduce((s, n) => s + n.cpuUsage, 0) / nodeUsages.length;
  const avgMem = nodeUsages.reduce((s, n) => s + n.memUsage, 0) / nodeUsages.length;

  // Events: Warning 事件数
  const warningEvents = MOCK_EVENTS.filter((e) => e.severity === "Warning");

  // --- Workloads ---
  const depReady = MOCK_DEPLOYMENTS.reduce((s, d) => s + parseReplicas(d.replicas).ready, 0);
  const depDesired = MOCK_DEPLOYMENTS.reduce((s, d) => s + parseReplicas(d.replicas).desired, 0);

  const stsReady = MOCK_STATEFULSETS.reduce((s, st) => s + st.ready, 0);
  const stsTotal = MOCK_STATEFULSETS.reduce((s, st) => s + st.replicas, 0);

  const dsReady = MOCK_DAEMONSETS.reduce((s, ds) => s + ds.ready, 0);
  const dsDesired = MOCK_DAEMONSETS.reduce((s, ds) => s + ds.desired, 0);

  const jobsSucceeded = MOCK_JOBS.filter((j) => j.complete).length;
  const jobsFailed = MOCK_JOBS.filter((j) => j.failed > 0).length;
  const jobsRunning = MOCK_JOBS.filter((j) => j.active > 0).length;

  // Pod 状态分布
  const podRunning = MOCK_PODS.filter((p) => p.phase === "Running").length;
  const podPending = MOCK_PODS.filter((p) => p.phase === "Pending").length;
  const podFailed = MOCK_PODS.filter((p) => p.phase === "Failed").length;
  const podSucceeded = MOCK_PODS.filter((p) => p.phase === "Succeeded").length;
  const podUnknown = MOCK_PODS.filter((p) => p.phase === "Unknown").length;

  // 峰值统计
  const peakCpuNode = nodeUsages.reduce((a, b) => a.cpuUsage > b.cpuUsage ? a : b);
  const peakMemNode = nodeUsages.reduce((a, b) => a.memUsage > b.memUsage ? a : b);

  // --- Alerts ---
  // 按 4h 窗口聚合告警趋势（最近 24h，6 个点）
  const now = new Date("2026-02-21T10:00:00Z");
  const trend = Array.from({ length: 6 }, (_, i) => {
    const windowEnd = new Date(now.getTime() - i * 4 * 3600_000);
    const windowStart = new Date(windowEnd.getTime() - 4 * 3600_000);
    const windowEvents = warningEvents.filter((e) => {
      const t = new Date(e.eventTime).getTime();
      return t >= windowStart.getTime() && t < windowEnd.getTime();
    });
    const kinds: Record<string, number> = {};
    windowEvents.forEach((e) => {
      kinds[e.kind] = (kinds[e.kind] || 0) + 1;
    });
    return { at: windowStart.toISOString(), kinds };
  }).reverse();

  // 最近告警（Warning 事件，按时间倒序，取前 10）
  const recentAlerts = [...warningEvents]
    .sort((a, b) => new Date(b.eventTime).getTime() - new Date(a.eventTime).getTime())
    .slice(0, 10)
    .map((e) => ({
      timestamp: e.eventTime,
      severity: "warning" as const,
      kind: e.kind,
      namespace: e.namespace,
      name: e.name,
      message: e.message,
      reason: e.reason,
    }));

  // --- Health ---
  const hasWarning = warningEvents.length > 0;
  const allNodesReady = readyNodes === totalNodes;
  const healthStatus = !allNodesReady ? "Degraded" as const
    : hasWarning ? "Healthy" as const  // 有 warning 但节点全 ready 仍为 Healthy
    : "Healthy" as const;

  return {
    clusterId,
    cards: {
      clusterHealth: {
        status: healthStatus,
        reason: healthStatus === "Degraded" ? "Some nodes are not ready" : undefined,
        nodeReadyPercent: Math.round(nodeReadyPct * 100) / 100,
        podReadyPercent: Math.round(podReadyPct * 100) / 100,
      },
      nodeReady: {
        total: totalNodes,
        ready: readyNodes,
        percent: Math.round(nodeReadyPct * 100) / 100,
      },
      cpuUsage: { percent: Math.round(avgCpu * 100) / 100 },
      memUsage: { percent: Math.round(avgMem * 100) / 100 },
      events24h: warningEvents.length,
    },
    workloads: {
      summary: {
        deployments: { total: MOCK_DEPLOYMENTS.length, ready: depReady >= depDesired ? MOCK_DEPLOYMENTS.length : MOCK_DEPLOYMENTS.length - 1 },
        daemonsets: { total: MOCK_DAEMONSETS.length, ready: dsReady >= dsDesired ? MOCK_DAEMONSETS.length : MOCK_DAEMONSETS.length - 1 },
        statefulsets: { total: MOCK_STATEFULSETS.length, ready: stsReady >= stsTotal ? MOCK_STATEFULSETS.length : MOCK_STATEFULSETS.length - 1 },
        jobs: { total: MOCK_JOBS.length, running: jobsRunning, succeeded: jobsSucceeded, failed: jobsFailed },
      },
      podStatus: {
        total: totalPods,
        running: podRunning,
        pending: podPending,
        failed: podFailed,
        succeeded: podSucceeded,
        unknown: podUnknown,
        runningPercent: totalPods > 0 ? Math.round((podRunning / totalPods) * 10000) / 100 : 0,
        pendingPercent: totalPods > 0 ? Math.round((podPending / totalPods) * 10000) / 100 : 0,
        failedPercent: totalPods > 0 ? Math.round((podFailed / totalPods) * 10000) / 100 : 0,
        succeededPercent: totalPods > 0 ? Math.round((podSucceeded / totalPods) * 10000) / 100 : 0,
      },
      peakStats: {
        peakCpu: peakCpuNode.cpuUsage,
        peakCpuNode: peakCpuNode.node,
        peakMem: peakMemNode.memUsage,
        peakMemNode: peakMemNode.node,
        hasData: true,
      },
    },
    alerts: {
      trend,
      totals: {
        critical: 0,
        warning: warningEvents.length,
        info: MOCK_EVENTS.filter((e) => e.severity === "Normal").length,
      },
      recent: recentAlerts,
    },
    nodes: {
      usage: nodeUsages,
    },
  };
}

export const MOCK_CLUSTER_OVERVIEW: ClusterOverview = buildClusterOverview();
