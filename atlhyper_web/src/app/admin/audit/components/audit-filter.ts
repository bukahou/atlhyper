import type { AuditLogItem } from "@/types/auth";

export type AuditResultFilter = "all" | "success" | "failed";

/** 审计列表的筛选条件（对已加载的记录在页面上即时筛选） */
export interface AuditFilter {
  /** 只看最近多少小时；0 = 不限 */
  timeRangeHours: number;
  /** 用户名包含这段文字（不分大小写）；空串 = 不限 */
  username: string;
  result: AuditResultFilter;
}

const HOUR_MS = 60 * 60 * 1000;

/**
 * 按筛选条件过滤审计记录。
 *
 * nowMs 由调用方给（记录取回来的时刻）：「最近 N 小时」以它为基准，
 * 同一份数据反复渲染时结果不变。
 */
export function filterAuditLogs(logs: AuditLogItem[], filter: AuditFilter, nowMs: number): AuditLogItem[] {
  const cutoffMs = filter.timeRangeHours > 0 ? nowMs - filter.timeRangeHours * HOUR_MS : null;
  const username = filter.username.toLowerCase();
  return logs.filter((log) => {
    if (cutoffMs !== null && new Date(log.timestamp).getTime() < cutoffMs) return false;
    if (username !== "" && !log.username.toLowerCase().includes(username)) return false;
    if (filter.result === "success" && !log.success) return false;
    if (filter.result === "failed" && log.success) return false;
    return true;
  });
}
