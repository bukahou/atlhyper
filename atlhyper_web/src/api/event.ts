/**
 * Event API
 *
 * 适配 Master V2 API
 */

import { get } from "./request";
import type { EventOverview, EventLog } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

interface EventListParams {
  cluster_id: string;
  namespace?: string;
  type?: string; // Normal, Warning
  reason?: string;
  involved_kind?: string; // Pod, Node, Deployment 等
  involved_name?: string;
  since?: string; // RFC3339 格式
  limit?: number;
  offset?: number;
}

// ============================================================
// 响应类型（后端返回 snake_case）
// ============================================================

// 后端返回的 involved_object 结构
interface InvolvedObjectApi {
  kind: string;
  namespace: string;
  name: string;
  uid: string;
}

// 后端返回的原始格式（Master V2 API）
interface EventApiItem {
  uid: string;
  name: string;
  namespace: string;
  kind: string;
  created_at: string;
  type: string;  // "Warning" | "Normal"
  reason: string;
  message: string;
  source: string;
  involved_object?: InvolvedObjectApi;
  count: number;
  first_timestamp: string;
  last_timestamp: string;
}

interface EventListResponse {
  message?: string;
  events: EventApiItem[];  // 后端返回 events 而非 data
  source?: string;
  total: number;
}

// ============================================================
// 数据转换
// ============================================================

/**
 * 将后端 API 返回的事件数据转换为前端类型
 */
function transformEventItem(apiItem: EventApiItem, clusterID: string): EventLog {
  // type: "Warning" -> severity: "warning", "Normal" -> "info"
  let severity = "info";
  if (apiItem.type === "Warning") {
    severity = "warning";
  } else if (apiItem.type === "Error") {
    severity = "error";
  }

  return {
    ClusterID: clusterID,
    Category: apiItem.reason || "",
    EventTime: apiItem.last_timestamp || apiItem.created_at || "",
    Kind: apiItem.involved_object?.kind || "Event",
    Message: apiItem.message || "",
    Name: apiItem.involved_object?.name || apiItem.name || "",
    Namespace: apiItem.namespace || "",
    Node: apiItem.source || "",
    Reason: apiItem.reason || "",
    Severity: severity,
    Time: apiItem.first_timestamp || apiItem.created_at || "",
  };
}

/**
 * 将事件列表转换为 EventOverview 格式
 */
function transformToEventOverview(apiEvents: EventApiItem[], clusterID: string): EventOverview {
  const events = apiEvents.map((e) => transformEventItem(e, clusterID));

  const kindsSet = new Set<string>();
  const categoriesSet = new Set<string>();
  let warning = 0;
  let error = 0;
  let info = 0;

  for (const e of events) {
    if (e.Kind) kindsSet.add(e.Kind);
    if (e.Reason) categoriesSet.add(e.Reason);

    // 根据 Severity 统计
    const severity = e.Severity?.toLowerCase() || "";
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
 * 只返回 Warning 类型事件
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

  const apiEvents = response.data.events || [];
  const overview = transformToEventOverview(apiEvents, data.ClusterID);

  return {
    ...response,
    data: {
      data: overview,
    },
  };
}

/**
 * 获取告警概览（历史，从数据库）
 * 用于告警页面，只显示 Warning 事件
 */
export async function getAlertOverview(data: { ClusterID: string }) {
  const response = await getAlertList({
    cluster_id: data.ClusterID,
  });

  const apiEvents = response.data.events || [];
  const overview = transformToEventOverview(apiEvents, data.ClusterID);

  return {
    ...response,
    data: {
      data: overview,
    },
  };
}

// ============================================================
// 兼容旧接口
// ============================================================

/** @deprecated 使用 getAlertOverview 替代 */
export function getEventLogs(data: { ClusterID: string; Namespace?: string; Type?: string }) {
  return getAlertOverview(data);
}
