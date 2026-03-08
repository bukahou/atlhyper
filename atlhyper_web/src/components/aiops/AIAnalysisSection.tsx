"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { Sparkles, RefreshCw, Loader2, Search, AlertTriangle } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { getAIReports, summarizeIncident, triggerAnalysis } from "@/api/aiops";
import type { AIReport } from "@/api/aiops";
import { AIReportCard } from "./AIReportCard";

interface AIAnalysisSectionProps {
  incidentId: string;
  incidentState: string;
}

export function AIAnalysisSection({ incidentId, incidentState }: AIAnalysisSectionProps) {
  const { t } = useI18n();
  const aiT = t.aiops.ai;
  const { isAuthenticated } = useAuthStore();
  const [reports, setReports] = useState<AIReport[]>([]);
  const [loading, setLoading] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [deepAnalyzing, setDeepAnalyzing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const loadReports = useCallback(async () => {
    if (!isAuthenticated) return;
    try {
      const data = await getAIReports(incidentId);
      setReports(data);
      return data;
    } catch {
      // silently fail — reports are optional
      return [];
    }
  }, [incidentId, isAuthenticated]);

  useEffect(() => {
    setLoading(true);
    loadReports().finally(() => setLoading(false));
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [loadReports]);

  const handleGenerate = async () => {
    setGenerating(true);
    setError(null);
    try {
      await summarizeIncident(incidentId);
      await loadReports();
    } catch {
      setError(aiT.analysisFailed);
    } finally {
      setGenerating(false);
    }
  };

  const handleDeepAnalysis = async () => {
    setDeepAnalyzing(true);
    setError(null);
    try {
      await triggerAnalysis(incidentId);
      // Poll for new report
      let attempts = 0;
      const currentCount = reports.length;
      pollRef.current = setInterval(async () => {
        attempts++;
        const updated = await loadReports();
        if ((updated && updated.length > currentCount) || attempts >= 12) {
          if (pollRef.current) clearInterval(pollRef.current);
          pollRef.current = null;
          setDeepAnalyzing(false);
        }
      }, 5000);
    } catch {
      setError(aiT.analysisFailed);
      setDeepAnalyzing(false);
    }
  };

  const canDeepAnalyze = incidentState === "warning" || incidentState === "incident";

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h4 className="text-xs font-semibold text-muted uppercase tracking-wider">
          {aiT.aiReports}
        </h4>
        <div className="flex items-center gap-2">
          {canDeepAnalyze && !deepAnalyzing && (
            <button
              onClick={handleDeepAnalysis}
              disabled={generating}
              className="flex items-center gap-1 text-xs text-purple-500 hover:text-purple-600 transition-colors disabled:opacity-50"
            >
              <Search className="w-3 h-3" />
              {aiT.triggerDeepAnalysis}
            </button>
          )}
          {reports.length > 0 && (
            <button
              onClick={handleGenerate}
              disabled={generating || deepAnalyzing}
              className="flex items-center gap-1 text-xs text-blue-500 hover:text-blue-600 transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-3 h-3 ${generating ? "animate-spin" : ""}`} />
              {aiT.reanalyze}
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 py-3 px-4 text-sm text-red-600 dark:text-red-400 bg-red-500/10 rounded-lg mb-3">
          <AlertTriangle className="w-4 h-4 shrink-0" />
          <span>{error}</span>
          <button onClick={handleGenerate} className="ml-auto text-xs underline hover:no-underline">
            {t.aiops.retry}
          </button>
        </div>
      )}

      {loading ? (
        <div className="flex items-center gap-2 py-4 text-sm text-muted">
          <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
        </div>
      ) : reports.length > 0 ? (
        <div className="space-y-2">
          {reports.map((report) => (
            <AIReportCard key={report.id} report={report} />
          ))}
        </div>
      ) : (
        <button
          onClick={handleGenerate}
          disabled={generating}
          className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-blue-600 dark:text-blue-400 bg-blue-500/10 hover:bg-blue-500/20 rounded-lg transition-colors disabled:opacity-50"
        >
          {generating ? (
            <Loader2 className="w-4 h-4 animate-spin" />
          ) : (
            <Sparkles className="w-4 h-4" />
          )}
          {generating ? aiT.analyzing : aiT.generateAnalysis}
        </button>
      )}

      {deepAnalyzing && (
        <div className="flex items-center gap-2 mt-3 py-2 text-sm text-purple-600 dark:text-purple-400">
          <Loader2 className="w-4 h-4 animate-spin" />
          {aiT.analysisInProgress}
        </div>
      )}
    </div>
  );
}
