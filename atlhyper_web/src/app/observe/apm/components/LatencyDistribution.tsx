"use client";

import { useEffect, useEffectEvent } from "react";
import type * as echarts from "echarts";
import { useEChart } from "@/hooks/useEChart";
import { useI18n } from "@/i18n/context";
import { getThemeColors } from "@/lib/chart-theme";
import type { LatencyBucket } from "@/types/model/apm";
import { safeHtml } from "@/lib/safe-html";

/** 共用的主题色 + 本图的柱色与当前样本标线色 */
function latencyThemeColors() {
  const isDark = document.documentElement.classList.contains("dark");
  return {
    ...getThemeColors(),
    barColor: isDark ? "#60a5fa" : "#93c5fd",
    highlightColor: "#22c55e",
  };
}

/** 主题切换后重新上色（只改与主题有关的颜色） */
function applyThemeColors(chart: echarts.ECharts) {
  const c = getThemeColors();
  chart.setOption({
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
      axisLabel: { color: c.textColor },
      splitLine: { lineStyle: { color: c.splitLineColor } },
    },
  });
}

function formatBucketLabel(ms: number): string {
  if (ms < 1) return `${(ms * 1000).toFixed(0)}μs`;
  if (ms < 1000) {
    return ms % 1 === 0 ? `${ms}ms` : `${ms.toFixed(1)}ms`;
  }
  const s = ms / 1000;
  return s % 1 === 0 ? `${s}s` : `${s.toFixed(1)}s`;
}

interface LatencyDistributionProps {
  title: string;
  totalTraces: number;
  buckets: LatencyBucket[];
  highlightBucket?: number;
  /** 拖选延迟区间（ES 同款 brush 下钻）；null = 清除选区。不传则只读。 */
  onSelectRange?: (range: { startMs: number; endMs: number } | null) => void;
}

export function LatencyDistribution({
  title,
  totalTraces,
  buckets,
  highlightBucket,
  onSelectRange,
}: LatencyDistributionProps) {
  const { t } = useI18n();
  const { containerRef, chart } = useEChart({ onThemeChange: applyThemeColors });
  // brush 事件是创建图表时注册一次的；回调里要读最新的 buckets / onSelectRange，但不想因此重建图表
  const reportBrushedBuckets = useEffectEvent((coordRange: [number, number] | undefined) => {
    if (!onSelectRange) return;
    if (!coordRange) { onSelectRange(null); return; }
    const lo = Math.max(0, Math.ceil(coordRange[0]));
    const hi = Math.min(buckets.length - 1, Math.floor(coordRange[1]));
    if (lo > hi || !buckets[lo]) { onSelectRange(null); return; }
    const end = buckets[hi].rangeEnd === Infinity ? Number.MAX_VALUE : buckets[hi].rangeEnd;
    onSelectRange({ startMs: buckets[lo].rangeStart, endMs: end });
  });

  // 框选事件：图表建好后注册一次（回调经 reportBrushedBuckets 读最新的 buckets / onSelectRange）
  useEffect(() => {
    if (!chart) return;
    // 直接拖选（不经过 toolbox 按钮）—— ES 的 click-and-drag 交互
    chart.dispatchAction({
      type: "takeGlobalCursor",
      key: "brush",
      brushOption: { brushType: "lineX", brushMode: "single" },
    });
    const handleBrushEnd = (raw: unknown) => {
      const params = raw as { areas?: { coordRange?: [number, number] }[] };
      reportBrushedBuckets(params.areas?.[0]?.coordRange);
    };
    chart.on("brushEnd", handleBrushEnd);
    return () => {
      chart.off("brushEnd", handleBrushEnd);
    };
  }, [chart]);

  useEffect(() => {
    if (!chart || buckets.length === 0) return;
    const c = latencyThemeColors();

    const labels = buckets.map((b) => formatBucketLabel(b.rangeStart));
    const values = buckets.map((b) => b.count);

    const markLineData: { xAxis: number; label: { show: boolean; formatter: string; position: string; color: string; fontSize: number } }[] = [];
    if (highlightBucket !== undefined && highlightBucket >= 0 && highlightBucket < buckets.length) {
      markLineData.push({
        xAxis: highlightBucket,
        label: {
          show: true,
          formatter: t.apm.currentSample,
          position: "insideEndTop",
          color: c.highlightColor,
          fontSize: 10,
        },
      });
    }

    chart.setOption(
      {
        brush: onSelectRange
          ? {
              toolbox: [],
              xAxisIndex: 0,
              brushType: "lineX",
              brushMode: "single",
              transformable: false,
              brushStyle: {
                color: "rgba(45, 212, 191, 0.10)",
                borderColor: "rgba(45, 212, 191, 0.6)",
                borderWidth: 1,
              },
              outOfBrush: { colorAlpha: 0.35 },
            }
          : undefined,
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
            return safeHtml`${formatBucketLabel(bucket.rangeStart)} – ${rangeEnd}<br/>${p.value} trace(s)`.toString();
          },
        },
        grid: { top: 16, right: 24, bottom: 40, left: 45 },
        xAxis: {
          type: "category",
          data: labels,
          axisLine: { lineStyle: { color: c.lineColor } },
          axisTick: { alignWithLabel: true },
          axisLabel: {
            color: c.textColor,
            fontSize: 10,
            rotate: 0,
            interval: (index: number) => {
              const ms = buckets[index]?.rangeStart ?? 0;
              // Kibana-style: show labels at key round values (ms)
              const roundValues = [
                1, 2, 3, 4, 5, 6, 8,
                10, 20, 30, 40, 50, 60, 80,
                100, 200, 300, 400, 500, 600, 800,
                1000, 2000, 3000, 4000, 5000, 6000, 8000,
                10000, 20000, 30000, 40000, 50000,
              ];
              return roundValues.includes(ms);
            },
          },
          splitLine: { show: false },
        },
        yAxis: {
          type: "value",
          minInterval: 1,
          axisLabel: { color: c.textColor, fontSize: 10 },
          splitLine: { lineStyle: { color: c.splitLineColor } },
        },
        series: [
          {
            type: "bar",
            data: values.map((v) => Math.max(v, 0)),
            itemStyle: {
              color: c.barColor,
              borderRadius: 0,
            },
            barGap: "0%",
            barCategoryGap: "30%",
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
    if (onSelectRange) {
      chart.dispatchAction({
        type: "takeGlobalCursor",
        key: "brush",
        brushOption: { brushType: "lineX", brushMode: "single" },
      });
    }
  }, [chart, buckets, highlightBucket, title, t.apm.currentSample, onSelectRange]);

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
