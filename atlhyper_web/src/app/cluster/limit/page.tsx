"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getLimitRangeList } from "@/datasource/cluster";
import type { LimitRangeItem } from "@/api/cluster-resources";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, FilterBar, type TableColumn } from "@/components/common";
import { useClusterStore } from "@/store/clusterStore";
import { Eye } from "lucide-react";
import { LimitRangeDetailModal } from "@/components/limit-range";

export default function LimitRangePage() {
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();
  const [items, setItems] = useState<LimitRangeItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // 筛选状态
  const [filters, setFilters] = useState({
    namespace: "",
    search: "",
  });

  // 详情弹窗状态
  const [selectedItem, setSelectedItem] = useState<LimitRangeItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getLimitRangeList({ cluster_id: currentClusterId });
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
    items.forEach((item) => {
      if (item.namespace) nsSet.add(item.namespace);
    });
    return Array.from(nsSet).sort();
  }, [items]);

  // StatsCards 统计
  const stats = useMemo(() => {
    const totalRules = items.reduce((sum, item) => sum + item.items.length, 0);
    return {
      total: items.length,
      namespaces: namespaces.length,
      rulesCount: totalRules,
    };
  }, [items, namespaces]);

  // 根据筛选条件过滤数据
  const filteredItems = useMemo(() => {
    return items.filter((item) => {
      if (filters.search && !item.name.toLowerCase().includes(filters.search.toLowerCase())) {
        return false;
      }
      if (filters.namespace && item.namespace !== filters.namespace) {
        return false;
      }
      return true;
    });
  }, [items, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const handleViewDetail = (item: LimitRangeItem) => {
    setSelectedItem(item);
    setDetailOpen(true);
  };

  const columns: TableColumn<LimitRangeItem>[] = [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (item) => (
        <div>
          <span className="font-medium text-default">{item.name || "-"}</span>
          <div className="text-xs text-muted">{item.age || "-"}</div>
        </div>
      ),
    },
    { key: "namespace", header: t.common.namespace },
    {
      key: "itemsCount",
      header: t.policyPage.itemsCount,
      render: (item) => String(item.items.length),
    },
    {
      key: "types",
      header: t.policyPage.types,
      mobileVisible: false,
      render: (item) => {
        const types = Array.from(new Set(item.items.map((entry) => entry.type)));
        return types.join(", ") || "-";
      },
    },
    {
      key: "age",
      header: t.policyPage.age,
      mobileVisible: false,
      render: (item) => item.age || "-",
    },
    {
      key: "action",
      header: t.common.action,
      mobileVisible: false,
      render: (item) => (
        <button
          onClick={(e) => {
            e.stopPropagation();
            handleViewDetail(item);
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
          title={t.nav.limitRange}
          description={t.policyPage.limitRangeDescription}
          autoRefreshSeconds={intervalSeconds}
        />

        {items.length > 0 && (
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <StatsCard label={t.common.total} value={stats.total} />
            <StatsCard label={t.common.namespace} value={stats.namespaces} iconColor="text-blue-500" />
            <StatsCard label={t.policyPage.itemsCount} value={stats.rulesCount} iconColor="text-purple-500" />
          </div>
        )}

        <FilterBar
          namespaces={namespaces}
          filters={filters}
          onFilterChange={handleFilterChange}
          searchPlaceholder={t.common.search + " LimitRange..."}
        />

        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={filteredItems}
            loading={loading}
            error={error}
            keyExtractor={(item, index) => `${index}-${item.namespace}/${item.name}`}
            onRowClick={handleViewDetail}
            pageSize={10}
          />
        </div>
      </div>

      {selectedItem && (
        <LimitRangeDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespace={selectedItem.namespace}
          name={selectedItem.name}
        />
      )}
    </Layout>
  );
}
