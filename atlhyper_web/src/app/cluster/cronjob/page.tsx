"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getCronJobList } from "@/datasource/cluster";
import type { CronJobItem } from "@/api/cluster-resources";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, StatusBadge, FilterBar, type TableColumn } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import { Eye } from "lucide-react";
import { CronJobDetailModal } from "@/components/cronjob";

export default function CronJobPage() {
  const { t } = useI18n();
  const [items, setItems] = useState<CronJobItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // 筛选状态
  const [filters, setFilters] = useState({
    namespace: "",
    search: "",
  });

  // 详情弹窗状态
  const [selectedItem, setSelectedItem] = useState<CronJobItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getCronJobList({ cluster_id: getCurrentClusterId() });
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
    const totalActive = items.reduce((sum, d) => sum + d.activeJobs, 0);
    const totalSuspended = items.filter((d) => d.suspend).length;
    return {
      total: items.length,
      active: totalActive,
      suspended: totalSuspended,
      namespaces: namespaces.length,
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

  const handleViewDetail = (item: CronJobItem) => {
    setSelectedItem(item);
    setDetailOpen(true);
  };

  const columns: TableColumn<CronJobItem>[] = [
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
      key: "schedule",
      header: t.cronjob.schedule,
      render: (d) => <span className="font-mono text-xs">{d.schedule || "-"}</span>,
    },
    {
      key: "suspend",
      header: t.cronjob.suspend,
      mobileVisible: false,
      render: (d) => {
        const label = d.suspend ? t.cronjob.suspended : t.common.enabled;
        const type = d.suspend ? "error" : "success";
        return <StatusBadge status={label} type={type} />;
      },
    },
    {
      key: "activeJobs",
      header: t.cronjob.activeJobs,
      mobileVisible: false,
      render: (d) => String(d.activeJobs),
    },
    {
      key: "lastSchedule",
      header: t.cronjob.lastSchedule,
      mobileVisible: false,
      render: (d) => d.lastScheduleTime ? new Date(d.lastScheduleTime).toLocaleString() : "-",
    },
    {
      key: "age",
      header: t.cronjob.age,
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
          title={t.nav.cronjob}
          description={t.cronjob.pageDescription}
          autoRefreshSeconds={intervalSeconds}
        />

        {items.length > 0 && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={stats.total} />
            <StatsCard label={t.job.active} value={stats.active} iconColor="text-blue-500" />
            <StatsCard label={t.cronjob.suspended} value={stats.suspended} iconColor="text-orange-500" />
            <StatsCard label={t.common.namespace} value={stats.namespaces} iconColor="text-purple-500" />
          </div>
        )}

        <FilterBar
          namespaces={namespaces}
          filters={filters}
          onFilterChange={handleFilterChange}
          searchPlaceholder={t.cronjob.searchPlaceholder}
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
        <CronJobDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespace={selectedItem.namespace}
          name={selectedItem.name}
        />
      )}
    </Layout>
  );
}
