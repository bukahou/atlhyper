"use client";

import { useQuery } from "@tanstack/react-query";
import { QueryView } from "@/components/common";
import { PermissionLock } from "@/components/common/PermissionLock";
import { useScopedState } from "@/hooks/useScopedState";
import { formatDurationMs } from "@/lib/format";
import type { TimeRangeParams } from "@/lib/time-range";
import { apmHttpStatsQuery, apmTracesQuery } from "@/queries/observe-apm";
import type { ApmTranslations } from "@/types/i18n";
import { StatusCodeChart } from "./StatusCodeChart";

/** 按状态码钻取时最多列多少条链路 */
const FILTERED_TRACE_LIMIT = 100;

interface StatusCodeFilter {
  code: number;
  method: string;
}

interface StatusCodeSectionProps {
  t: ApmTranslations;
  clusterId: string;
  serviceName: string;
  timeRange: TimeRangeParams;
  /** 是否有 traces.read；按状态码筛出的链路是内容级数据 */
  canReadTraces: boolean;
  onSelectTrace?: (traceId: string) => void;
}

/** HTTP 状态码分布；点某一行就在下方列出该状态码的链路 */
export function StatusCodeSection({ t, clusterId, serviceName, timeRange, canReadTraces, onSelectTrace }: StatusCodeSectionProps) {
  const httpStatsQuery = useQuery(apmHttpStatsQuery(clusterId, serviceName, timeRange));
  // 选中的状态码只对「这个服务 + 这段时间」有意义，换了任何一个就清掉
  const [statusFilter, setStatusFilter] = useScopedState<StatusCodeFilter | null>(
    `${clusterId}|${serviceName}|${JSON.stringify(timeRange)}`,
    null,
  );

  // 再点一次同一行 = 取消选中
  const handleStatusCodeClick = (code: number, method: string) => {
    const sameRow = statusFilter?.code === code && statusFilter?.method === method;
    setStatusFilter(sameRow ? null : { code, method });
  };

  return (
    <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
      <QueryView query={httpStatsQuery}>
        {(stats) => <StatusCodeChart t={t} stats={stats} onClickRow={handleStatusCodeClick} />}
      </QueryView>
      {/* 按状态码筛出的链路属内容级数据（traces.read） */}
      {statusFilter && (
        <PermissionLock permission="traces.read">
          <StatusFilteredTraces
            t={t}
            clusterId={clusterId}
            serviceName={serviceName}
            timeRange={timeRange}
            canReadTraces={canReadTraces}
            statusFilter={statusFilter}
            onSelectTrace={onSelectTrace}
          />
        </PermissionLock>
      )}
    </div>
  );
}

/** 某个状态码的链路列表。只在选中了某一行时挂载，所以查询也只在这时存在 */
function StatusFilteredTraces({
  t,
  clusterId,
  serviceName,
  timeRange,
  canReadTraces,
  statusFilter,
  onSelectTrace,
}: StatusCodeSectionProps & { statusFilter: StatusCodeFilter }) {
  const tracesQuery = useQuery(
    apmTracesQuery(
      clusterId,
      {
        service: serviceName,
        limit: FILTERED_TRACE_LIMIT,
        statusCode: String(statusFilter.code),
        method: statusFilter.method,
      },
      timeRange,
      canReadTraces,
    ),
  );

  return (
    <div className="mt-4 pt-4 border-t border-[var(--border-color)]">
      <QueryView
        query={tracesQuery}
        isEmpty={(page) => page.traces.length === 0}
        empty={<p className="text-xs text-muted text-center py-2">{t.noData}</p>}
      >
        {({ traces }) => (
          <>
            <h4 className="text-xs font-medium text-muted mb-2">
              {statusFilter.code} {statusFilter.method} — {traces.length} traces
            </h4>
            <div className="space-y-1 max-h-64 overflow-y-auto">
              {traces.map((tr) => (
                <div
                  key={tr.traceId}
                  onClick={() => onSelectTrace?.(tr.traceId)}
                  className="flex items-center justify-between px-2 py-1.5 rounded text-xs hover:bg-[var(--hover-bg)] cursor-pointer transition-colors"
                >
                  <span className="font-mono text-muted">{tr.traceId.slice(0, 16)}...</span>
                  <span className="text-default">{tr.rootOperation}</span>
                  <span className="text-muted">{formatDurationMs(tr.durationMs)}</span>
                  <span className="text-muted">{new Date(tr.timestamp).toLocaleTimeString()}</span>
                </div>
              ))}
            </div>
          </>
        )}
      </QueryView>
    </div>
  );
}
