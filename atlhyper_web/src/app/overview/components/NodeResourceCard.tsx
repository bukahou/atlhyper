"use client";

import { memo, useState } from "react";
import type { TransformedOverview } from "@/types/overview";
import type { useI18n } from "@/i18n/context";

interface NodeResourceCardProps {
  nodes: TransformedOverview["nodeUsages"];
  t: ReturnType<typeof useI18n>["t"];
}

export const NodeResourceCard = memo(function NodeResourceCard({ nodes, t }: NodeResourceCardProps) {
  const [sortBy, setSortBy] = useState<"cpu" | "memory">("cpu");

  const getUsageColor = (usage: number) => {
    if (usage >= 80) return "bg-red-500";
    if (usage >= 60) return "bg-yellow-500";
    return "bg-green-500";
  };

  const sortedNodes = [...nodes].sort((a, b) => {
    if (sortBy === "cpu") return b.cpuPercent - a.cpuPercent;
    return b.memoryPercent - a.memoryPercent;
  });

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5 h-[320px] flex flex-col">
      <div className="flex items-center justify-between mb-4 flex-shrink-0">
        <h3 className="text-lg font-semibold text-default">{t.overview.nodeResourceUsage}</h3>
        <div className="flex gap-1">
          <button
            onClick={() => setSortBy("cpu")}
            className={`px-2 py-1 text-xs rounded transition-colors ${
              sortBy === "cpu"
                ? "bg-orange-500 text-white"
                : "bg-[var(--background)] text-muted hover:text-default"
            }`}
          >
            CPU
          </button>
          <button
            onClick={() => setSortBy("memory")}
            className={`px-2 py-1 text-xs rounded transition-colors ${
              sortBy === "memory"
                ? "bg-green-500 text-white"
                : "bg-[var(--background)] text-muted hover:text-default"
            }`}
          >
            Memory
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-y-auto space-y-4 pr-2">
        {nodes.length === 0 ? (
          <div className="text-center py-8 text-muted">{t.overview.noNodeData}</div>
        ) : (
          sortedNodes.map((node) => (
            <div key={node.nodeName} className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-default">{node.nodeName}</span>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="flex justify-between text-xs mb-1">
                    <span className="text-muted">CPU</span>
                    <span className="transition-all duration-300">{node.cpuPercent.toFixed(1)}%</span>
                  </div>
                  <div className="h-1.5 bg-[var(--background)] rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all duration-300 ${getUsageColor(node.cpuPercent)}`}
                      style={{ width: `${Math.min(100, node.cpuPercent)}%` }}
                    />
                  </div>
                </div>
                <div>
                  <div className="flex justify-between text-xs mb-1">
                    <span className="text-muted">Memory</span>
                    <span className="transition-all duration-300">{node.memoryPercent.toFixed(1)}%</span>
                  </div>
                  <div className="h-1.5 bg-[var(--background)] rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all duration-300 ${getUsageColor(node.memoryPercent)}`}
                      style={{ width: `${Math.min(100, node.memoryPercent)}%` }}
                    />
                  </div>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
});
