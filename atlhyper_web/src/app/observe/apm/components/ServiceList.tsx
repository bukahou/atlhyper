"use client";

import { useState, useMemo } from "react";
import { Search, ArrowUpDown, ArrowUp, ArrowDown, ChevronLeft, ChevronRight, ChevronDown } from "lucide-react";
import type { APMService } from "@/types/model/apm";
import type { ApmTranslations, TableTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";

const PAGE_SIZE_OPTIONS = [10, 25, 50] as const;

interface ServiceListProps {
  t: ApmTranslations;
  tt: TableTranslations;
  serviceStats: APMService[];
  onSelectService: (serviceName: string) => void;
}

type SortKey = "name" | "avgDurationMs" | "rps" | "successRate" | "p50Ms" | "p99Ms";
type SortDir = "asc" | "desc";

export function ServiceList({
  t,
  tt,
  serviceStats,
  onSelectService,
}: ServiceListProps) {
  const [search, setSearch] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("rps");
  const [sortDir, setSortDir] = useState<SortDir>("desc");
  const [pageSize, setPageSize] = useState<number>(25);
  const [currentPage, setCurrentPage] = useState(1);
  const [showPageSizeDropdown, setShowPageSizeDropdown] = useState(false);

  const filtered = useMemo(() => {
    let list = serviceStats;
    if (search) {
      const q = search.toLowerCase();
      list = list.filter((s) => s.name.toLowerCase().includes(q) || s.namespace.toLowerCase().includes(q));
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
              <th className="text-left px-3 py-2.5 font-medium text-muted">
                {t.tags}
              </th>
              <th className="text-left px-3 py-2.5 font-medium text-muted">
                <button onClick={() => toggleSort("avgDurationMs")} className="flex items-center gap-1 hover:text-default transition-colors">
                  {t.latencyAvg}
                  <SortIcon col="avgDurationMs" />
                </button>
              </th>
              <th className="text-left px-3 py-2.5 font-medium text-muted">
                <button onClick={() => toggleSort("p50Ms")} className="flex items-center gap-1 hover:text-default transition-colors">
                  P50
                  <SortIcon col="p50Ms" />
                </button>
              </th>
              <th className="text-left px-3 py-2.5 font-medium text-muted">
                <button onClick={() => toggleSort("p99Ms")} className="flex items-center gap-1 hover:text-default transition-colors">
                  P99
                  <SortIcon col="p99Ms" />
                </button>
              </th>
              <th className="text-left px-3 py-2.5 font-medium text-muted">
                <button onClick={() => toggleSort("rps")} className="flex items-center gap-1 hover:text-default transition-colors">
                  RPS
                  <SortIcon col="rps" />
                </button>
              </th>
              <th className="text-left px-3 py-2.5 font-medium text-muted">
                <button onClick={() => toggleSort("successRate")} className="flex items-center gap-1 hover:text-default transition-colors">
                  {t.successRate}
                  <SortIcon col="successRate" />
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
                <td className="px-4 py-3">
                  <span className="text-primary hover:underline font-medium">
                    {svc.name}
                  </span>
                </td>
                <td className="px-3 py-3">
                  <div className="flex flex-wrap gap-1">
                    {svc.namespace && (
                      <span className="inline-block text-xs px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20">
                        {svc.namespace}
                      </span>
                    )}
                    {svc.environment && (
                      <span className="inline-block text-xs px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                        {svc.environment}
                      </span>
                    )}
                  </div>
                </td>
                <td className="px-3 py-3 text-default whitespace-nowrap">
                  {formatDurationMs(svc.avgDurationMs)}
                </td>
                <td className="px-3 py-3 text-default whitespace-nowrap">
                  {formatDurationMs(svc.p50Ms)}
                </td>
                <td className="px-3 py-3 text-default whitespace-nowrap">
                  {formatDurationMs(svc.p99Ms)}
                </td>
                <td className="px-3 py-3 text-default whitespace-nowrap">
                  {svc.rps.toFixed(3)}
                </td>
                <td className="px-3 py-3">
                  <span
                    className={
                      svc.successRate < 0.99 ? "text-orange-500" : "text-emerald-500"
                    }
                  >
                    {(svc.successRate * 100).toFixed(1)}%
                  </span>
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
