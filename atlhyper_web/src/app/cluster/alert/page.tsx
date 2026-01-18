"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getEventLogs } from "@/api/event";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, StatusBadge, type TableColumn, LoadingSpinner } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import {
  AlertTriangle,
  AlertCircle,
  Info,
  Filter,
  Eye,
  X,
} from "lucide-react";
import type { EventLog, EventOverview } from "@/types/cluster";

// 严重级别配置
const severityConfig: Record<string, { icon: typeof AlertCircle; color: string; badgeType: "error" | "warning" | "info" | "default" }> = {
  error: { icon: AlertCircle, color: "text-red-500", badgeType: "error" },
  warning: { icon: AlertTriangle, color: "text-yellow-500", badgeType: "warning" },
  info: { icon: Info, color: "text-blue-500", badgeType: "info" },
};

// 时间范围选项 - 动态生成以支持国际化
function getTimeRangeOptions(t: ReturnType<typeof useI18n>["t"]) {
  return [
    { value: 1, label: `${t.common.from} 1 ${t.common.date}` },
    { value: 3, label: `${t.common.from} 3 ${t.common.date}` },
    { value: 7, label: `${t.common.from} 7 ${t.common.date}` },
    { value: 14, label: `${t.common.from} 14 ${t.common.date}` },
    { value: 30, label: `${t.common.from} 30 ${t.common.date}` },
  ];
}

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
    <div className="relative flex-1 min-w-[200px]">
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
        className="w-full px-3 py-2 pr-8 bg-[var(--background)] border border-[var(--border-color)] rounded-lg text-sm text-default focus:outline-none focus:ring-1 focus:ring-primary appearance-none min-w-[120px]"
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

