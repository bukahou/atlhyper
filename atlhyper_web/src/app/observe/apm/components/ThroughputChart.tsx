"use client";

import { useRef, useEffect } from "react";
import * as echarts from "echarts";
import type { OperationStats } from "@/types/model/apm";

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
  operations: OperationStats[];
}

export function ThroughputChart({ title, operations }: ThroughputChartProps) {
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
    if (!chartRef.current || operations.length === 0) return;
    const c = getThemeColors();

    // Top 10 by rps descending
    const top = [...operations]
      .sort((a, b) => b.rps - a.rps)
      .slice(0, 10);

    const names = top.map((op) => {
      const n = op.operationName;
      return n.length > 20 ? n.slice(0, 17) + "..." : n;
    });

    // rps → tpm (requests per minute)
    const tpmValues = top.map((op) => +(op.rps * 60).toFixed(1));

    chartRef.current.setOption({
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "shadow" },
        backgroundColor: c.tooltipBg,
        borderColor: c.tooltipBorder,
        textStyle: { color: c.tooltipText, fontSize: 12 },
        formatter: (params: { dataIndex: number; value: number }[]) => {
          const idx = params[0]?.dataIndex ?? 0;
          const op = top[idx];
          return `<div style="font-weight:600;margin-bottom:4px">${op.operationName}</div>` +
            `${params[0].value} tpm (${op.rps.toFixed(2)} rps)`;
        },
      },
      grid: { top: 12, right: 16, bottom: 56, left: 40 },
      xAxis: {
        type: "category",
        data: names,
        axisLine: { lineStyle: { color: c.lineColor } },
        axisLabel: { color: c.textColor, fontSize: 10, rotate: 30, interval: 0 },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        name: "tpm",
        nameTextStyle: { color: c.textColor, fontSize: 10 },
        axisLabel: { color: c.textColor, fontSize: 10 },
        splitLine: { lineStyle: { color: c.splitLineColor } },
      },
      series: [{
        type: "bar",
        data: tpmValues,
        itemStyle: { color: "#22c55e", borderRadius: [2, 2, 0, 0] },
        barMaxWidth: 24,
      }],
      animation: false,
    }, true);
  }, [operations]);

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{title}</h4>
      <div ref={containerRef} className="w-full h-[200px]" />
    </div>
  );
}
