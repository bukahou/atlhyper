/**
 * 日志直方图的纯计算：级别配色、主题色、补齐空桶的时间线、X 轴标签
 */
import type { LogHistogramBucket } from "@/types/model/log";

export const SEVERITY_COLORS: Record<string, string> = {
  ERROR: "#ef4444",
  WARN: "#f59e0b",
  INFO: "#3b82f6",
  DEBUG: "#6b7280",
};

export const SEVERITY_ORDER = ["DEBUG", "INFO", "WARN", "ERROR"];

export function getThemeColors() {
  const isDark = document.documentElement.classList.contains("dark");
  return {
    textColor: isDark ? "#9ca3af" : "#6b7280",
    lineColor: isDark ? "#374151" : "#e5e7eb",
    splitLineColor: isDark ? "#1f2937" : "#f3f4f6",
    tooltipBg: isDark ? "#1f2937" : "#fff",
    tooltipBorder: isDark ? "#374151" : "#e5e7eb",
    tooltipText: isDark ? "#e5e7eb" : "#111827",
  };
}

/**
 * Build full timeline with empty buckets filled in.
 *
 * ClickHouse GROUP BY only returns non-empty buckets.
 * We need ALL buckets (including zeros) for the category axis to be uniformly spaced,
 * so brush selection accurately maps to time ranges.
 */
export function buildFullTimeline(
  data: LogHistogramBucket[],
  intervalMs: number,
): { allTimes: number[]; countMap: Map<string, number> } {
  // Collect data points
  const countMap = new Map<string, number>();
  let minTime = Infinity;
  let maxTime = -Infinity;
  for (const b of data) {
    const ts = new Date(b.timestamp).getTime();
    if (ts < minTime) minTime = ts;
    if (ts > maxTime) maxTime = ts;
    const key = `${ts}|${b.severity.toUpperCase()}`;
    countMap.set(key, (countMap.get(key) ?? 0) + b.count);
  }

  // Generate all bucket timestamps from min to max (fill gaps)
  const allTimes: number[] = [];
  for (let t = minTime; t <= maxTime; t += intervalMs) {
    allTimes.push(t);
  }
  // Edge case: ensure at least the data points are present
  if (allTimes.length === 0 && data.length > 0) {
    allTimes.push(minTime);
  }

  return { allTimes, countMap };
}

/** X 轴标签：跨度越长精度越粗（秒 → 分 → 日期 + 时间 → 日期） */
export function formatBucketLabels(bucketTimes: number[], viewSpanMs: number): string[] {
  const FIVE_MIN = 5 * 60_000;
  const ONE_DAY = 86_400_000;
  const THIRTY_DAYS = 30 * ONE_DAY;
  return bucketTimes.map((t) => {
    const d = new Date(t);
    if (viewSpanMs < FIVE_MIN) {
      return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
    }
    if (viewSpanMs < ONE_DAY) {
      return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    }
    if (viewSpanMs < THIRTY_DAYS) {
      const time = d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
      return `${d.getMonth() + 1}/${d.getDate()} ${time}`;
    }
    return `${d.getMonth() + 1}/${d.getDate()}`;
  });
}
