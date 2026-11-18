"use client";

import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import type { CommandHistory } from "@/api/commands";
import { PageHeader, DataTable, QueryView, StatusBadge, type TableColumn } from "@/components/common";
import { useCan } from "@/hooks/useCan";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useDebouncedValue } from "@/hooks/useDebouncedValue";
import { useScopedState } from "@/hooks/useScopedState";
import { useSessionReady } from "@/hooks/useSessionReady";
import { commandHistoryQuery } from "@/queries/admin";
import { REFRESH } from "@/queries/refresh";
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
import { PermissionLock } from "@/components/common/PermissionLock";

const PAGE_SIZE = 20;
const NO_COMMANDS: CommandHistory[] = [];

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
  const { clusterId } = useCurrentCluster();
  // 指令历史是内容级数据（commands.read）。权限加载完之前按没有权限算：不发请求，界面由 PermissionLock 占位
  const sessionReady = useSessionReady();
  const can = useCan();
  const canReadCommands = sessionReady && can("commands.read");

  // Filter state
  const [sourceFilter, setSourceFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [actionFilter, setActionFilter] = useState("");
  // 搜索框里的字随敲随显，进查询键的是去抖后的值：停下来才发请求
  const [searchInput, setSearchInput] = useState("");
  const searchTerm = useDebouncedValue(searchInput);

  // 页码依附于筛选条件：条件一变回到第一页
  const filterScope = JSON.stringify([sourceFilter, statusFilter, actionFilter, searchTerm]);
  const [page, setPage] = useScopedState(filterScope, 0);

  const historyQuery = useQuery(
    commandHistoryQuery(
      clusterId,
      {
        source: sourceFilter,
        status: statusFilter,
        action: actionFilter,
        search: searchTerm,
        limit: PAGE_SIZE,
        offset: page * PAGE_SIZE,
      },
      canReadCommands,
    ),
  );
  const commands = historyQuery.data?.commands ?? NO_COMMANDS;
  const total = historyQuery.data?.total ?? 0;

  // Detail modal
  const [selectedCommand, setSelectedCommand] = useState<CommandHistory | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const clearAllFilters = () => {
    setSourceFilter("");
    setStatusFilter("");
    setActionFilter("");
    setSearchInput("");
  };

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
          autoRefreshSeconds={REFRESH.standard / 1000}
        />

        <PermissionLock permission="commands.read" className="bg-card rounded-xl border border-[var(--border-color)]">
          <div className="space-y-6">
          <CommandFilterToolbar
            sourceFilter={sourceFilter}
            statusFilter={statusFilter}
            actionFilter={actionFilter}
            searchTerm={searchInput}
            onSourceChange={setSourceFilter}
            onStatusChange={setStatusFilter}
            onActionChange={setActionFilter}
            onSearchChange={setSearchInput}
            onClearAll={clearAllFilters}
            actionOptions={actionOptions}
            commandCount={commands.length}
            total={total}
            t={t}
          />

          {/* Data table */}
          <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
            <QueryView
              query={historyQuery}
              isEmpty={(history) => history.commands.length === 0}
              empty={
                <div className="text-center py-12 text-muted">
                  <Terminal className="w-12 h-12 mx-auto mb-3 opacity-50" />
                  <p>{t.commands.noCommands}</p>
                </div>
              }
            >
              {(history) => (
                <DataTable
                  columns={columns}
                  data={history.commands}
                  loading={false}
                  error=""
                  keyExtractor={(cmd) => cmd.commandId}
                />
              )}
            </QueryView>
          </div>

          <CommandPagination
            page={page}
            pageSize={PAGE_SIZE}
            total={total}
            onPageChange={setPage}
            t={t}
          />
          </div>
        </PermissionLock>
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
