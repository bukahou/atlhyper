"use client";

import { useEffect, useRef, useState, useMemo, memo, useCallback } from "react";
import * as echarts from "echarts";
import { TrendingUp, Clock, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { getNodeMetricsHistory } from "@/datasource/metrics";
import type { NodeMetricsSnapshot, MetricsDataPoint } from "@/types/node-metrics";

// ==================== Types & Constants ====================

/** Global time window — total data range */
type TimeWindow = "1h" | "6h" | "1d" | "7d";

/** Per-chart aggregation interval (seconds) */
type Interval = number;

const PALETTE = ["#F97316", "#10B981", "#3B82F6", "#8B5CF6", "#EF4444"];

const TIME_WINDOWS: { key: TimeWindow; label: string; hours: number }[] = [
  { key: "1h", label: "1h", hours: 1 },
  { key: "6h", label: "6h", hours: 6 },
  { key: "1d", label: "1d", hours: 24 },
  { key: "7d", label: "7d", hours: 168 },
];

/** Available intervals per window. label = display text, seconds = bucket size */
const INTERVALS_BY_WINDOW: Record<TimeWindow, { label: string; seconds: number }[]> = {
  "1h":  [{ label: "1m", seconds: 60 }, { label: "5m", seconds: 300 }, { label: "15m", seconds: 900 }],
  "6h":  [{ label: "5m", seconds: 300 }, { label: "15m", seconds: 900 }, { label: "30m", seconds: 1800 }],
  "1d":  [{ label: "15m", seconds: 900 }, { label: "1h", seconds: 3600 }, { label: "3h", seconds: 10800 }],
  "7d":  [{ label: "1h", seconds: 3600 }, { label: "6h", seconds: 21600 }, { label: "1d", seconds: 86400 }],
};

/** Sort function: extract the "current" value from a snapshot for ranking */
type SnapshotSortFn = (n: NodeMetricsSnapshot) => number;

const METRICS: {
  key: string;
  unit: string;
  extract: (d: MetricsDataPoint) => number;
  sortSnapshot: SnapshotSortFn;
}[] = [
  { key: "cpu", unit: "%", extract: (d) => d.cpuUsage, sortSnapshot: (n) => n.cpu.usagePercent },
  { key: "memory", unit: "%", extract: (d) => d.memUsage, sortSnapshot: (n) => n.memory.usagePercent },
  { key: "disk", unit: "%", extract: (d) => d.diskUsage, sortSnapshot: (n) => (n.disks.find((d) => d.mountPoint === "/") || n.disks[0])?.usagePercent || 0 },
  { key: "temp", unit: "°C", extract: (d) => d.temperature, sortSnapshot: (n) => n.temperature.cpuTemp },
];

// ==================== Helpers ====================

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

/** Downsample data by averaging within fixed-size time buckets */
function aggregateData(
  data: MetricsDataPoint[],
  extractFn: (d: MetricsDataPoint) => number,
  intervalSec: number,
): [number, number][] {
  if (data.length === 0) return [];
  // 30s raw = no aggregation needed
  if (intervalSec <= 30) {
    return data.map((d) => [d.timestamp, extractFn(d)]);
  }

  const intervalMs = intervalSec * 1000;
  const buckets = new Map<number, { sum: number; count: number }>();

  for (const d of data) {
    const bucketKey = Math.floor(d.timestamp / intervalMs) * intervalMs;
    const existing = buckets.get(bucketKey);
    const val = extractFn(d);
    if (existing) {
      existing.sum += val;
      existing.count += 1;
    } else {
      buckets.set(bucketKey, { sum: val, count: 1 });
    }
  }

  const result: [number, number][] = [];
  for (const [ts, { sum, count }] of buckets) {
    result.push([ts + intervalMs / 2, sum / count]); // center of bucket
  }
  result.sort((a, b) => a[0] - b[0]);
  return result;
}

// ==================== Single Metric Chart ====================

interface MetricChartProps {
  title: string;
  unit: string;
  nodeNames: string[];
  historyMap: Record<string, MetricsDataPoint[]>;
  extractFn: (d: MetricsDataPoint) => number;
  timeWindow: TimeWindow;
}

const MetricChart = memo(function MetricChart({
  title,
  unit,
  nodeNames,
  historyMap,
  extractFn,
  timeWindow,
}: MetricChartProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const chartRef = useRef<HTMLDivElement>(null);
  const chartInstance = useRef<echarts.ECharts | null>(null);

  const availableIntervals = INTERVALS_BY_WINDOW[timeWindow];
  const [intervalSec, setIntervalSec] = useState<Interval>(300);

  // Reset interval when window changes — prefer 5m if available
  useEffect(() => {
    const opts = INTERVALS_BY_WINDOW[timeWindow];
    const prefer = opts.find((o) => o.seconds === 300);
    setIntervalSec(prefer ? prefer.seconds : opts[0].seconds);
  }, [timeWindow]);

  // Init ECharts
  useEffect(() => {
    if (!chartRef.current) return;

    chartInstance.current = echarts.init(chartRef.current);
    const colors = getThemeColors();

    chartInstance.current.setOption({
      animation: true,
      animationDuration: 300,
      grid: { left: 45, right: 12, top: 10, bottom: 30 },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "line" },
        backgroundColor: colors.tooltipBg,
        borderColor: colors.tooltipBorder,
        textStyle: { color: colors.tooltipText, fontSize: 11 },
      },
      legend: { show: false },
      xAxis: {
        type: "time",
        axisLine: { lineStyle: { color: colors.lineColor } },
        axisLabel: { color: colors.textColor, fontSize: 9 },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        name: unit,
        nameTextStyle: { color: colors.textColor, fontSize: 9 },
        min: 0,
        axisLabel: { color: colors.textColor, fontSize: 9 },
        splitLine: { lineStyle: { color: colors.splitLineColor } },
      },
      series: [],
    });

    const handleResize = () => chartInstance.current?.resize();
    window.addEventListener("resize", handleResize);

    const observer = new MutationObserver(() => {
      if (!chartInstance.current) return;
      const c = getThemeColors();
      chartInstance.current.setOption({
        tooltip: {
          backgroundColor: c.tooltipBg,
          borderColor: c.tooltipBorder,
          textStyle: { color: c.tooltipText },
        },
        xAxis: {
          axisLine: { lineStyle: { color: c.lineColor } },
          axisLabel: { color: c.textColor },
        },
        yAxis: {
          nameTextStyle: { color: c.textColor },
          axisLabel: { color: c.textColor },
          splitLine: { lineStyle: { color: c.splitLineColor } },
        },
      });
    });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });

    return () => {
      window.removeEventListener("resize", handleResize);
      observer.disconnect();
      chartInstance.current?.dispose();
      chartInstance.current = null;
    };
  }, []);

  // Data update
  useEffect(() => {
    if (!chartInstance.current) return;

    const now = Date.now();
    const windowInfo = TIME_WINDOWS.find((w) => w.key === timeWindow)!;
    const rangeMs = windowInfo.hours * 3600000;

    const series = nodeNames.map((name, i) => {
      const raw = historyMap[name] || [];
      const filtered = raw.filter((d) => d.timestamp > now - rangeMs);
      const aggregated = aggregateData(filtered, extractFn, intervalSec);
      return {
        name,
        type: "line" as const,
        smooth: true,
        showSymbol: false,
        color: PALETTE[i % PALETTE.length],
        lineStyle: { width: 1.5 },
        data: aggregated,
      };
    });

    const allTimestamps = series.flatMap((s) => s.data.map((d) => d[0]));
    const xAxisMin = allTimestamps.length > 0 ? Math.min(...allTimestamps) : undefined;
    const xAxisMax = allTimestamps.length > 0 ? now : undefined;

    // Dynamic Y-axis
    const allValues = series.flatMap((s) => s.data.map((d) => d[1]));
    const dataMax = allValues.length > 0 ? Math.max(...allValues) : 0;
    const yMax = Math.max(10, Math.ceil(dataMax * 1.2));

    chartInstance.current.setOption({
      tooltip: {
        formatter: (params: unknown) => {
          const items = params as { seriesName: string; value: [number, number]; color: string }[];
          if (!items?.length) return "";
          const time = new Date(items[0].value[0]).toLocaleTimeString();
          const lines = items
            .filter((item) => item.value?.[1] != null)
            .map(
              (item) =>
                `<span style="color:${item.color}">●</span> ${item.seriesName}: ${item.value[1].toFixed(1)}${unit}`
            );
          return `${time}<br/>${lines.join("<br/>")}`;
        },
      },
      xAxis: { min: xAxisMin, max: xAxisMax, minInterval: intervalSec * 1000 },
      yAxis: { max: yMax },
      series,
    });
  }, [timeWindow, intervalSec, historyMap, nodeNames, extractFn, unit]);

  const hasData = nodeNames.some((name) => (historyMap[name]?.length || 0) > 0);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3">
      <div className="flex items-center justify-between mb-1">
        <span className="text-xs font-semibold text-default">{title}</span>
        {/* Per-chart interval selector */}
        <div className="flex gap-0.5">
          {availableIntervals.map((opt) => (
            <button
              key={opt.seconds}
              onClick={() => setIntervalSec(opt.seconds)}
              className={`px-1.5 py-0.5 text-[10px] rounded transition-colors ${
                intervalSec === opt.seconds
                  ? "bg-indigo-500/20 text-indigo-500 font-medium"
                  : "text-muted hover:text-default"
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>
      <div className="relative">
        <div ref={chartRef} style={{ width: "100%", height: "200px" }} />
        {!hasData && (
          <div className="absolute inset-0 flex items-center justify-center text-xs text-muted">
            {nm.overview.noData}
          </div>
        )}
      </div>
      {/* Per-chart legend */}
      <div className="flex flex-wrap gap-x-3 gap-y-0.5 mt-1">
        {nodeNames.map((name, i) => (
          <div key={name} className="flex items-center gap-1">
            <div className="w-2.5 h-0.5 rounded" style={{ backgroundColor: PALETTE[i % PALETTE.length] }} />
            <span className="text-[10px] text-muted">{name}</span>
          </div>
        ))}
      </div>
    </div>
  );
});

// ==================== Main Container ====================

interface ClusterOverviewChartProps {
  nodes: NodeMetricsSnapshot[];
  clusterId: string;
}

export const ClusterOverviewChart = memo(function ClusterOverviewChart({
  nodes,
  clusterId,
}: ClusterOverviewChartProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;

  const [timeWindow, setTimeWindow] = useState<TimeWindow>("1h");
  const [loading, setLoading] = useState(false);
  const [historyMap, setHistoryMap] = useState<Record<string, MetricsDataPoint[]>>({});

  // Top 5 nodes per metric (each metric sorts independently)
  const topNodesByMetric = useMemo(() => {
    const result: Record<string, string[]> = {};
    for (const m of METRICS) {
      result[m.key] = [...nodes]
        .sort((a, b) => m.sortSnapshot(b) - m.sortSnapshot(a))
        .slice(0, 5)
        .map((n) => n.nodeName);
    }
    return result;
  }, [nodes]);

  // Union of all top-5 lists for history fetching
  const allTopNodeNames = useMemo(() => {
    const set = new Set<string>();
    for (const names of Object.values(topNodesByMetric)) {
      for (const n of names) set.add(n);
    }
    return Array.from(set);
  }, [topNodesByMetric]);

  // Fetch hours based on window
  const fetchHours = useMemo(() => {
    return TIME_WINDOWS.find((w) => w.key === timeWindow)!.hours;
  }, [timeWindow]);

  // Fetch history for all unique top-5 nodes across metrics
  const fetchHistory = useCallback(async () => {
    if (allTopNodeNames.length === 0) return;
    setLoading(true);
    const map: Record<string, MetricsDataPoint[]> = {};
    for (const name of allTopNodeNames) {
      const result = await getNodeMetricsHistory(clusterId, name, fetchHours);
      map[result.nodeName] = result.data;
    }
    setHistoryMap(map);
    setLoading(false);
  }, [allTopNodeNames, fetchHours, clusterId]);

  useEffect(() => {
    fetchHistory();
  }, [fetchHistory]);

  const metricTitles: Record<string, string> = {
    cpu: nm.cpu.title,
    memory: nm.memory.title,
    disk: nm.disk.title,
    temp: nm.temperature.title,
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 bg-teal-500/10 rounded-lg">
            <TrendingUp className="w-5 h-5 text-teal-500" />
          </div>
          <div>
            <h3 className="text-base font-semibold text-default">{nm.overview.title}</h3>
            <span className="text-[10px] text-muted">{nm.overview.top5}</span>
          </div>
        </div>

        {/* Global time window */}
        <div className="flex items-center gap-1">
          <Clock className="w-4 h-4 text-muted" />
          {TIME_WINDOWS.map((w) => (
            <button
              key={w.key}
              onClick={() => setTimeWindow(w.key)}
              className={`px-2 py-1 text-xs rounded transition-colors ${
                timeWindow === w.key
                  ? "bg-teal-500 text-white"
                  : "bg-[var(--background)] text-muted hover:text-default"
              }`}
            >
              {w.label}
            </button>
          ))}
        </div>
      </div>

      {/* 2x2 Grid */}
      <div className="relative">
        {loading && (
          <div className="absolute inset-0 z-10 flex items-center justify-center bg-card/80 rounded-lg">
            <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
          </div>
        )}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
          {METRICS.map((m) => (
            <MetricChart
              key={m.key}
              title={metricTitles[m.key]}
              unit={m.unit}
              nodeNames={topNodesByMetric[m.key] || []}
              historyMap={historyMap}
              extractFn={m.extract}
              timeWindow={timeWindow}
            />
          ))}
        </div>
      </div>

    </div>
  );
});
