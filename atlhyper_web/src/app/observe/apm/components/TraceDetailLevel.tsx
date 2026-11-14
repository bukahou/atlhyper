"use client";

import { useState } from "react";
import { QueryView } from "@/components/common";
import { PermissionLock } from "@/components/common/PermissionLock";
import type { ApmTranslations } from "@/types/i18n";
import type { TraceDetail } from "@/types/model/apm";
import type { LatencyRange } from "../apm-view";
import type { TraceDetailData } from "../useTraceDetailData";
import { TraceWaterfall } from "./TraceWaterfall";

interface TraceDetailLevelProps {
  ta: ApmTranslations;
  /** 样本序列与当前链路的详情（见 useTraceDetailData） */
  traceData: TraceDetailData;
  /** 进入路径的服务：瀑布图初始聚焦它 */
  initialFocusService: string;
  onSelectTrace: (traceId: string) => void;
}

/** 后端没有返回内容、或返回的链路里一个 span 都没有 */
function isMissingTrace(detail: TraceDetail | null): boolean {
  return detail === null || detail.spans.length === 0;
}

/**
 * 第三层：一条链路的瀑布图，可在样本序列里前后翻。
 *
 * 链路是内容级数据（traces.read）：没有权限时整块显示锁定占位，查询也不会发出去。
 */
export function TraceDetailLevel({ ta, traceData, initialFocusService, onSelectTrace }: TraceDetailLevelProps) {
  const { fromOperation, samplesQuery, detailQuery, samples, currentIndex } = traceData;
  // 延迟选区存在这一层：瀑布图按 traceId 重新挂载，选区要跨样本保留
  const [latencyFilter, setLatencyFilter] = useState<LatencyRange | null>(null);

  const noTraces = <div className="flex items-center justify-center h-48 text-sm text-muted">{ta.noTraces}</div>;

  const traceWaterfall = (
    <QueryView query={detailQuery} requireCluster isEmpty={isMissingTrace} empty={noTraces}>
      {(detail) =>
        detail && (
          <TraceWaterfall
            key={detail.traceId}
            t={ta}
            trace={detail}
            allTraces={samples}
            currentTraceIndex={currentIndex}
            initialFocusService={initialFocusService}
            onNavigateTrace={(sampleIndex) => onSelectTrace(samples[sampleIndex].traceId)}
            latencyFilter={latencyFilter}
            onLatencyFilterChange={setLatencyFilter}
          />
        )
      }
    </QueryView>
  );

  return (
    <PermissionLock permission="traces.read" className="bg-card rounded-xl border border-[var(--border-color)]">
      {fromOperation ? (
        // 先有样本序列，才知道显示哪一条
        <QueryView query={samplesQuery} requireCluster isEmpty={(traces) => traces.length === 0} empty={noTraces}>
          {() => traceWaterfall}
        </QueryView>
      ) : (
        traceWaterfall
      )}
    </PermissionLock>
  );
}
