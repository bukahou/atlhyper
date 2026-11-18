"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import { Sparkles, RefreshCw, Loader2, Search, AlertTriangle } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { AIReportCard } from "./AIReportCard";
import { useCan } from "@/hooks/useCan";
import { useSessionReady } from "@/hooks/useSessionReady";
import { aiErrorMessage, aiCodeMessage } from "@/lib/ai-error";
import {
  aiReportsQuery,
  analysisWatchQuery,
  summarizeIncidentMutation,
  triggerAnalysisMutation,
} from "@/queries/aiops-ai";

interface AIAnalysisSectionProps {
  incidentId: string;
  incidentState: string;
}

export function AIAnalysisSection({ incidentId, incidentState }: AIAnalysisSectionProps) {
  const { t } = useI18n();
  const aiT = t.aiops.ai;
  // 报告与分析都需要 ai.analyze；权限加载完之前按没有权限算，不发请求
  const sessionReady = useSessionReady();
  const canAnalyze = useCan()("ai.analyze");

  const reportsQuery = useQuery(aiReportsQuery(incidentId, canAnalyze));
  const reports = reportsQuery.data ?? [];
  // 报告是可有可无的补充：取不到就当没有，不单独报错
  const loading = !sessionReady || (canAnalyze && reportsQuery.isPending);

  // 深度分析的跟踪：在跑的时候每 5 秒查一次，完成、失败或超时即停，离开详情自动停。
  // 打开详情时分析仍在进行（如重新打开弹窗）也会接着跟
  const analysisWatch = useQuery(analysisWatchQuery(incidentId, canAnalyze)).data;
  const summarize = useMutation(summarizeIncidentMutation(incidentId));
  const triggerDeepAnalysis = useMutation(triggerAnalysisMutation(incidentId));

  const generating = summarize.isPending;
  const deepAnalyzing = triggerDeepAnalysis.isPending || analysisWatch?.phase === "running";

  // 最近一次操作的失败原因：生成总结失败、触发分析失败、或分析以失败 / 中断 / 超时结束
  const error = summarize.isError
    ? aiErrorMessage(summarize.error, aiT)
    : triggerDeepAnalysis.isError
      ? aiErrorMessage(triggerDeepAnalysis.error, aiT)
      : analysisWatch?.phase === "failed"
        ? aiCodeMessage(analysisWatch.errorCode, aiT)
        : null;

  const handleGenerate = () => {
    triggerDeepAnalysis.reset();
    summarize.mutate();
  };

  const handleDeepAnalysis = () => {
    summarize.reset();
    triggerDeepAnalysis.mutate(); // 已在进行中也照常跟踪同一个任务
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
