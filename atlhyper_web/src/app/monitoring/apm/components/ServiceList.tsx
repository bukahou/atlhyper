"use client";

import { useState, useMemo } from "react";
import { Search, ArrowUpDown, ArrowUp, ArrowDown } from "lucide-react";
import type { TraceSummary } from "@/api/apm";
import type { ServiceStats } from "@/api/apm";
import type { ApmTranslations } from "@/types/i18n";
import { MiniSparkline } from "./MiniSparkline";

interface ServiceListProps {
  t: ApmTranslations;
  serviceStats: ServiceStats[];
  traces: TraceSummary[];
  onSelectService: (serviceName: string) => void;
}

function formatDuration(us: number): string {
  if (us < 1000) return `${us.toFixed(0)}μs`;
  if (us < 1_000_000) return `${(us / 1000).toFixed(1)}ms`;
  return `${(us / 1_000_000).toFixed(2)}s`;
}

type SortKey = "name" | "latencyAvg" | "throughput" | "errorRate";
type SortDir = "asc" | "desc";

export function ServiceList({
  t,
  serviceStats,
  traces,
  onSelectService,
}: ServiceListProps) {
  const [search, setSearch] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("throughput");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const filtered = useMemo(() => {
    let list = serviceStats;
    if (search) {
      const q = search.toLowerCase();
      list = list.filter((s) => s.name.toLowerCase().includes(q));
    }
    list = [...list].sort((a, b) => {
      const av = a[sortKey] ?? 0;
      const bv = b[sortKey] ?? 0;
      if (typeof av === "string" && typeof bv === "string") {
        return sortDir === "asc" ? av.localeCompare(bv) : bv.localeCompare(av);
      }
      return sortDir === "asc"
        ? (av as number) - (bv as number)
        : (bv as number) - (av as number);
    });
    return list;
  }, [serviceStats, search, sortKey, sortDir]);

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("desc");
    }
  };

  const SortIcon = ({ col }: { col: SortKey }) => {
    if (sortKey !== col) return <ArrowUpDown className="w-3 h-3 text-muted/50" />;
    return sortDir === "asc" ? (
      <ArrowUp className="w-3 h-3 text-primary" />
    ) : (
      <ArrowDown className="w-3 h-3 text-primary" />
    );
  };

  return (
    <div className="space-y-3">
      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t.searchServices}
          className="w-full pl-9 pr-4 py-2 text-sm rounded-lg border border-[var(--border-color)] bg-card text-default placeholder:text-muted focus:outline-none focus:ring-2 focus:ring-primary/30"
        />
      </div>

      {/* Table */}
      <div className="border border-[var(--border-color)] rounded-xl overflow-hidden bg-card">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-[var(--border-color)] bg-[var(--hover-bg)]">
              <th className="text-left px-4 py-2.5 font-medium text-muted">
                <button onClick={() => toggleSort("name")} className="flex items-center gap-1 hover:text-default transition-colors">
                  {t.serviceName}
                  <SortIcon col="name" />
                </button>
              </th>
              <th className="text-left px-3 py-2.5 font-medium text-muted w-24">
                {t.environment}
              </th>
              <th className="text-left px-3 py-2.5 font-medium text-muted">
                <button onClick={() => toggleSort("latencyAvg")} className="flex items-center gap-1 hover:text-default transition-colors">
                  {t.latencyAvg}
                  <SortIcon col="latencyAvg" />
                </button>
              </th>
              <th className="text-left px-3 py-2.5 font-medium text-muted">
                <button onClick={() => toggleSort("throughput")} className="flex items-center gap-1 hover:text-default transition-colors">
                  {t.throughput}
                  <SortIcon col="throughput" />
                </button>
              </th>
              <th className="text-left px-3 py-2.5 font-medium text-muted">
                <button onClick={() => toggleSort("errorRate")} className="flex items-center gap-1 hover:text-default transition-colors">
                  {t.errorRate}
                  <SortIcon col="errorRate" />
                </button>
              </th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((svc) => (
              <tr
                key={svc.name}
                onClick={() => onSelectService(svc.name)}
                className="border-b border-[var(--border-color)] last:border-b-0 hover:bg-[var(--hover-bg)] cursor-pointer transition-colors"
              >
                {/* Name */}
                <td className="px-4 py-3">
                  <span className="text-primary hover:underline font-medium">
                    {svc.name}
                  </span>
                </td>
                {/* Environment */}
                <td className="px-3 py-3">
                  <span className="inline-block text-xs px-2 py-0.5 rounded-full bg-[var(--hover-bg)] text-muted border border-[var(--border-color)]">
                    {svc.environment}
                  </span>
                </td>
                {/* Latency */}
                <td className="px-3 py-3">
                  <div className="flex items-center gap-2">
                    <span className="text-default whitespace-nowrap">
                      {formatDuration(svc.latencyAvg)}
                    </span>
                    <MiniSparkline
                      data={svc.latencyPoints}
                      type="line"
                      color="#6366f1"
                    />
                  </div>
                </td>
                {/* Throughput */}
                <td className="px-3 py-3">
                  <div className="flex items-center gap-2">
                    <span className="text-default whitespace-nowrap">
                      {svc.throughput.toFixed(1)} {t.tpm}
                    </span>
                    <MiniSparkline
                      data={svc.latencyPoints.map(() => svc.throughput)}
                      type="bar"
                      color="#22c55e"
                    />
                  </div>
                </td>
                {/* Error Rate */}
                <td className="px-3 py-3">
                  <div className="flex items-center gap-2">
                    <span
                      className={
                        svc.errorRate > 0 ? "text-orange-500" : "text-default"
                      }
                    >
                      {(svc.errorRate * 100).toFixed(1)}%
                    </span>
                    <MiniSparkline
                      data={svc.errorRatePoints}
                      type="line"
                      color={svc.errorRate > 0 ? "#f97316" : "#9ca3af"}
                    />
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {filtered.length === 0 && (
          <div className="text-center py-12 text-muted text-sm">
            {t.noTraces}
          </div>
        )}
      </div>
    </div>
  );
}
