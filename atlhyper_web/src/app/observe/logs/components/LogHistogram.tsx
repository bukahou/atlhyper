"use client";

import { useRef, useEffect, useEffectEvent } from "react";
import type * as echarts from "echarts";
import { useEChart } from "@/hooks/useEChart";
import { useI18n } from "@/i18n/context";
import { getThemeColors } from "@/lib/chart-theme";
import type { LogHistogramBucket } from "@/types/model/log";
import {
  SEVERITY_COLORS,
  SEVERITY_ORDER,
  buildFullTimeline,
  formatBucketLabels,
} from "./log-histogram-utils";

/** ECharts brush / brushEnd 事件里用到的部分 */
interface BrushEventParams {
  areas?: { coordRange: [number, number] }[];
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

export function LogHistogram(props: LogHistogramProps) {
  const { t } = useI18n();

  // 没有可画的数据时不挂图表，只留标题与占位（组件不隐藏）。
  // 图表拆成子组件：容器在才创建、不在就销毁（生命周期见 useEChart）
  if (props.data.length === 0 || props.intervalMs <= 0) {
    return (
      <div>
        <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-1">
          {props.title}
        </h4>
        <div className="w-full h-[120px] flex items-center justify-center text-sm text-muted border border-dashed border-[var(--border-color)] rounded-lg">
          {t.common.noData}
        </div>
      </div>
    );
  }

  return <LogHistogramChart {...props} />;
}

/** 主题切换后重新上色（只改与主题有关的颜色） */
function applyThemeColors(chart: echarts.ECharts) {
  const c = getThemeColors();
  chart.setOption({
    tooltip: { backgroundColor: c.tooltipBg, borderColor: c.tooltipBorder, textStyle: { color: c.tooltipText } },
    xAxis: { axisLine: { lineStyle: { color: c.lineColor } }, axisLabel: { color: c.textColor } },
    yAxis: { axisLabel: { color: c.textColor }, splitLine: { lineStyle: { color: c.splitLineColor } } },
  });
}

function LogHistogramChart({ data, intervalMs, title, timeSpanMs, selectedTimeRange, onTimeRangeSelect }: LogHistogramProps) {
  const { containerRef, chart } = useEChart({ onThemeChange: applyThemeColors });
  const bucketDataRef = useRef<{ times: number[]; intervalMs: number }>({ times: [], intervalMs: 0 });
  /** Suppress callback when programmatically manipulating brush */
  const suppressCallbackRef = useRef(false);

  // 图表事件是在创建图表时注册一次的，回调里要读到最新的 onTimeRangeSelect，又不能因此重建图表
  const reportBrushedRange = useEffectEvent((range: [number, number] | null) => {
    if (suppressCallbackRef.current) return;
    onTimeRangeSelect?.(range);
  });

  // =========================================================================
  // Effect 1: 框选事件（图表建好后注册一次；回调经 reportBrushedRange 读最新的 onTimeRangeSelect）
  // =========================================================================
  useEffect(() => {
    if (!chart) return;

    // brushEnd — only fires when user releases mouse (finished selecting)
    const handleBrushEnd = (rawParams: unknown) => {
      const areas = (rawParams as BrushEventParams).areas;
      if (!areas || areas.length === 0) {
        reportBrushedRange(null);
        return;
      }
      const [startIdx, endIdx] = areas[0].coordRange;
      const { times } = bucketDataRef.current;
      if (times.length === 0) return;
      const s = Math.max(0, Math.round(startIdx));
      const e = Math.min(times.length - 1, Math.round(endIdx));
      reportBrushedRange([times[s], times[e]]);
    };

    // brush with empty areas — user clicks to clear selection
    const handleBrush = (rawParams: unknown) => {
      const areas = (rawParams as BrushEventParams).areas;
      if (!areas || areas.length === 0) reportBrushedRange(null);
    };

    chart.on("brushEnd", handleBrushEnd);
    chart.on("brush", handleBrush);
    return () => {
      chart.off("brushEnd", handleBrushEnd);
      chart.off("brush", handleBrush);
    };
  }, [chart]);

  // =========================================================================
  // Effect 2: 数据渲染（不依赖 selectedTimeRange）
  // =========================================================================
  useEffect(() => {
    if (!chart) return;

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
    const xLabels = formatBucketLabels(allTimes, viewSpanMs);

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

    chart.setOption(
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
    chart.dispatchAction({
      type: "takeGlobalCursor",
      key: "brush",
      brushOption: { brushType: "lineX", brushMode: "single" },
    });

    // Re-enable callbacks after ECharts processes dispatches
    requestAnimationFrame(() => { suppressCallbackRef.current = false; });
  }, [chart, data, intervalMs, timeSpanMs]);

  // =========================================================================
  // Effect 3: Brush sync — 选区从外部变化（比如点掉筛选标签）或图表重画之后，
  //           把选区的高亮重新画上去。数据 effect 是整体重设图表的，会把高亮一起清掉，
  //           所以这里也要跟着 data 重跑（依赖里的 data / intervalMs 就是为此）。
  // =========================================================================
  useEffect(() => {
    if (!chart) return;

    suppressCallbackRef.current = true;

    if (selectedTimeRange) {
      const { times, intervalMs: bucketMs } = bucketDataRef.current;
      if (times.length > 0 && bucketMs > 0) {
        const [selStart, selEnd] = selectedTimeRange;
        let startIdx = 0;
        let endIdx = times.length - 1;
        for (let i = 0; i < times.length; i++) {
          if (times[i] + bucketMs > selStart) { startIdx = i; break; }
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
  }, [chart, selectedTimeRange, data, intervalMs]);

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-1">
        {title}
      </h4>
      <div ref={containerRef} className="w-full h-[120px]" />
    </div>
  );
}
