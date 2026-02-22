/**
 * 节点硬件指标 — Mock 查询函数
 *
 * 类型对齐 model_v3/metrics/node_metrics.go
 */

import type { NodeMetrics, Point, Summary } from "@/types/node-metrics";
import { MOCK_NODES } from "./data";

// ============================================================================
// 汇总计算 — 对齐 Go Summary (7 个字段)
// ============================================================================

function computeSummary(nodes: NodeMetrics[]): Summary {
  const n = nodes.length;
  if (n === 0) {
    return {
      totalNodes: 0, onlineNodes: 0,
      avgCpuPct: 0, avgMemPct: 0,
      maxCpuPct: 0, maxMemPct: 0,
      maxCpuTemp: 0,
    };
  }

  let sumCPU = 0, maxCPU = 0;
  let sumMem = 0, maxMem = 0;
  let maxTemp = 0;

  for (const node of nodes) {
    sumCPU += node.cpu.usagePct;
    maxCPU = Math.max(maxCPU, node.cpu.usagePct);

    sumMem += node.memory.usagePct;
    maxMem = Math.max(maxMem, node.memory.usagePct);

    maxTemp = Math.max(maxTemp, node.temperature.cpuTempC);
  }

  return {
    totalNodes: n,
    onlineNodes: n,
    avgCpuPct: sumCPU / n,
    avgMemPct: sumMem / n,
    maxCpuPct: maxCPU,
    maxMemPct: maxMem,
    maxCpuTemp: maxTemp,
  };
}

// ============================================================================
// 历史数据生成器（带波动模拟）— 返回 Point[]
// ============================================================================

/** 基于节点特征生成带自然波动的历史数据 */
function generateHistory(node: NodeMetrics, hours: number, metric: string): Point[] {
  const now = Date.now();
  const intervalMs = 30_000; // 30秒一个数据点
  const totalPoints = Math.floor((hours * 3600_000) / intervalMs);
  const data: Point[] = [];

  // 基准值
  const baseValues: Record<string, number> = {
    cpu: node.cpu.usagePct,
    memory: node.memory.usagePct,
    disk: (node.disks.find(d => d.mountPoint === "/") || node.disks[0])?.usagePct || 50,
    temp: node.temperature.cpuTempC,
  };
  const base = baseValues[metric] ?? baseValues.cpu;

  // 使用简单的正弦波 + 随机噪声模拟日变化
  for (let i = 0; i < totalPoints; i++) {
    const ts = now - (totalPoints - i) * intervalMs;
    const hourOfDay = new Date(ts).getHours();

    // 日间高、夜间低的权重 (0~1)
    const dayWeight = 0.6 + 0.4 * Math.sin((hourOfDay - 6) * Math.PI / 12);

    // 随机噪声 (-1 ~ 1)
    const noise = (Math.random() - 0.5) * 2;

    let value: number;
    switch (metric) {
      case "cpu":
        value = clamp(base * dayWeight + noise * 8, 2, 98);
        break;
      case "memory":
        value = clamp(base + noise * 3 + (dayWeight - 0.5) * 5, 10, 98);
        break;
      case "disk":
        value = clamp(base + (i / totalPoints) * 0.5 + noise * 0.3, 5, 98);
        break;
      case "temp":
        value = clamp(base * (0.7 + 0.3 * dayWeight) + noise * 3, 25, 95);
        break;
      default:
        value = clamp(base * dayWeight + noise * 5, 0, 100);
    }

    data.push({
      timestamp: new Date(ts).toISOString(),
      value: round(value),
    });
  }

  return data;
}

function clamp(v: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, v));
}

function round(v: number, d = 1): number {
  const f = 10 ** d;
  return Math.round(v * f) / f;
}

// ============================================================================
// 历史数据缓存
// ============================================================================

const historyCache: Record<string, { hours: number; data: Record<string, Point[]> }> = {};

function getCachedHistory(nodeName: string, hours: number): Record<string, Point[]> {
  const cached = historyCache[nodeName];
  if (cached && cached.hours >= hours) {
    const cutoff = Date.now() - hours * 3600_000;
    const result: Record<string, Point[]> = {};
    for (const [metric, points] of Object.entries(cached.data)) {
      result[metric] = points.filter(p => new Date(p.timestamp).getTime() >= cutoff);
    }
    return result;
  }

  const node = MOCK_NODES.find(n => n.nodeName === nodeName);
  if (!node) return {};

  const genHours = Math.max(hours, 168);
  const metrics = ["cpu", "memory", "disk", "temp"];
  const data: Record<string, Point[]> = {};
  for (const m of metrics) {
    data[m] = generateHistory(node, genHours, m);
  }
  historyCache[nodeName] = { hours: genHours, data };

  const cutoff = Date.now() - hours * 3600_000;
  const result: Record<string, Point[]> = {};
  for (const [metric, points] of Object.entries(data)) {
    result[metric] = points.filter(p => new Date(p.timestamp).getTime() >= cutoff);
  }
  return result;
}

// ============================================================================
// Mock API 函数
// ============================================================================

export interface MockClusterNodeMetricsResult {
  summary: Summary;
  nodes: NodeMetrics[];
}

export interface MockNodeMetricsHistoryResult {
  nodeName: string;
  start: Date;
  end: Date;
  data: Record<string, Point[]>;
}

/** 获取集群所有节点指标（含汇总） */
export function mockGetClusterNodeMetrics(): MockClusterNodeMetricsResult {
  return {
    summary: computeSummary(MOCK_NODES),
    nodes: MOCK_NODES,
  };
}

/** 获取单节点历史数据 */
export function mockGetNodeMetricsHistory(
  nodeName: string,
  hours: number = 24,
): MockNodeMetricsHistoryResult {
  const data = getCachedHistory(nodeName, hours);
  const now = new Date();
  const start = new Date(now.getTime() - hours * 3600_000);

  return {
    nodeName,
    start,
    end: now,
    data,
  };
}
