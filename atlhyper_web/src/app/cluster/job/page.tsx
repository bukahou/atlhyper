"use client";

import { useState, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getJobList, type JobItem } from "@/api/cluster-resources";
import { PageHeader, StatsCard, DataTable, StatusBadge, type TableColumn } from "@/components/common";
import { Filter, X } from "lucide-react";

// 筛选输入框
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

// 筛选下拉框
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

// 筛选栏
function FilterBar({
  namespaces,
  filters,
  onFilterChange,
}: {
  namespaces: string[];
  filters: { namespace: string; search: string };
  onFilterChange: (key: string, value: string) => void;
}) {
  const { t } = useI18n();
  const hasFilters = filters.namespace || filters.search;
  const activeCount = [filters.namespace, filters.search].filter(Boolean).length;

  return (
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
            onClick={() => {
              onFilterChange("namespace", "");
              onFilterChange("search", "");
            }}
            className="ml-auto flex items-center gap-1 text-xs text-muted hover:text-default transition-colors"
          >
            <X className="w-3 h-3" />
            {t.common.clearAll}
          </button>
        )}
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <FilterInput
          value={filters.search}
          onChange={(v) => onFilterChange("search", v)}
          onClear={() => onFilterChange("search", "")}
          placeholder={t.job.searchPlaceholder}
        />
        <FilterSelect
          value={filters.namespace}
          onChange={(v) => onFilterChange("namespace", v)}
          onClear={() => onFilterChange("namespace", "")}
          placeholder={t.deployment.allNamespaces}
          options={namespaces.map((ns) => ({ value: ns, label: ns }))}
        />
      </div>
    </div>
  );
}

export default function JobPage() {
  const { t } = useI18n();
  const [items] = useState<JobItem[]>(() => getJobList());
  const loading = false;
  const error = "";

  // 筛选状态
  const [filters, setFilters] = useState({
    namespace: "",
    search: "",
  });

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

  // 确定 Job 状态
  const getJobStatus = (job: JobItem) => {
    if (job.succeeded > 0 && job.active === 0) {
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
      key: "duration",
      header: t.job.duration,
      mobileVisible: false,
      render: (d) => d.duration || "-",
    },
    {
      key: "age",
      header: t.job.age,
      mobileVisible: false,
      render: (d) => d.age || "-",
    },
  ];

  return (
    <Layout>
      <div className="space-y-4">
        <PageHeader
          title={t.nav.job}
          description={t.job.pageDescription}
        />

        {items.length > 0 && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={stats.total} />
            <StatsCard label={t.job.active} value={stats.active} iconColor="text-blue-500" />
            <StatsCard label={t.job.succeeded} value={stats.succeeded} iconColor="text-green-500" />
            <StatsCard label={t.job.failed} value={stats.failed} iconColor="text-red-500" />
          </div>
        )}

        <FilterBar
          namespaces={namespaces}
          filters={filters}
          onFilterChange={handleFilterChange}
        />

        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={filteredItems}
            loading={loading}
            error={error}
            keyExtractor={(d, index) => `${index}-${d.namespace}/${d.name}`}
            pageSize={10}
          />
        </div>
      </div>
    </Layout>
  );
}
