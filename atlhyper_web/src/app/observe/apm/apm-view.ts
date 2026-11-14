/**
 * 链路页三层下钻的视图状态与相关的纯计算
 *
 * 服务列表 → 服务详情 → 链路详情。页面只存「现在在哪一层、看的是谁」，
 * 每一层要显示的数据由该层自己的查询去取。
 */
import { isErrorSpan } from "@/lib/otel";
import type { TraceDetail, TraceSummary } from "@/types/model/apm";

export interface TraceDetailView {
  level: "trace-detail";
  /** 进入路径上的服务；从 URL 深链进来时为空串，取到详情后用根 span 的 */
  serviceName: string;
  /** 从某个操作点进来时是该操作，此时按它取样本序列；直接点某条链路进来时为空串 */
  operationName: string;
  /** 选中的链路；刚从某个操作点进来、还没选过时为空串，显示样本序列的第一条 */
  traceId: string;
}

export type ApmView =
  | { level: "services" }
  | { level: "service-detail"; serviceName: string }
  | TraceDetailView;

/** 从 URL ?trace= 深链进来：只知道 traceId */
export function traceDetailViewFromLink(traceId: string): TraceDetailView {
  return { level: "trace-detail", serviceName: "", operationName: "", traceId };
}

/** 延迟分布图上 brush 选中的延迟区间：样本导航只在区间内的链路上移动 */
export interface LatencyRange {
  startMs: number;
  endMs: number;
}

export interface CurrentSample {
  /** 当前显示的链路；没有可显示的时为空串 */
  traceId: string;
  /** 它在样本序列里的位置；不在序列里时为 -1 */
  index: number;
}

/**
 * 样本序列里当前显示哪一条。
 *
 * 按 traceId 定位而不是按位置：位置只是用来显示「第几条」的，
 * 真正决定取哪条链路的是 traceId，序列变了也不会悄悄换成另一条。
 */
export function resolveCurrentSample(samples: TraceSummary[], selectedTraceId: string): CurrentSample {
  if (selectedTraceId === "") {
    return samples.length > 0 ? { traceId: samples[0].traceId, index: 0 } : { traceId: "", index: -1 };
  }
  return { traceId: selectedTraceId, index: samples.findIndex((sample) => sample.traceId === selectedTraceId) };
}

/**
 * 用链路详情合成它自己的摘要。
 *
 * 直接点某条链路（或从 URL 深链）进来时没有样本序列，这条链路自己就是唯一的样本，
 * 延迟分布与样本导航按它显示。没有 span 的链路合成不出来，返回 null。
 */
export function summarizeTraceDetail(detail: TraceDetail): TraceSummary | null {
  const rootSpan = detail.spans[0];
  if (rootSpan === undefined) return null;
  return {
    traceId: detail.traceId,
    rootService: rootSpan.serviceName,
    rootOperation: rootSpan.spanName,
    services: [...new Set(detail.spans.map((span) => span.serviceName))],
    durationMs: detail.durationMs,
    spanCount: detail.spanCount,
    serviceCount: detail.serviceCount,
    hasError: isErrorSpan(rootSpan),
    timestamp: rootSpan.timestamp,
  };
}

/**
 * 面包屑与初始聚焦用的链路视图：traceId 用当前显示的那条；
 * 进入路径上缺的服务 / 操作（深链、直接点链路）在取到详情后用根 span 的补上。
 */
export function displayedTraceView(
  view: TraceDetailView,
  currentTraceId: string,
  currentTraceSummary: TraceSummary | null,
): TraceDetailView {
  return {
    level: "trace-detail",
    serviceName: view.serviceName || currentTraceSummary?.rootService || "",
    operationName: view.operationName || currentTraceSummary?.rootOperation || "",
    traceId: currentTraceId,
  };
}
