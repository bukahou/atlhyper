"use client";

import { useState, useEffect, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import { RefreshCw, Loader2, WifiOff, AlertTriangle } from "lucide-react";

import { RiskGauge } from "./components/RiskGauge";
import { RiskTrendChart } from "./components/RiskTrendChart";
import { TopEntities } from "./components/TopEntities";

import { getClusterRisk, getClusterRiskTrend, getEntityRisks } from "@/api/aiops";
import type { ClusterRisk, EntityRisk, RiskTrendPoint } from "@/api/aiops";

export default function RiskPage() {
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());

  const [clusterRisk, setClusterRisk] = useState<ClusterRisk | null>(null);
  const [trendData, setTrendData] = useState<RiskTrendPoint[]>([]);
  const [entities, setEntities] = useState<EntityRisk[]>([]);

  const loadData = useCallback(
    async (showLoading = true) => {
      if (!currentClusterId) return;
      if (showLoading) setIsRefreshing(true);

      try {
        const [risk, trend, entityList] = await Promise.all([
          getClusterRisk(currentClusterId),
          getClusterRiskTrend(currentClusterId),
          getEntityRisks(currentClusterId, "r_final", 20),
        ]);
        setClusterRisk(risk);
        setTrendData(trend);
        setEntities(entityList);
        setError(null);
        setLastUpdate(new Date());
      } catch (err) {
        console.error("Failed to load risk data:", err);
        setError(t.aiops.loadFailed);
      } finally {
        setLoading(false);
        setIsRefreshing(false);
      }
    },
    [currentClusterId, t.aiops.loadFailed]
  );

  // 初始加载
  useEffect(() => {
    loadData();
  }, [loadData]);

  // 30s 自动刷新
  useEffect(() => {
    const interval = setInterval(() => loadData(false), 30000);
    return () => clearInterval(interval);
  }, [loadData]);

  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center h-96">
          <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      </Layout>
    );
  }

  if (!currentClusterId) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <WifiOff className="w-12 h-12 mb-4 text-muted" />
          <p className="text-default font-medium mb-2">{t.aiops.noCluster}</p>
          <p className="text-sm text-muted">{t.aiops.noClusterDesc}</p>
        </div>
      </Layout>
    );
  }

  if (error && !clusterRisk) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <AlertTriangle className="w-12 h-12 mb-4 text-yellow-500" />
          <p className="text-default font-medium mb-2">{error}</p>
          <button
            onClick={() => loadData(true)}
            className="mt-4 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            {t.aiops.retry}
          </button>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-6">
        {/* 标题栏 */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg sm:text-xl font-bold text-default">{t.aiops.riskDashboard}</h1>
            <p className="text-xs sm:text-sm text-muted mt-1">{t.aiops.pageDescription}</p>
          </div>
          <div className="flex items-center gap-2 sm:gap-3 flex-shrink-0">
            <span className="text-[10px] sm:text-xs text-muted hidden sm:block">
              {t.aiops.lastUpdate}: {lastUpdate.toLocaleTimeString()}
            </span>
            <button
              onClick={() => loadData(true)}
              disabled={isRefreshing}
              className="p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${isRefreshing ? "animate-spin" : ""}`} />
            </button>
          </div>
        </div>

        {/* 上方: RiskGauge + RiskTrendChart */}
        {clusterRisk && (
          <div className="grid grid-cols-1 lg:grid-cols-[320px_1fr] gap-4">
            <RiskGauge
              risk={clusterRisk.risk}
              level={clusterRisk.level}
              anomalyCount={clusterRisk.anomalyCount}
              totalEntities={clusterRisk.totalEntities}
            />
            <RiskTrendChart data={trendData} />
          </div>
        )}

        {/* 下方: TopEntities */}
        <TopEntities entities={entities} clusterId={currentClusterId} />
      </div>
    </Layout>
  );
}
