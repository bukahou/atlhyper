// types/hardware.ts
// 硬件健康矩阵（getHardwareHealth）：类型由 api/openapi.yaml 生成（types/api.gen.ts），这里只起常用的名字。
//
// 与 types/node-metrics.ts 的区别：那边是节点上报的原始读数，
// 这边是 Master 判定完的结果，每格自带 status。前端不做任何阈值比较。
// 格为 null = 该节点没有这个传感器 → 显示「无数据」，既不是正常也不是故障。

import type { components } from "./api.gen";

type Schemas = components["schemas"];

/** unknown = 没有这个读数（不是正常，也不是故障） */
export type HardwareStatus = Schemas["HardwareRow"]["overall"];

export type HardwareTempCell = Schemas["HardwareTempCell"];
export type HardwareUndervoltCell = Schemas["HardwareUndervoltCell"];
export type HardwareFanCell = Schemas["HardwareFanCell"];
export type HardwareFreqCell = Schemas["HardwareFreqCell"];
export type HardwareAwaitCell = Schemas["HardwareAwaitCell"];
export type HardwareSensorCell = Schemas["HardwareSensorCell"];
export type HardwareUsageCell = Schemas["HardwareUsageCell"];
export type HardwareRow = Schemas["HardwareRow"];
export type HardwareMaxTemp = Schemas["HardwareMaxTemp"];
export type HardwareMaxAwait = Schemas["HardwareMaxAwait"];
export type HardwareSummary = Schemas["HardwareSummary"];
export type HardwareHealth = Schemas["HardwareHealthResponse"];
