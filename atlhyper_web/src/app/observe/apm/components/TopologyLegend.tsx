import type { ApmTranslations } from "@/types/i18n";

interface TopologyStats {
  nodeCount: number;
  errorNodes: number;
  totalCalls: number;
  avgP99: number;
}

interface TopologyHeaderProps {
  t: ApmTranslations;
  stats: TopologyStats;
  expanded: boolean;
  onToggleExpand: () => void;
  formatDurationMs: (ms: number) => string;
}

export function TopologyHeader({ t, stats, expanded, onToggleExpand, formatDurationMs }: TopologyHeaderProps) {
  // Lazy import icons to keep bundle small — inline SVG avoids extra import
  return (
    <div className="px-4 py-2.5 border-b border-[var(--border-color)] flex items-center justify-between">
      <div className="flex items-center gap-4">
        <h3 className="text-sm font-medium text-default">{t.serviceTopology}</h3>
        <div className="flex items-center gap-2">
          <StatPill label={t.services} value={String(stats.nodeCount)} />
          <StatPill label={t.topoCalls} value={String(stats.totalCalls)} />
          <StatPill label="P99" value={formatDurationMs(stats.avgP99)} />
          {stats.errorNodes > 0 && (
            <StatPill label={t.topoErrorRate} value={String(stats.errorNodes)} variant="error" />
          )}
        </div>
      </div>
      <button
        onClick={onToggleExpand}
        className="p-1.5 rounded-lg hover:bg-[var(--hover-bg)] transition-colors text-muted"
      >
        {expanded ? (
          <ExpandIcon shrink />
        ) : (
          <ExpandIcon />
        )}
      </button>
    </div>
  );
}

function ExpandIcon({ shrink }: { shrink?: boolean }) {
  // Minimize2 / Maximize2 from lucide — inline to avoid importing full icon set
  if (shrink) {
    return (
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none"
        stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <polyline points="4 14 10 14 10 20" /><polyline points="20 10 14 10 14 4" />
        <line x1="14" y1="10" x2="21" y2="3" /><line x1="3" y1="21" x2="10" y2="14" />
      </svg>
    );
  }
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none"
      stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="15 3 21 3 21 9" /><polyline points="9 21 3 21 3 15" />
      <line x1="21" y1="3" x2="14" y2="10" /><line x1="3" y1="21" x2="10" y2="14" />
    </svg>
  );
}

export function TopologyLegend({ t }: { t: ApmTranslations }) {
  return (
    <div className="px-4 py-2 border-t border-[var(--border-color)] flex items-center gap-4 text-[10px] text-muted">
      <span className="flex items-center gap-1.5">
        <span className="w-3 h-3 rounded-full border-2 border-[#60a5fa] bg-[#60a5fa]/10 inline-block" />
        {t.nodeTypeService}
      </span>
      <span className="flex items-center gap-1.5">
        <span className="w-3 h-3 rotate-45 border-2 border-[#60a5fa] bg-[#60a5fa]/10 inline-block" style={{ borderRadius: 2 }} />
        {t.nodeTypeDatabase}
      </span>
      <span className="flex items-center gap-1.5">
        <span className="w-3 h-3 rounded-full border-2 border-[#f59e0b] bg-[#f59e0b]/10 inline-block" />
        {t.topoErrorRate} &gt;1%
      </span>
      <span className="flex items-center gap-1.5">
        <span className="w-3 h-3 rounded-full border-2 border-[#ef4444] bg-[#ef4444]/10 inline-block" />
        {t.topoErrorRate} &gt;5%
      </span>
    </div>
  );
}

export function StatPill({ label, value, variant }: { label: string; value: string; variant?: "error" }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[11px] ${
        variant === "error"
          ? "bg-red-500/10 text-red-400"
          : "bg-[var(--hover-bg)] text-muted"
      }`}
    >
      <span className="opacity-70">{label}</span>
      <span className="font-semibold text-default">{value}</span>
    </span>
  );
}
