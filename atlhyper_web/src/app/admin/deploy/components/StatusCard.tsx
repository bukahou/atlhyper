"use client";

import { useI18n } from "@/i18n/context";
import { Activity, CheckCircle, AlertTriangle, RefreshCw } from "lucide-react";
import { useState } from "react";
import type { MockPathStatus } from "@/mock/deploy/data";
import { Pagination, paginate } from "./Pagination";

interface StatusCardProps {
  statusList: MockPathStatus[];
  onSyncNow?: (path: string) => void;
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
  });
}

const STATUS_PAGE_SIZE = 5;

export function StatusCard({ statusList, onSyncNow }: StatusCardProps) {
  const { t } = useI18n();
  const dt = t.deployPage;
  const [page, setPage] = useState(0);

  const syncCount = statusList.filter((s) => s.inSync).length;
  const totalCount = statusList.length;
  const pagedList = paginate(statusList, page, STATUS_PAGE_SIZE);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)]">
        <div className="flex items-center gap-2">
          <Activity className="w-5 h-5 text-muted" />
          <h3 className="text-lg font-medium text-default">{dt.statusSection}</h3>
        </div>
        {totalCount > 0 && (
          <span className="text-sm text-muted">
            {syncCount}/{totalCount} {dt.statusInSync}
          </span>
        )}
      </div>

      {statusList.length === 0 ? (
        <div className="p-12 text-center">
          <Activity className="w-12 h-12 mx-auto mb-3 text-muted opacity-30" />
          <p className="text-muted">{dt.noStatus}</p>
          <p className="text-xs text-muted mt-1">{dt.noStatusHint}</p>
        </div>
      ) : (
        <div className="divide-y divide-[var(--border-color)]">
          {pagedList.map((item) => (
            <div
              key={item.path}
              className="flex items-center justify-between px-6 py-3 hover:bg-[var(--bg-secondary)] transition-colors"
            >
              <div className="flex items-center gap-3 min-w-0 flex-1">
                {item.inSync ? (
                  <CheckCircle className="w-5 h-5 text-emerald-500 flex-shrink-0" />
                ) : (
                  <AlertTriangle className="w-5 h-5 text-amber-500 flex-shrink-0" />
                )}
                <div className="min-w-0">
                  <code className="text-sm text-default">{item.path}</code>
                  <div className="flex items-center gap-2 mt-0.5">
                    <span className="text-xs px-1.5 py-0.5 rounded-full bg-violet-100 dark:bg-violet-900/30 text-violet-700 dark:text-violet-300">
                      {item.namespace}
                    </span>
                    <span className="text-xs text-muted">
                      {item.resourceCount} {dt.resourceCount}
                    </span>
                  </div>
                </div>
              </div>

              <div className="flex items-center gap-3 flex-shrink-0">
                <span
                  className={`text-sm font-medium ${
                    item.inSync
                      ? "text-emerald-600 dark:text-emerald-400"
                      : "text-amber-600 dark:text-amber-400"
                  }`}
                >
                  {item.inSync ? dt.statusInSync : dt.statusOutOfSync}
                </span>
                {!item.inSync && onSyncNow && (
                  <button
                    onClick={() => onSyncNow(item.path)}
                    className="flex items-center gap-1 px-2.5 py-1 text-xs rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors"
                  >
                    <RefreshCw className="w-3 h-3" />
                    {dt.syncNow}
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
      <Pagination
        page={page}
        pageSize={STATUS_PAGE_SIZE}
        total={totalCount}
        onPageChange={setPage}
        labels={t.table}
      />
    </div>
  );
}
