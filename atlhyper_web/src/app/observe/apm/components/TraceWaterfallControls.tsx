"use client";

import { ChevronRight, ChevronLeft, ChevronsLeft, ChevronsRight } from "lucide-react";
import { formatDurationMs, formatTimeAgo } from "@/lib/format";
import type { ApmTranslations } from "@/types/i18n";
import type { TraceSummary } from "@/types/model/apm";

// ============================================================
// 瀑布图上方的两组控件：样本导航、服务聚焦图例
// 纯展示 + 回调，状态都在 TraceWaterfall 里
// ============================================================

interface TraceSampleNavigatorProps {
  t: ApmTranslations;
  /** 可导航的样本（已按延迟选区筛过），i 是它在完整样本序列里的位置 */
  sampleEntries: { tr: TraceSummary; i: number }[];
  /** 当前样本在 sampleEntries 里的位置；不在其中时为 -1 */
  samplePos: number;
  currentTraceSummary: TraceSummary | undefined;
  onNavigateTrace: (index: number) => void;
}

/** 样本导航：第一条 / 上一条 / 第几条 / 下一条 / 最后一条 */
export function TraceSampleNavigator({
  t,
  sampleEntries,
  samplePos,
  currentTraceSummary,
  onNavigateTrace,
}: TraceSampleNavigatorProps) {
  return (
    <div className="flex items-center justify-between border border-[var(--border-color)] rounded-xl px-4 py-3 bg-card">
      <div className="flex items-center gap-3">
        <span className="text-sm font-medium text-default">{t.traceSample}</span>
        <div className="flex items-center gap-1">
          <button onClick={() => sampleEntries.length > 0 && onNavigateTrace(sampleEntries[0].i)} disabled={samplePos <= 0} className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors">
            <ChevronsLeft className="w-4 h-4 text-muted" />
          </button>
          <button onClick={() => samplePos > 0 && onNavigateTrace(sampleEntries[samplePos - 1].i)} disabled={samplePos <= 0} className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors">
            <ChevronLeft className="w-4 h-4 text-muted" />
          </button>
          <span className="text-sm text-default px-2 min-w-[60px] text-center">
            {samplePos >= 0 ? samplePos + 1 : "–"} / {sampleEntries.length}
          </span>
          <button onClick={() => samplePos >= 0 && samplePos < sampleEntries.length - 1 && onNavigateTrace(sampleEntries[samplePos + 1].i)} disabled={samplePos < 0 || samplePos >= sampleEntries.length - 1} className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors">
            <ChevronRight className="w-4 h-4 text-muted" />
          </button>
          <button onClick={() => sampleEntries.length > 0 && onNavigateTrace(sampleEntries[sampleEntries.length - 1].i)} disabled={samplePos < 0 || samplePos >= sampleEntries.length - 1} className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors">
            <ChevronsRight className="w-4 h-4 text-muted" />
          </button>
        </div>
      </div>
      {currentTraceSummary && (
        <div className="text-xs text-muted">
          {formatTimeAgo(currentTraceSummary.timestamp)} | {formatDurationMs(currentTraceSummary.durationMs)}
        </div>
      )}
    </div>
  );
}

interface ServiceFocusLegendProps {
  t: ApmTranslations;
  serviceColorMap: Map<string, string>;
  /** 各服务的 span 数与 self time 汇总 */
  serviceStats: Map<string, { spans: number; selfMs: number }>;
  /** 全部服务 self time 之和（占比的分母） */
  grandSelfMs: number;
  focusService: string | null;
  onFocusServiceChange: (service: string | null) => void;
}

/** 服务图例：点某个服务聚焦它（其余淡化），再点一次取消 */
export function ServiceFocusLegend({
  t,
  serviceColorMap,
  serviceStats,
  grandSelfMs,
  focusService,
  onFocusServiceChange: setFocusService,
}: ServiceFocusLegendProps) {
  return (
    <div className="flex flex-wrap items-center gap-2 px-4 py-2 border-b border-[var(--border-color)]">
      {[...serviceColorMap.entries()].map(([svc, color]) => {
        const isFocused = focusService === svc;
        const isDimmed = focusService !== null && !isFocused;
        return (
          <button
            key={svc}
            onClick={() => setFocusService(isFocused ? null : svc)}
            title={t.focusLegendHint}
            className={`flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full border transition-all ${
              isFocused
                ? "border-primary shadow-[inset_0_0_0_1px_var(--primary)]"
                : "border-[var(--border-color)] hover:border-[var(--text-muted)]"
            } ${isDimmed ? "opacity-40" : ""}`}
          >
            <span className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ backgroundColor: color }} />
            <span className="text-default">{svc}</span>
            <span className="text-[10px] text-muted font-mono tabular-nums">
              {formatDurationMs(serviceStats.get(svc)?.selfMs ?? 0)}
              {grandSelfMs > 0 && (
                <span className="ml-1 opacity-70">
                  {Math.round(((serviceStats.get(svc)?.selfMs ?? 0) / grandSelfMs) * 100)}%
                </span>
              )}
            </span>
          </button>
        );
      })}
      {/* 明暗编码说明：亮 = 自己在干活，暗 = 在等下游（那段时间的活在子行里） */}
      <div className="flex items-center gap-3 ml-auto text-[10px] text-muted">
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-2.5 rounded-sm" style={{ background: "var(--text-secondary)" }} />
          {t.legendSelfTime}
        </span>
        <span className="flex items-center gap-1.5">
          <span
            className="w-3 h-2.5 rounded-sm"
            style={{ background: "color-mix(in srgb, var(--text-secondary) 35%, transparent)" }}
          />
          {t.legendWaiting}
        </span>
        {focusService && (
          <button
            onClick={() => setFocusService(null)}
            className="text-xs text-primary hover:underline"
          >
            {t.viewFullTrace}
          </button>
        )}
      </div>
    </div>
  );
}
