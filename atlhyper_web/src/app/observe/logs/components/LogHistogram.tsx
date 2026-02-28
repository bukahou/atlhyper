"use client";

import { useRef, useEffect } from "react";
import * as echarts from "echarts";
import type { LogHistogramBucket } from "@/types/model/log";

const SEVERITY_COLORS: Record<string, string> = {
  ERROR: "#ef4444",
  WARN: "#f59e0b",
  INFO: "#3b82f6",
  DEBUG: "#6b7280",
};

const SEVERITY_ORDER = ["DEBUG", "INFO", "WARN", "ERROR"];

function getThemeColors() {
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

interface LogHistogramProps {
  data: LogHistogramBucket[];
  /** Bucket interval in milliseconds (from backend) */
  intervalMs: number;
  title: string;
  /** Time span in milliseconds (for X-axis label precision) */
  timeSpanMs?: number;
  selectedTimeRange: [number, number] | null;
  onTimeRangeSelect?: (range: [number, number] | null) => void;
}

/**
 * Build full timeline with empty buckets filled in.
 *
 * ClickHouse GROUP BY only returns non-empty buckets.
 * We need ALL buckets (including zeros) for the category axis to be uniformly spaced,
 * so brush selection accurately maps to time ranges.
 */
function buildFullTimeline(
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

export function LogHistogram({ data, intervalMs, title, timeSpanMs, selectedTimeRange, onTimeRangeSelect }: LogHistogramProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<echarts.ECharts | null>(null);
  const bucketDataRef = useRef<{ times: number[]; intervalMs: number }>({ times: [], intervalMs: 0 });
  const callbackRef = useRef(onTimeRangeSelect);
  callbackRef.current = onTimeRangeSelect;
  /** Suppress callback when programmatically manipulating brush */
  const suppressCallbackRef = useRef(false);

  // Resize + theme observer (attached once per chart instance)
  const listenersRef = useRef<{ resize: (() => void) | null; observer: MutationObserver | null }>({
    resize: null,
    observer: null,
  });

  // =========================================================================
  // Effect 1: Chart init + data rendering (does NOT depend on selectedTimeRange)
  // =========================================================================
  useEffect(() => {
    if (!containerRef.current || data.length === 0 || intervalMs <= 0) return;

    // Initialize chart if not yet created, or if container DOM changed
    if (chartRef.current && chartRef.current.getDom() !== containerRef.current) {
      if (listenersRef.current.resize) {
        window.removeEventListener("resize", listenersRef.current.resize);
        listenersRef.current.resize = null;
      }
      listenersRef.current.observer?.disconnect();
      listenersRef.current.observer = null;
      chartRef.current.dispose();
      chartRef.current = null;
    }
    if (!chartRef.current) {
      const chart = echarts.init(containerRef.current);
      chartRef.current = chart;

      const handleResize = () => chart.resize();
      window.addEventListener("resize", handleResize);
      listenersRef.current.resize = handleResize;

      // Theme observer
      const observer = new MutationObserver(() => {
        if (!chartRef.current) return;
        const c = getThemeColors();
        chartRef.current.setOption({
          tooltip: { backgroundColor: c.tooltipBg, borderColor: c.tooltipBorder, textStyle: { color: c.tooltipText } },
          xAxis: { axisLine: { lineStyle: { color: c.lineColor } }, axisLabel: { color: c.textColor } },
          yAxis: { axisLabel: { color: c.textColor }, splitLine: { lineStyle: { color: c.splitLineColor } } },
        });
      });
      observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
      listenersRef.current.observer = observer;

      // brushEnd — only fires when user releases mouse (finished selecting)
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      chart.on("brushEnd", (params: any) => {
        if (suppressCallbackRef.current) return;
        const areas = params.areas;
        if (!areas || areas.length === 0) {
          callbackRef.current?.(null);
          return;
        }
        const [startIdx, endIdx] = areas[0].coordRange;
        const { times, intervalMs: intMs } = bucketDataRef.current;
        if (times.length === 0) return;
        const s = Math.max(0, Math.round(startIdx));
        const e = Math.min(times.length - 1, Math.round(endIdx));
        callbackRef.current?.([times[s], times[e]]);
      });

      // brush with empty areas — user clicks to clear selection
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      chart.on("brush", (params: any) => {
        if (suppressCallbackRef.current) return;
        const areas = params.areas;
        if (!areas || areas.length === 0) {
          callbackRef.current?.(null);
        }
      });
    }

    // Suppress all brush callbacks during chart rebuild
    suppressCallbackRef.current = true;

    const c = getThemeColors();

    // Build full timeline with empty buckets filled in
    const { allTimes, countMap } = buildFullTimeline(data, intervalMs);
    bucketDataRef.current = { times: allTimes, intervalMs };

    // Build series data per severity
    const bucketsBySev: Record<string, number[]> = {};
    for (const sev of SEVERITY_ORDER) {
      bucketsBySev[sev] = allTimes.map((t) => countMap.get(`${t}|${sev}`) ?? 0);
    }

    // Adaptive label precision
    const viewSpanMs = timeSpanMs || (allTimes.length > 1 ? allTimes[allTimes.length - 1] - allTimes[0] : 0);
    const FIVE_MIN = 5 * 60_000;
    const ONE_DAY = 86_400_000;
    const THIRTY_DAYS = 30 * ONE_DAY;
    const xLabels = allTimes.map((t) => {
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

    const series = SEVERITY_ORDER.map((sev) => ({
      name: sev,
      type: "bar" as const,
      stack: "total",
      data: bucketsBySev[sev],
      itemStyle: { color: SEVERITY_COLORS[sev], borderRadius: 0 },
      barMaxWidth: 24,
    }));

    // Round top corners on the topmost visible series
    series[series.length - 1].itemStyle.borderRadius = [2, 2, 0, 0] as unknown as number;

    chartRef.current.setOption(
      {
        tooltip: {
          trigger: "axis",
          backgroundColor: c.tooltipBg,
          borderColor: c.tooltipBorder,
          textStyle: { color: c.tooltipText, fontSize: 12 },
        },
        legend: { show: false },
        grid: { top: 8, right: 12, bottom: 24, left: 36 },
        xAxis: {
          type: "category",
          data: xLabels,
          axisLine: { lineStyle: { color: c.lineColor } },
          axisLabel: { color: c.textColor, fontSize: 10 },
          splitLine: { show: false },
        },
        yAxis: {
          type: "value",
          axisLabel: { color: c.textColor, fontSize: 10 },
          splitLine: { lineStyle: { color: c.splitLineColor } },
        },
        brush: {
          xAxisIndex: 0,
          brushType: "lineX",
          brushMode: "single",
          transformable: false,
          removeOnClick: true,
          brushStyle: {
            color: "rgba(20, 184, 166, 0.12)",
            borderColor: "rgba(20, 184, 166, 0.5)",
            borderWidth: 1,
          },
          outOfBrush: {
            colorAlpha: 0.25,
          },
          throttleType: "debounce",
          throttleDelay: 200,
        },
        toolbox: { show: false },
        series,
        animation: false,
      },
      true
    );

    // Activate brush mode so user can drag immediately
    chartRef.current.dispatchAction({
      type: "takeGlobalCursor",
      key: "brush",
      brushOption: { brushType: "lineX", brushMode: "single" },
    });

    // Re-enable callbacks after ECharts processes dispatches
    requestAnimationFrame(() => { suppressCallbackRef.current = false; });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data, intervalMs, timeSpanMs]);

  // =========================================================================
  // Effect 2: Brush sync — re-apply brush highlight when selectedTimeRange
  //           changes externally (e.g. filter pill removed).
  // =========================================================================
  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;

    suppressCallbackRef.current = true;

    if (selectedTimeRange) {
      const { times, intervalMs: intMs } = bucketDataRef.current;
      if (times.length > 0 && intMs > 0) {
        const [selStart, selEnd] = selectedTimeRange;
        let startIdx = 0;
        let endIdx = times.length - 1;
        for (let i = 0; i < times.length; i++) {
          if (times[i] + intMs > selStart) { startIdx = i; break; }
        }
        for (let i = times.length - 1; i >= 0; i--) {
          if (times[i] <= selEnd) { endIdx = i; break; }
        }
        chart.dispatchAction({
          type: "brush",
          areas: [{ brushType: "lineX", coordRange: [startIdx, endIdx], xAxisIndex: 0 }],
        });
      }
    } else {
      chart.dispatchAction({ type: "brush", areas: [] });
    }

    requestAnimationFrame(() => { suppressCallbackRef.current = false; });
  }, [selectedTimeRange]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (listenersRef.current.resize) {
        window.removeEventListener("resize", listenersRef.current.resize);
      }
      listenersRef.current.observer?.disconnect();
      chartRef.current?.dispose();
      chartRef.current = null;
    };
  }, []);

  if (data.length === 0) return null;

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-1">
        {title}
      </h4>
      <div ref={containerRef} className="w-full h-[120px]" />
    </div>
  );
}
