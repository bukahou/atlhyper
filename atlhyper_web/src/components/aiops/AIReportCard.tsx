"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { getAIReportDetail } from "@/api/aiops";
import type { AIReport, AIReportDetail, Recommendation, SimilarMatch } from "@/api/aiops";
import { InvestigationTimeline } from "./InvestigationTimeline";

const ROLE_STYLES: Record<string, string> = {
  background: "bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300",
  analysis: "bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300",
};

const TRIGGER_MAP: Record<string, string> = {
  incident_created: "triggerIncidentCreated",
  state_changed: "triggerStateChanged",
  manual: "triggerManual",
  auto_escalation: "fromAnalysis",
};

const PRIORITY_COLORS: Record<number, string> = {
  1: "bg-red-500/15 text-red-600 dark:text-red-400",
  2: "bg-yellow-500/15 text-yellow-600 dark:text-yellow-400",
  3: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
};

interface AIReportCardProps {
  report: AIReport;
}

function formatTimeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export function AIReportCard({ report }: AIReportCardProps) {
  const { t } = useI18n();
  const aiT = t.aiops.ai;
  const [expanded, setExpanded] = useState(false);
  const [detail, setDetail] = useState<AIReportDetail | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);

  const handleToggle = async () => {
    if (!expanded && !detail) {
      setLoadingDetail(true);
      try {
        const d = await getAIReportDetail(report.id);
        setDetail(d);
      } catch (err) {
        console.error("Failed to load report detail:", err);
      } finally {
        setLoadingDetail(false);
      }
    }
    setExpanded(!expanded);
  };

  const triggerKey = TRIGGER_MAP[report.trigger] || report.trigger;
  const triggerLabel = (aiT as Record<string, string>)[triggerKey] || report.trigger;

  return (
    <div className="bg-card rounded-lg border border-[var(--border-color)] overflow-hidden">
      {/* Header */}
      <button
        onClick={handleToggle}
        className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-[var(--hover-bg)] transition-colors"
      >
        {expanded ? (
          <ChevronDown className="w-4 h-4 text-muted shrink-0" />
        ) : (
          <ChevronRight className="w-4 h-4 text-muted shrink-0" />
        )}

        <span className={`px-2 py-0.5 text-[10px] font-medium rounded-full ${ROLE_STYLES[report.role] || "bg-gray-100 text-gray-600"}`}>
          {report.role === "background" ? aiT.fromBackground : aiT.fromAnalysis}
        </span>

        <span className="text-[10px] text-muted">{triggerLabel}</span>

        <span className="flex-1 text-sm text-default truncate">{report.summary}</span>

        <span className="text-[10px] text-muted whitespace-nowrap">{formatTimeAgo(report.createdAt)}</span>
      </button>

      {/* Expanded Detail */}
      {expanded && (
        <div className="px-4 pb-4 space-y-3 border-t border-[var(--border-color)]">
          {loadingDetail ? (
            <div className="flex items-center gap-2 py-4 text-sm text-muted">
              <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
            </div>
          ) : detail ? (
            <>
              {/* Summary */}
              <div className="pt-3">
                <h5 className="text-xs font-semibold text-muted mb-1">{aiT.summary}</h5>
                <p className="text-sm text-default">{detail.summary}</p>
              </div>

              {/* Root Cause Analysis */}
              {detail.rootCauseAnalysis && (
                <div>
                  <h5 className="text-xs font-semibold text-muted mb-1">{aiT.rootCauseAnalysis}</h5>
                  <p className="text-sm text-default">{detail.rootCauseAnalysis}</p>
                </div>
              )}

              {/* Recommendations */}
              {detail.recommendations && (() => {
                let recs: Recommendation[] = [];
                try { recs = JSON.parse(detail.recommendations); } catch { /* ignore */ }
                return (
                  <div>
                    <h5 className="text-xs font-semibold text-muted mb-2">{aiT.recommendations}</h5>
                    {recs.length === 0 ? (
                      <p className="text-xs text-muted text-center py-2">{t.common.noData}</p>
                    ) : (
                    <div className="space-y-2">
                      {recs.map((rec, i) => (
                        <div key={i} className="bg-[var(--hover-bg)] rounded-lg p-3">
                          <div className="flex items-center gap-2 mb-1">
                            <span className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${PRIORITY_COLORS[rec.priority] ?? "bg-gray-500/15 text-gray-500"}`}>
                              P{rec.priority}
                            </span>
                            <span className="text-sm font-medium text-default">{rec.action}</span>
                          </div>
                          <div className="text-xs text-muted space-y-0.5 ml-8">
                            <p>{aiT.reason}: {rec.reason}</p>
                            <p>{aiT.impact}: {rec.impact}</p>
                          </div>
                        </div>
                      ))}
                    </div>
                    )}
                  </div>
                );
              })()}

              {/* Similar Incidents */}
              {detail.similarIncidents && (() => {
                let sims: SimilarMatch[] = [];
                try { sims = JSON.parse(detail.similarIncidents); } catch { /* ignore */ }
                return (
                  <div>
                    <h5 className="text-xs font-semibold text-muted mb-2">{aiT.similarIncidents}</h5>
                    {sims.length === 0 ? (
                      <p className="text-xs text-muted text-center py-2">{t.common.noData}</p>
                    ) : (
                    <div className="space-y-1.5">
                      {sims.map((sim) => (
                        <div key={sim.incidentId} className="flex items-center justify-between text-xs">
                          <span className="text-default font-mono">{sim.incidentId}</span>
                          <span className="text-muted">{sim.rootCause}</span>
                          <span className="text-muted">{sim.occurredAt}</span>
                        </div>
                      ))}
                    </div>
                    )}
                  </div>
                );
              })()}

              {/* Investigation Steps (analysis role only) */}
              {detail.investigationSteps && (
                <InvestigationTimeline stepsJson={detail.investigationSteps} />
              )}

              {/* Evidence Chain */}
              {detail.evidenceChain && (() => {
                let chain: string[] = [];
                try { chain = JSON.parse(detail.evidenceChain); } catch { /* ignore */ }
                return (
                  <div>
                    <h5 className="text-xs font-semibold text-muted mb-1">{aiT.evidenceChain}</h5>
                    {chain.length === 0 ? (
                      <p className="text-xs text-muted text-center py-2">{t.common.noData}</p>
                    ) : (
                    <ol className="list-decimal list-inside text-xs text-default space-y-0.5">
                      {chain.map((item, i) => (
                        <li key={i}>{item}</li>
                      ))}
                    </ol>
                    )}
                  </div>
                );
              })()}

              {/* Meta */}
              <p className="text-[10px] text-muted text-right">
                {report.providerName} · {report.model} · {(report.inputTokens + report.outputTokens).toLocaleString()} {aiT.tokens} · {(report.durationMs / 1000).toFixed(1)}s
              </p>
            </>
          ) : (
            <p className="text-sm text-muted py-2">{report.summary}</p>
          )}
        </div>
      )}
    </div>
  );
}
