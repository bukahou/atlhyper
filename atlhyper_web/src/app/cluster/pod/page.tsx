"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getPodOverview, restartPod } from "@/api/pod";
import { PageHeader, StatsCard, DataTable, StatusBadge, ConfirmDialog, type TableColumn } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { RotateCcw, Filter, X, Eye } from "lucide-react";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import type { PodItem, PodOverview } from "@/types/cluster";
import { PodDetailModal, PodLogsViewer } from "@/components/pod";

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

// 筛选栏组件
function FilterBar({
  namespaces,
  nodes,
  filters,
  onFilterChange,
}: {
  namespaces: string[];
  nodes: string[];
  filters: { namespace: string; node: string; status: string; search: string };
  onFilterChange: (key: string, value: string) => void;
}) {
  const hasFilters = filters.namespace || filters.node || filters.status || filters.search;
  const activeCount = [filters.namespace, filters.node, filters.status, filters.search].filter(Boolean).length;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
      <div className="flex items-center gap-2 mb-3">
        <Filter className="w-4 h-4 text-muted" />
        <span className="text-sm font-medium text-default">筛选</span>
        {activeCount > 0 && (
          <span className="px-1.5 py-0.5 text-xs bg-primary/10 text-primary rounded">
            {activeCount}
          </span>
        )}
        {hasFilters && (
          <button
            onClick={() => {
              onFilterChange("namespace", "");
              onFilterChange("node", "");
              onFilterChange("status", "");
              onFilterChange("search", "");
            }}
            className="ml-auto flex items-center gap-1 text-xs text-muted hover:text-default transition-colors"
          >
            <X className="w-3 h-3" />
            清除全部
          </button>
        )}
      </div>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {/* 搜索框 */}
        <FilterInput
          value={filters.search}
          onChange={(v) => onFilterChange("search", v)}
          onClear={() => onFilterChange("search", "")}
          placeholder="搜索 Pod 名称..."
        />

        {/* Namespace 筛选 */}
        <FilterSelect
          value={filters.namespace}
          onChange={(v) => onFilterChange("namespace", v)}
          onClear={() => onFilterChange("namespace", "")}
          placeholder="全部 Namespace"
          options={namespaces.map((ns) => ({ value: ns, label: ns }))}
        />

        {/* Node 筛选 */}
        <FilterSelect
          value={filters.node}
          onChange={(v) => onFilterChange("node", v)}
          onClear={() => onFilterChange("node", "")}
          placeholder="全部 Node"
          options={nodes.map((node) => ({ value: node, label: node }))}
        />

        {/* Status 筛选 */}
        <FilterSelect
          value={filters.status}
          onChange={(v) => onFilterChange("status", v)}
          onClear={() => onFilterChange("status", "")}
          placeholder="全部状态"
          options={[
            { value: "Running", label: "Running" },
            { value: "Pending", label: "Pending" },
            { value: "Failed", label: "Failed" },
            { value: "Succeeded", label: "Succeeded" },
            { value: "Unknown", label: "Unknown" },
          ]}
        />
      </div>
    </div>
  );
}

