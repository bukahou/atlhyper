"use client";

import { useI18n } from "@/i18n/context";

type ViewMode = "service" | "anomaly" | "full";

const VIEW_MODES: ViewMode[] = ["anomaly", "service", "full"];

const VIEW_LABEL_KEYS: Record<ViewMode, "viewService" | "viewAnomaly" | "viewFull"> = {
  service: "viewService",
  anomaly: "viewAnomaly",
  full: "viewFull",
};

interface TopologyToolbarProps {
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  allNamespaces: string[];
  nsFilter: Set<string>;
  onToggleNs: (ns: string) => void;
  onResetNsFilter: () => void;
}

export function TopologyToolbar({
  viewMode,
  onViewModeChange,
  allNamespaces,
  nsFilter,
  onToggleNs,
  onResetNsFilter,
}: TopologyToolbarProps) {
  const { t } = useI18n();

  // Service 视图只展示 service/ingress 形状图例
  const showAllShapes = viewMode !== "service";

  return (
    <>
      {/* 视图切换 + 图例 */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        {/* SegmentedControl */}
        <div className="flex rounded-lg border border-[var(--border-color)] overflow-hidden text-xs">
          {VIEW_MODES.map((mode) => (
            <button
              key={mode}
              onClick={() => onViewModeChange(mode)}
              className={`px-3 py-1.5 transition-colors ${
                viewMode === mode
                  ? "bg-blue-500 text-white"
                  : "bg-[var(--background)] text-muted hover:text-default"
              }`}
            >
              {t.aiops[VIEW_LABEL_KEYS[mode]]}
            </button>
          ))}
        </div>

        {/* 图例 */}
        <div className="flex flex-wrap gap-3 text-xs text-muted">
          {/* 形状图例 */}
          <span className="flex items-center gap-1.5">
            <span className="w-3.5 h-3.5 rounded-full border-2 border-current inline-block" /> Service
          </span>
          <span className="flex items-center gap-1.5">
            <svg className="w-3.5 h-3.5" viewBox="-10 -10 20 20">
              <polygon points="0,-7 7,0 0,7 -7,0" fill="none" stroke="currentColor" strokeWidth={1.5} />
            </svg>
            Ingress
          </span>
          {showAllShapes && (
            <>
              <span className="flex items-center gap-1.5">
                <span className="w-3.5 h-3.5 rounded-sm border-2 border-current inline-block" /> Pod
              </span>
              <span className="flex items-center gap-1.5">
                <svg className="w-3.5 h-3.5" viewBox="-10 -10 20 20">
                  <polygon points="0,-8 7,4 -7,4" fill="none" stroke="currentColor" strokeWidth={1.5} />
                </svg>
                Node
              </span>
            </>
          )}
          <span className="mx-1 text-[var(--border-color)]">|</span>
          <span className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-[#22c55e] inline-block" /> Healthy
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-[#eab308] inline-block" /> Warning
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-[#ef4444] inline-block" /> Critical
          </span>
        </div>
      </div>

      {/* Full 视图: Namespace 筛选 */}
      {viewMode === "full" && allNamespaces.length > 1 && (
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs text-muted">{t.common.namespace}:</span>
          {allNamespaces.map((ns) => {
            const isActive = nsFilter.size > 0 && nsFilter.has(ns);
            const isDimmed = nsFilter.size > 0 && !nsFilter.has(ns);
            return (
              <button
                key={ns}
                onClick={() => onToggleNs(ns)}
                className={`px-2.5 py-1 rounded-full text-xs transition-colors border ${
                  isActive
                    ? "bg-blue-500/15 text-blue-500 border-blue-500/30"
                    : isDimmed
                      ? "bg-[var(--background)] text-muted/40 border-[var(--border-color)] hover:text-muted"
                      : "bg-[var(--background)] text-muted border-[var(--border-color)] hover:text-default"
                }`}
              >
                {ns}
              </button>
            );
          })}
          {nsFilter.size > 0 && (
            <button
              onClick={onResetNsFilter}
              className="px-2 py-1 text-xs text-muted hover:text-default transition-colors"
            >
              {t.common.reset}
            </button>
          )}
        </div>
      )}
    </>
  );
}
