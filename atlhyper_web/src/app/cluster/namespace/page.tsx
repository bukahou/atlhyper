"use client";

import { useState, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getNamespaceOverview } from "@/api/namespace";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, StatusBadge, LoadingSpinner } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import { FolderTree, Box, Eye } from "lucide-react";
import type { NamespaceOverview, NamespaceItem } from "@/types/cluster";
import { NamespaceDetailModal } from "@/components/namespace";

function NamespaceCard({ ns, onViewDetail, t }: { ns: NamespaceItem; onViewDetail: () => void; t: ReturnType<typeof useI18n>["t"] }) {
  return (
    <div
      className="bg-card rounded-xl border border-[var(--border-color)] p-6 hover:border-primary/50 transition-colors cursor-pointer"
      onClick={onViewDetail}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-primary/10 rounded-lg">
            <FolderTree className="w-5 h-5 text-primary" />
          </div>
          <div>
            <h3 className="font-semibold text-default">{ns.name || "-"}</h3>
            <p className="text-sm text-muted mt-1">{ns.createdAt ? new Date(ns.createdAt).toLocaleDateString() : "-"}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <StatusBadge status={ns.status} />
          <button
            onClick={(e) => {
              e.stopPropagation();
              onViewDetail();
            }}
            className="p-2 hover-bg rounded-lg"
            title={t.namespace.viewDetails}
          >
            <Eye className="w-4 h-4 text-muted" />
          </button>
        </div>
      </div>
      <div className="mt-4 flex items-center gap-4 text-sm">
        <div className="flex items-center gap-1">
          <Box className="w-4 h-4 text-muted" />
          <span className="text-muted">{t.namespace.pods}:</span>
          <span className="font-medium">{ns.podCount ?? 0}</span>
        </div>
        <div className="flex items-center gap-1">
          <span className="text-muted">{t.namespace.labels}:</span>
          <span className="font-medium">{ns.labelCount ?? 0}</span>
        </div>
      </div>
    </div>
  );
}

export default function NamespacePage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<NamespaceOverview | null>(null);
  const [error, setError] = useState("");

  // 详情弹窗状态
  const [selectedNamespace, setSelectedNamespace] = useState<NamespaceItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getNamespaceOverview({ ClusterID: getCurrentClusterId() });
      setData(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, []);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // 查看详情
  const handleViewDetail = (ns: NamespaceItem) => {
    setSelectedNamespace(ns);
    setDetailOpen(true);
  };

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title={t.nav.namespace}
          description={t.namespace.pageDescription}
          autoRefreshSeconds={intervalSeconds}
        />

        {data && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={data.cards.totalNamespaces ?? 0} />
            <StatsCard label={t.status.active} value={data.cards.activeCount ?? 0} iconColor="text-green-500" />
            <StatsCard label={t.status.terminated} value={data.cards.terminating ?? 0} iconColor="text-yellow-500" />
            <StatsCard label={t.namespace.pods} value={data.cards.totalPods ?? 0} iconColor="text-blue-500" />
          </div>
        )}

        {loading ? (
          <LoadingSpinner />
        ) : error ? (
          <div className="text-center py-12 text-red-500">{error}</div>
        ) : !data?.rows?.length ? (
          <div className="text-center py-12 text-gray-500">{t.common.noData}</div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {data.rows.map((ns) => (
              <NamespaceCard
                key={ns.name}
                ns={ns}
                onViewDetail={() => handleViewDetail(ns)}
                t={t}
              />
            ))}
          </div>
        )}
      </div>

      {/* Namespace 详情弹窗 */}
      {selectedNamespace && (
        <NamespaceDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespaceName={selectedNamespace.name}
        />
      )}
    </Layout>
  );
}
