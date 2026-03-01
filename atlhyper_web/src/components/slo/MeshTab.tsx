"use client";

import { useState } from "react";
import { Network, Layers, Shield } from "lucide-react";
import { ServiceTopologyView } from "./MeshTopology";
import { ServiceListTable } from "./MeshServiceList";
import { ServiceDetailPanel } from "./MeshServiceDetail";
import type { MeshTopologyResponse } from "@/types/mesh";
import type { MeshTabTranslations } from "./MeshTypes";
import { timeRangeLabel } from "./MeshTypes";

// Re-export MeshTabTranslations for external consumers
export type { MeshTabTranslations } from "./MeshTypes";

// Main Mesh Tab
export function MeshTab({ topology, clusterId, timeRange, t }: {
  topology: MeshTopologyResponse | null;
  clusterId: string;
  timeRange: string;
  t: MeshTabTranslations;
}) {
  const [selectedServiceId, setSelectedServiceId] = useState<string | null>(null);

  if (!topology || topology.nodes.length === 0) {
    return (
      <div className="text-center py-8 text-sm text-muted">
        <Network className="w-8 h-8 mx-auto mb-2 opacity-50" />
        {t.noCallData}
      </div>
    );
  }

  const effectiveId = selectedServiceId ?? topology.nodes[0]?.id ?? null;
  const selectedNode = effectiveId ? topology.nodes.find(n => n.id === effectiveId) : null;

  // mTLS: check if any node has mTLS enabled
  const mtlsEnabled = topology.nodes.some(n => n.mtlsEnabled);

  return (
    <div className="space-y-4">
      {/* Topology Graph */}
      {topology.nodes.length > 1 && (
        <ServiceTopologyView topology={topology} onSelectNode={(node) => setSelectedServiceId(node.id)} timeRange={timeRange} t={t} />
      )}

      {/* Service Mesh Overview (table + detail) */}
      <div className="rounded-xl border border-[var(--border-color)] bg-card overflow-hidden">
        <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Layers className="w-4 h-4 text-primary" />
            <span className="text-sm font-semibold text-default">{t.meshOverview}</span>
            <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400">Linkerd</span>
            <span className="text-[10px] text-muted">{topology.nodes.length} services · {timeRangeLabel(timeRange)}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-[10px] font-medium ${mtlsEnabled ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400" : "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400"}`}>
              <Shield className="w-3 h-3" />
              {t.mtls} {mtlsEnabled ? "ON" : "OFF"}
            </span>
          </div>
        </div>
        <div className="flex flex-col lg:flex-row">
          <div className={`${selectedNode ? "lg:w-[400px] lg:border-r border-[var(--border-color)]" : "w-full"} p-4`}>
            <ServiceListTable nodes={topology.nodes} selectedId={effectiveId} onSelect={setSelectedServiceId} t={t} />
          </div>
          {selectedNode && (
            <div className="flex-1 p-4 bg-[var(--background)]">
              <ServiceDetailPanel node={selectedNode} topology={topology} clusterId={clusterId} timeRange={timeRange} t={t} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
