"use client";

import { useState, useEffect } from "react";
import { X, Loader2, Sparkles, RefreshCw, AlertTriangle } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { RiskBadge } from "@/components/aiops/RiskBadge";
import { EntityLink } from "@/components/aiops/EntityLink";
import { RootCauseCard } from "./RootCauseCard";
import { TimelineView } from "./TimelineView";
import { getIncidentDetail, summarizeIncident } from "@/api/aiops";
import type { IncidentDetail, SummarizeResponse } from "@/api/aiops";

const STATE_COLORS: Record<string, string> = {
  warning: "bg-yellow-500/15 text-yellow-600 dark:text-yellow-400",
  incident: "bg-red-500/15 text-red-600 dark:text-red-400",
  recovery: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
  stable: "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400",
};

interface IncidentDetailModalProps {
  incidentId: string | null;
  open: boolean;
  onClose: () => void;
}

function formatDuration(seconds: number, minuteLabel: string): string {
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes} ${minuteLabel}`;
  const hours = Math.floor(minutes / 60);
  const mins = minutes % 60;
  return `${hours}h ${mins}m`;
}

const ROLE_COLORS: Record<string, string> = {
  root_cause: "text-red-500",
  affected: "text-yellow-600 dark:text-yellow-400",
  symptom: "text-blue-500",
};

const PRIORITY_COLORS: Record<number, string> = {
  1: "bg-red-500/15 text-red-600 dark:text-red-400",
  2: "bg-yellow-500/15 text-yellow-600 dark:text-yellow-400",
  3: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
};

export function IncidentDetailModal({ incidentId, open, onClose }: IncidentDetailModalProps) {
  const { t } = useI18n();
  const [detail, setDetail] = useState<IncidentDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [aiAnalysis, setAiAnalysis] = useState<SummarizeResponse | null>(null);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiError, setAiError] = useState<string | null>(null);

  useEffect(() => {
    if (!incidentId || !open) {
      setDetail(null);
      setAiAnalysis(null);
      setAiError(null);
      return;
    }

    setLoading(true);
    getIncidentDetail(incidentId)
      .then(setDetail)
      .catch((err) => console.error("Failed to load incident detail:", err))
      .finally(() => setLoading(false));
  }, [incidentId, open]);

  const handleAiAnalyze = async () => {
    if (!incidentId) return;
    setAiLoading(true);
    setAiError(null);
    try {
      const res = await summarizeIncident(incidentId);
      setAiAnalysis(res);
    } catch {
      setAiError(t.aiops.ai.analysisFailed);
    } finally {
      setAiLoading(false);
    }
  };

  if (!open) return null;

  const rootCauseEntity = detail?.entities.find((e) => e.role === "root_cause");

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* 背景遮罩 */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onClose} />

      {/* 弹窗 */}
      <div className="relative bg-card border border-[var(--border-color)] rounded-2xl shadow-2xl w-full max-w-2xl max-h-[85vh] overflow-y-auto mx-4">
        {/* 头部 */}
        <div className="sticky top-0 bg-card border-b border-[var(--border-color)] px-6 py-4 flex items-center justify-between rounded-t-2xl">
          <div className="flex items-center gap-3">
            <h2 className="text-base font-bold text-default">
              {t.aiops.incidentId} {detail?.id ?? incidentId}
            </h2>
          </div>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* 内容 */}
        <div className="px-6 py-4 space-y-5">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
            </div>
          ) : detail ? (
            <>
              {/* 状态行 */}
              <div className="flex flex-wrap items-center gap-3 text-sm">
                <span className={`px-2 py-1 rounded-full text-xs font-medium ${STATE_COLORS[detail.state] ?? ""}`}>
                  {t.aiops.state[detail.state as keyof typeof t.aiops.state] ?? detail.state}
                </span>
                <RiskBadge level={detail.severity} size="md" />
                <span className="text-muted">
                  {t.aiops.duration}: <span className="text-default font-medium">{formatDuration(detail.durationS, t.aiops.minutes)}</span>
                </span>
                <span className="text-muted">
                  {t.aiops.peakRisk}: <span className="text-default font-mono font-medium">{detail.peakRisk.toFixed(1)}</span>
                </span>
              </div>

              {/* 根因卡片 */}
              <RootCauseCard entity={rootCauseEntity} />

              {/* 受影响实体 */}
              {detail.entities.length > 0 && (
                <div>
                  <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">
                    {t.aiops.affectedEntities}
                  </h4>
                  <div className="space-y-1.5">
                    {detail.entities.map((e) => (
                      <div key={e.entityKey} className="flex items-center gap-3 text-sm">
                        <EntityLink entityKey={e.entityKey} />
                        <span className={`text-xs font-medium ${ROLE_COLORS[e.role] ?? "text-muted"}`}>{e.role}</span>
                        <span className="text-xs text-muted">
                          R={e.rFinal.toFixed(1)}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* 时间线 */}
              <TimelineView timeline={detail.timeline} />

              {/* AI 分析 */}
              <div>
                <div className="flex items-center justify-between mb-3">
                  <h4 className="text-xs font-semibold text-muted uppercase tracking-wider">
                    {t.aiops.ai.analyze}
                  </h4>
                  {aiAnalysis && (
                    <button
                      onClick={handleAiAnalyze}
                      disabled={aiLoading}
                      className="flex items-center gap-1 text-xs text-blue-500 hover:text-blue-600 transition-colors disabled:opacity-50"
                    >
                      <RefreshCw className={`w-3 h-3 ${aiLoading ? "animate-spin" : ""}`} />
                      {t.aiops.ai.regenerate}
                    </button>
                  )}
                </div>

                {!aiAnalysis && !aiLoading && !aiError && (
                  <button
                    onClick={handleAiAnalyze}
                    className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-blue-600 dark:text-blue-400 bg-blue-500/10 hover:bg-blue-500/20 rounded-lg transition-colors"
                  >
                    <Sparkles className="w-4 h-4" />
                    {t.aiops.ai.analyze}
                  </button>
                )}

                {aiLoading && (
                  <div className="flex items-center gap-2 py-4 text-sm text-muted">
                    <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
                    {t.aiops.ai.analyzing}
                  </div>
                )}

                {aiError && (
                  <div className="flex items-center gap-2 py-3 px-4 text-sm text-red-600 dark:text-red-400 bg-red-500/10 rounded-lg">
                    <AlertTriangle className="w-4 h-4 shrink-0" />
                    <span>{aiError}</span>
                    <button onClick={handleAiAnalyze} className="ml-auto text-xs underline hover:no-underline">
                      {t.aiops.retry}
                    </button>
                  </div>
                )}

                {aiAnalysis && (
                  <div className="space-y-4 bg-[var(--hover-bg)] rounded-xl p-4">
                    {/* 摘要 */}
                    <div>
                      <h5 className="text-xs font-semibold text-muted mb-1">{t.aiops.ai.summary}</h5>
                      <p className="text-sm text-default">{aiAnalysis.summary}</p>
                    </div>

                    {/* 根因分析 */}
                    {aiAnalysis.rootCauseAnalysis && (
                      <div>
                        <h5 className="text-xs font-semibold text-muted mb-1">{t.aiops.ai.rootCauseAnalysis}</h5>
                        <p className="text-sm text-default">{aiAnalysis.rootCauseAnalysis}</p>
                      </div>
                    )}

                    {/* 处置建议 */}
                    {aiAnalysis.recommendations.length > 0 && (
                      <div>
                        <h5 className="text-xs font-semibold text-muted mb-2">{t.aiops.ai.recommendations}</h5>
                        <div className="space-y-2">
                          {aiAnalysis.recommendations.map((rec, i) => (
                            <div key={i} className="bg-card rounded-lg p-3 border border-[var(--border-color)]">
                              <div className="flex items-center gap-2 mb-1">
                                <span className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${PRIORITY_COLORS[rec.priority] ?? "bg-gray-500/15 text-gray-500"}`}>
                                  P{rec.priority}
                                </span>
                                <span className="text-sm font-medium text-default">{rec.action}</span>
                              </div>
                              <div className="text-xs text-muted space-y-0.5 ml-8">
                                <p>{t.aiops.ai.reason}: {rec.reason}</p>
                                <p>{t.aiops.ai.impact}: {rec.impact}</p>
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* 相似历史事件 */}
                    <div>
                      <h5 className="text-xs font-semibold text-muted mb-2">{t.aiops.ai.similarIncidents}</h5>
                      {aiAnalysis.similarIncidents.length > 0 ? (
                        <div className="space-y-1.5">
                          {aiAnalysis.similarIncidents.map((sim) => (
                            <div key={sim.incidentId} className="flex items-center justify-between text-xs">
                              <span className="text-default font-mono">{sim.incidentId}</span>
                              <span className="text-muted">{sim.rootCause}</span>
                              <span className="text-muted">{sim.occurredAt}</span>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <p className="text-xs text-muted">{t.aiops.ai.noSimilar}</p>
                      )}
                    </div>

                    {/* 生成时间 */}
                    <p className="text-[10px] text-muted text-right">
                      {t.aiops.ai.generatedAt}: {new Date(aiAnalysis.generatedAt * 1000).toLocaleString()}
                    </p>
                  </div>
                )}
              </div>
            </>
          ) : (
            <div className="py-8 text-center text-sm text-muted">{t.aiops.noData}</div>
          )}
        </div>
      </div>
    </div>
  );
}
