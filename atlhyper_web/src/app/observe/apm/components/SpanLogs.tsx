"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight, ChevronDown } from "lucide-react";
import { QueryView } from "@/components/common";
import { PermissionLock } from "@/components/common/PermissionLock";
import { useCan } from "@/hooks/useCan";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useScopedState } from "@/hooks/useScopedState";
import { useSessionReady } from "@/hooks/useSessionReady";
import { traceLogsQuery } from "@/queries/observe-logs";
import type { ApmTranslations } from "@/types/i18n";
import type { LogEntry } from "@/types/model/log";

// ============================================================
// SpanLogs — 整条 trace 的关联日志
//
// 默认全量（跨服务）：错误详情往往在上游服务的日志里，按当前 span 的
// 服务过滤会把根因挡在外面（2026-08-28 register 500 就是这么被挡的）。
// 服务维度只做前端 chip 过滤（已加载数据的即时过滤，不重新请求）。
// ERROR 行可展开结构化字段，exception.* 置顶标红。
// ============================================================

interface SpanLogsProps {
  t: ApmTranslations;
  traceId: string;
  /** 复用瀑布图的服务配色：每行日志前的服务色点与瀑布图一致 */
  serviceColorMap?: Map<string, string>;
  compact?: boolean;
}

/** exception.* 排最前，其余按字母序 —— 排查时最想看的字段不该藏在中间 */
function sortedAttrEntries(attrs: Record<string, string>): [string, string][] {
  return Object.entries(attrs).sort(([a], [b]) => {
    const ax = a.startsWith("exception.") ? 0 : 1;
    const bx = b.startsWith("exception.") ? 0 : 1;
    return ax !== bx ? ax - bx : a.localeCompare(b);
  });
}

/** ERROR 行默认展开 —— 排查者要的就是它（没有结构化字段的行没有可展开的内容） */
function isOpenByDefault(log: LogEntry): boolean {
  return log.severity === "ERROR" && Object.keys(log.attributes ?? {}).length > 0;
}

const NO_ROWS: ReadonlySet<number> = new Set();
const NO_SERVICES: ReadonlySet<string> = new Set();

export function SpanLogs(props: SpanLogsProps) {
  // 日志正文是内容级数据（logs.read）：没有权限时显示锁定占位，而不是「没有关联日志」
  return (
    <PermissionLock permission="logs.read">
      <TraceLogList {...props} />
    </PermissionLock>
  );
}

function TraceLogList({ t, traceId, serviceColorMap, compact }: SpanLogsProps) {
  const { clusterId } = useCurrentCluster();
  const sessionReady = useSessionReady();
  const can = useCan();
  const logsQuery = useQuery(
    traceLogsQuery(clusterId, traceId, compact ? 30 : 100, sessionReady && can("logs.read")),
  );
  // 隐藏的服务与手动开合过的行都是针对这一条链路的，换了链路就作废
  const [hiddenServices, setHiddenServices] = useScopedState(traceId, NO_SERVICES);
  /** 用户手动点过的行：与默认状态相反（默认展开的被收起，默认收起的被展开） */
  const [toggledRows, setToggledRows] = useScopedState(traceId, NO_ROWS);

  return (
    <QueryView
      query={logsQuery}
      isEmpty={(logs) => logs.length === 0}
      empty={
        <div className="flex items-center justify-center py-12 text-sm text-muted">
          {t.noCorrelatedLogs}
        </div>
      }
    >
      {(logs) => (
        <TraceLogRows
          logs={logs}
          serviceColorMap={serviceColorMap}
          compact={compact}
          hiddenServices={hiddenServices}
          toggledRows={toggledRows}
          onToggleService={(service) => setHiddenServices(toggleMember(hiddenServices, service))}
          onToggleRow={(row) => setToggledRows(toggleMember(toggledRows, row))}
        />
      )}
    </QueryView>
  );
}

/** 集合里有就去掉、没有就加上，返回新集合 */
function toggleMember<T>(members: ReadonlySet<T>, member: T): ReadonlySet<T> {
  const next = new Set(members);
  if (next.has(member)) next.delete(member);
  else next.add(member);
  return next;
}

