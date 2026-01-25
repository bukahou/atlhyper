"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getCommandHistory, type CommandHistory } from "@/api/commands";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import {
  PageHeader,
  DataTable,
  StatusBadge,
  type TableColumn,
  LoadingSpinner,
} from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import {
  Filter,
  Eye,
  X,
  Terminal,
  Bot,
  Globe,
  Clock,
  CheckCircle2,
  XCircle,
  Loader2,
  AlertCircle,
} from "lucide-react";

// 状态配置
const statusConfig: Record<
  string,
  { icon: typeof CheckCircle2; color: string; badgeType: "success" | "error" | "warning" | "info" | "default" }
> = {
  success: { icon: CheckCircle2, color: "text-green-500", badgeType: "success" },
  failed: { icon: XCircle, color: "text-red-500", badgeType: "error" },
  timeout: { icon: AlertCircle, color: "text-orange-500", badgeType: "warning" },
  running: { icon: Loader2, color: "text-blue-500", badgeType: "info" },
  pending: { icon: Clock, color: "text-gray-500", badgeType: "default" },
};

// 来源图标
const sourceIcons: Record<string, typeof Terminal> = {
  web: Globe,
  ai: Bot,
};

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

export default function CommandsPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [commands, setCommands] = useState<CommandHistory[]>([]);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState("");

  // 筛选状态
  const [sourceFilter, setSourceFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [actionFilter, setActionFilter] = useState("");
  const [searchTerm, setSearchTerm] = useState("");

  // 分页状态
  const [page, setPage] = useState(0);
  const pageSize = 20;

  // 详情弹窗
  const [selectedCommand, setSelectedCommand] = useState<CommandHistory | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  // 筛选辅助
  const activeFilterCount = [sourceFilter, statusFilter, actionFilter, searchTerm].filter(Boolean).length;
  const hasActiveFilters = activeFilterCount > 0;
  const clearAllFilters = () => {
    setSourceFilter("");
    setStatusFilter("");
    setActionFilter("");
    setSearchTerm("");
    setPage(0);
  };

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getCommandHistory({
        cluster_id: getCurrentClusterId(),
        source: sourceFilter,
        status: statusFilter,
        action: actionFilter,
        search: searchTerm,
        limit: pageSize,
        offset: page * pageSize,
      });
      setCommands(res.data.commands || []);
      setTotal(res.data.total || 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [sourceFilter, statusFilter, actionFilter, searchTerm, page, t.common.loadFailed]);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // 获取操作类型选项
  const actionOptions = useMemo(() => {
    const actions = Object.entries(t.commands.actions) as [string, string][];
    return actions.map(([value, label]) => ({ value, label }));
  }, [t.commands.actions]);

  // 查看详情
  const handleViewDetail = (cmd: CommandHistory) => {
    setSelectedCommand(cmd);
    setDetailOpen(true);
  };

  // 格式化耗时
  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  // 格式化目标
  const formatTarget = (cmd: CommandHistory) => {
    const parts = [];
    if (cmd.target_kind) parts.push(cmd.target_kind);
    if (cmd.target_namespace) parts.push(cmd.target_namespace);
    if (cmd.target_name) parts.push(cmd.target_name);
    return parts.join(" / ") || "-";
  };

  const columns: TableColumn<CommandHistory>[] = [
    {
      key: "time",
      header: t.common.time,
      render: (cmd) => (
        <span className="text-sm text-muted whitespace-nowrap">
          {cmd.created_at ? new Date(cmd.created_at).toLocaleString() : "-"}
        </span>
      ),
    },
    {
      key: "source",
      header: t.commands.source,
      render: (cmd) => {
        const Icon = sourceIcons[cmd.source] || Terminal;
        const label = t.commands.sources[cmd.source as keyof typeof t.commands.sources] || cmd.source;
        return (
          <span className="inline-flex items-center gap-1.5 px-2 py-1 text-xs bg-gray-100 dark:bg-gray-800 rounded">
            <Icon className="w-3 h-3" />
            {label}
          </span>
        );
      },
    },
    {
      key: "action",
      header: t.common.action,
      render: (cmd) => {
        const label = t.commands.actions[cmd.action as keyof typeof t.commands.actions] || cmd.action;
        return (
          <span className="inline-flex px-2 py-1 text-xs bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400 rounded">
            {label}
          </span>
        );
      },
    },
    {
      key: "target",
      header: t.commands.target,
      render: (cmd) => (
        <span className="text-sm text-default font-medium max-w-xs truncate block" title={formatTarget(cmd)}>
          {formatTarget(cmd)}
        </span>
      ),
    },
    {
      key: "status",
      header: t.common.status,
      render: (cmd) => {
        const config = statusConfig[cmd.status] || statusConfig.pending;
        const label = t.commands.statuses[cmd.status as keyof typeof t.commands.statuses] || cmd.status;
        return <StatusBadge status={label} type={config.badgeType} />;
      },
    },
    {
      key: "duration",
      header: t.commands.duration,
      render: (cmd) => (
        <span className="text-sm text-muted whitespace-nowrap">
          {cmd.duration_ms > 0 ? formatDuration(cmd.duration_ms) : "-"}
        </span>
      ),
    },
    {
      key: "actions",
      header: "",
      render: (cmd) => (
        <button
          onClick={() => handleViewDetail(cmd)}
          className="p-2 hover-bg rounded-lg"
          title={t.commands.viewDetails}
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
          title={t.nav.commands}
          description={t.commands.pageDescription}
          autoRefreshSeconds={intervalSeconds}
        />

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
            {/* 来源 */}
            <FilterSelect
              value={sourceFilter}
              onChange={(v) => {
                setSourceFilter(v);
                setPage(0);
              }}
              onClear={() => {
                setSourceFilter("");
                setPage(0);
              }}
              placeholder={t.commands.allSources}
              options={[
                { value: "web", label: t.commands.sources.web },
                { value: "ai", label: t.commands.sources.ai },
              ]}
            />

            {/* 状态 */}
            <FilterSelect
              value={statusFilter}
              onChange={(v) => {
                setStatusFilter(v);
                setPage(0);
              }}
              onClear={() => {
                setStatusFilter("");
                setPage(0);
              }}
              placeholder={t.commands.allStatus}
              options={[
                { value: "pending", label: t.commands.statuses.pending },
                { value: "running", label: t.commands.statuses.running },
                { value: "success", label: t.commands.statuses.success },
                { value: "failed", label: t.commands.statuses.failed },
                { value: "timeout", label: t.commands.statuses.timeout },
              ]}
            />

            {/* 操作 */}
            <FilterSelect
              value={actionFilter}
              onChange={(v) => {
                setActionFilter(v);
                setPage(0);
              }}
              onClear={() => {
                setActionFilter("");
                setPage(0);
              }}
              placeholder={t.commands.allActions}
              options={actionOptions}
            />

            {/* 搜索 */}
            <FilterInput
              value={searchTerm}
              onChange={(v) => {
                setSearchTerm(v);
                setPage(0);
              }}
              onClear={() => {
                setSearchTerm("");
                setPage(0);
              }}
              placeholder={t.commands.searchPlaceholder}
            />

            {/* 结果计数 */}
            <span className="text-sm text-muted whitespace-nowrap">
              {commands.length} / {total} {t.common.items}
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
          ) : commands.length === 0 ? (
            <div className="text-center py-12 text-muted">
              <Terminal className="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p>{t.commands.noCommands}</p>
            </div>
          ) : (
            <DataTable
              columns={columns}
              data={commands}
              loading={false}
              error=""
              keyExtractor={(cmd) => cmd.command_id}
            />
          )}
        </div>

        {/* 分页 */}
        {total > pageSize && (
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted">
              {t.table.showing} {page * pageSize + 1}-{Math.min((page + 1) * pageSize, total)} / {total} {t.table.entries}
            </span>
            <div className="flex gap-2">
              <button
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                disabled={page === 0}
                className="px-3 py-1 text-sm border border-[var(--border-color)] rounded-lg hover-bg disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {t.table.previousPage}
              </button>
              <button
                onClick={() => setPage((p) => p + 1)}
                disabled={(page + 1) * pageSize >= total}
                className="px-3 py-1 text-sm border border-[var(--border-color)] rounded-lg hover-bg disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {t.table.nextPage}
              </button>
            </div>
          </div>
        )}
      </div>

      {/* 详情弹窗 */}
      {detailOpen && selectedCommand && (
        <CommandDetailModal
          command={selectedCommand}
          onClose={() => setDetailOpen(false)}
          t={t}
        />
      )}
    </Layout>
  );
}

