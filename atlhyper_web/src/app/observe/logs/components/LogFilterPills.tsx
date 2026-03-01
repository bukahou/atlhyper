"use client";

import { X, Clock } from "lucide-react";

/** Pill color by filter type */
function pillStyle(type: "severity" | "service" | "scope", value?: string) {
  if (type === "severity") {
    switch (value?.toUpperCase()) {
      case "ERROR": return "bg-red-500/15 text-red-600 dark:text-red-400";
      case "WARN": return "bg-amber-500/15 text-amber-600 dark:text-amber-400";
      case "INFO": return "bg-blue-500/15 text-blue-600 dark:text-blue-400";
      case "DEBUG": return "bg-gray-500/15 text-gray-600 dark:text-gray-400";
    }
  }
  if (type === "service") return "bg-purple-500/15 text-purple-600 dark:text-purple-400";
  return "bg-gray-500/10 text-gray-600 dark:text-gray-400";
}

/** Format epoch ms pair to "HH:MM — HH:MM" */
function formatBrushRange(start: number, end: number): string {
  const fmt = (ts: number) =>
    new Date(ts).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  return `${fmt(start)} — ${fmt(end)}`;
}

interface LogFilterPillsProps {
  selectedSeverities: string[];
  selectedServices: string[];
  selectedScopes: string[];
  brushTimeRange: [number, number] | null;
  urlTraceId?: string;
  onRemoveSeverity: (v: string) => void;
  onRemoveService: (v: string) => void;
  onRemoveScope: (v: string) => void;
  onClearBrushTimeRange: () => void;
  onClearAll: () => void;
  clearFiltersLabel: string;
}

export function LogFilterPills({
  selectedSeverities,
  selectedServices,
  selectedScopes,
  brushTimeRange,
  urlTraceId,
  onRemoveSeverity,
  onRemoveService,
  onRemoveScope,
  onClearBrushTimeRange,
  onClearAll,
  clearFiltersLabel,
}: LogFilterPillsProps) {
  const hasAnyFilter =
    selectedServices.length > 0 ||
    selectedSeverities.length > 0 ||
    selectedScopes.length > 0 ||
    brushTimeRange !== null ||
    !!urlTraceId;

  if (!hasAnyFilter) return null;

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {/* Brush time range pill */}
      {brushTimeRange && (
        <button
          onClick={onClearBrushTimeRange}
          className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-primary/15 text-primary transition-colors hover:opacity-80"
        >
          <Clock className="w-3 h-3" />
          {formatBrushRange(brushTimeRange[0], brushTimeRange[1])}
          <X className="w-3 h-3" />
        </button>
      )}
      {selectedSeverities.map((v) => (
        <button
          key={`sev-${v}`}
          onClick={() => onRemoveSeverity(v)}
          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium transition-colors hover:opacity-80 ${pillStyle("severity", v)}`}
        >
          {v}
          <X className="w-3 h-3" />
        </button>
      ))}
      {selectedServices.map((v) => (
        <button
          key={`svc-${v}`}
          onClick={() => onRemoveService(v)}
          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium transition-colors hover:opacity-80 ${pillStyle("service")}`}
        >
          {v.replace("geass-", "")}
          <X className="w-3 h-3" />
        </button>
      ))}
      {selectedScopes.map((v) => (
        <button
          key={`scope-${v}`}
          onClick={() => onRemoveScope(v)}
          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium transition-colors hover:opacity-80 ${pillStyle("scope")}`}
        >
          {v.split(".").pop()}
          <X className="w-3 h-3" />
        </button>
      ))}
      {urlTraceId && (
        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-primary/15 text-primary">
          Trace: {urlTraceId.slice(0, 8)}...
        </span>
      )}
      <button
        onClick={onClearAll}
        className="text-xs text-muted hover:text-default transition-colors ml-1"
      >
        {clearFiltersLabel}
      </button>
    </div>
  );
}
