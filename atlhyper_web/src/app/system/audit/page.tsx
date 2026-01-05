"use client";

import { useState, useEffect, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { getAuditLogs } from "@/api/auth";
import { User, Clock, Activity, CheckCircle, XCircle, Filter } from "lucide-react";
import type { AuditLogItem } from "@/types/auth";
import type { AuditTranslations } from "@/types/i18n";
import { UserRole } from "@/types/auth";

// 角色标签配置
const roleLabels: Record<number, string> = {
  [UserRole.ADMIN]: "Admin",
  [UserRole.OPERATOR]: "Operator",
  [UserRole.VIEWER]: "Viewer",
};

// API 路径到翻译 key 的映射
type ActionKey = keyof AuditTranslations["actions"];
const actionKeyMap: Record<string, ActionKey> = {
  // 认证相关
  "auth.login": "login",
  "auto.uiapi/auth/login": "login",
  // Pod 操作
  "auto.uiapi/ops/pod/restart": "podRestart",
  "auto.uiapi/ops/pod/logs": "podLogs",
  // Node 操作
  "auto.uiapi/ops/node/cordon": "nodeCordon",
  "auto.uiapi/ops/node/uncordon": "nodeUncordon",
  // Workload 操作
  "auto.uiapi/ops/workload/scale": "deploymentScale",
  "auto.uiapi/ops/workload/updateImage": "deploymentUpdateImage",
  // 用户管理
  "auto.uiapi/auth/user/register": "userRegister",
  "auto.uiapi/auth/user/update-role": "userUpdateRole",
  "auto.uiapi/auth/user/delete": "userDelete",
  // 配置管理
  "auto.uiapi/config/slack/update": "slackConfigUpdate",
};

// 获取操作的翻译 key
function getActionKey(action: string): ActionKey {
  return actionKeyMap[action] || "unknown";
}

// 单条审计记录组件
function AuditItem({ log, auditT }: { log: AuditLogItem; auditT: AuditTranslations }) {
  const resultStyle = log.Success
    ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
    : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400";

  const actionKey = getActionKey(log.Action);
  const actionLabel = auditT.actions[actionKey];
  const roleLabel = roleLabels[log.Role] || `Role ${log.Role}`;

  return (
    <div className="flex gap-4">
      <div className="flex flex-col items-center">
        <div className={`w-3 h-3 rounded-full flex-shrink-0 ${log.Success ? "bg-green-500" : "bg-red-500"}`} />
        <div className="w-px flex-1 bg-[var(--border-color)]" />
      </div>
      <div className="flex-1 pb-6">
        <div className="flex items-center gap-4 mb-2 flex-wrap">
          <div className="flex items-center gap-2">
            <User className="w-4 h-4 text-gray-400" />
            <span className="font-medium text-default">{log.Username}</span>
            <span className="text-xs text-muted px-1.5 py-0.5 bg-[var(--background)] rounded">
              {roleLabel}
            </span>
          </div>
          <span className={`inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full ${resultStyle}`}>
            {log.Success ? (
              <CheckCircle className="w-3 h-3" />
            ) : (
              <XCircle className="w-3 h-3" />
            )}
            {log.Success ? auditT.successOnly : auditT.failedOnly}
          </span>
        </div>

        <div className="flex items-center gap-2 mb-2">
          <Activity className="w-4 h-4 text-primary" />
          <span className="text-default font-medium">{actionLabel}</span>
          {log.Status > 0 && (
            <span className={`text-xs px-1.5 py-0.5 rounded font-mono ${
              log.Status >= 400 ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400" : "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300"
            }`}>
              {log.Status}
            </span>
          )}
        </div>

        <div className="flex items-center gap-4 text-sm text-gray-500 flex-wrap">
          <div className="flex items-center gap-1">
            <Clock className="w-3 h-3" />
            {new Date(log.Timestamp).toLocaleString()}
          </div>
          <span>IP: {log.IP}</span>
        </div>
      </div>
    </div>
  );
}

export default function AuditPage() {
  const { t } = useI18n();
  const auditT = t.audit;
  const [logs, setLogs] = useState<AuditLogItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // 过滤器状态
  const [timeRange, setTimeRange] = useState(24); // 默认 24 小时
  const [filterUser, setFilterUser] = useState("");
  const [filterResult, setFilterResult] = useState<"all" | "success" | "failed">("all");

  // 时间范围选项
  const timeRanges = [
    { label: auditT.lastHour, value: 1 },
    { label: auditT.last24Hours, value: 24 },
    { label: auditT.last7Days, value: 168 },
    { label: auditT.allTime, value: 0 },
  ];

  const fetchLogs = useCallback(async () => {
    try {
      const res = await getAuditLogs();
      setLogs(res.data.data || []);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.error);
    } finally {
      setLoading(false);
    }
  }, [t.common.error]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  // 应用过滤器
  const filteredLogs = logs.filter((log) => {
    // 时间范围过滤
    if (timeRange > 0) {
      const logTime = new Date(log.Timestamp).getTime();
      const cutoff = Date.now() - timeRange * 60 * 60 * 1000;
      if (logTime < cutoff) return false;
    }

    // 用户过滤
    if (filterUser && !log.Username.toLowerCase().includes(filterUser.toLowerCase())) {
      return false;
    }

    // 结果过滤
    if (filterResult === "success" && !log.Success) return false;
    if (filterResult === "failed" && log.Success) return false;

    return true;
  });

  // 统计
  const stats = {
    total: filteredLogs.length,
    success: filteredLogs.filter((l) => l.Success).length,
    failed: filteredLogs.filter((l) => !l.Success).length,
  };

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.audit} description={auditT.description} />

        {/* 过滤器 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
          <div className="flex items-center gap-2 mb-3">
            <Filter className="w-4 h-4 text-muted" />
            <span className="text-sm font-medium text-default">{auditT.filterLabel}</span>
          </div>
          <div className="flex flex-wrap gap-4">
            {/* 时间范围 */}
            <div>
              <label className="block text-xs text-muted mb-1">{auditT.timeRange}</label>
              <select
                value={timeRange}
                onChange={(e) => setTimeRange(Number(e.target.value))}
                className="px-3 py-1.5 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default text-sm focus:ring-2 focus:ring-primary outline-none"
              >
                {timeRanges.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* 用户过滤 */}
            <div>
              <label className="block text-xs text-muted mb-1">{auditT.user}</label>
              <input
                type="text"
                placeholder={t.common.search + "..."}
                value={filterUser}
                onChange={(e) => setFilterUser(e.target.value)}
                className="px-3 py-1.5 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default text-sm focus:ring-2 focus:ring-primary outline-none w-40"
              />
            </div>

            {/* 结果过滤 */}
            <div>
              <label className="block text-xs text-muted mb-1">{auditT.result}</label>
              <select
                value={filterResult}
                onChange={(e) => setFilterResult(e.target.value as typeof filterResult)}
                className="px-3 py-1.5 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default text-sm focus:ring-2 focus:ring-primary outline-none"
              >
                <option value="all">{auditT.all}</option>
                <option value="success">{auditT.successOnly}</option>
                <option value="failed">{auditT.failedOnly}</option>
              </select>
            </div>

            {/* 统计信息 */}
            <div className="flex items-end gap-4 ml-auto text-sm">
              <span className="text-muted">
                {auditT.total} <span className="font-medium text-default">{stats.total}</span>
              </span>
              <span className="text-green-600 dark:text-green-400">
                {auditT.successCount} {stats.success}
              </span>
              <span className="text-red-600 dark:text-red-400">
                {auditT.failedCount} {stats.failed}
              </span>
            </div>
          </div>
        </div>

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
                <AuditItem key={log.ID} log={log} auditT={auditT} />
              ))}
            </div>
          )}
        </div>
      </div>
    </Layout>
  );
}
