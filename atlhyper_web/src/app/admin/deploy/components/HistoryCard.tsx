"use client";

import { useI18n } from "@/i18n/context";
import {
  History,
  RotateCcw,
  CheckCircle,
  XCircle,
  Clock,
  Eye,
  X,
  ExternalLink,
  GitPullRequest,
  FileDiff,
  Plus,
  Minus,
  User,
} from "lucide-react";
import { useState } from "react";
import type { MockDeployRecord, MockChangedFile, DeployTrigger } from "@/mock/deploy/data";
import { Pagination, paginate } from "./Pagination";

interface HistoryCardProps {
  history: MockDeployRecord[];
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  const time = d.toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
  });

  if (diffDays === 0) return time;
  if (diffDays === 1) return `1d ${time}`;
  return `${diffDays}d ${time}`;
}

function formatFullTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function StatusIcon({ status }: { status: string }) {
  switch (status) {
    case "success":
      return <CheckCircle className="w-4 h-4 text-emerald-500" />;
    case "failed":
      return <XCircle className="w-4 h-4 text-red-500" />;
    default:
      return <Clock className="w-4 h-4 text-amber-500" />;
  }
}

function TriggerBadge({
  trigger,
  labels,
}: {
  trigger: DeployTrigger;
  labels: { auto: string; manual: string; rollback: string };
}) {
  const config = {
    auto: {
      label: labels.auto,
      className:
        "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300",
    },
    manual: {
      label: labels.manual,
      className:
        "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300",
    },
    rollback: {
      label: labels.rollback,
      className:
        "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    },
  };
  const c = config[trigger];
  return (
    <span className={`text-xs px-1.5 py-0.5 rounded-full ${c.className}`}>
      {c.label}
    </span>
  );
}

function parseChangedFiles(json: string): MockChangedFile[] {
  try {
    return JSON.parse(json) || [];
  } catch {
    return [];
  }
}

function FileStatusIcon({ status }: { status: string }) {
  switch (status) {
    case "added":
      return <Plus className="w-3.5 h-3.5 text-emerald-500" />;
    case "removed":
      return <Minus className="w-3.5 h-3.5 text-red-500" />;
    default:
      return <FileDiff className="w-3.5 h-3.5 text-amber-500" />;
  }
}

function DetailModal({
  record,
  dt,
  triggerLabels,
  onClose,
}: {
  record: MockDeployRecord;
  dt: ReturnType<typeof useI18n>["t"]["deployPage"];
  triggerLabels: { auto: string; manual: string; rollback: string };
  onClose: () => void;
}) {
  const statusText =
    record.status === "success"
      ? dt.statusInSync
      : record.status === "failed"
        ? dt.statusFailed
        : dt.statusPending;

  const statusColor =
    record.status === "success"
      ? "text-emerald-600 dark:text-emerald-400"
      : record.status === "failed"
        ? "text-red-600 dark:text-red-400"
        : "text-amber-600 dark:text-amber-400";

  const changedFiles = parseChangedFiles(record.changedFiles);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative bg-card rounded-xl border border-[var(--border-color)] shadow-xl w-full max-w-2xl mx-4 max-h-[85vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)] flex-shrink-0">
          <h3 className="text-lg font-medium text-default">
            {dt.detailTitle}
          </h3>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-muted hover:text-default hover:bg-[var(--bg-secondary)] transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Body — scrollable */}
        <div className="px-6 py-5 space-y-4 overflow-y-auto flex-1">
          <DetailRow label={dt.detailPath}>
            <code className="text-sm bg-[var(--bg-secondary)] px-2 py-0.5 rounded text-default">
              {record.path}
            </code>
          </DetailRow>

          <DetailRow label={dt.detailNamespace}>
            <span className="text-xs px-1.5 py-0.5 rounded-full bg-violet-100 dark:bg-violet-900/30 text-violet-700 dark:text-violet-300">
              {record.namespace}
            </span>
          </DetailRow>

          <DetailRow label={dt.detailTrigger}>
            <TriggerBadge trigger={record.trigger} labels={triggerLabels} />
          </DetailRow>

          <DetailRow label={dt.detailDeployedAt}>
            <span className="text-sm text-default">
              {formatFullTime(record.deployedAt)}
            </span>
          </DetailRow>

          {/* Commit 信息区 */}
          <DetailRow label={dt.detailCommit}>
            <div className="flex items-center gap-2">
              <code className="text-sm bg-[var(--bg-secondary)] px-2 py-0.5 rounded text-default">
                {record.commitSha}
              </code>
              {record.compareUrl && (
                <a
                  href={record.compareUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-xs text-blue-600 dark:text-blue-400 hover:underline"
                >
                  <ExternalLink className="w-3 h-3" />
                  {dt.detailCompareLink}
                </a>
              )}
            </div>
          </DetailRow>

          <DetailRow label={dt.detailCommitMessage}>
            <p className="text-sm text-default">{record.commitMessage}</p>
          </DetailRow>

          {/* 作者 */}
          {record.commitAuthor && (
            <DetailRow label={dt.detailAuthor}>
              <div className="flex items-center gap-2">
                {record.commitAvatarUrl ? (
                  <img
                    src={record.commitAvatarUrl}
                    alt={record.commitAuthor}
                    className="w-5 h-5 rounded-full"
                  />
                ) : (
                  <User className="w-4 h-4 text-muted" />
                )}
                <span className="text-sm text-default">{record.commitAuthor}</span>
              </div>
            </DetailRow>
          )}

          {/* PR 信息 */}
          {record.prNumber > 0 && (
            <DetailRow label={dt.detailPR}>
              <div className="flex items-center gap-2">
                <GitPullRequest className="w-4 h-4 text-violet-500" />
                <span className="text-sm text-default">
                  #{record.prNumber} {record.prTitle}
                </span>
                {record.prUrl && (
                  <a
                    href={record.prUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-600 dark:text-blue-400 hover:underline"
                  >
                    <ExternalLink className="w-3 h-3" />
                  </a>
                )}
              </div>
            </DetailRow>
          )}

          <DetailRow label={dt.detailDuration}>
            <span className="text-sm text-default">
              {formatDuration(record.durationMs)}
            </span>
          </DetailRow>

          <DetailRow label={dt.detailResourceChanged}>
            <span className="text-sm text-default">
              {record.resourceChanged}/{record.resourceTotal}
            </span>
          </DetailRow>

          <DetailRow label={dt.detailStatus}>
            <div className="flex items-center gap-1.5">
              <StatusIcon status={record.status} />
              <span className={`text-sm font-medium ${statusColor}`}>
                {statusText}
              </span>
            </div>
          </DetailRow>

          {record.errorMessage && (
            <DetailRow label={dt.detailErrorMessage}>
              <p className="text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 px-3 py-2 rounded-lg">
                {record.errorMessage}
              </p>
            </DetailRow>
          )}

          {/* 变更文件列表 */}
          {changedFiles.length > 0 && (
            <div className="pt-2 border-t border-[var(--border-color)]">
              <div className="flex items-center gap-2 mb-3">
                <FileDiff className="w-4 h-4 text-muted" />
                <span className="text-sm font-medium text-default">
                  {dt.detailChangedFiles}
                  <span className="text-muted ml-1">({changedFiles.length})</span>
                </span>
              </div>
              <div className="space-y-1 max-h-48 overflow-y-auto">
                {changedFiles.map((file, idx) => (
                  <div
                    key={idx}
                    className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-[var(--bg-secondary)] text-sm"
                  >
                    <FileStatusIcon status={file.status} />
                    <span className="text-default flex-1 min-w-0 truncate font-mono text-xs">
                      {file.filename}
                    </span>
                    <span className="flex items-center gap-1 text-xs flex-shrink-0">
                      <span className="text-emerald-600 dark:text-emerald-400">+{file.additions}</span>
                      <span className="text-red-500">-{file.deletions}</span>
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function DetailRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex gap-3">
      <span className="text-sm text-muted w-24 flex-shrink-0 pt-0.5">
        {label}
      </span>
      <div className="flex-1 min-w-0">{children}</div>
    </div>
  );
}

const HISTORY_PAGE_SIZE = 8;

export function HistoryCard({ history }: HistoryCardProps) {
  const { t } = useI18n();
  const dt = t.deployPage;
  const [selectedRecord, setSelectedRecord] = useState<MockDeployRecord | null>(
    null,
  );
  const [page, setPage] = useState(0);
  const pagedHistory = paginate(history, page, HISTORY_PAGE_SIZE);

  const triggerLabels = {
    auto: dt.triggerAuto,
    manual: dt.triggerManual,
    rollback: dt.triggerRollback,
  };

  return (
    <>
      <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
        <div className="flex items-center gap-2 px-6 py-4 border-b border-[var(--border-color)]">
          <History className="w-5 h-5 text-muted" />
          <h3 className="text-lg font-medium text-default">
            {dt.historySection}
          </h3>
        </div>

        {history.length === 0 ? (
          <div className="p-12 text-center">
            <History className="w-12 h-12 mx-auto mb-3 text-muted opacity-30" />
            <p className="text-muted">{dt.noHistory}</p>
            <p className="text-xs text-muted mt-1">{dt.noHistoryHint}</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="text-left text-sm text-muted border-b border-[var(--border-color)]">
                  <th className="px-6 py-3 font-medium">{dt.timeCol}</th>
                  <th className="px-6 py-3 font-medium">{dt.pathCol2}</th>
                  <th className="px-6 py-3 font-medium">{dt.triggerCol}</th>
                  <th className="px-6 py-3 font-medium">{dt.commitCol}</th>
                  <th className="px-6 py-3 font-medium">{dt.statusCol2}</th>
                  <th className="px-6 py-3 font-medium"></th>
                </tr>
              </thead>
              <tbody>
                {pagedHistory.map((record) => (
                  <tr
                    key={record.id}
                    className="border-b border-[var(--border-color)] last:border-0 hover:bg-[var(--bg-secondary)] transition-colors"
                  >
                    <td className="px-6 py-3">
                      <span className="text-sm text-muted whitespace-nowrap">
                        {formatTime(record.deployedAt)}
                      </span>
                    </td>
                    <td className="px-6 py-3">
                      <code className="text-sm text-default">
                        {record.path}
                      </code>
                    </td>
                    <td className="px-6 py-3">
                      <TriggerBadge
                        trigger={record.trigger}
                        labels={triggerLabels}
                      />
                    </td>
                    <td className="px-6 py-3">
                      <code className="text-xs bg-[var(--bg-secondary)] px-1.5 py-0.5 rounded text-default">
                        {record.commitSha}
                      </code>
                    </td>
                    <td className="px-6 py-3">
                      <StatusIcon status={record.status} />
                    </td>
                    <td className="px-6 py-3">
                      <div className="flex items-center gap-1.5">
                        <button
                          onClick={() => setSelectedRecord(record)}
                          className="flex items-center gap-1 px-2 py-1 text-xs rounded border border-[var(--border-color)] text-muted hover:text-default hover:bg-[var(--bg-secondary)] transition-colors"
                        >
                          <Eye className="w-3 h-3" />
                          {dt.viewDetail}
                        </button>
                        {record.trigger !== "rollback" && (
                          <button className="flex items-center gap-1 px-2 py-1 text-xs rounded border border-[var(--border-color)] text-muted hover:text-default hover:bg-[var(--bg-secondary)] transition-colors">
                            <RotateCcw className="w-3 h-3" />
                            {dt.rollbackBtn}
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <Pagination
          page={page}
          pageSize={HISTORY_PAGE_SIZE}
          total={history.length}
          onPageChange={setPage}
          labels={t.table}
        />
      </div>

      {/* 详情弹窗 */}
      {selectedRecord && (
        <DetailModal
          record={selectedRecord}
          dt={dt}
          triggerLabels={triggerLabels}
          onClose={() => setSelectedRecord(null)}
        />
      )}
    </>
  );
}