export default function AlertPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<EventOverview | null>(null);
  const [error, setError] = useState("");

  // 筛选状态
  const [timeRange, setTimeRange] = useState(1);
  const [severityFilter, setSeverityFilter] = useState("");
  const [kindFilter, setKindFilter] = useState("");
  const [namespaceFilter, setNamespaceFilter] = useState("");
  const [searchTerm, setSearchTerm] = useState("");

  // 详情弹窗
  const [selectedEvent, setSelectedEvent] = useState<EventLog | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  // 筛选辅助
  const activeFilterCount = [severityFilter, kindFilter, namespaceFilter, searchTerm].filter(Boolean).length;
  const hasActiveFilters = activeFilterCount > 0;
  const clearAllFilters = () => {
    setSeverityFilter("");
    setKindFilter("");
    setNamespaceFilter("");
    setSearchTerm("");
  };

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getEventLogs({
        ClusterID: getCurrentClusterId(),
      });
      setData(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, []);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // 获取唯一的 Kind 和 Namespace 列表
  const { kinds, namespaces } = useMemo(() => {
    if (!data?.rows) return { kinds: [], namespaces: [] };
    const kindSet = new Set<string>();
    const nsSet = new Set<string>();
    data.rows.forEach((e) => {
      if (e.Kind) kindSet.add(e.Kind);
      if (e.Namespace) nsSet.add(e.Namespace);
    });
    return {
      kinds: Array.from(kindSet).sort(),
      namespaces: Array.from(nsSet).sort(),
    };
  }, [data?.rows]);

  // 过滤数据
  const filteredRows = useMemo(() => {
    if (!data?.rows) return [];
    return data.rows.filter((e) => {
      // 严重级别过滤
      if (severityFilter) {
        const sev = (e.Severity || "info").toLowerCase();
        if (sev !== severityFilter) return false;
      }
      // Kind 过滤
      if (kindFilter && e.Kind !== kindFilter) return false;
      // Namespace 过滤
      if (namespaceFilter && e.Namespace !== namespaceFilter) return false;
      // 搜索过滤
      if (searchTerm) {
        const term = searchTerm.toLowerCase();
        return (
          e.Name?.toLowerCase().includes(term) ||
          e.Message?.toLowerCase().includes(term) ||
          e.Reason?.toLowerCase().includes(term)
        );
      }
      return true;
    });
  }, [data?.rows, severityFilter, kindFilter, namespaceFilter, searchTerm]);

  // 查看详情
  const handleViewDetail = (event: EventLog) => {
    setSelectedEvent(event);
    setDetailOpen(true);
  };

  const columns: TableColumn<EventLog>[] = [
    {
      key: "time",
      header: t.common.time,
      render: (e) => (
        <span className="text-sm text-muted whitespace-nowrap">
          {e.EventTime ? new Date(e.EventTime).toLocaleString() : "-"}
        </span>
      ),
    },
    {
      key: "severity",
      header: t.alert.severity,
      render: (e) => {
        const sev = (e.Severity || "info").toLowerCase();
        const config = severityConfig[sev] || severityConfig.info;
        return <StatusBadge status={sev.toUpperCase()} type={config.badgeType} />;
      },
    },
    {
      key: "kind",
      header: t.alert.type,
      render: (e) => (
        <span className="inline-flex px-2 py-1 text-xs bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400 rounded">
          {e.Kind || "-"}
        </span>
      ),
    },
    {
      key: "name",
      header: t.common.name,
      render: (e) => <span className="font-medium text-default">{e.Name || "-"}</span>,
    },
    {
      key: "namespace",
      header: t.common.namespace,
      render: (e) => <span className="text-sm text-muted">{e.Namespace || "-"}</span>,
    },
    {
      key: "reason",
      header: t.alert.source,
      render: (e) => (
        <span className="inline-flex px-2 py-1 text-xs bg-gray-100 dark:bg-gray-800 rounded">
          {e.Reason || "-"}
        </span>
      ),
    },
    {
      key: "message",
      header: t.alert.message,
      render: (e) => (
        <span className="text-sm text-secondary max-w-xs truncate block" title={e.Message}>
          {e.Message || "-"}
        </span>
      ),
    },
    {
      key: "actions",
      header: "",
      render: (e) => (
        <button
          onClick={() => handleViewDetail(e)}
          className="p-2 hover-bg rounded-lg"
          title={t.alert.viewDetails}
        >
          <Eye className="w-4 h-4 text-muted hover:text-primary" />
        </button>
      ),
    },
  ];

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title={t.nav.alert}
          description={t.alert.pageDescription}
          autoRefreshSeconds={intervalSeconds}
        />

        {/* 统计卡片 */}
        {data && (
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-4">
            <StatsCard label={t.common.total} value={data.cards.totalAlerts ?? 0} />
            <StatsCard label={t.alert.critical} value={data.cards.error ?? 0} icon={AlertCircle} iconColor="text-red-500" />
            <StatsCard label={t.alert.warning} value={data.cards.warning ?? 0} icon={AlertTriangle} iconColor="text-yellow-500" />
            <StatsCard label={t.alert.info} value={data.cards.info ?? 0} icon={Info} iconColor="text-blue-500" />
            <StatsCard label={t.common.total} value={data.cards.totalEvents ?? 0} iconColor="text-purple-500" />
            <StatsCard label={t.alert.type} value={data.cards.categoriesCount ?? 0} iconColor="text-green-500" />
            <StatsCard label={t.alert.type} value={data.cards.kindsCount ?? 0} iconColor="text-orange-500" />
          </div>
        )}

        {/* 筛选工具栏 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
          {/* 标题栏 */}
          <div className="flex items-center gap-2 mb-3">
            <Filter className="w-4 h-4 text-muted" />
            <span className="text-sm font-medium text-default">{t.common.filter}</span>
            {activeFilterCount > 0 && (
              <span className="px-1.5 py-0.5 text-xs bg-primary/10 text-primary rounded">
                {activeFilterCount}
              </span>
            )}
            {hasActiveFilters && (
              <button
                onClick={clearAllFilters}
                className="ml-auto flex items-center gap-1 text-xs text-muted hover:text-default transition-colors"
              >
                <X className="w-3 h-3" />
                {t.common.clearAll}
              </button>
            )}
          </div>

          {/* 筛选控件 */}
          <div className="flex flex-wrap gap-3 items-center">
            {/* 时间范围 */}
            <div className="relative">
              <select
                value={timeRange}
                onChange={(e) => setTimeRange(Number(e.target.value))}
                className="px-3 py-2 pr-8 bg-[var(--background)] border border-[var(--border-color)] rounded-lg text-sm text-default focus:outline-none focus:ring-1 focus:ring-primary appearance-none min-w-[130px]"
              >
                {getTimeRangeOptions(t).map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
              <div className="absolute right-2 top-1/2 -translate-y-1/2 pointer-events-none">
                <svg className="w-4 h-4 text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </div>

            {/* 严重级别 */}
            <FilterSelect
              value={severityFilter}
              onChange={setSeverityFilter}
              onClear={() => setSeverityFilter("")}
              placeholder={t.alert.allSeverities}
              options={[
                { value: "error", label: "Error" },
                { value: "warning", label: "Warning" },
                { value: "info", label: "Info" },
              ]}
            />

            {/* Kind 过滤 */}
            <FilterSelect
              value={kindFilter}
              onChange={setKindFilter}
              onClear={() => setKindFilter("")}
              placeholder={t.alert.allTypes}
              options={kinds.map((k) => ({ value: k, label: k }))}
            />

            {/* Namespace 过滤 */}
            <FilterSelect
              value={namespaceFilter}
              onChange={setNamespaceFilter}
              onClear={() => setNamespaceFilter("")}
              placeholder={t.pod.allNamespaces}
              options={namespaces.map((ns) => ({ value: ns, label: ns }))}
            />

            {/* 搜索 */}
            <FilterInput
              value={searchTerm}
              onChange={setSearchTerm}
              onClear={() => setSearchTerm("")}
              placeholder={t.alert.searchPlaceholder}
            />

            {/* 结果计数 */}
            <span className="text-sm text-muted whitespace-nowrap">
              {filteredRows.length} / {data?.rows?.length || 0} {t.common.items}
            </span>
          </div>
        </div>

        {/* 数据表格 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          {loading ? (
            <div className="py-12">
              <LoadingSpinner />
            </div>
          ) : error ? (
            <div className="text-center py-12 text-red-500">{error}</div>
          ) : (
            <DataTable
              columns={columns}
              data={filteredRows}
              loading={false}
              error=""
              keyExtractor={(e, index) => `${index}-${e.ClusterID}-${e.Kind}-${e.Namespace}-${e.Name}-${e.EventTime}`}
            />
          )}
        </div>
      </div>

      {/* 详情弹窗 */}
      {detailOpen && selectedEvent && (
        <EventDetailModal
          event={selectedEvent}
          onClose={() => setDetailOpen(false)}
          t={t}
        />
      )}
    </Layout>
  );
}

