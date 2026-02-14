"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getPVList, type PVItem } from "@/api/cluster-resources";
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

// 筛选栏
function FilterBar({
  filters,
  onFilterChange,
}: {
  filters: { search: string };
  onFilterChange: (key: string, value: string) => void;
}) {
  const { t } = useI18n();
  const hasFilters = filters.search;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
      <div className="flex items-center gap-2 mb-3">
        <Filter className="w-4 h-4 text-muted" />
        <span className="text-sm font-medium text-default">{t.common.filter}</span>
        {hasFilters && (
          <>
            <span className="px-1.5 py-0.5 text-xs bg-primary/10 text-primary rounded">1</span>
            <button
              onClick={() => onFilterChange("search", "")}
              className="ml-auto flex items-center gap-1 text-xs text-muted hover:text-default transition-colors"
            >
              <X className="w-3 h-3" />
              {t.common.clearAll}
            </button>
          </>
        )}
      </div>
      <div className="grid grid-cols-1 gap-3">
        <FilterInput
          value={filters.search}
          onChange={(v) => onFilterChange("search", v)}
          onClear={() => onFilterChange("search", "")}
          placeholder={t.common.search + " PV..."}
        />
      </div>
    </div>
  );
}

export default function PVPage() {
  const { t } = useI18n();
  const [items, setItems] = useState<PVItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // 筛选状态
  const [filters, setFilters] = useState({
    search: "",
  });

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getPVList({ cluster_id: getCurrentClusterId() });
      setItems(res.data.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, []);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // StatsCards 统计
  const stats = useMemo(() => {
    const bound = items.filter((pv) => pv.phase === "Bound").length;
    const available = items.filter((pv) => pv.phase === "Available").length;
    const other = items.length - bound - available;
    return {
      total: items.length,
      bound,
      available,
      other,
    };
  }, [items]);

  // 根据筛选条件过滤数据
  const filteredItems = useMemo(() => {
    return items.filter((pv) => {
      if (filters.search && !pv.name.toLowerCase().includes(filters.search.toLowerCase())) {
        return false;
      }
      return true;
    });
  }, [items, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const columns: TableColumn<PVItem>[] = [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (pv) => (
        <div>
          <span className="font-medium text-default">{pv.name || "-"}</span>
          <div className="text-xs text-muted">{pv.age || "-"}</div>
        </div>
      ),
    },
    {
      key: "capacity",
      header: t.storagePage.capacity,
      render: (pv) => pv.capacity || "-",
    },
    {
      key: "phase",
      header: t.storagePage.phase,
      render: (pv) => {
        const typeMap: Record<string, "success" | "info" | "warning" | "error"> = {
          Bound: "success",
          Available: "info",
          Released: "warning",
          Failed: "error",
        };
        return <StatusBadge status={pv.phase} type={typeMap[pv.phase] || "info"} />;
      },
    },
    {
      key: "storageClass",
      header: t.storagePage.storageClass,
      mobileVisible: false,
      render: (pv) => pv.storageClass || "-",
    },
    {
      key: "reclaimPolicy",
      header: t.storagePage.reclaimPolicy,
      mobileVisible: false,
      render: (pv) => pv.reclaimPolicy || "-",
    },
    {
      key: "accessModes",
      header: t.storagePage.accessModes,
      mobileVisible: false,
      render: (pv) => (pv.accessModes && pv.accessModes.length > 0 ? pv.accessModes.join(", ") : "-"),
    },
    {
      key: "age",
      header: t.storagePage.age,
      mobileVisible: false,
      render: (pv) => pv.age || "-",
    },
  ];

  return (
    <Layout>
      <div className="space-y-4">
        <PageHeader title={t.nav.pv} description={t.storagePage.pvDescription} autoRefreshSeconds={intervalSeconds} />

        {items.length > 0 && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={stats.total} />
            <StatsCard label={t.storagePage.bound} value={stats.bound} iconColor="text-green-500" />
            <StatsCard label={t.storagePage.available} value={stats.available} iconColor="text-blue-500" />
            <StatsCard
              label={t.storagePage.released + "/" + t.common.error}
              value={stats.other}
              iconColor="text-yellow-500"
            />
          </div>
        )}

        <FilterBar filters={filters} onFilterChange={handleFilterChange} />

        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={filteredItems}
            loading={loading}
            error={error}
            keyExtractor={(pv, index) => `${index}-${pv.name}`}
            pageSize={10}
          />
        </div>
      </div>
    </Layout>
  );
}
