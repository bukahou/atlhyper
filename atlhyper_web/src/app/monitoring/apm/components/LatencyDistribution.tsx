"use client";

import { useRef, useEffect } from "react";
import * as echarts from "echarts";
import type { LatencyBucket } from "@/api/apm";

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

function formatBucketLabel(us: number): string {
  if (us < 1000) return `${us}μs`;
  if (us < 1_000_000) return `${(us / 1000).toFixed(0)}ms`;
  return `${(us / 1_000_000).toFixed(1)}s`;
}

interface LatencyDistributionProps {
  title: string;
  buckets: LatencyBucket[];
  highlightBucket?: number; // index of currently selected trace's bucket
}

export function LatencyDistribution({ title, buckets, highlightBucket }: LatencyDistributionProps) {
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
    if (!chartRef.current || buckets.length === 0) return;
    const c = getThemeColors();

    const labels = buckets.map((b) => formatBucketLabel(b.rangeStart));
    const values = buckets.map((b) => b.count);
    const colors = buckets.map((_, i) =>
      i === highlightBucket ? "#6366f1" : "#94a3b8"
    );

    chartRef.current.setOption({
      tooltip: {
        trigger: "axis",
        backgroundColor: c.tooltipBg,
        borderColor: c.tooltipBorder,
        textStyle: { color: c.tooltipText, fontSize: 12 },
        formatter: (params: { dataIndex: number; value: number }[]) => {
          const p = params[0];
          const bucket = buckets[p.dataIndex];
          return `${formatBucketLabel(bucket.rangeStart)} - ${formatBucketLabel(bucket.rangeEnd)}<br/>Count: ${p.value}`;
        },
      },
      grid: { top: 8, right: 16, bottom: 32, left: 40 },
      xAxis: {
        type: "category",
        data: labels,
        axisLine: { lineStyle: { color: c.lineColor } },
        axisLabel: { color: c.textColor, fontSize: 10, rotate: 30 },
        splitLine: { show: false },
      },
      yAxis: {
        type: "log",
        min: 1,
        axisLabel: { color: c.textColor, fontSize: 10 },
        splitLine: { lineStyle: { color: c.splitLineColor } },
      },
      series: [{
        type: "bar",
        data: values.map((v, i) => ({
          value: Math.max(v, 0),
          itemStyle: { color: colors[i], borderRadius: [2, 2, 0, 0] },
        })),
        barMaxWidth: 30,
      }],
      animation: false,
    }, true);
  }, [buckets, highlightBucket]);

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{title}</h4>
      <div ref={containerRef} className="w-full h-[160px]" />
    </div>
  );
}