export default function PodPage() {
  const { t } = useI18n();
  const requireAuth = useRequireAuth();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<PodOverview | null>(null);
  const [error, setError] = useState("");

  // 筛选状态
  const [filters, setFilters] = useState({
    namespace: "",
    node: "",
    status: "",
    search: "",
  });

  // 详情弹窗状态
  const [selectedPod, setSelectedPod] = useState<PodItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  // 日志查看器状态
  const [logsOpen, setLogsOpen] = useState(false);
  const [logsContainer, setLogsContainer] = useState("");

  // 重启确认状态
  const [restartTarget, setRestartTarget] = useState<PodItem | null>(null);
  const [restartLoading, setRestartLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getPodOverview({ ClusterID: getCurrentClusterId() });
      setData(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, []);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // 打开详情
  const handleRowClick = (pod: PodItem) => {
    setSelectedPod(pod);
    setDetailOpen(true);
  };

  // 从详情页打开日志（需要先登录）
  const handleViewLogs = (containerName: string) => {
    requireAuth(() => {
      setLogsContainer(containerName);
      setLogsOpen(true);
    });
  };

  // 显示重启确认（需要先登录）
  const handleRestartClick = (pod: PodItem) => {
    requireAuth(() => setRestartTarget(pod));
  };

  // 确认重启
  const handleRestartConfirm = async () => {
    if (!restartTarget) return;
    setRestartLoading(true);
    try {
      await restartPod({
        ClusterID: getCurrentClusterId(),
        Namespace: restartTarget.namespace,
        Pod: restartTarget.name,
      });
      setRestartTarget(null);
      // 延迟2秒后刷新，给后端处理时间
      setTimeout(() => fetchData(), 2000);
    } catch (err) {
      console.error("Restart failed:", err);
    } finally {
      setRestartLoading(false);
    }
  };

  // 提取唯一的 namespaces 和 nodes
  const { namespaces, nodes } = useMemo(() => {
    const pods = data?.pods || [];
    const nsSet = new Set<string>();
    const nodeSet = new Set<string>();

    pods.forEach((pod) => {
      if (pod.namespace) nsSet.add(pod.namespace);
      if (pod.node) nodeSet.add(pod.node);
    });

    return {
      namespaces: Array.from(nsSet).sort(),
      nodes: Array.from(nodeSet).sort(),
    };
  }, [data?.pods]);

  // 根据筛选条件过滤数据
  const filteredPods = useMemo(() => {
    const pods = data?.pods || [];

    return pods.filter((pod) => {
      // 搜索名称
      if (filters.search && !pod.name.toLowerCase().includes(filters.search.toLowerCase())) {
        return false;
      }
      // Namespace 筛选
      if (filters.namespace && pod.namespace !== filters.namespace) {
        return false;
      }
      // Node 筛选
      if (filters.node && pod.node !== filters.node) {
        return false;
      }
      // Status 筛选
      if (filters.status && pod.phase !== filters.status) {
        return false;
      }
      return true;
    });
  }, [data?.pods, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const columns: TableColumn<PodItem>[] = [
    {
      key: "name",
      header: t.common.name,
      render: (pod) => (
        <div>
          <span className="font-medium text-default">{pod.name}</span>
          <div className="text-xs text-muted">{pod.deployment}</div>
        </div>
      ),
    },
    { key: "namespace", header: t.common.namespace },
    {
      key: "phase",
      header: t.common.status,
      render: (pod) => <StatusBadge status={pod.phase} />,
    },
    {
      key: "ready",
      header: "Ready",
      render: (pod) => <span className="font-mono text-sm">{pod.ready}</span>,
    },
    { key: "node", header: "Node" },
    {
      key: "cpu",
      header: "CPU",
      render: (pod) => (
        <div className="text-sm">
          <span>{pod.cpuText}</span>
          <span className="text-muted ml-1">({pod.cpuPercentText})</span>
        </div>
      ),
    },
    {
      key: "memory",
      header: "Memory",
      render: (pod) => (
        <div className="text-sm">
          <span>{pod.memoryText}</span>
          <span className="text-muted ml-1">({pod.memPercentText})</span>
        </div>
      ),
    },
    { key: "restarts", header: "Restarts" },
    {
      key: "action",
      header: t.common.action,
      render: (pod) => (
        <div className="flex items-center gap-1">
          <button
            onClick={(e) => {
              e.stopPropagation();
              handleRowClick(pod);
            }}
            className="p-2 hover-bg rounded-lg"
            title="查看详情"
          >
            <Eye className="w-4 h-4 text-muted" />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              handleRestartClick(pod);
            }}
            className="p-2 hover-bg rounded-lg"
            title="重启 Pod"
          >
            <RotateCcw className="w-4 h-4 text-muted" />
          </button>
        </div>
      ),
    },
  ];

  const totalPods = data ? data.cards.running + data.cards.pending + data.cards.failed + data.cards.unknown : 0;

  return (
    <Layout>
      <div className="space-y-4">
        <PageHeader
          title={t.nav.pod}
          description="Pod 资源监控与管理"
          autoRefreshSeconds={intervalSeconds}
        />

        {data && (
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            <StatsCard label={t.common.total} value={totalPods} />
            <StatsCard label={t.status.running} value={data.cards.running} iconColor="text-green-500" />
            <StatsCard label={t.status.pending} value={data.cards.pending} iconColor="text-yellow-500" />
            <StatsCard label={t.status.failed} value={data.cards.failed} iconColor="text-red-500" />
            <StatsCard label="Unknown" value={data.cards.unknown} iconColor="text-gray-500" />
          </div>
        )}

        {/* 筛选栏 */}
        <FilterBar
          namespaces={namespaces}
          nodes={nodes}
          filters={filters}
          onFilterChange={handleFilterChange}
        />

        {/* 数据表格 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={filteredPods}
            loading={loading}
            error={error}
            keyExtractor={(pod, index) => `${index}-${pod.namespace}/${pod.name}`}
            onRowClick={handleRowClick}
            pageSize={10}
          />
        </div>
      </div>

      {/* Pod 详情弹窗 */}
      {selectedPod && (
        <PodDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespace={selectedPod.namespace}
          podName={selectedPod.name}
          onViewLogs={handleViewLogs}
        />
      )}

      {/* 日志查看器 */}
      {selectedPod && (
        <PodLogsViewer
          isOpen={logsOpen}
          onClose={() => setLogsOpen(false)}
          namespace={selectedPod.namespace}
          podName={selectedPod.name}
          containerName={logsContainer}
        />
      )}

      {/* 重启确认对话框 */}
      <ConfirmDialog
        isOpen={!!restartTarget}
        onClose={() => setRestartTarget(null)}
        onConfirm={handleRestartConfirm}
        title="确认重启 Pod"
        message={`确定要重启 Pod "${restartTarget?.name}" 吗？这将导致 Pod 短暂不可用。`}
        confirmText="重启"
        cancelText="取消"
        loading={restartLoading}
        variant="warning"
      />
    </Layout>
  );
}
