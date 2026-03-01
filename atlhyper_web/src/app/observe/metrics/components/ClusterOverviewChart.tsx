"use client";

import { useEffect, useState, useMemo, useCallback, memo } from "react";
import { TrendingUp, Clock, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { getNodeMetricsHistory } from "@/datasource/metrics";
import type { NodeMetrics, Point } from "@/types/node-metrics";
import type { TimeWindow } from "./chartConstants";
import { TIME_WINDOWS, METRICS } from "./chartConstants";
import { MetricChart } from "./MetricChart";

// ==================== Main Container ====================

interface ClusterOverviewChartProps {
  nodes: NodeMetrics[];
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
  /** Per-node history: { nodeName: { metricKey: Point[] } } */
  const [historyMap, setHistoryMap] = useState<Record<string, Record<string, Point[]>>>({});

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
    const map: Record<string, Record<string, Point[]>> = {};
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
              metricKey={m.key}
              nodeNames={topNodesByMetric[m.key] || []}
              historyMap={historyMap}
              timeWindow={timeWindow}
            />
          ))}
        </div>
      </div>

    </div>
  );
});
