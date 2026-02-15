"use client";

import { useState, useEffect, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import { RefreshCw, Loader2, WifiOff, AlertTriangle } from "lucide-react";

import { IncidentStats } from "./components/IncidentStats";
import { IncidentList } from "./components/IncidentList";
import { IncidentDetailModal } from "./components/IncidentDetailModal";

import { getIncidents, getIncidentStats } from "@/api/aiops";
import type { Incident, IncidentStats as IncidentStatsType } from "@/api/aiops";

const STATE_FILTERS = ["", "warning", "incident", "recovery", "stable"] as const;

export default function IncidentsPage() {
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());

  const [stats, setStats] = useState<IncidentStatsType | null>(null);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [stateFilter, setStateFilter] = useState("");
  const [selectedIncidentId, setSelectedIncidentId] = useState<string | null>(null);

  const loadData = useCallback(
    async (showLoading = true) => {
      if (!currentClusterId) return;
      if (showLoading) setIsRefreshing(true);

      try {
        const [statsData, incidentData] = await Promise.all([
          getIncidentStats(currentClusterId),
          getIncidents({
            cluster: currentClusterId,
            state: stateFilter || undefined,
            limit: 20,
          }),
        ]);
        setStats(statsData);
        setIncidents(incidentData);
        setError(null);
        setLastUpdate(new Date());
      } catch (err) {
        console.error("Failed to load incidents:", err);
        setError(t.aiops.loadFailed);
      } finally {
        setLoading(false);
        setIsRefreshing(false);
      }
    },
    [currentClusterId, stateFilter, t.aiops.loadFailed]
  );

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

  if (error && !stats) {
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
            <h1 className="text-lg sm:text-xl font-bold text-default">{t.aiops.incidents}</h1>
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

        {/* 统计卡片 */}
        {stats && <IncidentStats stats={stats} />}

        {/* 过滤栏 */}
        <div className="flex flex-wrap gap-2">
          {STATE_FILTERS.map((s) => (
            <button
              key={s || "all"}
              onClick={() => setStateFilter(s)}
              className={`px-3 py-1.5 text-xs rounded-lg border transition-colors ${
                stateFilter === s
                  ? "bg-indigo-500 text-white border-indigo-500"
                  : "bg-card text-muted border-[var(--border-color)] hover:text-default"
              }`}
            >
              {s ? (t.aiops.state[s as keyof typeof t.aiops.state] ?? s) : t.common.all}
            </button>
          ))}
        </div>

        {/* 事件列表 */}
        <IncidentList incidents={incidents} onSelect={setSelectedIncidentId} />

        {/* 详情弹窗 */}
        <IncidentDetailModal
          incidentId={selectedIncidentId}
          open={!!selectedIncidentId}
          onClose={() => setSelectedIncidentId(null)}
        />
      </div>
    </Layout>
  );
}
