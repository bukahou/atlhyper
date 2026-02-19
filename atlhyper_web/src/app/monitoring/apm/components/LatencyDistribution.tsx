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
    barColor: isDark ? "#60a5fa" : "#93c5fd", // light blue like Kibana
    highlightColor: "#22c55e", // green marker for current sample
  };
}

function formatBucketLabel(us: number): string {
  if (us < 1000) return `${us}μs`;
  if (us < 1_000_000) {
    const ms = us / 1000;
    return ms % 1 === 0 ? `${ms}ms` : `${ms.toFixed(1)}ms`;
  }
  const s = us / 1_000_000;
  return s % 1 === 0 ? `${s}s` : `${s.toFixed(1)}s`;
}

interface LatencyDistributionProps {
  title: string;
  totalTraces: number;
  buckets: LatencyBucket[];
  highlightBucket?: number;
}

export function LatencyDistribution({
  title,
  totalTraces,
  buckets,
  highlightBucket,
}: LatencyDistributionProps) {
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
        tooltip: {
          backgroundColor: c.tooltipBg,
          borderColor: c.tooltipBorder,
          textStyle: { color: c.tooltipText },
        },
        xAxis: {
          axisLine: { lineStyle: { color: c.lineColor } },
          axisLabel: { color: c.textColor },
          nameTextStyle: { color: c.textColor },
        },
        yAxis: {
          axisLabel: { color: c.textColor },
          nameTextStyle: { color: c.textColor },
          splitLine: { lineStyle: { color: c.splitLineColor } },
        },
      });
    });
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class"],
    });

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

    // Build markLine for current sample highlight (green vertical line)
    const markLineData: { xAxis: number; label: { show: boolean; formatter: string; position: string; color: string; fontSize: number } }[] = [];
    if (highlightBucket !== undefined && highlightBucket >= 0 && highlightBucket < buckets.length) {
      markLineData.push({
        xAxis: highlightBucket,
        label: {
          show: true,
          formatter: title.includes("分布") ? "当前样例" : "current",
          position: "insideEndTop",
          color: c.highlightColor,
          fontSize: 10,
        },
      });
    }

    chartRef.current.setOption(
      {
        tooltip: {
          trigger: "axis",
          backgroundColor: c.tooltipBg,
          borderColor: c.tooltipBorder,
          textStyle: { color: c.tooltipText, fontSize: 12 },
          formatter: (params: { dataIndex: number; value: number }[]) => {
            const p = params[0];
            const bucket = buckets[p.dataIndex];
            const rangeEnd =
              bucket.rangeEnd === Infinity
                ? "+"
                : formatBucketLabel(bucket.rangeEnd);
            return `${formatBucketLabel(bucket.rangeStart)} – ${rangeEnd}<br/>${p.value} trace(s)`;
          },
        },
        grid: { top: 16, right: 24, bottom: 40, left: 45 },
        xAxis: {
          type: "category",
          data: labels,
          name: "",
          axisLine: { lineStyle: { color: c.lineColor } },
          axisTick: { alignWithLabel: true },
          axisLabel: {
            color: c.textColor,
            fontSize: 10,
            rotate: 0,
            interval: (index: number) => {
              // Show a label every ~4-5 ticks, or if it's a round number
              if (buckets.length <= 12) return true;
              const us = buckets[index]?.rangeStart ?? 0;
              // Show labels at: 1ms, 2ms, 5ms, 10ms, 20ms, 50ms, 100ms, 200ms, 500ms, 1s, 2s, 5s, 10s, 20s, 50s
              const roundValues = [
                0, 1000, 2000, 5000, 10000, 20000, 50000, 100000, 200000,
                500000, 1000000, 2000000, 5000000, 10000000, 20000000,
                50000000,
              ];
              return roundValues.includes(us);
            },
          },
          splitLine: { show: false },
        },
        yAxis: {
          type: "log",
          min: 1,
          name: "",
          nameTextStyle: { color: c.textColor, fontSize: 10 },
          axisLabel: { color: c.textColor, fontSize: 10 },
          splitLine: { lineStyle: { color: c.splitLineColor } },
        },
        series: [
          {
            type: "bar",
            data: values.map((v) => Math.max(v, 0)),
            itemStyle: {
              color: c.barColor,
              borderRadius: [1, 1, 0, 0],
            },
            barMinWidth: 4,
            barCategoryGap: "20%",
            markLine:
              markLineData.length > 0
                ? {
                    silent: true,
                    symbol: "none",
                    lineStyle: {
                      color: c.highlightColor,
                      width: 2,
                      type: "solid",
                    },
                    data: markLineData,
                  }
                : undefined,
          },
        ],
        animation: false,
      },
      true
    );
  }, [buckets, highlightBucket, title]);

  return (
    <div>
      <div className="flex items-center gap-2 mb-2">
        <h4 className="text-sm font-semibold text-primary">{title}</h4>
        <span className="text-xs text-muted">
          | {totalTraces} traces
        </span>
      </div>
      <div ref={containerRef} className="w-full h-[180px]" />
    </div>
  );
}
