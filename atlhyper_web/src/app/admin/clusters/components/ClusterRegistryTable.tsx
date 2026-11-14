"use client";

import { AlertTriangle, KeyRound, Pencil, Plus, Trash2 } from "lucide-react";
import type { ClusterRegistryItem, ClusterRegistryStatus } from "@/api/cluster-registry";
import { DataTable, StatusBadge, type TableColumn } from "@/components/common";
import type { ClusterRegistryTranslations, CommonTranslations } from "@/types/i18n";

interface ClusterRegistryTableProps {
  clusters: ClusterRegistryItem[];
  /** 演示数据：操作按钮照常显示但不可用 */
  readOnly: boolean;
  onRegister: (cluster: ClusterRegistryItem) => void;
  onEditProfile: (cluster: ClusterRegistryItem) => void;
  onIssueToken: (cluster: ClusterRegistryItem) => void;
  onUnregister: (cluster: ClusterRegistryItem) => void;
  t: ClusterRegistryTranslations;
  commonT: CommonTranslations;
}

const STATUS_BADGE_TYPE: Record<ClusterRegistryStatus, "success" | "error" | "default"> = {
  online: "success",
  offline: "error",
  never: "default",
};

const ACTION_BUTTON_CLASS =
  "p-2 rounded-lg hover-bg text-muted hover:text-primary transition-colors disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:text-muted";

function formatTimestamp(isoTimestamp: string): string {
  return new Date(isoTimestamp).toLocaleString();
}

/** 集群注册表：一行一个集群。未登记但正在上报的集群用警示色标出，并给出「登记」入口 */
export function ClusterRegistryTable({
  clusters,
  readOnly,
  onRegister,
  onEditProfile,
  onIssueToken,
  onUnregister,
  t,
  commonT,
}: ClusterRegistryTableProps) {
  const statusLabel: Record<ClusterRegistryStatus, string> = {
    online: t.statusOnline,
    offline: t.statusOffline,
    never: t.statusNever,
  };

  const columns: TableColumn<ClusterRegistryItem>[] = [
    {
      key: "clusterId",
      header: t.clusterId,
      mobileTitle: true,
      render: (cluster) => (
        <span className="inline-flex items-center gap-1.5 font-mono text-sm text-default">
          {!cluster.registered && (
            <AlertTriangle className="w-3.5 h-3.5 text-amber-500 flex-shrink-0" aria-label={t.unregistered} />
          )}
          {cluster.clusterId}
        </span>
      ),
    },
    {
      key: "name",
      header: t.displayName,
      render: (cluster) => <span className="text-sm text-default">{cluster.name || "-"}</span>,
    },
    {
      key: "description",
      header: t.description,
      mobileVisible: false,
      render: (cluster) => (
        <span className="text-sm text-muted max-w-xs truncate block" title={cluster.description}>
          {cluster.description || "-"}
        </span>
      ),
    },
    {
      key: "status",
      header: t.status,
      render: (cluster) => (
        <StatusBadge status={statusLabel[cluster.status] ?? cluster.status} type={STATUS_BADGE_TYPE[cluster.status]} />
      ),
    },
    {
      key: "lastSeen",
      header: t.lastSeen,
      mobileVisible: false,
      render: (cluster) => (
        <span className="text-sm text-muted whitespace-nowrap">
          {cluster.lastSeen ? formatTimestamp(cluster.lastSeen) : t.neverReported}
        </span>
      ),
    },
    {
      key: "registration",
      header: t.registration,
      render: (cluster) =>
        cluster.registered ? (
          <StatusBadge status={t.registered} type="info" />
        ) : (
          <span title={t.unregisteredHint}>
            <StatusBadge status={t.unregistered} type="warning" />
          </span>
        ),
    },
    {
      key: "token",
      header: t.token,
      render: (cluster) =>
        cluster.tokenIssued ? (
          <div className="text-sm">
            <code className="font-mono text-default">{cluster.tokenPrefix}…</code>
            {cluster.tokenIssuedAt && (
              <div className="text-xs text-muted whitespace-nowrap">
                {t.tokenIssuedAt.replace("{time}", formatTimestamp(cluster.tokenIssuedAt))}
              </div>
            )}
          </div>
        ) : (
          <span className="text-sm text-muted">{t.tokenNotIssued}</span>
        ),
    },
    {
      key: "agentAuth",
      header: t.agentAuth,
      render: (cluster) =>
        cluster.authenticated ? (
          <StatusBadge status={t.agentAuthenticated} type="success" />
        ) : (
          // 在线却没带令牌才值得警示；不在线的集群没带令牌是理所当然
          <StatusBadge status={t.agentUnauthenticated} type={cluster.status === "online" ? "warning" : "default"} />
        ),
    },
    {
      key: "actions",
      header: commonT.action,
      render: (cluster) =>
        cluster.registered ? (
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => onEditProfile(cluster)}
              disabled={readOnly}
              className={ACTION_BUTTON_CLASS}
              title={t.editProfile}
              aria-label={t.editProfile}
            >
              <Pencil className="w-4 h-4" />
            </button>
            <button
              type="button"
              onClick={() => onIssueToken(cluster)}
              disabled={readOnly}
              className={ACTION_BUTTON_CLASS}
              title={cluster.tokenIssued ? t.rotateToken : t.issueToken}
              aria-label={cluster.tokenIssued ? t.rotateToken : t.issueToken}
            >
              <KeyRound className="w-4 h-4" />
            </button>
            <button
              type="button"
              onClick={() => onUnregister(cluster)}
              disabled={readOnly}
              className={`${ACTION_BUTTON_CLASS} hover:text-red-500`}
              title={t.unregister}
              aria-label={t.unregister}
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => onRegister(cluster)}
            disabled={readOnly}
            className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-lg bg-amber-500 hover:bg-amber-600 text-white transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            <Plus className="w-3.5 h-3.5" />
            {t.register}
          </button>
        ),
    },
  ];

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <DataTable
        columns={columns}
        data={clusters}
        loading={false}
        error=""
        keyExtractor={(cluster) => cluster.clusterId}
        pageSize={20}
      />
    </div>
  );
}
