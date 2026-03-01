"use client";

import { useState, useEffect, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { getAuditLogs } from "@/api/auth";
import { useAuthStore } from "@/store/authStore";
import { Eye } from "lucide-react";
import type { AuditLogItem } from "@/types/auth";
import { UserRole } from "@/types/auth";
import { MOCK_AUDIT_LOGS } from "./components/mock-data";
import { AuditItem } from "./components/AuditItem";
import { AuditFilterBar } from "./components/AuditFilterBar";
import type { FilterResult } from "./components/AuditFilterBar";

export default function AuditPage() {
  const { t } = useI18n();
  const auditT = t.audit;
  const { isAuthenticated, user } = useAuthStore();
  const [logs, setLogs] = useState<AuditLogItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // 判断是否有权限查看真实数据（需要 Operator 权限）
  const hasPermission = isAuthenticated && user && user.role >= UserRole.OPERATOR;
  const isDemo = !hasPermission;

  // 过滤器状态
  const [timeRange, setTimeRange] = useState(24); // 默认 24 小时
  const [filterUser, setFilterUser] = useState("");
  const [filterResult, setFilterResult] = useState<FilterResult>("all");

  const fetchLogs = useCallback(async () => {
    // 无权限时使用 mock 数据
    if (!hasPermission) {
      setLogs(MOCK_AUDIT_LOGS);
      setLoading(false);
      return;
    }

    try {
      const res = await getAuditLogs();
      setLogs(res.data.data || []);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.error);
    } finally {
      setLoading(false);
    }
  }, [t.common.error, hasPermission]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  // 应用过滤器
  const filteredLogs = logs.filter((log) => {
    // 时间范围过滤
    if (timeRange > 0) {
      const logTime = new Date(log.timestamp).getTime();
      const cutoff = Date.now() - timeRange * 60 * 60 * 1000;
      if (logTime < cutoff) return false;
    }

    // 用户过滤
    if (filterUser && !log.username.toLowerCase().includes(filterUser.toLowerCase())) {
      return false;
    }

    // 结果过滤
    if (filterResult === "success" && !log.success) return false;
    if (filterResult === "failed" && log.success) return false;

    return true;
  });

  // 统计
  const stats = {
    total: filteredLogs.length,
    success: filteredLogs.filter((l) => l.success).length,
    failed: filteredLogs.filter((l) => !l.success).length,
  };

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.audit} description={auditT.description} />

        {/* 演示模式提示 */}
        {isDemo && (
          <div className="flex items-center gap-3 p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-xl">
            <Eye className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0" />
            <div>
              <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
                {t.common.demoMode}
              </p>
              <p className="text-xs text-amber-600 dark:text-amber-400">
                {t.common.demoModeHintAudit}
              </p>
            </div>
          </div>
        )}

        {/* 过滤器 */}
        <AuditFilterBar
          auditT={auditT}
          commonT={t.common}
          timeRange={timeRange}
          onTimeRangeChange={setTimeRange}
          filterUser={filterUser}
          onFilterUserChange={setFilterUser}
          filterResult={filterResult}
          onFilterResultChange={setFilterResult}
          stats={stats}
        />

        {/* 审计日志列表 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] p-6">
          {loading ? (
            <div className="py-12">
              <LoadingSpinner />
            </div>
          ) : error ? (
            <div className="text-center py-12 text-red-500">{error}</div>
          ) : filteredLogs.length === 0 ? (
            <div className="text-center py-12 text-muted">{auditT.noRecords}</div>
          ) : (
            <div className="space-y-0">
              {filteredLogs.map((log) => (
                <AuditItem key={log.id} log={log} auditT={auditT} />
              ))}
            </div>
          )}
        </div>
      </div>
    </Layout>
  );
}
