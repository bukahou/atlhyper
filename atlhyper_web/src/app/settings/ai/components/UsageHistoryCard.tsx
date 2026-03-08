"use client";

import { useEffect, useState } from "react";
import { Clock, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { getAIReports, type AIReportItem } from "@/api/ai-provider";

const ROLE_COLORS: Record<string, string> = {
  background: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300",
  chat: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300",
  analysis: "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300",
};

const PAGE_SIZE = 10;

function formatDuration(ms: number): string {
  if (ms <= 0) return "-";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function formatTokens(n: number): string {
  if (n <= 0) return "-";
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}

export function UsageHistoryCard() {
  const { t } = useI18n();
  const aiT = t.aiSettingsPage;
  const { isAuthenticated } = useAuthStore();
  const [reports, setReports] = useState<AIReportItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [roleFilter, setRoleFilter] = useState("");

  const fetchReports = async (offset: number, append: boolean) => {
    if (!isAuthenticated) {
      setLoading(false);
      return;
    }
    try {
      const res = await getAIReports({ role: roleFilter || undefined, limit: PAGE_SIZE, offset });
      if (append) {
        setReports((prev) => [...prev, ...res.data.data]);
      } else {
        setReports(res.data.data || []);
      }
      setTotal(res.data.total);
    } catch {
      // ignore
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  };

  useEffect(() => {
    setLoading(true);
    setReports([]);
    fetchReports(0, false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, roleFilter]);

  const handleLoadMore = () => {
    setLoadingMore(true);
    fetchReports(reports.length, true);
  };

  const hasMore = reports.length < total;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-6 py-4 border-b border-[var(--border-color)] flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium text-default">{aiT.usageHistory}</h3>
          <p className="text-sm text-muted mt-1">{aiT.usageHistoryDesc}</p>
        </div>
        <select
          value={roleFilter}
          onChange={(e) => setRoleFilter(e.target.value)}
          className="px-3 py-1.5 text-sm rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-default"
        >
          <option value="">{aiT.allRoles}</option>
          <option value="background">{aiT.roleBackground}</option>
          <option value="chat">{aiT.roleChat}</option>
          <option value="analysis">{aiT.roleAnalysis}</option>
        </select>
      </div>

      <div className="p-6">
        {loading ? (
          <div className="flex justify-center py-8">
            <Loader2 className="w-5 h-5 animate-spin text-muted" />
          </div>
        ) : reports.length === 0 ? (
          <div className="text-center py-8 text-muted">
            <Clock className="w-10 h-10 mx-auto mb-2 opacity-30" />
            <p>{aiT.noReports}</p>
          </div>
        ) : (
          <>
            {/* Table */}
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-xs text-muted border-b border-[var(--border-color)]">
                    <th className="pb-2 pr-3 font-medium">{aiT.trigger}</th>
                    <th className="pb-2 pr-3 font-medium">{aiT.model}</th>
                    <th className="pb-2 pr-3 font-medium text-right">{aiT.inputTokens}</th>
                    <th className="pb-2 pr-3 font-medium text-right">{aiT.outputTokens}</th>
                    <th className="pb-2 pr-3 font-medium text-right">{aiT.duration}</th>
                    <th className="pb-2 font-medium text-right">{t.common.time}</th>
                  </tr>
                </thead>
                <tbody>
                  {reports.map((r) => (
                    <tr key={r.id} className="border-b border-[var(--border-color)] last:border-0">
                      <td className="py-2.5 pr-3">
                        <div className="flex items-center gap-2">
                          <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${ROLE_COLORS[r.role] || "bg-gray-100 text-gray-600"}`}>
                            {r.role}
                          </span>
                          <span className="text-muted text-xs">{r.trigger}</span>
                        </div>
                      </td>
                      <td className="py-2.5 pr-3 text-default">
                        {r.providerName ? `${r.providerName} / ${r.model}` : "-"}
                      </td>
                      <td className="py-2.5 pr-3 text-right tabular-nums">{formatTokens(r.inputTokens)}</td>
                      <td className="py-2.5 pr-3 text-right tabular-nums">{formatTokens(r.outputTokens)}</td>
                      <td className="py-2.5 pr-3 text-right tabular-nums">{formatDuration(r.durationMs)}</td>
                      <td className="py-2.5 text-right text-muted text-xs">
                        {new Date(r.createdAt).toLocaleString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Load More */}
            {hasMore && (
              <div className="mt-4 text-center">
                <button
                  onClick={handleLoadMore}
                  disabled={loadingMore}
                  className="px-4 py-2 text-sm text-violet-600 hover:text-violet-700 disabled:opacity-50"
                >
                  {loadingMore ? (
                    <Loader2 className="w-4 h-4 animate-spin inline mr-1" />
                  ) : null}
                  {aiT.loadMore} ({reports.length}/{total})
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
