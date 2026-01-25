"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Layout } from "@/components/layout/Layout";
import { PageHeader, StatusPage } from "@/components/common";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import { getClusterOverview } from "@/api/overview";
import type { RecentAlert } from "@/types/overview";
import {
  AlertTriangle,
  AlertCircle,
  Info,
  Bot,
  RefreshCw,
  Inbox,
} from "lucide-react";

// 告警唯一标识
function alertKey(alert: RecentAlert): string {
  return `${alert.kind}/${alert.namespace}/${alert.name}/${alert.reason}/${alert.timestamp}`;
}

// 严重性图标
function SeverityIcon({ severity }: { severity: string }) {
  switch (severity) {
    case "critical":
      return <AlertCircle className="w-4 h-4 text-red-500" />;
    case "warning":
      return <AlertTriangle className="w-4 h-4 text-yellow-500" />;
    default:
      return <Info className="w-4 h-4 text-blue-500" />;
  }
}

// 严重性颜色
function severityColor(severity: string): string {
  switch (severity) {
    case "critical":
      return "text-red-600 bg-red-50 dark:bg-red-900/20";
    case "warning":
      return "text-yellow-600 bg-yellow-50 dark:bg-yellow-900/20";
    default:
      return "text-blue-600 bg-blue-50 dark:bg-blue-900/20";
  }
}

export default function AlertsPage() {
  const { t } = useI18n();
  const router = useRouter();
  const { currentClusterId } = useClusterStore();

  const [alerts, setAlerts] = useState<RecentAlert[]>([]);
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // 加载告警数据
  const loadAlerts = useCallback(async () => {
    if (!currentClusterId) return;

    setLoading(true);
    setError(null);

    try {
      const res = await getClusterOverview({ cluster_id: currentClusterId });
      setAlerts(res.data.data.alerts.recent || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
      setAlerts([]);
    } finally {
      setLoading(false);
    }
  }, [currentClusterId]);

  useEffect(() => {
    loadAlerts();
  }, [loadAlerts]);

  // 切换选择
  const handleToggle = (alert: RecentAlert) => {
    const key = alertKey(alert);
    const newSet = new Set(selectedKeys);
    if (newSet.has(key)) {
      newSet.delete(key);
    } else {
      newSet.add(key);
    }
    setSelectedKeys(newSet);
  };

  // 全选/取消全选
  const handleToggleAll = () => {
    if (selectedKeys.size === alerts.length) {
      setSelectedKeys(new Set());
    } else {
      setSelectedKeys(new Set(alerts.map(alertKey)));
    }
  };

  // 获取选中的告警列表
  const getSelectedAlerts = (): RecentAlert[] => {
    return alerts.filter((a) => selectedKeys.has(alertKey(a)));
  };

  // AI 分析
  const handleAnalyze = () => {
    const selected = getSelectedAlerts();
    if (selected.length === 0) return;

    // 存入 sessionStorage
    sessionStorage.setItem("alertContext", JSON.stringify(selected));
    // 跳转到 AI Chat
    router.push("/workbench/ai?from=alerts");
  };

  // 格式化时间
  const formatTime = (timestamp: string) => {
    try {
      return new Date(timestamp).toLocaleString("zh-CN", {
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      });
    } catch {
      return timestamp;
    }
  };

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title={t.nav.alerts}
          description="查看集群告警事件，支持多选后使用 AI 进行分析"
          actions={
            <div className="flex items-center gap-3">
              <button
                onClick={loadAlerts}
                disabled={loading}
                className="flex items-center gap-2 px-3 py-2 text-sm text-muted hover:text-default transition-colors disabled:opacity-50"
              >
                <RefreshCw
                  className={`w-4 h-4 ${loading ? "animate-spin" : ""}`}
                />
                刷新
              </button>
              <button
                onClick={handleAnalyze}
                disabled={selectedKeys.size === 0}
                className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Bot className="w-4 h-4" />
                AI 分析 ({selectedKeys.size})
              </button>
            </div>
          }
        />

        {/* 错误提示 */}
        {error && (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 text-red-600 dark:text-red-400">
            {error}
          </div>
        )}

        {/* 告警表格 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          {loading ? (
            <div className="p-12 flex items-center justify-center">
              <RefreshCw className="w-6 h-6 text-muted animate-spin" />
            </div>
          ) : alerts.length === 0 ? (
            <StatusPage
              icon={Inbox}
              title="暂无告警"
              description="当前集群运行正常，没有告警事件"
            />
          ) : (
            <table className="w-full">
              <thead>
                <tr className="bg-secondary/50 border-b border-[var(--border-color)]">
                  <th className="p-3 w-10">
                    <input
                      type="checkbox"
                      checked={
                        selectedKeys.size === alerts.length &&
                        alerts.length > 0
                      }
                      onChange={handleToggleAll}
                      className="w-4 h-4 rounded border-gray-300 text-emerald-600 focus:ring-emerald-500"
                    />
                  </th>
                  <th className="p-3 text-left text-sm font-medium text-muted">
                    时间
                  </th>
                  <th className="p-3 text-left text-sm font-medium text-muted">
                    级别
                  </th>
                  <th className="p-3 text-left text-sm font-medium text-muted">
                    资源类型
                  </th>
                  <th className="p-3 text-left text-sm font-medium text-muted">
                    命名空间
                  </th>
                  <th className="p-3 text-left text-sm font-medium text-muted">
                    资源名称
                  </th>
                  <th className="p-3 text-left text-sm font-medium text-muted">
                    原因
                  </th>
                  <th className="p-3 text-left text-sm font-medium text-muted">
                    消息
                  </th>
                </tr>
              </thead>
              <tbody>
                {alerts.map((alert, idx) => {
                  const key = alertKey(alert);
                  const isSelected = selectedKeys.has(key);

                  return (
                    <tr
                      key={idx}
                      className={`border-b border-[var(--border-color)] hover:bg-secondary/30 transition-colors cursor-pointer ${
                        isSelected ? "bg-emerald-50/50 dark:bg-emerald-900/10" : ""
                      }`}
                      onClick={() => handleToggle(alert)}
                    >
                      <td className="p-3">
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => handleToggle(alert)}
                          onClick={(e) => e.stopPropagation()}
                          className="w-4 h-4 rounded border-gray-300 text-emerald-600 focus:ring-emerald-500"
                        />
                      </td>
                      <td className="p-3 text-sm text-muted whitespace-nowrap">
                        {formatTime(alert.timestamp)}
                      </td>
                      <td className="p-3">
                        <span
                          className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium ${severityColor(
                            alert.severity
                          )}`}
                        >
                          <SeverityIcon severity={alert.severity} />
                          {alert.severity}
                        </span>
                      </td>
                      <td className="p-3 text-sm text-default font-mono">
                        {alert.kind}
                      </td>
                      <td className="p-3 text-sm text-default font-mono">
                        {alert.namespace || "-"}
                      </td>
                      <td className="p-3 text-sm text-default font-mono max-w-[200px] truncate">
                        {alert.name}
                      </td>
                      <td className="p-3 text-sm text-default">
                        {alert.reason}
                      </td>
                      <td className="p-3 text-sm text-muted max-w-[300px] truncate">
                        {alert.message}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </div>

        {/* 底部说明 */}
        {alerts.length > 0 && (
          <p className="text-sm text-muted">
            共 {alerts.length} 条告警，已选择 {selectedKeys.size} 条。
            选择告警后点击 "AI 分析" 按钮，AI 将查询相关资源状态、日志和事件进行诊断。
          </p>
        )}
      </div>
    </Layout>
  );
}
