/**
 * Event API
 *
 * 后端已返回 camelCase 响应，前端直接使用
 */

import { get } from "./request";
import type { components, operations } from "@/types/api.gen";
import type { EventOverview, EventLog } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

/** 查询参数：由规格生成，后端不认的参数这里也没有 */
type EventListParams = operations["listEvents"]["parameters"]["query"];

// ============================================================
// 响应类型（后端已返回 camelCase）
// ============================================================

type EventListResponse = components["schemas"]["EventListResponse"];

// ============================================================
// API 方法
// ============================================================

/**
 * 获取事件列表（实时，从 DataHub）
 * GET /api/v2/events?cluster_id=xxx&namespace=xxx&type=xxx
 */
// 按资源查事件（K8s 原生事件时间线，来自 /events/by-resource）
// 用于资源详情页：某个 Pod/Deployment 最近发生了什么
export function getEventsByResource(
  params: { cluster_id: string; kind: string; namespace?: string; name: string },
  signal?: AbortSignal,
) {
  return get<{ events: EventLog[] }>("/api/v2/events/by-resource", { ...params }, signal);
}

export function getEventList(params: EventListParams, signal?: AbortSignal) {
  return get<EventListResponse>("/api/v2/events", params, signal);
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
export async function getEventOverview(
  data: { ClusterID: string; Type?: string },
  signal?: AbortSignal,
) {
  const response = await getEventList(
    {
      cluster_id: data.ClusterID,
      type: data.Type,
    },
    signal,
  );

  return {
    ...response,
    data: {
      data: { cards: response.data.cards, rows: response.data.events || [] } satisfies EventOverview,
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

  return {
    ...response,
    data: {
      data: { cards: response.data.cards, rows: response.data.events || [] } satisfies EventOverview,
    },
  };
}

/** @deprecated 使用 getAlertOverview 替代 */
export function getEventLogs(data: { ClusterID: string; Namespace?: string; Type?: string }) {
  return getAlertOverview(data);
}
