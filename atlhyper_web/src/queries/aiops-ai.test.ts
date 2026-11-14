import { QueryClient, QueryObserver } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const api = vi.hoisted(() => ({
  getAIReports: vi.fn(),
  getAIReportDetail: vi.fn(),
  getAnalysisStatus: vi.fn(),
  summarizeIncident: vi.fn(),
  triggerAnalysis: vi.fn(),
}));
vi.mock("@/api/aiops", () => api);

import { ANALYSIS_POLL_INTERVAL_MS, ANALYSIS_POLL_MAX_MS, type AnalysisWatch } from "@/lib/analysis-poll";
import {
  aiReportDetailQuery,
  aiReportsQuery,
  analysisWatchQuery,
  summarizeIncidentMutation,
  triggerAnalysisMutation,
} from "./aiops-ai";
import { aiopsKeys } from "./keys";

const watchKey = aiopsKeys.analysisStatus("inc-1");
const reportsKey = aiopsKeys.aiReports("inc-1");

/** 不保留缓存的查询要有观察者才留得住数据：模拟详情弹窗开着 */
function observeWatch(client: QueryClient) {
  const observer = new QueryObserver(client, { ...analysisWatchQuery("inc-1", true), enabled: false });
  return observer.subscribe(() => {});
}

/** 直接把跟踪状态放进缓存（先声明成 AnalysisWatch 再传：联合类型的字面量直接传给 setQueryData 推断不出来） */
function setWatch(client: QueryClient, watch: AnalysisWatch) {
  client.setQueryData(watchKey, watch);
}

function status(state: "running" | "succeeded" | "failed" | "none", errorCode?: string) {
  return { incidentId: "inc-1", state, errorCode };
}

beforeEach(() => {
  for (const call of Object.values(api)) call.mockReset();
});

