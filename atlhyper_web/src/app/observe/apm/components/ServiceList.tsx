"use client";

import { useState, useMemo } from "react";
import { Search, ArrowUpDown, ArrowUp, ArrowDown, ChevronLeft, ChevronRight, ChevronDown } from "lucide-react";
import type { TraceSummary } from "@/api/apm";
import type { ServiceStats } from "@/api/apm";
import type { ApmTranslations } from "@/types/i18n";
import type { TableTranslations } from "@/types/i18n";
import { MiniSparkline } from "./MiniSparkline";

const PAGE_SIZE_OPTIONS = [10, 25, 50] as const;

interface ServiceListProps {
  t: ApmTranslations;
  tt: TableTranslations;
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
  tt,
  serviceStats,
  traces,
  onSelectService,
}: ServiceListProps) {
  const [search, setSearch] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("throughput");
  const [sortDir, setSortDir] = useState<SortDir>("desc");
  const [pageSize, setPageSize] = useState<number>(25);
  const [currentPage, setCurrentPage] = useState(1);
  const [showPageSizeDropdown, setShowPageSizeDropdown] = useState(false);

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

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const safePage = Math.min(currentPage, totalPages);
  const paged = filtered.slice((safePage - 1) * pageSize, safePage * pageSize);

  // Reset to page 1 when search changes
  const handleSearch = (v: string) => { setSearch(v); setCurrentPage(1); };

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
          onChange={(e) => handleSearch(e.target.value)}
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
            {paged.map((svc) => (
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

        {paged.length === 0 && (
          <div className="text-center py-12 text-muted text-sm">
            {t.noTraces}
          </div>
        )}

        {/* Pagination footer */}
        {filtered.length > 0 && (
          <div className="flex items-center justify-between px-4 py-2.5 border-t border-[var(--border-color)] text-xs text-muted">
            {/* Rows per page */}
            <div className="relative flex items-center gap-1.5">
              <span>{tt.rowsPerPage}:</span>
              <button
                onClick={() => setShowPageSizeDropdown((v) => !v)}
                className="flex items-center gap-0.5 px-1.5 py-0.5 rounded hover:bg-[var(--hover-bg)] transition-colors text-default"
              >
                {pageSize}
                <ChevronDown className="w-3 h-3 text-muted" />
              </button>
              {showPageSizeDropdown && (
                <>
                  <div className="fixed inset-0 z-40" onClick={() => setShowPageSizeDropdown(false)} />
                  <div className="absolute left-0 bottom-full mb-1 z-50 py-1 rounded-lg border border-[var(--border-color)] bg-card shadow-lg">
                    {PAGE_SIZE_OPTIONS.map((size) => (
                      <button
                        key={size}
                        onClick={() => { setPageSize(size); setCurrentPage(1); setShowPageSizeDropdown(false); }}
                        className={`block w-full text-left px-4 py-1.5 text-xs transition-colors ${
                          pageSize === size ? "text-primary bg-primary/5" : "text-default hover:bg-[var(--hover-bg)]"
                        }`}
                      >
                        {size}
                      </button>
                    ))}
                  </div>
                </>
              )}
            </div>

            {/* Page navigation */}
            <div className="flex items-center gap-1">
              <button
                onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                disabled={safePage <= 1}
                className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors"
              >
                <ChevronLeft className="w-3.5 h-3.5" />
              </button>
              {Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
                <button
                  key={p}
                  onClick={() => setCurrentPage(p)}
                  className={`min-w-[24px] h-6 rounded text-xs transition-colors ${
                    p === safePage ? "text-primary font-semibold" : "text-muted hover:bg-[var(--hover-bg)]"
                  }`}
                >
                  {p}
                </button>
              ))}
              <button
                onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                disabled={safePage >= totalPages}
                className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors"
              >
                <ChevronRight className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
