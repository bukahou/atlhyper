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
  title: string;
  selectedTimeRange: [number, number] | null;
  onTimeRangeSelect?: (range: [number, number] | null) => void;
}

export function LogHistogram({ data, title, selectedTimeRange, onTimeRangeSelect }: LogHistogramProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<echarts.ECharts | null>(null);
  const bucketDataRef = useRef<{ times: number[]; bucketSize: number }>({ times: [], bucketSize: 0 });
  const callbackRef = useRef(onTimeRangeSelect);
  callbackRef.current = onTimeRangeSelect;
  /** Suppress callback when programmatically re-applying brush */
  const suppressCallbackRef = useRef(false);

  // Resize + theme observer (attached once per chart instance)
  const listenersRef = useRef<{ resize: (() => void) | null; observer: MutationObserver | null }>({
    resize: null,
    observer: null,
  });

  // Init chart + update data in a single effect to avoid race condition:
  // when data goes from empty → non-empty the container div appears for the first time,
  // so chart init must happen in the same effect that uses [data] as dependency.
  useEffect(() => {
    if (!containerRef.current || data.length === 0) return;

    // Initialize chart if not yet created (or if container changed)
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
        const { times, bucketSize } = bucketDataRef.current;
        if (times.length === 0) return;
        const s = Math.max(0, Math.round(startIdx));
        const e = Math.min(times.length - 1, Math.round(endIdx));
        callbackRef.current?.([times[s], times[e] + bucketSize]);
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
    const c = getThemeColors();

    // Sort by time
    const sorted = [...data].sort(
      (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    );

    const minTime = new Date(sorted[0].timestamp).getTime();
    const maxTime = new Date(sorted[sorted.length - 1].timestamp).getTime();
    const bucketCount = Math.min(Math.max(Math.ceil(sorted.length / 5), 5), 30);
    const bucketSize = Math.max((maxTime - minTime) / bucketCount, 1);

    // Initialize buckets per severity
    const bucketsBySev: Record<string, number[]> = {};
    for (const sev of SEVERITY_ORDER) {
      bucketsBySev[sev] = new Array(bucketCount).fill(0);
    }
    const bucketTimes: number[] = [];
    for (let i = 0; i < bucketCount; i++) {
      bucketTimes.push(minTime + i * bucketSize);
    }
    bucketDataRef.current = { times: bucketTimes, bucketSize };

    for (const item of sorted) {
      const ts = new Date(item.timestamp).getTime();
      const idx = Math.min(Math.floor((ts - minTime) / bucketSize), bucketCount - 1);
      const sev = item.severity.toUpperCase();
      if (bucketsBySev[sev]) {
        bucketsBySev[sev][idx]++;
      }
    }

    // If data spans multiple days, include date in label
    const spanDays = (maxTime - minTime) > 24 * 60 * 60 * 1000;
    const xLabels = bucketTimes.map((t) => {
      const d = new Date(t);
      const time = d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
      if (spanDays) {
        return `${d.getMonth() + 1}/${d.getDate()} ${time}`;
      }
      return time;
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

    // Re-apply brush highlight if there's an active selection (suppress callback to avoid loop)
    suppressCallbackRef.current = true;
    if (selectedTimeRange) {
      const { times, bucketSize } = bucketDataRef.current;
      if (times.length > 0) {
        const [selStart, selEnd] = selectedTimeRange;
        // Find bucket indices that overlap with the selection
        let startIdx = 0;
        let endIdx = times.length - 1;
        for (let i = 0; i < times.length; i++) {
          if (times[i] + bucketSize >= selStart) { startIdx = i; break; }
        }
        for (let i = times.length - 1; i >= 0; i--) {
          if (times[i] <= selEnd) { endIdx = i; break; }
        }
        chartRef.current.dispatchAction({
          type: "brush",
          areas: [{ brushType: "lineX", coordRange: [startIdx, endIdx], xAxisIndex: 0 }],
        });
      }
    } else {
      chartRef.current.dispatchAction({ type: "brush", areas: [] });
    }
    // Re-enable callback after echarts processes the dispatch synchronously
    requestAnimationFrame(() => { suppressCallbackRef.current = false; });
  }, [data, selectedTimeRange]);

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
