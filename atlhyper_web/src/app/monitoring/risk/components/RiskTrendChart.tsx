"use client";

import { useMemo } from "react";
import { useI18n } from "@/i18n/context";
import type { RiskTrendPoint } from "@/api/aiops";

interface RiskTrendChartProps {
  data: RiskTrendPoint[];
}

const CHART_HEIGHT = 160;
const CHART_PADDING = { top: 10, right: 10, bottom: 24, left: 36 };

function riskColor(risk: number): string {
  if (risk >= 80) return "#ef4444";
  if (risk >= 50) return "#eab308";
  if (risk >= 20) return "#3b82f6";
  return "#22c55e";
}

export function RiskTrendChart({ data }: RiskTrendChartProps) {
  const { t } = useI18n();

  const { pathD, areaD, xLabels, yLabels, latestColor } = useMemo(() => {
    if (data.length < 2) {
      return { pathD: "", areaD: "", xLabels: [], yLabels: [], latestColor: "#22c55e" };
    }

    const w = 100; // 使用 viewBox 百分比
    const h = CHART_HEIGHT;
    const plotW = w - CHART_PADDING.left - CHART_PADDING.right;
    const plotH = h - CHART_PADDING.top - CHART_PADDING.bottom;

    const minT = data[0].timestamp;
    const maxT = data[data.length - 1].timestamp;
    const rangeT = maxT - minT || 1;

    const points = data.map((d) => ({
      x: CHART_PADDING.left + ((d.timestamp - minT) / rangeT) * plotW,
      y: CHART_PADDING.top + plotH - (d.risk / 100) * plotH,
    }));

    const lineD = points.map((p, i) => `${i === 0 ? "M" : "L"}${p.x.toFixed(2)},${p.y.toFixed(2)}`).join(" ");
    const bottom = CHART_PADDING.top + plotH;
    const area = `${lineD} L${points[points.length - 1].x.toFixed(2)},${bottom} L${points[0].x.toFixed(2)},${bottom} Z`;

    // X轴标签: 显示4个时间点
    const xlabels = [0, Math.floor(data.length / 3), Math.floor((data.length * 2) / 3), data.length - 1].map((i) => {
      const d = new Date(data[i].timestamp);
      return {
        x: points[i].x,
        label: `${d.getHours().toString().padStart(2, "0")}:${d.getMinutes().toString().padStart(2, "0")}`,
      };
    });

    // Y轴标签
    const ylabels = [0, 25, 50, 75, 100].map((v) => ({
      y: CHART_PADDING.top + plotH - (v / 100) * plotH,
      label: v.toString(),
    }));

    return {
      pathD: lineD,
      areaD: area,
      xLabels: xlabels,
      yLabels: ylabels,
      latestColor: riskColor(data[data.length - 1].risk),
    };
  }, [data]);

  if (data.length < 2) {
    return (
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-5 flex items-center justify-center h-[220px]">
        <span className="text-sm text-muted">{t.aiops.noData}</span>
      </div>
    );
  }

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5">
      <h3 className="text-sm font-semibold text-default mb-3">{t.aiops.riskTrend}</h3>
      <svg viewBox={`0 0 100 ${CHART_HEIGHT}`} className="w-full" preserveAspectRatio="none">
        {/* 背景分区 */}
        <rect
          x={CHART_PADDING.left}
          y={CHART_PADDING.top}
          width={100 - CHART_PADDING.left - CHART_PADDING.right}
          height={(CHART_HEIGHT - CHART_PADDING.top - CHART_PADDING.bottom) * 0.2}
          fill="#ef4444"
          opacity={0.05}
        />
        <rect
          x={CHART_PADDING.left}
          y={CHART_PADDING.top + (CHART_HEIGHT - CHART_PADDING.top - CHART_PADDING.bottom) * 0.2}
          width={100 - CHART_PADDING.left - CHART_PADDING.right}
          height={(CHART_HEIGHT - CHART_PADDING.top - CHART_PADDING.bottom) * 0.3}
          fill="#eab308"
          opacity={0.05}
        />

        {/* 网格线 */}
        {[25, 50, 75].map((v) => {
          const y = CHART_PADDING.top + (CHART_HEIGHT - CHART_PADDING.top - CHART_PADDING.bottom) * (1 - v / 100);
          return (
            <line
              key={v}
              x1={CHART_PADDING.left}
              y1={y}
              x2={100 - CHART_PADDING.right}
              y2={y}
              stroke="var(--border-color)"
              strokeWidth={0.15}
              strokeDasharray="1,1"
            />
          );
        })}

        {/* 面积 */}
        <path d={areaD} fill={latestColor} opacity={0.1} />

        {/* 折线 */}
        <path d={pathD} fill="none" stroke={latestColor} strokeWidth={0.5} />

        {/* Y轴标签 */}
        {yLabels.map((yl) => (
          <text key={yl.label} x={CHART_PADDING.left - 2} y={yl.y + 1} textAnchor="end" fontSize={3} fill="var(--text-muted)">
            {yl.label}
          </text>
        ))}

        {/* X轴标签 */}
        {xLabels.map((xl, i) => (
          <text key={i} x={xl.x} y={CHART_HEIGHT - 4} textAnchor="middle" fontSize={3} fill="var(--text-muted)">
            {xl.label}
          </text>
        ))}
      </svg>
    </div>
  );
}
