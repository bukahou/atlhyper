"use client";

import { Layout } from "@/components/layout/Layout";
import { useClusterStore } from "@/store/clusterStore";
import { useI18n } from "@/i18n/context";
import { DatabaseZap } from "lucide-react";

interface OTelGuardProps {
  children: React.ReactNode;
}

/**
 * OTelGuard 包裹 Observe 页面（Metrics / APM / Logs / SLO）
 * 当当前集群未部署 OTel + ClickHouse 时，展示引导页面而非报错
 */
export function OTelGuard({ children }: OTelGuardProps) {
  const { t } = useI18n();
  const { isOTelAvailable } = useClusterStore();

  if (!isOTelAvailable()) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <div className="p-4 rounded-2xl bg-violet-500/10 mb-6">
            <DatabaseZap className="w-12 h-12 text-violet-500" />
          </div>
          <h2 className="text-lg font-semibold text-default mb-2">
            {t.common.otelRequired}
          </h2>
          <p className="text-sm text-muted max-w-md mb-3">
            {t.common.otelRequiredDesc}
          </p>
          <p className="text-xs text-muted/70 max-w-md">
            {t.common.otelRequiredHint}
          </p>
        </div>
      </Layout>
    );
  }

  return <>{children}</>;
}
