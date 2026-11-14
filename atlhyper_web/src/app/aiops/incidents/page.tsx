"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Layout } from "@/components/layout/Layout";
import { QueryView } from "@/components/common";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useManualRefetch } from "@/hooks/useManualRefetch";
import { useI18n } from "@/i18n/context";
import { incidentsQuery, incidentStatsQuery } from "@/queries/aiops";
import { aiopsKeys } from "@/queries/keys";
import { RefreshCw } from "lucide-react";

import { IncidentStats } from "./components/IncidentStats";
import { IncidentList } from "./components/IncidentList";
import { IncidentDetailModal } from "./components/IncidentDetailModal";

const STATE_FILTERS = ["", "warning", "incident", "recovery", "stable"] as const;

export default function IncidentsPage() {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  const [stateFilter, setStateFilter] = useState("");
  const [selectedIncidentId, setSelectedIncidentId] = useState<string | null>(null);

  const statsQuery = useQuery(incidentStatsQuery(clusterId));
  const listQuery = useQuery(incidentsQuery(clusterId, stateFilter));
  const { refetch: refreshIncidents, isRefetching } = useManualRefetch(aiopsKeys.all);

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
            {statsQuery.dataUpdatedAt > 0 && (
              <span className="text-[10px] sm:text-xs text-muted hidden sm:block">
                {t.aiops.lastUpdate}: {new Date(statsQuery.dataUpdatedAt).toLocaleTimeString()}
              </span>
            )}
            <button
              onClick={refreshIncidents}
              disabled={isRefetching}
              title={t.common.refresh}
              className="p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${isRefetching ? "animate-spin" : ""}`} />
            </button>
          </div>
        </div>

        <QueryView query={statsQuery} requireCluster>
          {(stats) => (
            <>
              {/* 统计卡片 */}
              <IncidentStats stats={stats} />

              {/* 过滤栏 */}
              <div className="flex flex-wrap gap-2">
                {STATE_FILTERS.map((s) => (
                  <button
                    key={s || "all"}
                    onClick={() => setStateFilter(s)}
                    title={s ? (t.aiops.stateDesc[s as keyof typeof t.aiops.stateDesc] ?? "") : ""}
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

              {/* 事件列表：换状态筛选时先留着上一份，新结果到了再换 */}
              <QueryView query={listQuery}>
                {(incidents) => <IncidentList incidents={incidents} onSelect={setSelectedIncidentId} />}
              </QueryView>
            </>
          )}
        </QueryView>

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
