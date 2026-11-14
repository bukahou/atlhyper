"use client";

import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import type { JobItem } from "@/api/cluster-resources";
import { PageHeader, QueryView, StatsCard, DataTable, StatusBadge, type TableColumn } from "@/components/common";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { jobListQuery } from "@/queries/cluster-workload";
import { REFRESH } from "@/queries/refresh";
import { Eye } from "lucide-react";
import { JobDetailModal } from "@/components/job";
import { JobFilterBar } from "./components/JobFilterBar";

// 还没有数据时始终用同一个空数组：每次渲染换一个新数组会让下面的 useMemo 全部白算
const NO_ITEMS: JobItem[] = [];

export default function JobPage() {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  const query = useQuery(jobListQuery(clusterId));
  const items = query.data ?? NO_ITEMS;

  // Filter state
  const [filters, setFilters] = useState({
    namespace: "",
    search: "",
  });

  // Detail modal state
  const [selectedItem, setSelectedItem] = useState<JobItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  // Extract unique namespaces
  const namespaces = useMemo(() => {
    const nsSet = new Set<string>();
    items.forEach((d) => {
      if (d.namespace) nsSet.add(d.namespace);
    });
    return Array.from(nsSet).sort();
  }, [items]);

  // Stats cards
  const stats = useMemo(() => {
    const totalActive = items.reduce((sum, d) => sum + d.active, 0);
    const totalSucceeded = items.reduce((sum, d) => sum + d.succeeded, 0);
    const totalFailed = items.reduce((sum, d) => sum + d.failed, 0);
    return {
      total: items.length,
      active: totalActive,
      succeeded: totalSucceeded,
      failed: totalFailed,
    };
  }, [items]);

  // Filter items
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

  const handleViewDetail = (item: JobItem) => {
    setSelectedItem(item);
    setDetailOpen(true);
  };

  // Determine Job status
  const getJobStatus = (job: JobItem) => {
    if (job.complete) {
      return { label: t.job.statusComplete, type: "success" as const };
    }
    if (job.active > 0) {
      return { label: t.job.statusRunning, type: "warning" as const };
    }
    if (job.failed > 0) {
      return { label: t.job.statusFailed, type: "error" as const };
    }
    return { label: t.job.statusComplete, type: "success" as const };
  };

  const columns: TableColumn<JobItem>[] = [
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
      key: "status",
      header: t.common.status,
      render: (d) => {
        const { label, type } = getJobStatus(d);
        return <StatusBadge status={label} type={type} />;
      },
    },
    {
      key: "active",
      header: t.job.active,
      mobileVisible: false,
      render: (d) => String(d.active),
    },
    {
      key: "succeeded",
      header: t.job.succeeded,
      mobileVisible: false,
      render: (d) => String(d.succeeded),
    },
    {
      key: "failed",
      header: t.job.failed,
      mobileVisible: false,
      render: (d) => String(d.failed),
    },
    {
      key: "age",
      header: t.job.age,
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
          title={t.nav.job}
          description={t.job.pageDescription}
          autoRefreshSeconds={REFRESH.standard / 1000}
        />

        {items.length > 0 && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={stats.total} />
            <StatsCard label={t.job.active} value={stats.active} iconColor="text-blue-500" />
            <StatsCard label={t.job.succeeded} value={stats.succeeded} iconColor="text-green-500" />
            <StatsCard label={t.job.failed} value={stats.failed} iconColor="text-red-500" />
          </div>
        )}

        <JobFilterBar
          namespaces={namespaces}
          filters={filters}
          onFilterChange={handleFilterChange}
        />

        <QueryView query={query} requireCluster>
          {() => (
            <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
              <DataTable
                columns={columns}
                data={filteredItems}
                loading={false}
                error=""
                keyExtractor={(d, index) => `${index}-${d.namespace}/${d.name}`}
                onRowClick={handleViewDetail}
                pageSize={10}
              />
            </div>
          )}
        </QueryView>
      </div>

      {selectedItem && (
        <JobDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespace={selectedItem.namespace}
          name={selectedItem.name}
        />
      )}
    </Layout>
  );
}
