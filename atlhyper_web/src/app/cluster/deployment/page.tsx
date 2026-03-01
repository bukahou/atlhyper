"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getDeploymentOverview } from "@/datasource/cluster";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, StatusBadge, FilterBar, type TableColumn } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import { Eye } from "lucide-react";
import type { DeploymentItem, DeploymentOverview } from "@/types/cluster";
import { DeploymentDetailModal } from "@/components/deployment";

export default function DeploymentPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<DeploymentOverview | null>(null);
  const [error, setError] = useState("");

  // 筛选状态
  const [filters, setFilters] = useState({
    namespace: "",
    search: "",
  });

  // 详情弹窗状态
  const [selectedDeployment, setSelectedDeployment] = useState<DeploymentItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getDeploymentOverview({ ClusterID: getCurrentClusterId() });
      setData(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, []);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // 提取唯一的 namespaces
  const namespaces = useMemo(() => {
    const rows = data?.rows || [];
    const nsSet = new Set<string>();
    rows.forEach((d) => {
      if (d.namespace) nsSet.add(d.namespace);
    });
    return Array.from(nsSet).sort();
  }, [data?.rows]);

  // 根据筛选条件过滤数据
  const filteredDeployments = useMemo(() => {
    const rows = data?.rows || [];
    return rows.filter((d) => {
      // 搜索名称
      if (filters.search && !d.name.toLowerCase().includes(filters.search.toLowerCase())) {
        return false;
      }
      // Namespace 筛选
      if (filters.namespace && d.namespace !== filters.namespace) {
        return false;
      }
      return true;
    });
  }, [data?.rows, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  // 查看详情
  const handleViewDetail = (deployment: DeploymentItem) => {
    setSelectedDeployment(deployment);
    setDetailOpen(true);
  };

  const columns: TableColumn<DeploymentItem>[] = [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (d) => (
        <div>
          <span className="font-medium text-default">{d.name || "-"}</span>
          <div className="text-xs text-muted truncate max-w-[200px]" title={d.image || ""}>{d.image || "-"}</div>
        </div>
      ),
    },
    { key: "namespace", header: t.common.namespace },
    {
      key: "replicas",
      header: t.deployment.replicas,
      render: (d) => {
        if (!d.replicas) return <StatusBadge status="-" type="default" />;
        const parts = d.replicas.split("/");
        const ready = parseInt(parts[0], 10) || 0;
        const total = parseInt(parts[1], 10) || 0;
        const type = ready === total ? "success" : ready === 0 ? "error" : "warning";
        return <StatusBadge status={d.replicas} type={type} />;
      },
    },
    {
      key: "createdAt",
      header: t.common.createdAt,
      mobileVisible: false,
      render: (d) => d.createdAt ? new Date(d.createdAt).toLocaleString() : "-",
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
          title={t.deployment.viewDetails}
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
          title={t.nav.deployment}
          description={t.deployment.pageDescription}
          autoRefreshSeconds={intervalSeconds}
        />

        {data && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={data.cards.totalDeployments} />
            <StatsCard label={t.common.namespace} value={data.cards.namespaces} iconColor="text-blue-500" />
            <StatsCard label={t.deployment.replicas} value={data.cards.totalReplicas} iconColor="text-purple-500" />
            <StatsCard label={t.deployment.readyReplicas} value={data.cards.readyReplicas} iconColor="text-green-500" />
          </div>
        )}

        {/* 筛选栏 */}
        <FilterBar
          namespaces={namespaces}
          filters={filters}
          onFilterChange={handleFilterChange}
          searchPlaceholder={t.deployment.searchPlaceholder}
        />

        {/* 数据表格 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={filteredDeployments}
            loading={loading}
            error={error}
            keyExtractor={(d, index) => `${index}-${d.namespace}/${d.name}`}
            onRowClick={handleViewDetail}
            pageSize={10}
          />
        </div>
      </div>

      {/* Deployment 详情弹窗 */}
      {selectedDeployment && (
        <DeploymentDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespace={selectedDeployment.namespace}
          deploymentName={selectedDeployment.name}
          onUpdated={fetchData}
        />
      )}
    </Layout>
  );
}
