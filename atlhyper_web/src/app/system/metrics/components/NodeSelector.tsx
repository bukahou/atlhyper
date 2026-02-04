"use client";

import { memo } from "react";
import { Server, CheckCircle, XCircle, Activity } from "lucide-react";
import type { NodeListItem } from "@/types/node-metrics";

interface NodeSelectorProps {
  nodes: NodeListItem[];
  selectedNode: string;
  onSelect: (nodeName: string) => void;
}

export const NodeSelector = memo(function NodeSelector({
  nodes,
  selectedNode,
  onSelect,
}: NodeSelectorProps) {
  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
      <div className="flex items-center gap-2 mb-4">
        <Server className="w-5 h-5 text-blue-500" />
        <h3 className="text-base font-semibold text-default">Nodes</h3>
        <span className="text-xs text-muted bg-[var(--background)] px-2 py-0.5 rounded-full">
          {nodes.length}
        </span>
      </div>

      <div className="space-y-2">
        {nodes.map((node) => (
          <button
            key={node.name}
            onClick={() => onSelect(node.name)}
            disabled={!node.hasMetrics}
            className={`w-full flex items-center gap-3 p-3 rounded-lg transition-all ${
              selectedNode === node.name
                ? "bg-blue-500/10 border border-blue-500/50"
                : node.hasMetrics
                ? "bg-[var(--background)] border border-transparent hover:border-[var(--border-color)]"
                : "bg-[var(--background)] border border-transparent opacity-50 cursor-not-allowed"
            }`}
          >
            {/* 状态图标 */}
            {node.status === "Ready" ? (
              <CheckCircle className="w-4 h-4 text-green-500 flex-shrink-0" />
            ) : (
              <XCircle className="w-4 h-4 text-red-500 flex-shrink-0" />
            )}

            {/* 节点信息 */}
            <div className="flex-1 text-left min-w-0">
              <div className="text-sm font-medium text-default truncate">
                {node.name}
              </div>
              <div className="flex items-center gap-2 mt-0.5">
                {node.roles.map((role) => (
                  <span
                    key={role}
                    className="text-xs text-muted bg-[var(--background)] px-1.5 py-0.5 rounded"
                  >
                    {role}
                  </span>
                ))}
              </div>
            </div>

            {/* Metrics 状态 */}
            {node.hasMetrics ? (
              <Activity className="w-4 h-4 text-green-500 flex-shrink-0" />
            ) : (
              <span className="text-xs text-muted">No metrics</span>
            )}
          </button>
        ))}
      </div>
    </div>
  );
});
