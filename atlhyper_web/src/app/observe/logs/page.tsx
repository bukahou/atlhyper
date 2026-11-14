"use client";

import { useState, useMemo } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";
import { Layout } from "@/components/layout/Layout";
import { QueryView, TimeRangePicker } from "@/components/common";
import { OTelGuard } from "@/components/observe/OTelGuard";
import { SignalFreshnessBadge } from "@/components/observe/SignalFreshnessBadge";
import { useCan } from "@/hooks/useCan";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useDebouncedValue } from "@/hooks/useDebouncedValue";
import { useManualRefetch } from "@/hooks/useManualRefetch";
import { useObserveTimeRange } from "@/hooks/useObserveTimeRange";
import { useScopedState } from "@/hooks/useScopedState";
import { useSessionReady } from "@/hooks/useSessionReady";
import { useSignalFreshness } from "@/hooks/useSignalFreshness";
import { useI18n } from "@/i18n/context";
import { toSpanMs, toTimeRangeParams } from "@/lib/time-range";
import { logsKeys } from "@/queries/keys";
import { logHistogramQuery, logListQuery, resolveLogListRange, type BrushedBuckets } from "@/queries/observe-logs";
import { REFRESH } from "@/queries/refresh";
import type { LogEntry } from "@/types/model/log";
import { LogDetailDrawer } from "./components/LogDetail";
import { LogExplorer } from "./components/LogExplorer";

const PAGE_SIZE = 50;

export default function LogsPage() {
  return (
    <OTelGuard>
      <LogsPageContent />
    </OTelGuard>
  );
}

function LogsPageContent() {
  const { t } = useI18n();
  const tl = t.logs;
  const { clusterId } = useCurrentCluster();
  // 日志正文需要 logs.read。权限加载完之前一律按没有权限算：不发请求，界面由 PermissionLock 显示加载中
  const sessionReady = useSessionReady();
  const can = useCan();
  const canReadLogs = sessionReady && can("logs.read");

  // URL ?trace= / ?spanId= params
  const searchParams = useSearchParams();
  // 参数名统一为 trace —— APM 页与 LogDetail 的跳转链接一直用它，
  // 这里此前读的是 traceId，等于三个地方两套叫法。保留旧名兼容已有链接。
  const urlTraceId = searchParams.get("trace") || searchParams.get("traceId") || undefined;
  const urlSpanId = searchParams.get("spanId") || undefined;

  // 搜索框里的字随敲随显，进查询键的是去抖后的值：停下来才发请求
  const [searchInput, setSearchInput] = useState("");
  const search = useDebouncedValue(searchInput);
  // 从其他信号跳转过来时预填过滤条件（如 SLO → 某服务的 ERROR 日志）
  const urlService = searchParams.get("service") || "";
  const urlSeverity = searchParams.get("severity") || "";
  const [selectedServices, setSelectedServices] = useState<string[]>(urlService ? [urlService] : []);
  const [selectedSeverities, setSelectedSeverities] = useState<string[]>(urlSeverity ? [urlSeverity] : []);
  const [selectedScopes, setSelectedScopes] = useState<string[]>([]);

  // 时间轴是全局的 —— 从 APM 切过来时保持同一段时间，不用重选
  const { selection: timeSelection, setSelection: setTimeSelection } = useObserveTimeRange("full");
  const freshness = useSignalFreshness("logs");

  // 筛选条件 + 时间范围：直方图按它取数，也是 brush 选区与页码所依附的范围
  const filterParams = useMemo(
    () => ({
      search,
      services: selectedServices,
      severities: selectedSeverities,
      scopes: selectedScopes,
      ...toTimeRangeParams(timeSelection),
    }),
    [search, selectedServices, selectedSeverities, selectedScopes, timeSelection],
  );
  const filterScope = JSON.stringify(filterParams);

  // 筛选条件一变，brush 选区作废；筛选条件或选区一变，页码回到第一页
  const [brush, setBrush] = useScopedState<BrushedBuckets | null>(filterScope, null);
  const [page, setPage] = useScopedState(`${filterScope}|${JSON.stringify(brush)}`, 1);

  const histogramQuery = useQuery(logHistogramQuery(clusterId, filterParams));
  // 列表的查询条件里没有任何来自直方图返回的东西：选区的桶宽在选中那一刻就存进了 brush，
  // 所以直方图刷新不会让列表重新请求，首屏两者各发一次
  const listQuery = useQuery(
    logListQuery(
      clusterId,
      {
        ...filterParams,
        ...resolveLogListRange(toTimeRangeParams(timeSelection), brush),
        limit: PAGE_SIZE,
        offset: (page - 1) * PAGE_SIZE,
        traceId: urlTraceId,
        spanId: urlSpanId,
      },
      canReadLogs,
    ),
  );
  const { refetch: refreshLogs } = useManualRefetch(logsKeys.all);

  // Log detail drawer
  const [selectedEntry, setSelectedEntry] = useState<LogEntry | null>(null);
  const [selectedIdx, setSelectedIdx] = useState<number | null>(null);

  const handleSelectEntry = (entry: LogEntry, idx: number) => {
    if (selectedIdx === idx) {
      setSelectedEntry(null);
      setSelectedIdx(null);
    } else {
      setSelectedEntry(entry);
      setSelectedIdx(idx);
    }
  };
  const handleCloseDrawer = () => {
    setSelectedEntry(null);
    setSelectedIdx(null);
  };
  const handlePageChange = (nextPage: number) => {
    setPage(nextPage);
    handleCloseDrawer();
  };

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-5 overflow-hidden">
        {/* Header */}
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-xl sm:text-2xl font-bold text-default">{tl.pageTitle}</h1>
            <p className="text-xs text-muted mt-1">{tl.pageDescription}</p>
          </div>

          <div className="flex items-center gap-2">
            <SignalFreshnessBadge item={freshness} />
            <TimeRangePicker
              value={timeSelection}
              onChange={setTimeSelection}
              t={tl}
            />
            <span className="text-xs text-muted bg-[var(--background)] px-2 py-1 rounded">
              {REFRESH.moderate / 1000}s
            </span>
            <button
              onClick={refreshLogs}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg bg-primary text-white hover:bg-primary/90 transition-colors"
            >
              <RefreshCw className="w-3.5 h-3.5" />
              {t.common.refresh}
            </button>
          </div>
        </div>

        {/* 直方图只有计数、所有人可见，用它决定整页的加载 / 出错 / 无集群状态 */}
        <QueryView query={histogramQuery} requireCluster>
          {(histogram) => (
            <LogExplorer
              tl={tl}
              histogram={histogram}
              listQuery={listQuery}
              timeSpanMs={toSpanMs(timeSelection)}
              searchInput={searchInput}
              onSearchInputChange={setSearchInput}
              searchHighlight={search}
              selectedServices={selectedServices}
              selectedSeverities={selectedSeverities}
              selectedScopes={selectedScopes}
              onServicesChange={setSelectedServices}
              onSeveritiesChange={setSelectedSeverities}
              onScopesChange={setSelectedScopes}
              brush={brush}
              onBrushChange={setBrush}
              urlTraceId={urlTraceId}
              page={page}
              pageSize={PAGE_SIZE}
              onPageChange={handlePageChange}
              selectedIdx={selectedIdx}
              onSelectEntry={handleSelectEntry}
            />
          )}
        </QueryView>
      </div>

      {/* Log detail drawer */}
      <LogDetailDrawer
        entry={selectedEntry}
        onClose={handleCloseDrawer}
        t={tl}
      />
    </Layout>
  );
}
