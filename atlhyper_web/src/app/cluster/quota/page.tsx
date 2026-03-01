"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getResourceQuotaList } from "@/datasource/cluster";
import type { ResourceQuotaItem } from "@/api/cluster-resources";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, FilterBar, type TableColumn } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import { Eye } from "lucide-react";
import { ResourceQuotaDetailModal } from "@/components/resource-quota";

// 格式化 Hard/Used 列
function formatHardUsed(item: ResourceQuotaItem): string {
  const keyResources = ["requests.cpu", "requests.memory", "limits.cpu", "limits.memory", "pods"];
  const parts: string[] = [];

  keyResources.forEach((key) => {
    const hard = item.hard?.[key];
    const used = item.used?.[key];
    if (hard || used) {
      const displayKey = key.replace("requests.", "").replace("limits.", "");
      parts.push(`${displayKey}: ${used || "0"}/${hard || "-"}`);
    }
  });

  return parts.join(", ") || "-";
}

export default function ResourceQuotaPage() {
  const { t } = useI18n();
  const [items, setItems] = useState<ResourceQuotaItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // 筛选状态
  const [filters, setFilters] = useState({
    namespace: "",
    search: "",
  });

  // 详情弹窗状态
  const [selectedItem, setSelectedItem] = useState<ResourceQuotaItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getResourceQuotaList({ cluster_id: getCurrentClusterId() });
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
    const resourceTypesSet = new Set<string>();
    items.forEach((item) => {
      if (item.hard) {
        Object.keys(item.hard).forEach((key) => resourceTypesSet.add(key));
      }
    });
    return {
      total: items.length,
      namespaces: namespaces.length,
      resourceTypes: resourceTypesSet.size,
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

  const handleViewDetail = (item: ResourceQuotaItem) => {
    setSelectedItem(item);
    setDetailOpen(true);
  };

  const columns: TableColumn<ResourceQuotaItem>[] = [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (d) => (
        <div>
          <span className="font-medium text-default">{d.name || "-"}</span>
          <div className="text-xs text-muted">{d.age || "-"}</div>
        </div>
      ),
    },
    { key: "namespace", header: t.common.namespace },
    {
      key: "hardUsed",
      header: t.policyPage.hardUsed,
      render: (d) => (
        <div className="text-xs">
          {formatHardUsed(d)}
        </div>
      ),
    },
    {
      key: "age",
      header: t.policyPage.age,
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
          title={t.nav.resourceQuota}
          description={t.policyPage.resourceQuotaDescription}
          autoRefreshSeconds={intervalSeconds}
        />

        {items.length > 0 && (
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <StatsCard label={t.common.total} value={stats.total} />
            <StatsCard label={t.common.namespace} value={stats.namespaces} iconColor="text-blue-500" />
            <StatsCard label={t.policyPage.resourceTypes} value={stats.resourceTypes} iconColor="text-purple-500" />
          </div>
        )}

        <FilterBar
          namespaces={namespaces}
          filters={filters}
          onFilterChange={handleFilterChange}
          searchPlaceholder={t.common.search + " ResourceQuota..."}
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
        <ResourceQuotaDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespace={selectedItem.namespace}
          name={selectedItem.name}
        />
      )}
    </Layout>
  );
}