// 命令详情弹窗
function CommandDetailModal({
  command,
  onClose,
  t,
}: {
  command: CommandHistory;
  onClose: () => void;
  t: ReturnType<typeof useI18n>["t"];
}) {
  const config = statusConfig[command.status] || statusConfig.pending;
  const Icon = config.icon;
  const statusLabel = t.commands.statuses[command.status as keyof typeof t.commands.statuses] || command.status;
  const actionLabel = t.commands.actions[command.action as keyof typeof t.commands.actions] || command.action;
  const sourceLabel = t.commands.sources[command.source as keyof typeof t.commands.sources] || command.source;

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  const details = [
    { label: t.commands.commandId, value: command.command_id },
    { label: t.commands.source, value: sourceLabel },
    { label: t.common.action, value: actionLabel },
    { label: t.common.type, value: command.target_kind },
    { label: t.common.namespace, value: command.target_namespace },
    { label: t.common.name, value: command.target_name },
    { label: t.common.status, value: statusLabel },
    { label: t.commands.duration, value: command.duration_ms > 0 ? formatDuration(command.duration_ms) : "-" },
    { label: t.commands.createdAt, value: command.created_at ? new Date(command.created_at).toLocaleString() : "-" },
    { label: t.commands.startedAt, value: command.started_at ? new Date(command.started_at).toLocaleString() : "-" },
    { label: t.commands.finishedAt, value: command.finished_at ? new Date(command.finished_at).toLocaleString() : "-" },
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
              <StatusBadge status={statusLabel} type={config.badgeType} />
            </div>
          </div>
          <button onClick={onClose} className="p-2 hover-bg rounded-lg">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
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

          {/* 参数 */}
          {command.params && (
            <div>
              <h3 className="text-sm font-semibold text-default mb-2">{t.commands.params}</h3>
              <div className="bg-[var(--background)] rounded-lg p-4">
                <pre className="text-sm text-default whitespace-pre-wrap font-mono">
                  {(() => {
                    try {
                      return JSON.stringify(JSON.parse(command.params), null, 2);
                    } catch {
                      return command.params;
                    }
                  })()}
                </pre>
              </div>
            </div>
          )}

          {/* 结果 */}
          {command.result && (
            <div>
              <h3 className="text-sm font-semibold text-default mb-2">{t.commands.result}</h3>
              <div className="bg-[var(--background)] rounded-lg p-4">
                <pre className="text-sm text-default whitespace-pre-wrap font-mono">
                  {(() => {
                    try {
                      return JSON.stringify(JSON.parse(command.result), null, 2);
                    } catch {
                      return command.result;
                    }
                  })()}
                </pre>
              </div>
            </div>
          )}

          {/* 错误信息 */}
          {command.error_message && (
            <div>
              <h3 className="text-sm font-semibold text-red-500 mb-2">{t.commands.errorMessage}</h3>
              <div className="bg-red-50 dark:bg-red-900/20 rounded-lg p-4 border border-red-200 dark:border-red-800">
                <p className="text-sm text-red-600 dark:text-red-400 whitespace-pre-wrap">
                  {command.error_message}
                </p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
