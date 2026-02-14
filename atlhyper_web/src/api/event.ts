/**
 * Event API
 *
 * 后端已返回 camelCase 响应，前端直接使用
 */

import { get } from "./request";
import type { EventOverview, EventLog, EventCards } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

interface EventListParams {
  cluster_id: string;
  namespace?: string;
  type?: string; // Normal, Warning
  reason?: string;
  involved_kind?: string;
  involved_name?: string;
  since?: string; // RFC3339 格式
  limit?: number;
  offset?: number;
  source?: string;
}

// ============================================================
// 响应类型（后端已返回 camelCase）
// ============================================================

interface EventListResponse {
  events: EventLog[];
  total: number;
  source?: string;
  limit?: number;
  offset?: number;
}

// ============================================================
// 聚合工具（从事件列表生成概览统计）
// ============================================================

function aggregateOverview(events: EventLog[]): EventOverview {
  const kindsSet = new Set<string>();
  const categoriesSet = new Set<string>();
  let warning = 0;
  let error = 0;
  let info = 0;

  for (const e of events) {
    if (e.kind) kindsSet.add(e.kind);
    if (e.reason) categoriesSet.add(e.reason);

    const severity = e.severity?.toLowerCase() || "";
    if (severity === "warning") {
      warning++;
    } else if (severity === "error") {
      error++;
    } else {
      info++;
    }
  }

  return {
    cards: {
      totalAlerts: warning + error,
      totalEvents: events.length,
      warning,
      error,
      info,
      kindsCount: kindsSet.size,
      categoriesCount: categoriesSet.size,
    },
    rows: events,
  };
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取事件列表（实时，从 DataHub）
 * GET /api/v2/events?cluster_id=xxx&namespace=xxx&type=xxx
 */
export function getEventList(params: EventListParams) {
  return get<EventListResponse>("/api/v2/events", params);
}

/**
 * 获取告警列表（历史，从数据库）
 * GET /api/v2/events?cluster_id=xxx&source=history
 */
export function getAlertList(params: EventListParams) {
  return get<EventListResponse>("/api/v2/events", {
    ...params,
    source: "history",
  });
}

/**
 * 获取事件概览（实时，从 DataHub）
 */
export async function getEventOverview(data: { ClusterID: string; Namespace?: string; Type?: string }) {
  const response = await getEventList({
    cluster_id: data.ClusterID,
    namespace: data.Namespace,
    type: data.Type,
  });

  const events = response.data.events || [];

  return {
    ...response,
    data: {
      data: aggregateOverview(events),
    },
  };
}

/**
 * 获取告警概览（历史，从数据库）
 */
export async function getAlertOverview(data: { ClusterID: string }) {
  const response = await getAlertList({
    cluster_id: data.ClusterID,
  });

  const events = response.data.events || [];

  return {
    ...response,
    data: {
      data: aggregateOverview(events),
    },
  };
}

/** @deprecated 使用 getAlertOverview 替代 */
export function getEventLogs(data: { ClusterID: string; Namespace?: string; Type?: string }) {
  return getAlertOverview(data);
}