// 事件详情弹窗
function EventDetailModal({ event, onClose, t }: { event: EventLog; onClose: () => void; t: ReturnType<typeof useI18n>["t"] }) {
  const sev = (event.Severity || "info").toLowerCase();
  const config = severityConfig[sev] || severityConfig.info;
  const Icon = config.icon;

  const details = [
    { label: t.nav.cluster, value: event.ClusterID },
    { label: t.alert.type, value: event.Kind },
    { label: t.common.name, value: event.Name },
    { label: t.common.namespace, value: event.Namespace },
    { label: t.nav.node, value: event.Node },
    { label: t.alert.source, value: event.Reason },
    { label: t.alert.type, value: event.Category },
    { label: t.alert.timestamp, value: event.EventTime ? new Date(event.EventTime).toLocaleString() : "-" },
    { label: t.common.time, value: event.Time ? new Date(event.Time).toLocaleString() : "-" },
  ];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-card rounded-xl border border-[var(--border-color)] shadow-xl w-full max-w-2xl mx-4 max-h-[80vh] overflow-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-3">
            <Icon className={`w-6 h-6 ${config.color}`} />
            <div>
              <h2 className="text-lg font-semibold text-default">{t.common.details}</h2>
              <StatusBadge status={sev.toUpperCase()} type={config.badgeType} />
            </div>
          </div>
          <button onClick={onClose} className="p-2 hover-bg rounded-lg">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {/* 消息 */}
          <div>
            <h3 className="text-sm font-semibold text-default mb-2">{t.alert.message}</h3>
            <div className="bg-[var(--background)] rounded-lg p-4">
              <p className="text-sm text-default whitespace-pre-wrap">{event.Message || t.common.noData}</p>
            </div>
          </div>

          {/* 详细信息 */}
          <div>
            <h3 className="text-sm font-semibold text-default mb-3">{t.common.details}</h3>
            <div className="grid grid-cols-2 gap-4">
              {details.map((item, i) => (
                <div key={i} className="bg-[var(--background)] rounded-lg p-3">
                  <div className="text-xs text-muted mb-1">{item.label}</div>
                  <div className="text-sm text-default font-medium break-all">{item.value || "-"}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
