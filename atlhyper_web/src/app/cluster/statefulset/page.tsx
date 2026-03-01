"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getStatefulSetList } from "@/datasource/cluster";
import type { StatefulSetListItem } from "@/api/workload";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, StatusBadge, FilterBar, type TableColumn } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import { Eye } from "lucide-react";
import { StatefulSetDetailModal } from "@/components/statefulset";

export default function StatefulSetPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<StatefulSetListItem[]>([]);
  const [error, setError] = useState("");

  // 筛选状态
  const [filters, setFilters] = useState({
    namespace: "",
    search: "",
  });

  // 详情弹窗状态
  const [selectedItem, setSelectedItem] = useState<StatefulSetListItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getStatefulSetList({ cluster_id: getCurrentClusterId() });
      setItems(res.data.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, []);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // 提取唯一的 namespaces
  const namespaces = useMemo(() => {
    const nsSet = new Set<string>();
    items.forEach((d) => {
      if (d.namespace) nsSet.add(d.namespace);
    });
    return Array.from(nsSet).sort();
  }, [items]);

  // StatsCards 统计
  const stats = useMemo(() => {
    const totalReplicas = items.reduce((sum, d) => sum + d.replicas, 0);
    const totalReady = items.reduce((sum, d) => sum + d.ready, 0);
    return {
      total: items.length,
      namespaces: namespaces.length,
      replicas: totalReplicas,
      ready: totalReady,
    };
  }, [items, namespaces]);

  // 根据筛选条件过滤数据
  const filteredItems = useMemo(() => {
    return items.filter((d) => {
      if (filters.search && !d.name.toLowerCase().includes(filters.search.toLowerCase())) {
        return false;
      }
      if (filters.namespace && d.namespace !== filters.namespace) {
        return false;
      }
      return true;
    });
  }, [items, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const handleViewDetail = (item: StatefulSetListItem) => {
    setSelectedItem(item);
    setDetailOpen(true);
  };

  const columns: TableColumn<StatefulSetListItem>[] = [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (d) => (
        <div>
          <span className="font-medium text-default">{d.name || "-"}</span>
          {d.serviceName && (
            <div className="text-xs text-muted truncate max-w-[200px]" title={d.serviceName}>
              {t.statefulset.serviceName}: {d.serviceName}
            </div>
          )}
        </div>
      ),
    },
    { key: "namespace", header: t.common.namespace },
    {
      key: "ready",
      header: `${t.statefulset.readyReplicas}/${t.statefulset.replicas}`,
      render: (d) => {
        const label = `${d.ready}/${d.replicas}`;
        const type = d.ready === d.replicas ? "success" : d.ready === 0 ? "error" : "warning";
        return <StatusBadge status={label} type={type} />;
      },
    },
    {
      key: "updated",
      header: t.statefulset.updatedReplicas,
      mobileVisible: false,
      render: (d) => String(d.updated),
    },
    {
      key: "age",
      header: t.statefulset.age,
      mobileVisible: false,
      render: (d) => d.age || "-",
    },
    {
      key: "action",
      header: t.common.action,
      mobileVisible: false,
      render: (d) => (
        <button
          onClick={(e) => {
            e.stopPropagation();
            handleViewDetail(d);
          }}
          className="p-2 hover-bg rounded-lg"
          title={t.common.details}
        >
          <Eye className="w-4 h-4 text-muted" />
        </button>
      ),
    },
  ];

  return (
    <Layout>
      <div className="space-y-4">
        <PageHeader
          title={t.nav.statefulset}
          description={t.statefulset.pageDescription}
          autoRefreshSeconds={intervalSeconds}
        />

        {items.length > 0 && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={stats.total} />
            <StatsCard label={t.common.namespace} value={stats.namespaces} iconColor="text-blue-500" />
            <StatsCard label={t.statefulset.replicas} value={stats.replicas} iconColor="text-purple-500" />
            <StatsCard label={t.statefulset.readyReplicas} value={stats.ready} iconColor="text-green-500" />
          </div>
        )}

        <FilterBar
          namespaces={namespaces}
          filters={filters}
          onFilterChange={handleFilterChange}
          searchPlaceholder={t.common.search + " StatefulSet..."}
        />

        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={filteredItems}
            loading={loading}
            error={error}
            keyExtractor={(d, index) => `${index}-${d.namespace}/${d.name}`}
            onRowClick={handleViewDetail}
            pageSize={10}
          />
        </div>
      </div>

      {selectedItem && (
        <StatefulSetDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespace={selectedItem.namespace}
          name={selectedItem.name}
        />
      )}
    </Layout>
  );
}
