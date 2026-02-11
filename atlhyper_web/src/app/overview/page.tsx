"use client";

import { useState, useEffect, useRef } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getClusterOverview } from "@/api/overview";
import { getClusterList } from "@/api/cluster";
import { getSLODomainsV2 } from "@/api/slo";
import { LoadingSpinner, PageHeader } from "@/components/common";
import { Server, Cpu, HardDrive, AlertTriangle } from "lucide-react";
import type { TransformedOverview } from "@/types/overview";
import type { DomainSLOListResponseV2 } from "@/types/slo";

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
import { emptyData, transformOverview } from "./utils";

// 内置刷新间隔：10秒
const REFRESH_INTERVAL = 10000;

export default function OverviewPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<TransformedOverview>(emptyData);
  const [sloData, setSloData] = useState<DomainSLOListResponseV2 | null>(null);
  const [error, setError] = useState("");
  const isMountedRef = useRef(true);
  const isFirstLoadRef = useRef(true);

  // 告警详情弹窗状态
  const [selectedAlert, setSelectedAlert] = useState<AlertItem | null>(null);

  // 异步获取数据（静默刷新，不影响 UI）
  useEffect(() => {
    isMountedRef.current = true;

    const fetchData = async () => {
      try {
        // 先获取集群列表，使用第一个可用的集群
        const clusterRes = await getClusterList();
        const clusters = clusterRes.data?.clusters || [];
        if (clusters.length === 0) {
          if (isMountedRef.current && isFirstLoadRef.current) {
            setError(t.common.noCluster);
          }
          return;
        }

        const clusterId = clusters[0].cluster_id;
        const [res, sloRes] = await Promise.all([
          getClusterOverview({ cluster_id: clusterId }),
          getSLODomainsV2({ clusterId }).catch(() => null),
        ]);
        if (isMountedRef.current) {
          setData(transformOverview(res.data?.data));
          setSloData(sloRes?.data ?? null);
          setError("");
        }
      } catch (err) {
        if (isMountedRef.current) {
          // 静默处理错误，保留现有数据
          console.warn("[Overview] Fetch error:", err);
          // 仅首次加载时显示错误
          if (isFirstLoadRef.current) {
            setError(err instanceof Error ? err.message : "Failed to load data");
          }
        }
      } finally {
        if (isMountedRef.current) {
          setLoading(false);
          isFirstLoadRef.current = false;
        }
      }
    };

    // 立即执行一次
    fetchData();

    // 设置 10s 定时刷新
    const intervalId = setInterval(fetchData, REFRESH_INTERVAL);

    return () => {
      isMountedRef.current = false;
      clearInterval(intervalId);
    };
  }, []);

  if (loading) {
    return (
      <Layout>
        <LoadingSpinner />
      </Layout>
    );
  }

  if (error && data === emptyData) {
    return (
      <Layout>
        <div className="text-center py-12 text-red-500">{error}</div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.overview} />

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
          <SloOverviewCard data={sloData} t={t} />
        </div>

        {/* Bottom Section */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <NodeResourceCard nodes={data.nodeUsages} t={t} />
          <RecentAlertsCard alerts={data.recentAlerts} onAlertClick={setSelectedAlert} t={t} />
        </div>
      </div>

      {/* 告警详情弹窗 */}
      <AlertDetailModal alert={selectedAlert} onClose={() => setSelectedAlert(null)} t={t} />
    </Layout>
  );
}
