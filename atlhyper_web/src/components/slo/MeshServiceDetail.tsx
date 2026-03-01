"use client";

import { useState, useEffect } from "react";
import { Network, Server, ArrowRight, BarChart3, Loader2 } from "lucide-react";
import { getNamespaceColor, formatLatency, formatRPS } from "./common";
import { getMeshServiceDetail } from "@/datasource/mesh";
import { MiniLatencyHistogram } from "./MeshHistogram";
import type { MeshServiceNode, MeshTopologyResponse, MeshServiceDetailResponse } from "@/types/mesh";
import type { MeshTabTranslations } from "./MeshTypes";
import { timeRangeLabel } from "./MeshTypes";

// Status code colors — match by first character, supports "200"/"2xx" etc.
const statusColorMap: Record<string, { bar: string; bg: string; text: string }> = {
  "2": { bar: "bg-emerald-500", bg: "bg-emerald-50 dark:bg-emerald-900/20", text: "text-emerald-700 dark:text-emerald-400" },
  "3": { bar: "bg-blue-500", bg: "bg-blue-50 dark:bg-blue-900/20", text: "text-blue-700 dark:text-blue-400" },
  "4": { bar: "bg-amber-500", bg: "bg-amber-50 dark:bg-amber-900/20", text: "text-amber-700 dark:text-amber-400" },
  "5": { bar: "bg-red-500", bg: "bg-red-50 dark:bg-red-900/20", text: "text-red-700 dark:text-red-400" },
};
const defaultStatusColor = statusColorMap["2"];
function getStatusColor(code: string) { return statusColorMap[code[0]] || defaultStatusColor; }

