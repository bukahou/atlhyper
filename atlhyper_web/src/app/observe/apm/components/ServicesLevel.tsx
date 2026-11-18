"use client";

import { useQuery } from "@tanstack/react-query";
import { QueryView } from "@/components/common";
import type { TimeRangeParams } from "@/lib/time-range";
import { apmServicesQuery, apmTopologyQuery } from "@/queries/observe-apm";
import type { ApmTranslations, TableTranslations } from "@/types/i18n";
import { ServiceList } from "./ServiceList";
import { ServiceTopology } from "./ServiceTopology";

interface ServicesLevelProps {
  ta: ApmTranslations;
  tt: TableTranslations;
  clusterId: string;
  timeRange: TimeRangeParams;
  onSelectService: (serviceName: string) => void;
}

/**
 * 第一层：服务拓扑 + 服务列表。
 *
 * 查询写在这一层的组件里：只有显示这一层时它们才在轮询，下钻到别的层就停。
 */
export function ServicesLevel({ ta, tt, clusterId, timeRange, onSelectService }: ServicesLevelProps) {
  const servicesQuery = useQuery(apmServicesQuery(clusterId, timeRange));
  const topologyQuery = useQuery(apmTopologyQuery(clusterId, timeRange));

  return (
    <QueryView query={servicesQuery} requireCluster>
      {(services) => (
        <>
          <QueryView query={topologyQuery}>
            {(topology) => <ServiceTopology t={ta} topology={topology} onSelectService={onSelectService} />}
          </QueryView>
          <ServiceList t={ta} tt={tt} serviceStats={services} onSelectService={onSelectService} />
        </>
      )}
    </QueryView>
  );
}
