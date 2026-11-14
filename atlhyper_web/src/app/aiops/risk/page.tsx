"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Layout } from "@/components/layout/Layout";
import { QueryView } from "@/components/common";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useManualRefetch } from "@/hooks/useManualRefetch";
import { useI18n } from "@/i18n/context";
import { clusterRiskQuery, entityRisksQuery } from "@/queries/aiops";
import { aiopsKeys } from "@/queries/keys";
import { REFRESH } from "@/queries/refresh";
import { RefreshCw } from "lucide-react";

import { RiskGauge } from "./components/RiskGauge";
import { TopEntities } from "./components/TopEntities";

export default function RiskPage() {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  const [entityLimit, setEntityLimit] = useState(20);

  const clusterRisk = useQuery(clusterRiskQuery(clusterId));
  const entityRisks = useQuery(entityRisksQuery(clusterId, entityLimit, REFRESH.slow));
  const { refetch: refreshRisk, isRefetching } = useManualRefetch(aiopsKeys.all);

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
            {clusterRisk.dataUpdatedAt > 0 && (
              <span className="text-[10px] sm:text-xs text-muted hidden sm:block">
                {t.aiops.lastUpdate}: {new Date(clusterRisk.dataUpdatedAt).toLocaleTimeString()}
              </span>
            )}
            <button
              onClick={refreshRisk}
              disabled={isRefetching}
              title={t.common.refresh}
              className="p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${isRefetching ? "animate-spin" : ""}`} />
            </button>
          </div>
        </div>

        <QueryView query={clusterRisk} requireCluster>
          {(risk) => (
            <>
              {/* 上方: RiskGauge */}
              <RiskGauge
                risk={risk.risk}
                level={risk.level}
                anomalyCount={risk.anomalyCount}
                totalEntities={risk.totalEntities}
              />

              {/* 下方: TopEntities */}
              <QueryView query={entityRisks}>
                {(entities) => (
                  <TopEntities
                    entities={entities}
                    clusterId={clusterId}
                    limit={entityLimit}
                    onLimitChange={setEntityLimit}
                  />
                )}
              </QueryView>
            </>
          )}
        </QueryView>
      </div>
    </Layout>
  );
}
