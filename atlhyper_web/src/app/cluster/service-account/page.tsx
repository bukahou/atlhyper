"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getServiceAccountList, type ServiceAccountItem } from "@/api/cluster-resources";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, StatusBadge, type TableColumn } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
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
          placeholder={t.common.search + " ServiceAccount..."}
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

export default function ServiceAccountPage() {
  const { t } = useI18n();
  const [items, setItems] = useState<ServiceAccountItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // 筛选状态
  const [filters, setFilters] = useState({
    namespace: "",
    search: "",
  });

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getServiceAccountList({ cluster_id: getCurrentClusterId() });
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
    const withSecrets = items.filter((item) => item.secretsCount > 0).length;
    return {
      total: items.length,
      namespaces: namespaces.length,
      withSecrets,
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

  const columns: TableColumn<ServiceAccountItem>[] = [
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
      key: "secretsCount",
      header: t.policyPage.secrets,
      render: (item) => String(item.secretsCount),
    },
    {
      key: "imagePullSecretsCount",
      header: t.policyPage.imagePullSecrets,
      mobileVisible: false,
      render: (item) => String(item.imagePullSecretsCount),
    },
    {
      key: "automountToken",
      header: t.policyPage.automountToken,
      mobileVisible: false,
      render: (item) => (
        <StatusBadge
          status={item.automountServiceAccountToken ? t.common.yes : t.common.no}
          type={item.automountServiceAccountToken ? "success" : "default"}
        />
      ),
    },
    {
      key: "age",
      header: t.policyPage.age,
      mobileVisible: false,
      render: (item) => item.age || "-",
    },
  ];

  return (
    <Layout>
      <div className="space-y-4">
        <PageHeader
          title={t.nav.serviceAccount}
          description={t.policyPage.serviceAccountDescription}
          autoRefreshSeconds={intervalSeconds}
        />

        {items.length > 0 && (
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <StatsCard label={t.common.total} value={stats.total} />
            <StatsCard label={t.common.namespace} value={stats.namespaces} iconColor="text-blue-500" />
            <StatsCard label={t.policyPage.withSecrets} value={stats.withSecrets} iconColor="text-purple-500" />
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
            keyExtractor={(item, index) => `${index}-${item.namespace}/${item.name}`}
            pageSize={10}
          />
        </div>
      </div>
    </Layout>
  );
}
