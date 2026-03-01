import type { Point } from "@/types/node-metrics";
import { MIN_AUTO_POINTS } from "./chartConstants";

// ==================== Theme ====================

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

// ==================== Data Processing ====================

/** Downsample Point[] by averaging within fixed-size time buckets */
export function aggregatePoints(
  points: Point[],
  intervalSec: number,
): [number, number][] {
  if (points.length === 0) return [];
  // 30s raw = no aggregation needed
  if (intervalSec <= 30) {
    return points.map((p) => [new Date(p.timestamp).getTime(), p.value]);
  }

  const intervalMs = intervalSec * 1000;
  const buckets = new Map<number, { sum: number; count: number }>();

  for (const p of points) {
    const ts = new Date(p.timestamp).getTime();
    const bucketKey = Math.floor(ts / intervalMs) * intervalMs;
    const existing = buckets.get(bucketKey);
    if (existing) {
      existing.sum += p.value;
      existing.count += 1;
    } else {
      buckets.set(bucketKey, { sum: p.value, count: 1 });
    }
  }

  const result: [number, number][] = [];
  for (const [ts, { sum, count }] of buckets) {
    result.push([ts + intervalMs / 2, sum / count]); // center of bucket
  }
  result.sort((a, b) => a[0] - b[0]);
  return result;
}

/** Pick the largest interval that yields >= MIN_AUTO_POINTS from the data span */
export function pickAdaptiveInterval(
  dataSpanMs: number,
  intervals: { label: string; seconds: number }[],
): number {
  // Iterate from largest to smallest
  for (let i = intervals.length - 1; i >= 0; i--) {
    if (dataSpanMs / (intervals[i].seconds * 1000) >= MIN_AUTO_POINTS) {
      return intervals[i].seconds;
    }
  }
  return intervals[0].seconds; // fallback: smallest
}
