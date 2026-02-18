"use client";

import { EntityLink } from "@/components/aiops/EntityLink";
import type { CausalTreeNode } from "@/api/aiops";
import type { AIOpsTranslations } from "@/types/i18n";

interface CausalTreeNodeViewProps {
  node: CausalTreeNode;
  depth: number;
  t: AIOpsTranslations;
}

export function CausalTreeNodeView({ node, depth, t }: CausalTreeNodeViewProps) {
  const indent = depth * 16;
  const directionLabel = node.direction === "upstream" ? "↑" : "↓";
  const edgeLabel = node.edgeType ? `(${node.edgeType})` : "";

  return (
    <div style={{ marginLeft: indent }}>
      <div className="flex items-center gap-2 text-xs py-0.5">
        <span className="text-muted">{directionLabel}</span>
        <EntityLink entityKey={node.entityKey} showType={false} />
        <span className="text-[10px] text-muted">{edgeLabel}</span>
        {node.rFinal > 0 && (
          <span className="font-mono text-default text-[10px]">R={node.rFinal.toFixed(2)}</span>
        )}
      </div>
      {node.metrics?.map((m, i) => (
        <div
          key={i}
          className="flex items-center gap-2 text-[10px] text-muted"
          style={{ marginLeft: indent + 16 }}
        >
          <span className="font-mono">{m.metricName}</span>
          <span>{m.deviation.toFixed(1)}σ</span>
        </div>
      ))}
      {node.children?.map((child, i) => (
        <CausalTreeNodeView key={i} node={child} depth={depth + 1} t={t} />
      ))}
    </div>
  );
}
