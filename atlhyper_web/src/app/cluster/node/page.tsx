"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getNodeOverview } from "@/datasource/cluster";
import { PageHeader, StatsCard, DataTable, StatusBadge, type TableColumn } from "@/components/common";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { getCurrentClusterId } from "@/config/cluster";
import { Filter, X } from "lucide-react";
import type { NodeItem, NodeOverview } from "@/types/cluster";
import { NodeDetailModal } from "@/components/node";

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

export default function NodePage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<NodeOverview | null>(null);
  const [error, setError] = useState("");

  // 筛选状态
  const [filters, setFilters] = useState({
    search: "",
    status: "",
    architecture: "",
    schedulable: "",
  });

  // 详情弹窗状态
  const [selectedNode, setSelectedNode] = useState<NodeItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getNodeOverview({ ClusterID: getCurrentClusterId() });
      setData(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [t.common.loadFailed]);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // 查看详情
  const handleRowClick = (node: NodeItem) => {
    setSelectedNode(node);
    setDetailOpen(true);
  };

  // 动态提取架构选项
  const architectureOptions = useMemo(() => {
    const rows = data?.rows || [];
    const archSet = new Set<string>();
    rows.forEach((node) => {
      if (node.architecture) archSet.add(node.architecture);
    });
    return Array.from(archSet).sort().map((a) => ({ value: a, label: a }));
  }, [data?.rows]);

  // 客户端筛选
  const filteredNodes = useMemo(() => {
    const rows = data?.rows || [];
    return rows.filter((node) => {
      if (filters.search && !node.name.toLowerCase().includes(filters.search.toLowerCase())) {
        return false;
      }
      if (filters.status) {
        const isReady = node.ready ? "Ready" : "NotReady";
        if (isReady !== filters.status) return false;
      }
      if (filters.architecture && node.architecture !== filters.architecture) {
        return false;
      }
      if (filters.schedulable) {
        const sched = node.schedulable ? "Schedulable" : "Unschedulable";
        if (sched !== filters.schedulable) return false;
      }
      return true;
    });
  }, [data?.rows, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const hasFilters = filters.search || filters.status || filters.architecture || filters.schedulable;
  const activeCount = [filters.search, filters.status, filters.architecture, filters.schedulable].filter(Boolean).length;

  const columns: TableColumn<NodeItem>[] = [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (node) => <span className="font-medium text-default">{node.name}</span>,
    },
    {
      key: "status",
      header: t.common.status,
      render: (node) => (
        <div className="flex items-center gap-1.5">
          <StatusBadge status={node.ready ? "Ready" : "NotReady"} />
          <StatusBadge
            status={node.schedulable !== false ? "Schedulable" : "Unschedulable"}
            type={node.schedulable !== false ? "success" : "warning"}
          />
        </div>
      ),
    },
    {
      key: "architecture",
      header: t.node.architecture,
      mobileVisible: false,
      render: (node) => node.architecture ? <StatusBadge status={node.architecture} type="info" /> : <span className="text-muted">-</span>,
    },
    {
      key: "cpu",
      header: "CPU",
      render: (node) => <span className="text-sm">{node.cpuCores != null ? `${node.cpuCores} cores` : "-"}</span>,
    },
    {
      key: "memory",
      header: "Memory",
      render: (node) => <span className="text-sm">{node.memoryGiB != null ? `${node.memoryGiB.toFixed(1)} GiB` : "-"}</span>,
    },
    {
      key: "ip",
      header: "IP",
      mobileVisible: false,
      render: (node) => <span className="font-mono text-xs">{node.internalIP || "-"}</span>,
    },
    {
      key: "os",
      header: "OS",
      mobileVisible: false,
      render: (node) => <span className="truncate max-w-[180px] block text-sm" title={node.osImage}>{node.osImage || "-"}</span>,
    },
  ];

  return (
    <Layout>
      <div className="space-y-4">
        <PageHeader title={t.nav.node} description={t.node.pageDescription} autoRefreshSeconds={intervalSeconds} />

        {data && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={data.cards.totalNodes ?? 0} />
            <StatsCard label={t.status.ready} value={data.cards.readyNodes ?? 0} iconColor="text-green-500" />
            <StatsCard label="Total CPU" value={data.cards.totalCPU ?? 0} iconColor="text-blue-500" />
            <StatsCard label="Total Memory" value={data.cards.totalMemoryGiB != null ? `${data.cards.totalMemoryGiB.toFixed(1)} GiB` : "-"} iconColor="text-purple-500" />
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
                onClick={() => setFilters({ search: "", status: "", architecture: "", schedulable: "" })}
                className="ml-auto flex items-center gap-1 text-xs text-muted hover:text-default transition-colors"
              >
                <X className="w-3 h-3" />
                {t.common.clearAll}
              </button>
            )}
          </div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <FilterInput
              value={filters.search}
              onChange={(v) => handleFilterChange("search", v)}
              onClear={() => handleFilterChange("search", "")}
              placeholder={t.node.searchPlaceholder}
            />
            <FilterSelect
              value={filters.status}
              onChange={(v) => handleFilterChange("status", v)}
              onClear={() => handleFilterChange("status", "")}
              placeholder={t.node.allStatus}
              options={[
                { value: "Ready", label: "Ready" },
                { value: "NotReady", label: "NotReady" },
              ]}
            />
            <FilterSelect
              value={filters.architecture}
              onChange={(v) => handleFilterChange("architecture", v)}
              onClear={() => handleFilterChange("architecture", "")}
              placeholder={t.node.allArchitectures}
              options={architectureOptions}
            />
            <FilterSelect
              value={filters.schedulable}
              onChange={(v) => handleFilterChange("schedulable", v)}
              onClear={() => handleFilterChange("schedulable", "")}
              placeholder={t.node.allSchedulable}
              options={[
                { value: "Schedulable", label: t.node.schedulable },
                { value: "Unschedulable", label: t.node.unschedulable },
              ]}
            />
          </div>
        </div>

        {/* 数据表格 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={filteredNodes}
            loading={loading}
            error={error}
            keyExtractor={(node) => node.name}
            onRowClick={handleRowClick}
            pageSize={10}
          />
        </div>
      </div>

      {/* Node 详情弹窗 */}
      {selectedNode && (
        <NodeDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          nodeName={selectedNode.name}
          onNodeChanged={fetchData}
        />
      )}
    </Layout>
  );
}
