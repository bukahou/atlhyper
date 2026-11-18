/**
 * AIOps 的取数描述 —— 事件的 AI 报告与深度分析
 *
 * 这些接口需要 ai.analyze 权限：有没有权限由调用方用 useCan 得到后传进来，工厂本身不读权限状态。
 */
import { mutationOptions, queryOptions } from "@tanstack/react-query";
import {
  getAIReportDetail,
  getAIReports,
  getAnalysisStatus,
  summarizeIncident,
  triggerAnalysis,
} from "@/api/aiops";
import {
  advanceAnalysisWatch,
  analysisWatchAfterStatusError,
  analysisWatchPollInterval,
  IDLE_ANALYSIS_WATCH,
  type AnalysisWatch,
} from "@/lib/analysis-poll";
import { aiopsKeys } from "./keys";

/** 某个事件的 AI 报告列表 */
export function aiReportsQuery(incidentId: string, canAnalyze: boolean) {
  return queryOptions({
    queryKey: aiopsKeys.aiReports(incidentId),
    queryFn: ({ signal }) => getAIReports(incidentId, signal),
    enabled: canAnalyze && incidentId !== "",
  });
}

/** 一份 AI 报告的正文。报告生成后不再变化，取过一次就不重取；expanded 为假（卡片收着）时不取 */
export function aiReportDetailQuery(reportId: number, expanded: boolean) {
  return queryOptions({
    queryKey: aiopsKeys.aiReportDetail(reportId),
    queryFn: ({ signal }) => getAIReportDetail(reportId, signal),
    staleTime: Infinity,
    enabled: expanded,
  });
}

/**
 * 跟踪某个事件的深度分析：在跑的时候每 5 秒查一次，完成、失败或超时即停。
 *
 * 缓存里放的不是后端的原始状态，而是前端的跟踪状态（AnalysisWatch）——
 * 只有看到过 running，之后的结束才提示结果（见 lib/analysis-poll.ts）。
 * - refetchInterval 用函数形式：跟踪状态是 running 才继续轮询；组件卸载后没有观察者，轮询自动停止
 * - 分析成功的那一次查询顺带让报告列表重新取数
 * - 缓存不保留（gcTime 0）：关掉详情再打开时从头判断，不重放上一次的失败提示
 */
export function analysisWatchQuery(incidentId: string, canAnalyze: boolean) {
  return queryOptions({
    queryKey: aiopsKeys.analysisStatus(incidentId),
    queryFn: async ({ signal, client, queryKey }): Promise<AnalysisWatch> => {
      // 上一次的跟踪状态就是这个查询自己缓存里的数据
      const previous = client.getQueryData<AnalysisWatch>(queryKey);
      let next: AnalysisWatch;
      try {
        next = advanceAnalysisWatch(previous, await getAnalysisStatus(incidentId, signal), Date.now());
      } catch (statusError) {
        // 被取消的请求原样抛出；其余失败在跟踪期间不算数，继续等到前端时限
        const kept = signal.aborted ? null : analysisWatchAfterStatusError(previous, Date.now());
        if (kept === null) throw statusError;
        return kept;
      }
      if (previous?.phase === "running" && next.phase === "succeeded") {
        void client.invalidateQueries({ queryKey: aiopsKeys.aiReports(incidentId) });
      }
      return next;
    },
    refetchInterval: (query) => analysisWatchPollInterval(query.state.data),
    gcTime: 0,
    enabled: canAnalyze && incidentId !== "",
  });
}

/**
 * 生成（或重新生成）事件的 AI 总结。
 *
 * 开始时把上一次深度分析的失败提示作废；成功后等报告列表重新取完才算结束，
 * 这样按钮转圈停下的那一刻，新报告已经在列表里。
 */
export function summarizeIncidentMutation(incidentId: string) {
  return mutationOptions({
    mutationFn: () => summarizeIncident(incidentId),
    onMutate: (_variables, context) => {
      context.client.setQueryData<AnalysisWatch>(aiopsKeys.analysisStatus(incidentId), (watch) =>
        watch?.phase === "failed" ? IDLE_ANALYSIS_WATCH : watch,
      );
    },
    onSuccess: (_summary, _variables, _onMutateResult, context) =>
      context.client.invalidateQueries({ queryKey: aiopsKeys.aiReports(incidentId) }),
  });
}

/**
 * 触发深度分析。
 *
 * 触发成功（包括「已在进行中」）就把跟踪状态置为 running，轮询随即开始，不必等下一次查询来发现它在跑。
 */
export function triggerAnalysisMutation(incidentId: string) {
  return mutationOptions({
    mutationFn: () => triggerAnalysis(incidentId),
    onSuccess: (_accepted, _variables, _onMutateResult, context) => {
      const running: AnalysisWatch = { phase: "running", startedAtMs: Date.now() };
      context.client.setQueryData<AnalysisWatch>(aiopsKeys.analysisStatus(incidentId), running);
    },
  });
}