// Service Detail Panel
export function ServiceDetailPanel({ node, topology, clusterId, timeRange, t }: {
  node: MeshServiceNode;
  topology: MeshTopologyResponse;
  clusterId: string;
  timeRange: string;
  t: MeshTabTranslations;
}) {
  const nsColor = getNamespaceColor(node.namespace);
  const inbound = topology.edges.filter(e => e.target === node.id);
  const outbound = topology.edges.filter(e => e.source === node.id);
  const [detail, setDetail] = useState<MeshServiceDetailResponse | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  // Lazy-load detail data when node changes
  useEffect(() => {
    let cancelled = false;
    setDetailLoading(true);
    getMeshServiceDetail({ clusterId, namespace: node.namespace, name: node.name, timeRange })
      .then(res => { if (!cancelled) setDetail(res.data); })
      .catch(() => { if (!cancelled) setDetail(null); })
      .finally(() => { if (!cancelled) setDetailLoading(false); });
    return () => { cancelled = true; };
  }, [node.id, clusterId, node.namespace, node.name, timeRange]);

  const statusCodes = detail?.statusCodes?.filter(s => s.count > 0) ?? [];
  const allLatencyBuckets = detail?.latencyBuckets ?? [];
  const latencyBuckets = allLatencyBuckets.filter(b => b.count > 0);
  const totalStatusRequests = statusCodes.reduce((sum, s) => sum + s.count, 0);

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 rounded-full flex items-center justify-center text-white shadow-md" style={{ backgroundColor: nsColor.fill }}>
          <Server className="w-4 h-4" />
        </div>
        <div>
          <div className="flex items-center gap-2">
            <span className="font-semibold text-default">{node.name}</span>
            <span className={`px-2 py-0.5 rounded-full text-[10px] font-medium ${
              node.status === "healthy" ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400" :
              node.status === "warning" ? "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400" :
              "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
            }`}>
              {node.status === "healthy" ? t.healthy : node.status === "warning" ? t.warning : t.critical}
            </span>
          </div>
          <div className="text-xs text-muted mt-0.5">{node.namespace}</div>
        </div>
      </div>

      {/* Golden Metrics */}
      <div className="grid grid-cols-2 lg:grid-cols-3 gap-3">
        {[
          { label: t.rps, value: formatRPS(node.rps), unit: "/s" },
          { label: t.p50Latency, value: formatLatency(node.p50Latency), unit: "" },
          { label: t.p95Latency, value: formatLatency(node.p95Latency), unit: "" },
          { label: t.p99Latency, value: formatLatency(node.p99Latency), unit: "" },
          { label: t.errorRate, value: node.errorRate.toFixed(2), unit: "%", color: node.errorRate > 0.5 ? "text-red-500" : "text-emerald-500" },
          { label: t.totalRequests, value: node.totalRequests.toLocaleString(), unit: "" },
        ].map((m, i) => (
          <div key={i} className="p-3 rounded-lg bg-[var(--hover-bg)]">
            <div className="text-[10px] text-muted mb-1">{m.label}</div>
            <div className={`text-lg font-bold ${m.color || "text-default"}`}>{m.value}<span className="text-xs font-normal text-muted">{m.unit}</span></div>
          </div>
        ))}
      </div>

      {/* Call Relations */}
      <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center gap-3">
          <Network className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">{t.callRelation}</span>
        </div>
        <div className="p-4 space-y-3">
          {inbound.length > 0 && (
            <div>
              <div className="text-[10px] text-muted font-medium uppercase tracking-wider mb-2">{t.inbound} ({inbound.length})</div>
              <div className="flex flex-wrap gap-2">
                {inbound.map((edge, idx) => {
                  const srcNode = topology.nodes.find(n => n.id === edge.source);
                  if (!srcNode) return null;
                  return (
                    <div key={idx} className="flex items-center gap-2 px-2.5 py-1.5 rounded-full bg-slate-100 dark:bg-slate-800 text-xs">
                      <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: getNamespaceColor(srcNode.namespace).fill }} />
                      <span className="font-medium text-default">{srcNode.name}</span>
                      <ArrowRight className="w-3 h-3 text-slate-400" />
                      <span className="text-muted">{formatRPS(edge.rps)}/s · avg {formatLatency(edge.avgLatency)}</span>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
          {outbound.length > 0 && (
            <div>
              <div className="text-[10px] text-muted font-medium uppercase tracking-wider mb-2">{t.outbound} ({outbound.length})</div>
              <div className="flex flex-wrap gap-2">
                {outbound.map((edge, idx) => {
                  const tgtNode = topology.nodes.find(n => n.id === edge.target);
                  if (!tgtNode) return null;
                  return (
                    <div key={idx} className="flex items-center gap-2 px-2.5 py-1.5 rounded-full bg-slate-100 dark:bg-slate-800 text-xs">
                      <ArrowRight className="w-3 h-3 text-cyan-600" />
                      <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: getNamespaceColor(tgtNode.namespace).fill }} />
                      <span className="font-medium text-default">{tgtNode.name}</span>
                      <span className="text-muted">{formatRPS(edge.rps)}/s · avg {formatLatency(edge.avgLatency)}</span>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
          {inbound.length === 0 && outbound.length === 0 && (
            <div className="text-xs text-muted text-center py-4">{t.noCallData}</div>
          )}
        </div>
      </div>

      {/* Detail loading indicator */}
      {detailLoading && (
        <div className="flex items-center justify-center py-4 gap-2 text-sm text-muted">
          <Loader2 className="w-4 h-4 animate-spin" />
          {t.loading}
        </div>
      )}

      {/* Status Code + Latency Distribution — side by side */}
      <div className="flex flex-col lg:flex-row gap-4">
          {/* Status Code Distribution */}
          <div className={`bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden lg:w-[40%]`}>
            <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center gap-2">
              <BarChart3 className="w-4 h-4 text-primary flex-shrink-0" />
              <span className="text-sm font-medium text-default truncate">{t.statusCodeBreakdown}</span>
              <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400 flex-shrink-0">Linkerd</span>
            </div>
            {statusCodes.length > 0 ? (
              <div className="p-4 space-y-2">
                <div className="text-[10px] text-muted mb-2">{totalStatusRequests.toLocaleString()} {t.requests} · {timeRangeLabel(timeRange)}</div>
                {statusCodes.map((s) => {
                  const percent = totalStatusRequests > 0 ? (s.count / totalStatusRequests) * 100 : 0;
                  const maxCount = Math.max(...statusCodes.map(sc => sc.count), 1);
                  const barWidth = (s.count / maxCount) * 100;
                  const colors = getStatusColor(s.code);
                  return (
                    <div key={s.code} className="flex items-center gap-2">
                      <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded font-semibold w-10 text-center ${colors.text} ${colors.bg}`}>{s.code}</span>
                      <div className="flex-1 h-4 bg-[var(--hover-bg)] rounded-sm overflow-hidden">
                        <div className={`h-full rounded-sm ${colors.bar} opacity-80`} style={{ width: `${barWidth}%` }} />
                      </div>
                      <div className="text-right flex items-center gap-1 justify-end flex-shrink-0">
                        <span className="text-xs font-medium text-default">{percent.toFixed(1)}%</span>
                        <span className="text-[10px] text-muted">({s.count.toLocaleString()})</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="p-4 text-center text-xs text-muted py-8">{t.noCallData}</div>
            )}
          </div>

          {/* Latency Distribution Histogram */}
          <div className={`bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden lg:w-[60%]`}>
            <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <BarChart3 className="w-4 h-4 text-primary flex-shrink-0" />
                <span className="text-sm font-medium text-default truncate">{t.latencyDistribution}</span>
                <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400 flex-shrink-0">Linkerd</span>
              </div>
              {latencyBuckets.length > 0 && (
                <div className="flex items-center gap-1.5 flex-shrink-0">
                  <span className="px-1.5 py-0.5 bg-blue-100 dark:bg-blue-900/30 rounded text-[9px] text-blue-700 dark:text-blue-400 font-medium">
                    P50 {formatLatency(node.p50Latency)}
                  </span>
                  <span className="px-1.5 py-0.5 bg-amber-100 dark:bg-amber-900/30 rounded text-[9px] text-amber-700 dark:text-amber-400 font-medium">
                    P95 {formatLatency(node.p95Latency)}
                  </span>
                  <span className="px-1.5 py-0.5 bg-red-100 dark:bg-red-900/30 rounded text-[9px] text-red-700 dark:text-red-400 font-medium">
                    P99 {formatLatency(node.p99Latency)}
                  </span>
                </div>
              )}
            </div>
            {latencyBuckets.length > 0 ? (
              <div className="px-4 pt-3 pb-10">
                <MiniLatencyHistogram
                  buckets={latencyBuckets}
                  allBuckets={allLatencyBuckets}
                  p50={node.p50Latency}
                  p95={node.p95Latency}
                  p99={node.p99Latency}
                  t={t}
                />
              </div>
            ) : (
              <div className="p-4 text-center text-xs text-muted py-8">{t.noCallData}</div>
            )}
          </div>
        </div>
    </div>
  );
}
