"use client";

import { useRef, useEffect } from "react";
import * as echarts from "echarts";
import type { TraceSummary } from "@/types/model/apm";

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

interface ThroughputChartProps {
  title: string;
  traces: TraceSummary[];
}

export function ThroughputChart({ title, traces }: ThroughputChartProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<echarts.ECharts | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;
    const chart = echarts.init(containerRef.current);
    chartRef.current = chart;

    const handleResize = () => chart.resize();
    window.addEventListener("resize", handleResize);

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

    return () => {
      window.removeEventListener("resize", handleResize);
      observer.disconnect();
      chart.dispose();
    };
  }, []);

  useEffect(() => {
    if (!chartRef.current || traces.length === 0) return;
    const c = getThemeColors();

    // Group traces into time buckets using ISO timestamp
    const sorted = [...traces].sort((a, b) =>
      new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    );
    const bucketCount = Math.min(sorted.length, 20);
    const minTime = new Date(sorted[0].timestamp).getTime();
    const maxTime = new Date(sorted[sorted.length - 1].timestamp).getTime();
    const bucketSize = Math.max((maxTime - minTime) / bucketCount, 1);

    const buckets: { time: number; count: number }[] = [];
    for (let i = 0; i < bucketCount; i++) {
      buckets.push({ time: minTime + i * bucketSize, count: 0 });
    }
    for (const t of sorted) {
      const ts = new Date(t.timestamp).getTime();
      const idx = Math.min(Math.floor((ts - minTime) / bucketSize), bucketCount - 1);
      buckets[idx].count++;
    }

    chartRef.current.setOption({
      tooltip: {
        trigger: "axis",
        backgroundColor: c.tooltipBg,
        borderColor: c.tooltipBorder,
        textStyle: { color: c.tooltipText, fontSize: 12 },
      },
      grid: { top: 12, right: 16, bottom: 32, left: 40 },
      xAxis: {
        type: "category",
        data: buckets.map((b) => new Date(b.time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })),
        axisLine: { lineStyle: { color: c.lineColor } },
        axisLabel: { color: c.textColor, fontSize: 10, rotate: 0 },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        nameTextStyle: { color: c.textColor, fontSize: 10 },
        axisLabel: { color: c.textColor, fontSize: 10 },
        splitLine: { lineStyle: { color: c.splitLineColor } },
      },
      series: [{
        type: "bar",
        data: buckets.map((b) => b.count),
        itemStyle: { color: "#22c55e", borderRadius: [2, 2, 0, 0] },
        barMaxWidth: 20,
      }],
      animation: false,
    }, true);
  }, [traces]);

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{title}</h4>
      <div ref={containerRef} className="w-full h-[200px]" />
    </div>
  );
}
