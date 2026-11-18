"use client";

import { useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { Layout } from "@/components/layout/Layout";
import { OTelGuard } from "@/components/observe/OTelGuard";
import { useCan } from "@/hooks/useCan";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useManualRefetch } from "@/hooks/useManualRefetch";
import { useObserveTimeRange } from "@/hooks/useObserveTimeRange";
import { useSessionReady } from "@/hooks/useSessionReady";
import { useSignalFreshness } from "@/hooks/useSignalFreshness";
import { useI18n } from "@/i18n/context";
import { toTimeRangeParams } from "@/lib/time-range";
import { apmKeys } from "@/queries/keys";
import { displayedTraceView, traceDetailViewFromLink, type ApmView } from "./apm-view";
import { ApmPageHeader } from "./components/ApmPageHeader";
import { ServiceDetailLevel } from "./components/ServiceDetailLevel";
import { ServicesLevel } from "./components/ServicesLevel";
import { TraceDetailLevel } from "./components/TraceDetailLevel";
import { useTraceDetailData } from "./useTraceDetailData";

export default function ApmPage() {
  return (
    <OTelGuard>
      <ApmPageContent />
    </OTelGuard>
  );
}

function ApmPageContent() {
  const { t } = useI18n();
  const ta = t.apm;
  const { clusterId } = useCurrentCluster();
  // 链路列表 / 详情 / 相关性需要 traces.read。权限加载完之前一律按没有权限算：不发请求
  const sessionReady = useSessionReady();
  const can = useCan();
  const canReadTraces = sessionReady && can("traces.read");

  // 时间轴是全局的 —— 切到 Logs 查同一时刻的日志时不用重选
  const { selection: timeSelection, setSelection: setTimeSelection } = useObserveTimeRange("full");
  const freshness = useSignalFreshness("traces");
  const timeRange = useMemo(() => toTimeRangeParams(timeSelection), [timeSelection]);

  // 跨信号关联：URL ?trace=xxx 直接进入链路详情
  const linkedTraceId = useSearchParams().get("trace") ?? "";
  const [view, setView] = useState<ApmView>(() =>
    linkedTraceId ? traceDetailViewFromLink(linkedTraceId) : { level: "services" },
  );
  // 停在本页期间 ?trace= 变了（浏览器前进 / 后退）：跟着跳到新的那条。
  // 记住处理过的值，用户从深链的链路退回服务列表后不会被拉回去
  const [handledLinkedTraceId, setHandledLinkedTraceId] = useState(linkedTraceId);
  if (linkedTraceId !== handledLinkedTraceId) {
    setHandledLinkedTraceId(linkedTraceId);
    if (linkedTraceId) setView(traceDetailViewFromLink(linkedTraceId));
  }

  const traceData = useTraceDetailData(clusterId, view, timeRange, canReadTraces);
  const { refetch: refreshApm, isRefetching } = useManualRefetch(apmKeys.all);

  const goToServices = () => setView({ level: "services" });
  const goToService = (serviceName: string) => setView({ level: "service-detail", serviceName });

  // 面包屑上的链路层：traceId 用当前显示的那条；深链进来时服务与操作要等详情取到才知道
  const headerView =
    view.level === "trace-detail" ? displayedTraceView(view, traceData.currentTraceId, traceData.currentSummary) : view;

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-5">
        <ApmPageHeader
          ta={ta}
          tc={t.common}
          view={headerView}
          timeSelection={timeSelection}
          freshness={freshness}
          isRefreshing={isRefetching}
          onTimeChange={setTimeSelection}
          onRefresh={refreshApm}
          onGoToServices={goToServices}
          onGoToService={goToService}
        />

        {view.level === "services" && (
          <ServicesLevel ta={ta} tt={t.table} clusterId={clusterId} timeRange={timeRange} onSelectService={goToService} />
        )}

        {view.level === "service-detail" && (
          <ServiceDetailLevel
            ta={ta}
            clusterId={clusterId}
            serviceName={view.serviceName}
            timeRange={timeRange}
            canReadTraces={canReadTraces}
            // 点事务行：进入链路详情，由那一层按这个操作取样本序列
            onSelectOperation={(operationName) =>
              setView({ level: "trace-detail", serviceName: view.serviceName, operationName, traceId: "" })
            }
            onNavigateToService={goToService}
            // 直接点某条链路（慢链路、出错链路、按状态码筛出的链路）
            onSelectTrace={(traceId) =>
              setView({ level: "trace-detail", serviceName: view.serviceName, operationName: "", traceId })
            }
          />
        )}

        {view.level === "trace-detail" && (
          <TraceDetailLevel
            // 换了进入路径（另一个服务 / 操作）就是另一组样本，这一层的状态（延迟选区）重新开始
            key={`${view.serviceName}|${view.operationName}`}
            ta={ta}
            traceData={traceData}
            initialFocusService={headerView.level === "trace-detail" ? headerView.serviceName : ""}
            onSelectTrace={(traceId) => setView({ ...view, traceId })}
          />
        )}
      </div>
    </Layout>
  );
}
