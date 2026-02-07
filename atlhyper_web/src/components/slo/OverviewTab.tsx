"use client";

import { useRef, useState, useMemo } from "react";
import { TrendingUp, Target } from "lucide-react";
import { ErrorBudgetBar, formatNumber } from "./common";
import type { SLOMetrics } from "@/types/slo";

interface OverviewTabTranslations {
  availability: string;
  p95Latency: string;
  p99Latency: string;
  errorRate: string;
  totalRequests: string;
  errorBudget: string;
  target: string;
  throughput: string;
  sloTrend: string;
  errorBudgetBurn: string;
  current: string;
  estimatedExhaust: string;
}

// SLO Trend Chart (SVG area chart)
function HistoryChart({ history, t }: {
  history: { timestamp: string; p95_latency: number; error_rate: number }[];
  t: OverviewTabTranslations;
}) {
  const [activeMetric, setActiveMetric] = useState<"p95" | "error">("p95");
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const svgRef = useRef<SVGSVGElement>(null);

  const metrics = [
    { id: "p95" as const, label: t.p95Latency, unit: "ms", color: "#0891b2" },
    { id: "error" as const, label: t.errorRate, unit: "%", color: "#ef4444" },
  ];
  const currentMetric = metrics.find(m => m.id === activeMetric)!;

  const values = history.map(p => activeMetric === "p95" ? p.p95_latency : p.error_rate);
  const rawMin = Math.min(...values);
  const rawMax = Math.max(...values);
  const minVal = rawMin - (rawMax - rawMin) * 0.05;
  const maxVal = rawMax + (rawMax - rawMin) * 0.05;
  const range = maxVal - minVal || 1;

  const width = 660, height = 180;
  const padLeft = 55, padRight = 5, padTop = 10, padBottom = 25;
  const chartH = height - padTop - padBottom;
  const chartW = width - padLeft - padRight;

  const points = values.map((v, i) => ({
    x: padLeft + (i / Math.max(values.length - 1, 1)) * chartW,
    y: padTop + (1 - (v - minVal) / range) * chartH,
    value: v,
    timestamp: history[i].timestamp,
  }));

  const linePath = points.map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`).join(" ");
  const areaPath = points.length > 1
    ? `${linePath} L ${points[points.length - 1].x} ${padTop + chartH} L ${points[0].x} ${padTop + chartH} Z`
    : "";
  const gradientId = `hist-grad-${activeMetric}`;

  const formatVal = (v: number) => activeMetric === "p95" ? Math.round(v) + "ms" : v.toFixed(3) + "%";

  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
      <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <TrendingUp className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">{t.sloTrend}</span>
        </div>
        <div className="flex items-center gap-1 p-0.5 rounded-lg bg-slate-100 dark:bg-slate-800">
          {metrics.map(m => (
            <button
              key={m.id}
              onClick={() => setActiveMetric(m.id)}
              className={`px-2.5 py-1 text-[10px] rounded-md transition-colors ${
                activeMetric === m.id ? "bg-white dark:bg-slate-700 text-default shadow-sm font-medium" : "text-muted hover:text-default"
              }`}
            >{m.label}</button>
          ))}
        </div>
      </div>
      <div className="p-4">
        <svg
          ref={svgRef}
          viewBox={`0 0 ${width} ${height}`}
          className="w-full h-auto"
          onMouseLeave={() => setHoveredIndex(null)}
          onMouseMove={(e) => {
            const svg = svgRef.current;
            if (!svg) return;
            const rect = svg.getBoundingClientRect();
            const mouseX = ((e.clientX - rect.left) / rect.width) * width;
            let closest = 0, minDist = Infinity;
            points.forEach((p, i) => { const d = Math.abs(p.x - mouseX); if (d < minDist) { minDist = d; closest = i; } });
            setHoveredIndex(closest);
          }}
        >
          <defs>
            <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={currentMetric.color} stopOpacity="0.3" />
              <stop offset="100%" stopColor={currentMetric.color} stopOpacity="0.02" />
            </linearGradient>
          </defs>
          {[0, 0.25, 0.5, 0.75, 1].map((r, i) => {
            const y = padTop + r * chartH;
            const val = maxVal - r * range;
            return (
              <g key={i}>
                <line x1={padLeft} y1={y} x2={padLeft + chartW} y2={y} stroke="#e2e8f0" strokeWidth="0.5" strokeDasharray={i === 0 || i === 4 ? "0" : "3 3"} className="dark:stroke-slate-700" />
                <text x={padLeft - 6} y={y + 3} textAnchor="end" className="text-[9px] fill-slate-400">{formatVal(val)}</text>
              </g>
            );
          })}
          {points.length > 1 && <path d={areaPath} fill={`url(#${gradientId})`} />}
          {points.length > 1 && <path d={linePath} fill="none" stroke={currentMetric.color} strokeWidth="2" strokeLinejoin="round" />}
          {points.map((p, i) => (
            <circle key={i} cx={p.x} cy={p.y} r={hoveredIndex === i ? 4 : 0} fill={currentMetric.color} stroke="white" strokeWidth="2" />
          ))}
          {hoveredIndex !== null && points[hoveredIndex] && (
            <g>
              <line x1={points[hoveredIndex].x} y1={padTop} x2={points[hoveredIndex].x} y2={padTop + chartH} stroke="#94a3b8" strokeWidth="0.5" strokeDasharray="3 3" />
              <rect x={Math.min(points[hoveredIndex].x - 50, width - 105)} y={Math.max(points[hoveredIndex].y - 38, padTop)} width="100" height="30" rx="4" fill="#1e293b" opacity="0.95" />
              <text x={Math.min(points[hoveredIndex].x - 50, width - 105) + 50} y={Math.max(points[hoveredIndex].y - 38, padTop) + 13} textAnchor="middle" className="text-[9px] fill-white font-medium">
                {formatVal(points[hoveredIndex].value)}
              </text>
              <text x={Math.min(points[hoveredIndex].x - 50, width - 105) + 50} y={Math.max(points[hoveredIndex].y - 38, padTop) + 24} textAnchor="middle" className="text-[8px] fill-slate-400">
                {new Date(points[hoveredIndex].timestamp).toLocaleString("zh-CN", { month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" })}
              </text>
            </g>
          )}
        </svg>
      </div>
    </div>
  );
}

// Error Budget Burn Chart
function ErrorBudgetBurnChart({ history, t }: {
  history: { timestamp: string; error_budget: number }[];
  t: OverviewTabTranslations;
}) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const values = history.map(p => p.error_budget);
  const width = 600, height = 160;
  const padX = 0, padTop = 10, padBottom = 25;
  const chartH = height - padTop - padBottom;
  const chartW = width - padX * 2;

  const points = values.map((v, i) => ({
    x: padX + (i / Math.max(values.length - 1, 1)) * chartW,
    y: padTop + (1 - v / 100) * chartH,
    value: v,
    timestamp: history[i].timestamp,
  }));
  const linePath = points.map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`).join(" ");

  const currentBudget = values[values.length - 1] ?? 0;
  const budgetColor = currentBudget > 50 ? "#10b981" : currentBudget > 20 ? "#f59e0b" : "#ef4444";

  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
      <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Target className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">{t.errorBudgetBurn}</span>
        </div>
        <div className="flex items-center gap-3 text-[11px]">
          <span className="text-muted">{t.current}</span>
          <span className="font-semibold" style={{ color: budgetColor }}>{currentBudget.toFixed(1)}%</span>
        </div>
      </div>
      <div className="p-4">
        <svg
          ref={svgRef}
          viewBox={`0 0 ${width} ${height}`}
          className="w-full h-auto"
          onMouseLeave={() => setHoveredIndex(null)}
          onMouseMove={(e) => {
            const svg = svgRef.current;
            if (!svg) return;
            const rect = svg.getBoundingClientRect();
            const mouseX = ((e.clientX - rect.left) / rect.width) * width;
            let closest = 0, minDist = Infinity;
            points.forEach((p, i) => { const d = Math.abs(p.x - mouseX); if (d < minDist) { minDist = d; closest = i; } });
            setHoveredIndex(closest);
          }}
        >
          <defs>
            <linearGradient id="budget-grad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#10b981" stopOpacity="0.15" />
              <stop offset="50%" stopColor="#f59e0b" stopOpacity="0.15" />
              <stop offset="100%" stopColor="#ef4444" stopOpacity="0.15" />
            </linearGradient>
          </defs>
          <rect x={padX} y={padTop} width={chartW} height={chartH} fill="url(#budget-grad)" rx="4" />
          {[0, 25, 50, 75, 100].map((v, i) => {
            const y = padTop + (1 - v / 100) * chartH;
            return (
              <g key={i}>
                <line x1={padX} y1={y} x2={padX + chartW} y2={y} stroke="#e2e8f0" strokeWidth="0.5" strokeDasharray="3 3" className="dark:stroke-slate-700" />
                <text x={padX + chartW + 4} y={y + 3} className="text-[9px] fill-slate-400">{v}%</text>
              </g>
            );
          })}
          {points.length > 1 && <path d={linePath} fill="none" stroke={budgetColor} strokeWidth="2.5" strokeLinejoin="round" />}
          {points.map((p, i) => (
            <circle key={i} cx={p.x} cy={p.y} r={hoveredIndex === i ? 4 : 0} fill={budgetColor} stroke="white" strokeWidth="2" />
          ))}
          {hoveredIndex !== null && points[hoveredIndex] && (
            <g>
              <line x1={points[hoveredIndex].x} y1={padTop} x2={points[hoveredIndex].x} y2={padTop + chartH} stroke="#94a3b8" strokeWidth="0.5" strokeDasharray="3 3" />
              <rect x={Math.min(points[hoveredIndex].x - 40, width - 85)} y={Math.max(points[hoveredIndex].y - 38, padTop)} width="80" height="30" rx="4" fill="#1e293b" opacity="0.95" />
              <text x={Math.min(points[hoveredIndex].x - 40, width - 85) + 40} y={Math.max(points[hoveredIndex].y - 38, padTop) + 13} textAnchor="middle" className="text-[9px] fill-white font-medium">
                {points[hoveredIndex].value.toFixed(1)}%
              </text>
            </g>
          )}
        </svg>
      </div>
    </div>
  );
}

// Main Overview Tab
export function OverviewTab({ summary, errorBudgetRemaining, history, t }: {
  summary: SLOMetrics | null;
  errorBudgetRemaining: number;
  history?: { timestamp: string; p95_latency: number; p99_latency: number; error_rate: number; availability: number; rps: number }[];
  t: OverviewTabTranslations;
}) {
  const availability = summary?.availability ?? 0;
  const p95Latency = summary?.p95_latency ?? 0;
  const p99Latency = summary?.p99_latency ?? 0;
  const errorRate = summary?.error_rate ?? 0;
  const rps = summary?.requests_per_sec ?? 0;
  const totalRequests = summary?.total_requests ?? 0;

  // Simulate error budget history from domain history if available
  const budgetHistory = useMemo(() => {
    if (!history || history.length === 0) return [];
    return history.map(h => ({
      timestamp: h.timestamp,
      error_budget: Math.max(0, Math.min(100, 100 - h.error_rate * 20)),
    }));
  }, [history]);

  return (
    <div className="space-y-4">
      {/* Golden Metrics */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-xs text-muted mb-1">{t.availability}</div>
          <div className="text-lg font-bold text-default">{availability.toFixed(3)}%</div>
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-xs text-muted mb-1">{t.p95Latency} / {t.p99Latency}</div>
          <div className="text-lg font-bold text-default">{p95Latency}ms / {p99Latency}ms</div>
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-xs text-muted mb-1">{t.errorRate}</div>
          <div className="text-lg font-bold text-default">{errorRate.toFixed(3)}%</div>
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-xs text-muted mb-1">{t.totalRequests}</div>
          <div className="text-lg font-bold text-default">{formatNumber(totalRequests)}</div>
          <div className="text-xs text-muted mt-1">{rps.toFixed(2)} {t.throughput}</div>
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-xs text-muted mb-1">{t.errorBudget}</div>
          <div className={`text-lg font-bold ${
            errorBudgetRemaining > 50 ? "text-emerald-500" :
            errorBudgetRemaining > 20 ? "text-amber-500" : "text-red-500"
          }`}>{errorBudgetRemaining.toFixed(1)}%</div>
          <div className="mt-1"><ErrorBudgetBar percent={errorBudgetRemaining} /></div>
        </div>
      </div>

      {/* Charts */}
      {history && history.length > 2 && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <HistoryChart history={history} t={t} />
          {budgetHistory.length > 2 && (
            <ErrorBudgetBurnChart history={budgetHistory} t={t} />
          )}
        </div>
      )}
    </div>
  );
}
