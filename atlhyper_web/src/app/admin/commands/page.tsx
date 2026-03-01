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
  Terminal,
  Bot,
  Globe,
  Eye,
  CheckCircle2,
  XCircle,
  Loader2,
  AlertCircle,
  Clock,
} from "lucide-react";

import { CommandDetailModal, CommandFilterToolbar, CommandPagination } from "./components";

// Status config
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

// Source icons
const sourceIcons: Record<string, typeof Terminal> = {
  web: Globe,
  ai: Bot,
};

export default function CommandsPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [commands, setCommands] = useState<CommandHistory[]>([]);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState("");

  // Filter state
  const [sourceFilter, setSourceFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [actionFilter, setActionFilter] = useState("");
  const [searchTerm, setSearchTerm] = useState("");

  // Pagination
  const [page, setPage] = useState(0);
  const pageSize = 20;

  // Detail modal
  const [selectedCommand, setSelectedCommand] = useState<CommandHistory | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

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

  const actionOptions = useMemo(() => {
    const actions = Object.entries(t.commands.actions) as [string, string][];
    return actions.map(([value, label]) => ({ value, label }));
  }, [t.commands.actions]);

  const handleViewDetail = (cmd: CommandHistory) => {
    setSelectedCommand(cmd);
    setDetailOpen(true);
  };

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  const formatTarget = (cmd: CommandHistory) => {
    const parts = [];
    if (cmd.targetKind) parts.push(cmd.targetKind);
    if (cmd.targetNamespace) parts.push(cmd.targetNamespace);
    if (cmd.targetName) parts.push(cmd.targetName);
    return parts.join(" / ") || "-";
  };

  const columns: TableColumn<CommandHistory>[] = [
    {
      key: "time",
      header: t.common.time,
      mobileVisible: false,
      render: (cmd) => (
        <span className="text-sm text-muted whitespace-nowrap">
          {cmd.createdAt ? new Date(cmd.createdAt).toLocaleString() : "-"}
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
      mobileTitle: true,
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
      mobileVisible: false,
      render: (cmd) => (
        <span className="text-sm text-muted whitespace-nowrap">
          {cmd.durationMs > 0 ? formatDuration(cmd.durationMs) : "-"}
        </span>
      ),
    },
    {
      key: "actions",
      header: "",
      mobileVisible: false,
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

        <CommandFilterToolbar
          sourceFilter={sourceFilter}
          statusFilter={statusFilter}
          actionFilter={actionFilter}
          searchTerm={searchTerm}
          onSourceChange={(v) => { setSourceFilter(v); setPage(0); }}
          onStatusChange={(v) => { setStatusFilter(v); setPage(0); }}
          onActionChange={(v) => { setActionFilter(v); setPage(0); }}
          onSearchChange={(v) => { setSearchTerm(v); setPage(0); }}
          onClearAll={clearAllFilters}
          actionOptions={actionOptions}
          commandCount={commands.length}
          total={total}
          t={t}
        />

        {/* Data table */}
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
              keyExtractor={(cmd) => cmd.commandId}
            />
          )}
        </div>

        <CommandPagination
          page={page}
          pageSize={pageSize}
          total={total}
          onPageChange={setPage}
          t={t}
        />
      </div>

      {/* Detail modal */}
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
