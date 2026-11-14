import { describe, expect, it } from "vitest";
import { STATUS_CODE } from "@/lib/otel";
import type { Span, TraceDetail, TraceSummary } from "@/types/model/apm";
import { displayedTraceView, resolveCurrentSample, summarizeTraceDetail, traceDetailViewFromLink } from "./apm-view";

function sample(traceId: string): TraceSummary {
  return {
    traceId,
    rootService: "gateway",
    rootOperation: "GET /api/users",
    services: ["gateway"],
    durationMs: 12,
    spanCount: 1,
    serviceCount: 1,
    hasError: false,
    timestamp: "2026-01-01T00:00:00Z",
  };
}

function span(overrides: Partial<Span>): Span {
  return {
    spanId: "s1",
    serviceName: "gateway",
    spanName: "GET /api/users",
    statusCode: STATUS_CODE.ok,
    timestamp: "2026-01-01T00:00:00Z",
    ...overrides,
  } as Span;
}

describe("resolveCurrentSample", () => {
  const samples = [sample("t1"), sample("t2"), sample("t3")];

  it("刚从某个操作点进来、还没选过链路时，显示第一条样本", () => {
    expect(resolveCurrentSample(samples, "")).toEqual({ traceId: "t1", index: 0 });
  });

  it("选过链路后按 traceId 定位，不按位置：样本序列变了也不会显示成另一条", () => {
    expect(resolveCurrentSample(samples, "t3")).toEqual({ traceId: "t3", index: 2 });
    expect(resolveCurrentSample([sample("t0"), ...samples], "t3")).toEqual({ traceId: "t3", index: 3 });
  });

  it("选中的链路不在样本序列里时仍然显示它，只是没有位置", () => {
    expect(resolveCurrentSample(samples, "t9")).toEqual({ traceId: "t9", index: -1 });
  });

  it("样本还没取到（或一条都没有）时没有可显示的链路", () => {
    expect(resolveCurrentSample([], "")).toEqual({ traceId: "", index: -1 });
  });
});

describe("summarizeTraceDetail", () => {
  it("用根 span 合成这条链路的摘要", () => {
    const detail: TraceDetail = {
      traceId: "t1",
      durationMs: 373,
      serviceCount: 2,
      spanCount: 3,
      spans: [
        span({ spanId: "root" }),
        span({ spanId: "child-1", serviceName: "user" }),
        span({ spanId: "child-2", serviceName: "user" }),
      ],
    };

    expect(summarizeTraceDetail(detail)).toEqual({
      traceId: "t1",
      rootService: "gateway",
      rootOperation: "GET /api/users",
      services: ["gateway", "user"],
      durationMs: 373,
      spanCount: 3,
      serviceCount: 2,
      hasError: false,
      timestamp: "2026-01-01T00:00:00Z",
    });
  });

  it("根 span 出错时摘要标记为出错", () => {
    const detail: TraceDetail = {
      traceId: "t1",
      durationMs: 1,
      serviceCount: 1,
      spanCount: 1,
      spans: [span({ statusCode: STATUS_CODE.error })],
    };
    expect(summarizeTraceDetail(detail)?.hasError).toBe(true);
  });

  it("没有 span 的链路合成不出摘要", () => {
    expect(summarizeTraceDetail({ traceId: "t1", durationMs: 0, serviceCount: 0, spanCount: 0, spans: [] })).toBeNull();
  });
});

describe("displayedTraceView", () => {
  const rootSummary = sample("t1");

  it("从 URL 深链进来时不知道服务与操作，取到详情后用根 span 的补上", () => {
    expect(displayedTraceView(traceDetailViewFromLink("t1"), "t1", rootSummary)).toEqual({
      level: "trace-detail",
      serviceName: "gateway",
      operationName: "GET /api/users",
      traceId: "t1",
    });
  });

  it("详情还没取到时先显示已知的部分", () => {
    expect(displayedTraceView(traceDetailViewFromLink("t1"), "t1", null)).toEqual({
      level: "trace-detail",
      serviceName: "",
      operationName: "",
      traceId: "t1",
    });
  });

  it("从某个服务的某个操作点进来时，保留进入路径上的服务与操作，traceId 用当前显示的那条", () => {
    const fromOperation = { level: "trace-detail" as const, serviceName: "user", operationName: "SELECT users", traceId: "" };
    expect(displayedTraceView(fromOperation, "t7", rootSummary)).toEqual({
      level: "trace-detail",
      serviceName: "user",
      operationName: "SELECT users",
      traceId: "t7",
    });
  });
});
