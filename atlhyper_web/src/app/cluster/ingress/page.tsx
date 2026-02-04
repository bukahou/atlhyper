"use client";

import { useState, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getIngressOverview } from "@/api/ingress";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, type TableColumn } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import { Globe, Lock, Eye } from "lucide-react";
import type { IngressItem, IngressOverview } from "@/types/cluster";
import { IngressDetailModal } from "@/components/ingress";

export default function IngressPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<IngressOverview | null>(null);
  const [error, setError] = useState("");

  // 详情弹窗状态
  const [selectedIngress, setSelectedIngress] = useState<{ namespace: string; name: string } | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getIngressOverview({ ClusterID: getCurrentClusterId() });
      setData(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, []);

  const { refresh, intervalSeconds } = useAutoRefresh(fetchData);

  const columns: TableColumn<IngressItem>[] = [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (ing) => (
        <div className="flex items-center gap-2">
          <Globe className="w-4 h-4 text-primary" />
          <span className="font-medium text-default">{ing.name || "-"}</span>
        </div>
      ),
    },
    { key: "namespace", header: t.common.namespace },
    {
      key: "host",
      header: t.ingress.host,
      render: (ing) => (
        <span className="inline-flex px-2 py-1 text-xs bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400 rounded">
          {ing.host || "-"}
        </span>
      ),
    },
    {
      key: "path",
      header: t.ingress.path,
      mobileVisible: false,
      render: (ing) => <span className="font-mono text-sm">{ing.path || "-"}</span>,
    },
    {
      key: "service",
      header: t.nav.service,
      render: (ing) => (
        <span className="text-sm">{ing.serviceName || "-"}:{ing.servicePort || "-"}</span>
      ),
    },
    {
      key: "tls",
      header: t.ingress.tls,
      render: (ing) => ing.tls && ing.tls !== "-" ? (
        <div className="flex items-center gap-1 text-green-600">
          <Lock className="w-3 h-3" />
          <span className="text-xs">{t.common.yes}</span>
        </div>
      ) : (
        <span className="text-xs text-muted">{t.common.no}</span>
      ),
    },
    {
      key: "actions",
      header: "",
      mobileVisible: false,
      render: (ing) => (
        <button
          onClick={() => handleViewDetail(ing)}
          className="p-2 hover-bg rounded-lg"
          title={t.ingress.viewDetails}
        >
          <Eye className="w-4 h-4 text-muted hover:text-primary" />
        </button>
      ),
    },
  ];

  // 查看详情
  const handleViewDetail = (ing: IngressItem) => {
    setSelectedIngress({ namespace: ing.namespace, name: ing.name });
    setDetailOpen(true);
  };

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title={t.nav.ingress}
          description={t.ingress.pageDescription}
          autoRefreshSeconds={intervalSeconds}
          onRefresh={refresh}
        />

        {data && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={data.cards.totalIngresses ?? 0} />
            <StatsCard label={t.ingress.host} value={data.cards.usedHosts ?? 0} iconColor="text-blue-500" />
            <StatsCard label={t.ingress.tls} value={data.cards.tlsCerts ?? 0} iconColor="text-green-500" />
            <StatsCard label={t.ingress.path} value={data.cards.totalPaths ?? 0} iconColor="text-purple-500" />
          </div>
        )}

        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={data?.rows || []}
            loading={loading}
            error={error}
            keyExtractor={(ing, index) => `${index}-${ing.namespace}/${ing.name}/${ing.host}${ing.path}`}
          />
        </div>
      </div>

      {/* Ingress 详情弹窗 */}
      {selectedIngress && (
        <IngressDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespace={selectedIngress.namespace}
          ingressName={selectedIngress.name}
        />
      )}
    </Layout>
  );
}
