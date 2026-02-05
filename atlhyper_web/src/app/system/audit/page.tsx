"use client";

import { useState, useEffect, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { getAuditLogs } from "@/api/auth";
import { useAuthStore } from "@/store/authStore";
import { User, Clock, Activity, CheckCircle, XCircle, Filter, Eye } from "lucide-react";
import type { AuditLogItem } from "@/types/auth";
import type { AuditTranslations } from "@/types/i18n";
import { UserRole } from "@/types/auth";

// Mock 数据（展示用）
const MOCK_AUDIT_LOGS: AuditLogItem[] = [
  {
    id: 1,
    timestamp: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
    userId: 1,
    username: "admin",
    role: UserRole.ADMIN,
    source: "web",
    action: "login",
    resource: "user",
    method: "POST",
    success: true,
    ip: "192.168.1.100",
    status: 200,
    durationMs: 125,
  },
  {
    id: 2,
    timestamp: new Date(Date.now() - 1000 * 60 * 15).toISOString(),
    userId: 2,
    username: "operator",
    role: UserRole.OPERATOR,
    source: "web",
    action: "execute",
    resource: "deployment",
    method: "POST",
    success: true,
    ip: "192.168.1.101",
    status: 200,
    durationMs: 342,
  },
  {
    id: 3,
    timestamp: new Date(Date.now() - 1000 * 60 * 30).toISOString(),
    userId: 3,
    username: "viewer",
    role: UserRole.VIEWER,
    source: "api",
    action: "read",
    resource: "pod",
    method: "GET",
    success: false,
    ip: "192.168.1.102",
    status: 403,
    durationMs: 15,
  },
  {
    id: 4,
    timestamp: new Date(Date.now() - 1000 * 60 * 45).toISOString(),
    userId: 1,
    username: "admin",
    role: UserRole.ADMIN,
    source: "web",
    action: "create",
    resource: "user",
    method: "POST",
    success: true,
    ip: "192.168.1.100",
    status: 201,
    durationMs: 89,
  },
  {
    id: 5,
    timestamp: new Date(Date.now() - 1000 * 60 * 60).toISOString(),
    userId: 2,
    username: "operator",
    role: UserRole.OPERATOR,
    source: "api",
    action: "execute",
    resource: "pod",
    method: "POST",
    success: true,
    ip: "192.168.1.101",
    status: 200,
    durationMs: 1250,
  },
  {
    id: 6,
    timestamp: new Date(Date.now() - 1000 * 60 * 90).toISOString(),
    userId: 0,
    username: "guest",
    role: 0,
    source: "web",
    action: "login",
    resource: "user",
    method: "POST",
    success: false,
    ip: "10.0.0.50",
    status: 401,
    durationMs: 45,
  },
];

// 获取角色标签
function getRoleLabel(role: number, auditT: AuditTranslations): string {
  if (role === 0) return auditT.roles.guest;
  if (role === UserRole.VIEWER) return auditT.roles.viewer;
  if (role === UserRole.OPERATOR) return auditT.roles.operator;
  if (role === UserRole.ADMIN) return auditT.roles.admin;
  return `Role ${role}`;
}

// 获取资源标签
function getResourceLabel(resource: string, auditT: AuditTranslations): string {
  const key = resource as keyof typeof auditT.resources;
  return auditT.resources[key] || resource;
}

// 获取操作的显示标签
function getActionLabel(action: string, resource: string, auditT: AuditTranslations): string {
  // 构造 actionLabels 的键名，例如 login + user → loginUser
  const labelKey = `${action}${resource.charAt(0).toUpperCase()}${resource.slice(1)}` as keyof typeof auditT.actionLabels;
  const label = auditT.actionLabels[labelKey];
  if (label) return label;

  // 回退：显示 action + resource
  const resourceName = getResourceLabel(resource, auditT);
  const actionKey = action as keyof typeof auditT.actionNames;
  const actionName = auditT.actionNames[actionKey] || action;
  return `${actionName} ${resourceName}`;
}

// 单条审计记录组件
function AuditItem({ log, auditT }: { log: AuditLogItem; auditT: AuditTranslations }) {
  const resultStyle = log.success
    ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
    : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400";

  const actionLabel = getActionLabel(log.action, log.resource, auditT);
  const roleLabel = getRoleLabel(log.role, auditT);
  const resourceLabel = getResourceLabel(log.resource, auditT);

  return (
    <div className="flex gap-4">
      <div className="flex flex-col items-center">
        <div className={`w-3 h-3 rounded-full flex-shrink-0 ${log.success ? "bg-green-500" : "bg-red-500"}`} />
        <div className="w-px flex-1 bg-[var(--border-color)]" />
      </div>
      <div className="flex-1 pb-6">
        <div className="flex items-center gap-4 mb-2 flex-wrap">
          <div className="flex items-center gap-2">
            <User className="w-4 h-4 text-gray-400" />
            <span className="font-medium text-default">{log.username}</span>
            <span className="text-xs text-muted px-1.5 py-0.5 bg-[var(--background)] rounded">
              {roleLabel}
            </span>
          </div>
          <span className={`inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full ${resultStyle}`}>
            {log.success ? (
              <CheckCircle className="w-3 h-3" />
            ) : (
              <XCircle className="w-3 h-3" />
            )}
            {log.success ? auditT.successOnly : auditT.failedOnly}
          </span>
        </div>

        <div className="flex items-center gap-2 mb-2">
          <Activity className="w-4 h-4 text-primary" />
          <span className="text-default font-medium">{actionLabel}</span>
          <span className="text-xs px-1.5 py-0.5 rounded bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
            {resourceLabel}
          </span>
          {log.status > 0 && (
            <span className={`text-xs px-1.5 py-0.5 rounded font-mono ${
              log.status >= 400 ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400" : "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300"
            }`}>
              {log.status}
            </span>
          )}
        </div>

        <div className="flex items-center gap-4 text-sm text-gray-500 flex-wrap">
          <div className="flex items-center gap-1">
            <Clock className="w-3 h-3" />
            {new Date(log.timestamp).toLocaleString()}
          </div>
          <span>IP: {log.ip}</span>
        </div>
      </div>
    </div>
  );
}

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
  const [filterResult, setFilterResult] = useState<"all" | "success" | "failed">("all");

  // 时间范围选项
  const timeRanges = [
    { label: auditT.lastHour, value: 1 },
    { label: auditT.last24Hours, value: 24 },
    { label: auditT.last7Days, value: 168 },
    { label: auditT.allTime, value: 0 },
  ];

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
                {t.locale === "zh" ? "演示模式" : "デモモード"}
              </p>
              <p className="text-xs text-amber-600 dark:text-amber-400">
                {t.locale === "zh"
                  ? "当前展示的是示例数据。登录并获得 Operator 权限后可查看真实审计日志。"
                  : "サンプルデータを表示中です。Operator 権限でログインすると実際の監査ログを確認できます。"}
              </p>
            </div>
          </div>
        )}

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
                <AuditItem key={log.id} log={log} auditT={auditT} />
              ))}
            </div>
          )}
        </div>
      </div>
    </Layout>
  );
}
