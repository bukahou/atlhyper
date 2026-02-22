"use client";

import { memo } from "react";
import { Server, Activity } from "lucide-react";
import type { NodeMetrics } from "@/types/node-metrics";
import { useI18n } from "@/i18n/context";

interface NodeSelectorProps {
  nodes: NodeMetrics[];
  selectedNode: string;
  onSelect: (nodeName: string) => void;
}

export const NodeSelector = memo(function NodeSelector({
  nodes,
  selectedNode,
  onSelect,
}: NodeSelectorProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
      <div className="flex items-center gap-2 mb-4">
        <Server className="w-5 h-5 text-blue-500" />
        <h3 className="text-base font-semibold text-default">{nm.summary.nodes}</h3>
        <span className="text-xs text-muted bg-[var(--background)] px-2 py-0.5 rounded-full">
          {nodes.length}
        </span>
      </div>

      <div className="space-y-2">
        {nodes.map((node) => (
          <button
            key={node.nodeName}
            onClick={() => onSelect(node.nodeName)}
            className={`w-full flex items-center gap-3 p-3 rounded-lg transition-all ${
              selectedNode === node.nodeName
                ? "bg-blue-500/10 border border-blue-500/50"
                : "bg-[var(--background)] border border-transparent hover:border-[var(--border-color)]"
            }`}
          >
            <div className="flex-1 text-left min-w-0">
              <div className="text-sm font-medium text-default truncate">
                {node.nodeName}
              </div>
              <div className="text-xs text-muted mt-0.5">{node.nodeIP}</div>
            </div>
            <Activity className="w-4 h-4 text-green-500 flex-shrink-0" />
          </button>
        ))}
      </div>
    </div>
  );
});