interface TraceLogRowsProps {
  logs: LogEntry[];
  serviceColorMap?: Map<string, string>;
  compact?: boolean;
  hiddenServices: ReadonlySet<string>;
  toggledRows: ReadonlySet<number>;
  onToggleService: (service: string) => void;
  onToggleRow: (row: number) => void;
}

function TraceLogRows({
  logs,
  serviceColorMap,
  compact,
  hiddenServices,
  toggledRows,
  onToggleService: toggleService,
  onToggleRow: toggleExpand,
}: TraceLogRowsProps) {
  const services = useMemo(() => [...new Set(logs.map((l) => l.serviceName))], [logs]);
  const visibleLogs = useMemo(
    () => logs.map((log, i) => ({ log, i })).filter(({ log }) => !hiddenServices.has(log.serviceName)),
    [logs, hiddenServices]
  );

  return (
    <div>
      {/* 服务 chip 过滤（多服务时才显示，前端即时过滤） */}
      {!compact && services.length > 1 && (
        <div className="flex flex-wrap gap-1.5 px-4 py-2 border-b border-[var(--border-color)]/40">
          {services.map((svc) => {
            const color = serviceColorMap?.get(svc) ?? "#94a3b8";
            const hidden = hiddenServices.has(svc);
            return (
              <button
                key={svc}
                onClick={() => toggleService(svc)}
                className={`flex items-center gap-1.5 text-[11px] px-2 py-0.5 rounded-full border transition-all ${
                  hidden ? "opacity-40 border-[var(--border-color)]" : "border-[var(--border-color)] hover:border-[var(--text-muted)]"
                }`}
              >
                <span className="w-2 h-2 rounded-full" style={{ backgroundColor: color }} />
                <span className="text-default">{svc}</span>
              </button>
            );
          })}
        </div>
      )}

      <div className={`overflow-auto ${compact ? "max-h-[300px]" : "max-h-[500px]"}`}>
        {visibleLogs.map(({ log, i }) => {
          const severityClass =
            log.severity === "ERROR" ? "bg-red-500/10 text-red-500" :
            log.severity === "WARN" ? "bg-amber-500/10 text-amber-500" :
            log.severity === "DEBUG" ? "bg-gray-500/10 text-gray-500" :
            "bg-blue-500/10 text-blue-500";
          const color = serviceColorMap?.get(log.serviceName) ?? "#94a3b8";
          const attrEntries = sortedAttrEntries(log.attributes ?? {});
          const expandable = attrEntries.length > 0;
          const isOpen = isOpenByDefault(log) !== toggledRows.has(i);
          return (
            <div key={i} className="border-b border-[var(--border-color)]/20">
              <div
                onClick={() => expandable && toggleExpand(i)}
                className={`flex items-start gap-2 px-4 py-2 text-xs hover:bg-[var(--hover-bg)] ${expandable ? "cursor-pointer" : ""}`}
              >
                <span className="text-[10px] text-muted flex-shrink-0 w-[70px] pt-0.5 font-mono">
                  {new Date(log.timestamp).toLocaleTimeString()}
                </span>
                <span className="w-2 h-2 rounded-full flex-shrink-0 mt-1" style={{ backgroundColor: color }} />
                <span className={`px-1.5 py-0.5 rounded text-[10px] font-semibold flex-shrink-0 ${severityClass}`}>
                  {log.severity}
                </span>
                <span className="text-muted flex-shrink-0 max-w-[120px] truncate">{log.serviceName}</span>
                <span className="text-default break-all flex-1">{log.body}</span>
                {expandable && (
                  <span className="flex-shrink-0 text-muted pt-0.5">
                    {isOpen ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
                  </span>
                )}
              </div>
              {expandable && isOpen && (
                <div className="mx-4 mb-2 ml-[100px] border border-[var(--border-color)]/60 rounded-md overflow-hidden">
                  {attrEntries.map(([k, v]) => {
                    const isEx = k.startsWith("exception.");
                    return (
                      <div key={k} className="grid grid-cols-[160px_1fr] text-[11px] font-mono border-b border-[var(--border-color)]/40 last:border-b-0">
                        <span className={`px-2.5 py-1 border-r border-[var(--border-color)]/40 ${isEx ? "text-red-400/70" : "text-muted"}`}>{k}</span>
                        <span className={`px-2.5 py-1 break-all ${isEx ? "text-red-400" : "text-default"}`}>{v}</span>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
