"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getServiceOverview } from "@/api/service";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, StatusBadge, type TableColumn } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import { Filter, X, Eye } from "lucide-react";
import type { ServiceItem, ServiceOverview } from "@/types/cluster";
import { ServiceDetailModal } from "@/components/service";

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
        className="w-full px-3 py-2 pr-8 bg-[var(--background)] border border-[var(--border-color)] rounded-lg text-sm text-default placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-primary"
      />
      {value && (
        <button
          onClick={onClear}
          className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted hover:text-default transition-colors"
        >
          <X className="w-3 h-3" />
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
        className="w-full px-3 py-2 pr-8 bg-[var(--background)] border border-[var(--border-color)] rounded-lg text-sm text-default focus:outline-none focus:ring-1 focus:ring-primary appearance-none"
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
          className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted hover:text-default transition-colors z-10"
        >
          <X className="w-3 h-3" />
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
  types,
  filters,
  onFilterChange,
}: {
  namespaces: string[];
  types: string[];
  filters: { namespace: string; type: string; search: string };
  onFilterChange: (key: string, value: string) => void;
}) {
  const { t } = useI18n();
  const hasFilters = filters.namespace || filters.type || filters.search;
  const activeCount = [filters.namespace, filters.type, filters.search].filter(Boolean).length;

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
              onFilterChange("type", "");
              onFilterChange("search", "");
            }}
            className="ml-auto flex items-center gap-1 text-xs text-muted hover:text-default transition-colors"
          >
            <X className="w-3 h-3" />
            {t.common.clearAll}
          </button>
        )}
      </div>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
        <FilterInput
          value={filters.search}
          onChange={(v) => onFilterChange("search", v)}
          onClear={() => onFilterChange("search", "")}
          placeholder={t.service.searchPlaceholder}
        />
        <FilterSelect
          value={filters.namespace}
          onChange={(v) => onFilterChange("namespace", v)}
          onClear={() => onFilterChange("namespace", "")}
          placeholder={t.service.allNamespaces}
          options={namespaces.map((ns) => ({ value: ns, label: ns }))}
        />
        <FilterSelect
          value={filters.type}
          onChange={(v) => onFilterChange("type", v)}
          onClear={() => onFilterChange("type", "")}
          placeholder={t.service.allTypes}
          options={types.map((tp) => ({ value: tp, label: tp }))}
        />
      </div>
    </div>
  );
}

export default function ServicePage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<ServiceOverview | null>(null);
  const [error, setError] = useState("");

  // 筛选状态
  const [filters, setFilters] = useState({
    namespace: "",
    type: "",
    search: "",
  });

  // 详情弹窗状态
  const [selectedService, setSelectedService] = useState<ServiceItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getServiceOverview({ ClusterID: getCurrentClusterId() });
      setData(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, []);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // 提取唯一的 namespaces 和 types
  const { namespaces, types } = useMemo(() => {
    const rows = data?.rows || [];
    const nsSet = new Set<string>();
    const typeSet = new Set<string>();

    rows.forEach((s) => {
      if (s.namespace) nsSet.add(s.namespace);
      if (s.type) typeSet.add(s.type);
    });

    return {
      namespaces: Array.from(nsSet).sort(),
      types: Array.from(typeSet).sort(),
    };
  }, [data?.rows]);

  // 根据筛选条件过滤数据
  const filteredServices = useMemo(() => {
    const rows = data?.rows || [];
    return rows.filter((s) => {
      // 搜索名称
      if (filters.search && !s.name.toLowerCase().includes(filters.search.toLowerCase())) {
        return false;
      }
      // Namespace 筛选
      if (filters.namespace && s.namespace !== filters.namespace) {
        return false;
      }
      // Type 筛选
      if (filters.type && s.type !== filters.type) {
        return false;
      }
      return true;
    });
  }, [data?.rows, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  // 查看详情
  const handleViewDetail = (service: ServiceItem) => {
    setSelectedService(service);
    setDetailOpen(true);
  };

  const getTypeStatus = (type: string): "success" | "info" | "default" => {
    if (type === "LoadBalancer") return "success";
    if (type === "NodePort") return "info";
    return "default";
  };

  const columns: TableColumn<ServiceItem>[] = [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (s) => <span className="font-medium text-default">{s.name || "-"}</span>,
    },
    { key: "namespace", header: t.common.namespace },
    {
      key: "type",
      header: t.service.serviceType,
      render: (s) => <StatusBadge status={s.type || "-"} type={getTypeStatus(s.type || "")} />,
    },
    {
      key: "clusterIP",
      header: t.service.clusterIP,
      mobileVisible: false,
      render: (s) => <span className="font-mono text-sm">{s.clusterIP || "-"}</span>,
    },
    {
      key: "ports",
      header: t.service.ports,
      render: (s) => <span className="text-sm">{s.ports || "-"}</span>,
    },
    {
      key: "selector",
      header: t.service.selector,
      mobileVisible: false,
      render: (s) => (
        <span className="text-xs text-muted truncate max-w-[150px] block" title={s.selector || ""}>
          {s.selector || "-"}
        </span>
      ),
    },
    {
      key: "action",
      header: t.common.action,
      mobileVisible: false,
      render: (s) => (
        <button
          onClick={(e) => {
            e.stopPropagation();
            handleViewDetail(s);
          }}
          className="p-2 hover-bg rounded-lg"
          title={t.service.viewDetails}
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
          title={t.nav.service}
          description={t.service.pageDescription}
          autoRefreshSeconds={intervalSeconds}
        />

        {data && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={data.cards.totalServices} />
            <StatsCard label="External" value={data.cards.externalServices} iconColor="text-green-500" />
            <StatsCard label="Internal" value={data.cards.internalServices} iconColor="text-blue-500" />
            <StatsCard label="Headless" value={data.cards.headlessServices} iconColor="text-gray-500" />
          </div>
        )}

        {/* 筛选栏 */}
        <FilterBar
          namespaces={namespaces}
          types={types}
          filters={filters}
          onFilterChange={handleFilterChange}
        />

        {/* 数据表格 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={filteredServices}
            loading={loading}
            error={error}
            keyExtractor={(s, index) => `${index}-${s.namespace}/${s.name}`}
            onRowClick={handleViewDetail}
            pageSize={10}
          />
        </div>
      </div>

      {/* Service 详情弹窗 */}
      {selectedService && (
        <ServiceDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespace={selectedService.namespace}
          serviceName={selectedService.name}
        />
      )}
    </Layout>
  );
}
