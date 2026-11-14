"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, QueryView } from "@/components/common";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { clusterOverviewQuery, overviewSLODomainsQuery } from "@/queries/overview";
import { Server, Cpu, HardDrive, AlertTriangle } from "lucide-react";

// 组件
import {
  HealthCard,
  StatCard,
  NodeResourceCard,
  RecentAlertsCard,
  WorkloadSummaryCard,
  SloOverviewCard,
  AlertDetailModal,
} from "./components";
import type { AlertItem } from "./components";

// 工具函数
import { transformOverview } from "./utils";

export default function OverviewPage() {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  const overviewQuery = useQuery(clusterOverviewQuery(clusterId));
  // SLO 摘要是另一份查询：集群没接 OTel 时它取不到，卡片显示「暂无数据」，不影响概览
  const sloQuery = useQuery(overviewSLODomainsQuery(clusterId));
  const overview = overviewQuery.data;
  const data = useMemo(() => transformOverview(overview), [overview]);

  // 告警详情弹窗状态
  const [selectedAlert, setSelectedAlert] = useState<AlertItem | null>(null);

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.overview} />

        <QueryView query={overviewQuery} requireCluster>
          {() => (
            <>
              {/* Top Stats Cards */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
                <HealthCard data={data.healthCard} t={t} />
                <StatCard
                  title={t.overview.nodes}
                  value={`${data.nodesCard.readyNodes} / ${data.nodesCard.totalNodes}`}
                  subText={`${t.status.ready}: ${data.nodesCard.nodeReadyPct.toFixed(1)}%`}
                  icon={Server}
                  percent={data.nodesCard.nodeReadyPct}
                  accentColor="#6366F1"
                />
                <StatCard
                  title={t.overview.clusterAvgCpu}
                  value={data.cpuCard.percent}
                  icon={Cpu}
                  percent={data.cpuCard.percent}
                  accentColor="#F97316"
                />
                <StatCard
                  title={t.overview.clusterAvgMem}
                  value={data.memCard.percent}
                  icon={HardDrive}
                  percent={data.memCard.percent}
                  accentColor="#10B981"
                />
                <StatCard
                  title={t.overview.alerts}
                  value={data.alertsTotal}
                  subText="24h"
                  icon={AlertTriangle}
                  accentColor="#EF4444"
                />
              </div>

              {/* Charts Section */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <WorkloadSummaryCard workloads={data.workloads} podStatus={data.podStatus} peakStats={data.peakStats} t={t} />
                <SloOverviewCard data={sloQuery.data ?? null} t={t} />
              </div>

              {/* Bottom Section */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <NodeResourceCard nodes={data.nodeUsages} t={t} />
                <RecentAlertsCard alerts={data.recentAlerts} onAlertClick={setSelectedAlert} t={t} />
              </div>
            </>
          )}
        </QueryView>
      </div>

      {/* 告警详情弹窗 */}
      <AlertDetailModal alert={selectedAlert} onClose={() => setSelectedAlert(null)} t={t} />
    </Layout>
  );
}
