/**
 * 时间范围选择器类型定义
 */

export type PresetKey = "15min" | "1h" | "24h" | "7d" | "15d" | "30d";
export type RelativeUnit = "m" | "h" | "d";

export interface PresetTimeRange {
  mode: "preset";
  preset: PresetKey;
}

export interface CustomTimeRange {
  mode: "custom";
  value: number;
  unit: RelativeUnit;
}

export interface AbsoluteTimeRange {
  mode: "absolute";
  start: number; // epoch ms
  end: number;   // epoch ms
}

export type TimeRangeSelection = PresetTimeRange | CustomTimeRange | AbsoluteTimeRange;
