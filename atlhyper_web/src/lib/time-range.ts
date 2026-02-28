/**
 * 时间范围选择器工具函数
 */

import type { PresetKey, RelativeUnit, TimeRangeSelection } from "@/types/time-range";

/** 预设 key → Go duration string (用于 API since 参数) */
const PRESET_SINCE: Record<PresetKey, string> = {
  "15min": "15m",
  "1h": "1h",
  "24h": "24h",
  "7d": "168h",
  "15d": "360h",
  "30d": "720h",
};

/** 预设 key → 毫秒数 */
const PRESET_MS: Record<PresetKey, number> = {
  "15min": 15 * 60_000,
  "1h": 3_600_000,
  "24h": 86_400_000,
  "7d": 7 * 86_400_000,
  "15d": 15 * 86_400_000,
  "30d": 30 * 86_400_000,
};

/** 单位 → 毫秒 */
const UNIT_MS: Record<RelativeUnit, number> = {
  m: 60_000,
  h: 3_600_000,
  d: 86_400_000,
};

/**
 * 转换为 Go duration 字符串 (since 参数)
 * absolute 模式返回 undefined（使用 start_time/end_time 代替）
 */
export function toSince(sel: TimeRangeSelection): string | undefined {
  switch (sel.mode) {
    case "preset":
      return PRESET_SINCE[sel.preset];
    case "custom": {
      // 统一转换为分钟（Go duration 格式）
      const totalMs = sel.value * UNIT_MS[sel.unit];
      const totalMinutes = Math.round(totalMs / 60_000);
      if (totalMinutes >= 60 && totalMinutes % 60 === 0) {
        return `${totalMinutes / 60}h`;
      }
      return `${totalMinutes}m`;
    }
    case "absolute":
      return undefined;
  }
}

/**
 * 转换为绝对时间参数 (start_time/end_time ISO 字符串)
 * 非 absolute 模式返回空对象
 */
export function toAbsoluteParams(sel: TimeRangeSelection): { startTime?: string; endTime?: string } {
  if (sel.mode !== "absolute") return {};
  return {
    startTime: new Date(sel.start).toISOString(),
    endTime: new Date(sel.end).toISOString(),
  };
}

/**
 * 转换为时间跨度毫秒数（用于直方图 X 轴标签精度判断）
 */
export function toSpanMs(sel: TimeRangeSelection): number {
  switch (sel.mode) {
    case "preset":
      return PRESET_MS[sel.preset];
    case "custom":
      return sel.value * UNIT_MS[sel.unit];
    case "absolute":
      return sel.end - sel.start;
  }
}

/**
 * 生成显示标签
 */
export function toDisplayLabel(
  sel: TimeRangeSelection,
  presetLabels: Record<PresetKey, string>,
  unitLabels: { minutes: string; hours: string; days: string },
): string {
  switch (sel.mode) {
    case "preset":
      return presetLabels[sel.preset];
    case "custom": {
      const unitLabel =
        sel.unit === "m" ? unitLabels.minutes :
        sel.unit === "h" ? unitLabels.hours :
        unitLabels.days;
      return `${sel.value} ${unitLabel}`;
    }
    case "absolute": {
      const fmt = (ts: number) => {
        const d = new Date(ts);
        const month = d.getMonth() + 1;
        const day = d.getDate();
        const h = String(d.getHours()).padStart(2, "0");
        const m = String(d.getMinutes()).padStart(2, "0");
        return `${month}/${day} ${h}:${m}`;
      };
      return `${fmt(sel.start)} — ${fmt(sel.end)}`;
    }
  }
}