describe("aiReportsQuery / aiReportDetailQuery", () => {
  it("没有 ai.analyze 权限时不发请求", () => {
    expect(aiReportsQuery("inc-1", false).enabled).toBe(false);
    expect(aiReportsQuery("inc-1", true).enabled).toBe(true);
    expect(aiReportsQuery("", true).enabled).toBe(false);
  });

  it("报告正文只在卡片展开时取，取过不再重取", () => {
    expect(aiReportDetailQuery(7, false).enabled).toBe(false);
    expect(aiReportDetailQuery(7, true).enabled).toBe(true);
    expect(aiReportDetailQuery(7, true).staleTime).toBe(Infinity);
    expect(aiReportDetailQuery(7, true).queryKey).toEqual(["aiops", "ai-report-detail", { reportId: 7 }]);
  });

  it("把取消信号交给接口", async () => {
    api.getAIReports.mockResolvedValue([]);
    api.getAIReportDetail.mockResolvedValue({ id: 7 });
    const client = new QueryClient();

    await client.fetchQuery(aiReportsQuery("inc-1", true));
    await client.fetchQuery(aiReportDetailQuery(7, true));

    expect(api.getAIReports.mock.calls[0][0]).toBe("inc-1");
    expect(api.getAIReports.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
    expect(api.getAIReportDetail.mock.calls[0][0]).toBe(7);
    expect(api.getAIReportDetail.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
  });
});

describe("analysisWatchQuery", () => {
  it("没有 ai.analyze 权限时不启用", () => {
    expect(analysisWatchQuery("inc-1", false).enabled).toBe(false);
    expect(analysisWatchQuery("inc-1", true).enabled).toBe(true);
  });

  it("轮询间隔是函数：在跑才每 5 秒查一次，完成 / 失败 / 没在跟即停", () => {
    const interval = analysisWatchQuery("inc-1", true).refetchInterval as (query: {
      state: { data: AnalysisWatch | undefined };
    }) => number | false;
    expect(interval({ state: { data: { phase: "running", startedAtMs: 0 } } })).toBe(ANALYSIS_POLL_INTERVAL_MS);
    expect(interval({ state: { data: { phase: "succeeded" } } })).toBe(false);
    expect(interval({ state: { data: { phase: "failed", errorCode: "ai_timeout" } } })).toBe(false);
    expect(interval({ state: { data: { phase: "idle" } } })).toBe(false);
    expect(interval({ state: { data: undefined } })).toBe(false);
  });

  it("打开详情时分析仍在进行 → 接着跟；过去的结果不提示", async () => {
    api.getAnalysisStatus.mockResolvedValueOnce(status("running"));
    const running = await new QueryClient().fetchQuery(analysisWatchQuery("inc-1", true));
    expect(running.phase).toBe("running");
    expect(api.getAnalysisStatus.mock.calls[0][0]).toBe("inc-1");
    expect(api.getAnalysisStatus.mock.calls[0][1]).toBeInstanceOf(AbortSignal);

    api.getAnalysisStatus.mockResolvedValueOnce(status("failed", "ai_upstream"));
    expect(await new QueryClient().fetchQuery(analysisWatchQuery("inc-1", true))).toEqual({ phase: "idle" });
  });

  it("跟踪期间分析成功 → 结束，并让报告列表重新取数", async () => {
    const client = new QueryClient();
    const stopObserving = observeWatch(client);
    client.setQueryData(reportsKey, []);
    setWatch(client, { phase: "running", startedAtMs: Date.now() });
    api.getAnalysisStatus.mockResolvedValue(status("succeeded"));

    expect(await client.fetchQuery(analysisWatchQuery("inc-1", true))).toEqual({ phase: "succeeded" });
    expect(client.getQueryState(reportsKey)?.isInvalidated).toBe(true);
    stopObserving();
  });

  it("跟踪期间分析失败 → 带错误码结束，不动报告列表", async () => {
    const client = new QueryClient();
    const stopObserving = observeWatch(client);
    client.setQueryData(reportsKey, []);
    setWatch(client, { phase: "running", startedAtMs: Date.now() });
    api.getAnalysisStatus.mockResolvedValue(status("failed", "ai_budget_exhausted"));

    expect(await client.fetchQuery(analysisWatchQuery("inc-1", true))).toEqual({
      phase: "failed",
      errorCode: "ai_budget_exhausted",
    });
    expect(client.getQueryState(reportsKey)?.isInvalidated).toBe(false);
    stopObserving();
  });

  it("跟踪期间状态接口暂时不可用 → 继续等；超过时限 → 按超时结束", async () => {
    const client = new QueryClient();
    const stopObserving = observeWatch(client);
    api.getAnalysisStatus.mockRejectedValue(new Error("502"));

    const watching: AnalysisWatch = { phase: "running", startedAtMs: Date.now() };
    setWatch(client, watching);
    expect(await client.fetchQuery(analysisWatchQuery("inc-1", true))).toEqual(watching);

    setWatch(client, { phase: "running", startedAtMs: Date.now() - ANALYSIS_POLL_MAX_MS });
    expect(await client.fetchQuery(analysisWatchQuery("inc-1", true))).toEqual({
      phase: "failed",
      errorCode: "ai_timeout",
    });
    stopObserving();
  });

  it("没在跟的时候状态接口失败 → 照常报错", async () => {
    api.getAnalysisStatus.mockRejectedValue(new Error("502"));
    await expect(
      new QueryClient().fetchQuery({ ...analysisWatchQuery("inc-1", true), retry: false }),
    ).rejects.toThrow("502");
  });
});

describe("triggerAnalysisMutation / summarizeIncidentMutation", () => {
  it("触发成功即把跟踪状态置为 running，轮询随即开始", async () => {
    api.triggerAnalysis.mockResolvedValue({ message: "ok", alreadyRunning: false });
    const client = new QueryClient();
    const stopObserving = observeWatch(client);

    await client.getMutationCache().build(client, triggerAnalysisMutation("inc-1")).execute();

    expect(api.triggerAnalysis).toHaveBeenCalledWith("inc-1");
    expect(client.getQueryData<AnalysisWatch>(watchKey)?.phase).toBe("running");
    stopObserving();
  });

  it("触发失败不进入跟踪", async () => {
    api.triggerAnalysis.mockRejectedValue(new Error("forbidden"));
    const client = new QueryClient();
    const stopObserving = observeWatch(client);

    await expect(
      client.getMutationCache().build(client, triggerAnalysisMutation("inc-1")).execute(),
    ).rejects.toThrow("forbidden");
    expect(client.getQueryData(watchKey)).toBeUndefined();
    stopObserving();
  });

  it("生成总结：开始时作废上一次分析的失败提示，成功后让报告列表重新取数", async () => {
    api.summarizeIncident.mockResolvedValue({ incidentId: "inc-1" });
    const client = new QueryClient();
    const stopObserving = observeWatch(client);
    client.setQueryData(reportsKey, []);
    setWatch(client, { phase: "failed", errorCode: "ai_timeout" });

    await client.getMutationCache().build(client, summarizeIncidentMutation("inc-1")).execute();

    expect(api.summarizeIncident).toHaveBeenCalledWith("inc-1");
    expect(client.getQueryData(watchKey)).toEqual({ phase: "idle" });
    expect(client.getQueryState(reportsKey)?.isInvalidated).toBe(true);
    stopObserving();
  });

  it("生成总结失败不刷新报告列表", async () => {
    api.summarizeIncident.mockRejectedValue(new Error("budget"));
    const client = new QueryClient();
    client.setQueryData(reportsKey, []);

    await expect(
      client.getMutationCache().build(client, summarizeIncidentMutation("inc-1")).execute(),
    ).rejects.toThrow("budget");
    expect(client.getQueryState(reportsKey)?.isInvalidated).toBe(false);
  });
});
