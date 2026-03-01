"use client";

import { useState, useMemo } from "react";
import { Shield } from "lucide-react";
import { getNamespaceColor, formatLatency, formatRPS } from "./common";
import type { MeshServiceNode } from "@/types/mesh";
import type { MeshTabTranslations } from "./MeshTypes";

// Service List Table (sortable)
export function ServiceListTable({ nodes, selectedId, onSelect, t }: {
  nodes: MeshServiceNode[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  t: MeshTabTranslations;
}) {
  const [sortKey, setSortKey] = useState<"name" | "rps" | "p95Latency" | "errorRate">("rps");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");

  const toggleSort = (key: typeof sortKey) => {
    if (sortKey === key) setSortDir(d => d === "asc" ? "desc" : "asc");
    else { setSortKey(key); setSortDir("desc"); }
  };

  const sorted = useMemo(() => {
    const arr = [...nodes];
    arr.sort((a, b) => {
      const aVal = sortKey === "name" ? a.name : a[sortKey];
      const bVal = sortKey === "name" ? b.name : b[sortKey];
      if (typeof aVal === "string" && typeof bVal === "string") return sortDir === "asc" ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
      return sortDir === "asc" ? (aVal as number) - (bVal as number) : (bVal as number) - (aVal as number);
    });
    return arr;
  }, [nodes, sortKey, sortDir]);

  const SortHeader = ({ label, field }: { label: string; field: typeof sortKey }) => (
    <button onClick={() => toggleSort(field)}
      className={`flex items-center gap-1 text-[10px] font-medium uppercase tracking-wider ${sortKey === field ? "text-primary" : "text-muted hover:text-default"}`}>
      {label}
      {sortKey === field && <span className="text-[8px]">{sortDir === "asc" ? "\u25B2" : "\u25BC"}</span>}
    </button>
  );

  return (
    <div className="overflow-auto">
      <table className="w-full text-xs">
        <thead>
          <tr className="border-b border-[var(--border-color)]">
            <th className="text-left py-2 px-2"><SortHeader label={t.service} field="name" /></th>
            <th className="text-right py-2 px-2"><SortHeader label={t.rps} field="rps" /></th>
            <th className="text-right py-2 px-2"><SortHeader label="P95" field="p95Latency" /></th>
            <th className="text-right py-2 px-2"><SortHeader label={t.errorRate} field="errorRate" /></th>
            <th className="text-center py-2 px-2"><span className="text-[10px] font-medium uppercase tracking-wider text-muted">{t.mtls}</span></th>
            <th className="text-center py-2 px-2"><span className="text-[10px] font-medium uppercase tracking-wider text-muted">{t.status}</span></th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((node) => {
            const nsColor = getNamespaceColor(node.namespace);
            return (
              <tr key={node.id} onClick={() => onSelect(node.id)}
                className={`cursor-pointer transition-colors border-b border-[var(--border-color)] ${selectedId === node.id ? "bg-primary/5 dark:bg-primary/10" : "hover:bg-[var(--hover-bg)]"}`}>
                <td className="py-2.5 px-2">
                  <div className="flex items-center gap-2">
                    <span className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ backgroundColor: nsColor.fill }} />
                    <div>
                      <div className="font-medium text-default">{node.name}</div>
                      <div className="text-[10px] text-muted">{node.namespace}</div>
                    </div>
                  </div>
                </td>
                <td className="text-right py-2.5 px-2 font-medium text-default">{formatRPS(node.rps)}<span className="text-muted">/s</span></td>
                <td className="text-right py-2.5 px-2 font-medium text-default">{formatLatency(node.p95Latency)}</td>
                <td className="text-right py-2.5 px-2">
                  <span className={node.errorRate > 0.5 ? "text-red-500 font-semibold" : "text-default font-medium"}>{node.errorRate.toFixed(2)}%</span>
                </td>
                <td className="text-center py-2.5 px-2">
                  <span className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium ${node.mtlsEnabled ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400" : "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400"}`}>
                    <Shield className="w-3 h-3" />
                    {node.mtlsEnabled ? "ON" : "OFF"}
                  </span>
                </td>
                <td className="text-center py-2.5 px-2">
                  <span className={`inline-block w-2 h-2 rounded-full ${node.status === "healthy" ? "bg-emerald-500" : node.status === "warning" ? "bg-amber-500" : "bg-red-500"}`} />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
