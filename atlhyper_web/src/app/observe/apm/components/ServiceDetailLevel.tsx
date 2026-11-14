"use client";

import { useQuery } from "@tanstack/react-query";
import { QueryView } from "@/components/common";
import type { TimeRangeParams } from "@/lib/time-range";
import { apmOperationsQuery, apmTopologyQuery } from "@/queries/observe-apm";
import type { ApmTranslations } from "@/types/i18n";
import { ServiceOverview } from "./ServiceOverview";

interface ServiceDetailLevelProps {
  ta: ApmTranslations;
  clusterId: string;
  serviceName: string;
  timeRange: TimeRangeParams;
  /** 是否有 traces.read；该服务的链路列表与相关性是内容级数据 */
  canReadTraces: boolean;
  onSelectOperation: (operation: string) => void;
  onNavigateToService: (serviceName: string) => void;
  onSelectTrace: (traceId: string) => void;
}

/**
 * 第二层：某一个服务的概览（指标、趋势、事务、依赖、错误）。
 *
 * 操作统计与拓扑在这一层轮询；拓扑与第一层是同一个键，从服务列表点进来时直接用已有的缓存。
 */
export function ServiceDetailLevel({
  ta,
  clusterId,
  serviceName,
  timeRange,
  canReadTraces,
  onSelectOperation,
  onNavigateToService,
  onSelectTrace,
}: ServiceDetailLevelProps) {
  const operationsQuery = useQuery(apmOperationsQuery(clusterId, timeRange));
  const topologyQuery = useQuery(apmTopologyQuery(clusterId, timeRange));

  return (
    <QueryView query={operationsQuery} requireCluster>
      {(operations) => (
        <ServiceOverview
          t={ta}
          serviceName={serviceName}
          operations={operations}
          topologyQuery={topologyQuery}
          clusterId={clusterId}
          timeRange={timeRange}
          canReadTraces={canReadTraces}
          onSelectOperation={onSelectOperation}
          onNavigateToService={onNavigateToService}
          onSelectTrace={onSelectTrace}
        />
      )}
    </QueryView>
  );
}
