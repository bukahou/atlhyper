"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getNamespaceOverview } from "@/datasource/cluster";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, StatusBadge, type TableColumn } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import { Filter, X } from "lucide-react";
import type { NamespaceOverview, NamespaceItem } from "@/types/cluster";
import { NamespaceDetailModal } from "@/components/namespace";

// 带清除按钮的筛选输入框
function FilterInput({
  value,
  onChange,
  onClear,
  placeholder,
}: {
  value: string;
  onChange: (value: string) => void;
  onClear: () => void;
  placeholder: string;
}) {
  return (
    <div className="relative">
      <input
        type="text"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2.5 sm:py-2 pr-8 bg-[var(--background)] border border-[var(--border-color)] rounded-lg text-sm text-default placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-primary"
      />
      {value && (
        <button
          onClick={onClear}
          className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 text-muted hover:text-default transition-colors"
        >
          <X className="w-4 h-4 sm:w-3 sm:h-3" />
        </button>
      )}
    </div>
  );
}

// 带清除按钮的筛选下拉框
function FilterSelect({
  value,
  onChange,
  onClear,
  placeholder,
  options,
}: {
  value: string;
  onChange: (value: string) => void;
  onClear: () => void;
  placeholder: string;
  options: { value: string; label: string }[];
}) {
  return (
    <div className="relative">
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2.5 sm:py-2 pr-8 bg-[var(--background)] border border-[var(--border-color)] rounded-lg text-sm text-default focus:outline-none focus:ring-1 focus:ring-primary appearance-none"
      >
        <option value="">{placeholder}</option>
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      {value ? (
        <button
          onClick={(e) => {
            e.preventDefault();
            onClear();
          }}
          className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 text-muted hover:text-default transition-colors z-10"
        >
          <X className="w-4 h-4 sm:w-3 sm:h-3" />
        </button>
      ) : (
        <div className="absolute right-2 top-1/2 -translate-y-1/2 pointer-events-none">
          <svg className="w-4 h-4 text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      )}
    </div>
  );
}

export default function NamespacePage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<NamespaceOverview | null>(null);
  const [error, setError] = useState("");

  // 筛选状态
  const [filters, setFilters] = useState({
    search: "",
    status: "",
  });

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
  }, [t.common.loadFailed]);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  const handleRowClick = (ns: NamespaceItem) => {
    setSelectedNamespace(ns);
    setDetailOpen(true);
  };

  // 客户端筛选
  const filteredNamespaces = useMemo(() => {
    const rows = data?.rows || [];
    return rows.filter((ns) => {
      if (filters.search && !ns.name.toLowerCase().includes(filters.search.toLowerCase())) {
        return false;
      }
      if (filters.status && ns.status !== filters.status) {
        return false;
      }
      return true;
    });
  }, [data?.rows, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const hasFilters = filters.search || filters.status;
  const activeCount = [filters.search, filters.status].filter(Boolean).length;

  const columns: TableColumn<NamespaceItem>[] = [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (ns) => <span className="font-medium text-default">{ns.name}</span>,
    },
    {
      key: "status",
      header: t.common.status,
      render: (ns) => <StatusBadge status={ns.status || "Unknown"} />,
    },
    {
      key: "podCount",
      header: t.namespace.pods,
      render: (ns) => <span className="text-sm">{ns.podCount ?? 0}</span>,
    },
    {
      key: "labels",
      header: t.namespace.labels,
      mobileVisible: false,
      render: (ns) => <span className="text-sm text-muted">{ns.labelCount ?? 0}</span>,
    },
    {
      key: "annotations",
      header: t.namespace.annotations,
      mobileVisible: false,
      render: (ns) => <span className="text-sm text-muted">{ns.annotationCount ?? 0}</span>,
    },
    {
      key: "createdAt",
      header: t.common.createdAt,
      mobileVisible: false,
      render: (ns) => (
        <span className="text-sm text-muted">
          {ns.createdAt ? new Date(ns.createdAt).toLocaleDateString() : "-"}
        </span>
      ),
    },
  ];

  return (
    <Layout>
      <div className="space-y-4">
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

        {/* 筛选栏 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
          <div className="flex items-center gap-2 mb-3">
            <Filter className="w-4 h-4 text-muted" />
            <span className="text-sm font-medium text-default">{t.common.filter}</span>
            {activeCount > 0 && (
              <span className="px-1.5 py-0.5 text-xs bg-primary/10 text-primary rounded">
                {activeCount}
              </span>
            )}
            {hasFilters && (
              <button
                onClick={() => setFilters({ search: "", status: "" })}
                className="ml-auto flex items-center gap-1 text-xs text-muted hover:text-default transition-colors"
              >
                <X className="w-3 h-3" />
                {t.common.clearAll}
              </button>
            )}
          </div>
          <div className="grid grid-cols-2 gap-3">
            <FilterInput
              value={filters.search}
              onChange={(v) => handleFilterChange("search", v)}
              onClear={() => handleFilterChange("search", "")}
              placeholder={t.namespace.searchPlaceholder}
            />
            <FilterSelect
              value={filters.status}
              onChange={(v) => handleFilterChange("status", v)}
              onClear={() => handleFilterChange("status", "")}
              placeholder={t.namespace.allStatus}
              options={[
                { value: "Active", label: t.status.active },
                { value: "Terminating", label: t.status.terminated },
              ]}
            />
          </div>
        </div>

        {/* 数据表格 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={filteredNamespaces}
            loading={loading}
            error={error}
            keyExtractor={(ns) => ns.name}
            onRowClick={handleRowClick}
            pageSize={10}
          />
        </div>
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
