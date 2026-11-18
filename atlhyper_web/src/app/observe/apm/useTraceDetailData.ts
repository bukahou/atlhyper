/**
 * 链路详情层要的数据：样本序列 + 当前这条链路的详情
 *
 * 页面在任何一层都会调用它（hooks 不能有条件地调用），但不在链路详情层时
 * 两个查询都不启用 —— 没有操作名、没有 traceId，工厂的启用条件自然不成立。
 */
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import type { TimeRangeParams } from "@/lib/time-range";
import { apmOperationSamplesQuery, apmTraceDetailQuery } from "@/queries/observe-apm";
import type { TraceSummary } from "@/types/model/apm";
import { resolveCurrentSample, summarizeTraceDetail, type ApmView } from "./apm-view";

const NO_SAMPLES: TraceSummary[] = [];

export function useTraceDetailData(
  clusterId: string,
  view: ApmView,
  timeRange: TimeRangeParams,
  canReadTraces: boolean,
) {
  const traceView = view.level === "trace-detail" ? view : null;
  // 从某个操作点进来：按操作取样本序列，在序列里前后翻。
  // 直接点某条链路（或从 URL 深链）进来：没有序列，这条链路自己就是唯一的样本
  const fromOperation = traceView !== null && traceView.operationName !== "";

  const samplesQuery = useQuery(
    apmOperationSamplesQuery(
      clusterId,
      { service: traceView?.serviceName ?? "", operation: traceView?.operationName ?? "" },
      timeRange,
      canReadTraces,
    ),
  );
  const operationSamples = fromOperation ? samplesQuery.data : undefined;
  const current = resolveCurrentSample(operationSamples ?? NO_SAMPLES, traceView?.traceId ?? "");

  // traceId 在键里：连续点不同的链路时，显示的一定是最后点的那条
  const detailQuery = useQuery(apmTraceDetailQuery(clusterId, current.traceId, canReadTraces));
  const detail = detailQuery.data;

  const samples = useMemo(() => {
    if (fromOperation) return operationSamples ?? NO_SAMPLES;
    const ownSummary = detail ? summarizeTraceDetail(detail) : null;
    return ownSummary ? [ownSummary] : NO_SAMPLES;
  }, [fromOperation, operationSamples, detail]);

  const currentIndex = fromOperation ? current.index : samples.length > 0 ? 0 : -1;

  return {
    fromOperation,
    samplesQuery,
    detailQuery,
    /** 样本序列（延迟分布与前后翻用） */
    samples,
    /** 当前显示的链路；样本还没取到时为空串 */
    currentTraceId: current.traceId,
    /** 当前链路在样本序列里的位置；不在序列里时为 -1 */
    currentIndex,
    /** 当前链路的摘要；还不知道时为 null */
    currentSummary: currentIndex >= 0 ? samples[currentIndex] : null,
  };
}

export type TraceDetailData = ReturnType<typeof useTraceDetailData>;
