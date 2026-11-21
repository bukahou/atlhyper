// types/node-metrics.ts
// 节点指标类型：节点指标列表接口（listNodeMetrics）的类型由 api/openapi.yaml 生成（types/api.gen.ts），
// 这里只给生成的类型起常用的名字；规格以外的接口（趋势、汇总）仍在下方手写。

import type { components } from "./api.gen";

type Schemas = components["schemas"];

export type NodeCPU = Schemas["NodeCPU"];
export type NodeMemory = Schemas["NodeMemory"];
export type NodeDisk = Schemas["NodeDisk"];
export type NodeNetwork = Schemas["NodeNetwork"];
export type NodeTemperature = Schemas["NodeTemperature"];
export type TempSensor = Schemas["TempSensor"];
export type NodePSI = Schemas["NodePSI"];
export type NodeTCP = Schemas["NodeTCP"];
export type NodeSystem = Schemas["NodeSystem"];
export type NodeVMStat = Schemas["NodeVMStat"];
export type NodeSoftnet = Schemas["NodeSoftnet"];
export type NodeHardware = Schemas["NodeHardware"];
export type FanSensor = Schemas["FanSensor"];
export type CoolingDevice = Schemas["CoolingDevice"];

export type NodeDiskLevels = Schemas["NodeDiskLevels"];
export type NodePSILevels = Schemas["NodePSILevels"];
export type NodeVMStatLevels = Schemas["NodeVMStatLevels"];
export type NodeLevels = Schemas["NodeLevels"];

/** 节点指标列表的一项：上报的读数 + 后端派生值 + 后端判定的等级 */
export type NodeMetricsView = Schemas["NodeMetricsView"];

/** 上报的读数部分（不含后端派生值与等级） */
export type NodeMetrics = Omit<
  NodeMetricsView,
  "primaryDiskIndex" | "cpuLoadPerCore" | "fileFdUsagePct" | "conntrackUsagePct" | "levels"
>;

// ============================================================================
// 时序数据（趋势图用） — 对齐 Go Point / Series
// ============================================================================
export interface Point {
  timestamp: string;   // ISO 8601
  value: number;
}

export interface Series {
  metric: string;
  labels?: Record<string, string>;
  points: Point[];
}

// ============================================================================
// 集群节点指标概览 — 对齐 Go Summary
// ============================================================================
export interface Summary {
  totalNodes: number;
  onlineNodes: number;
  avgCpuPct: number;
  avgMemPct: number;
  maxCpuPct: number;
  maxMemPct: number;
  maxCpuTemp: number;
}
