/**
 * Observe Landing Page API
 */

import { get } from "./request";
import type { ObserveResponse } from "./observe-common";
import type { LandingPageResponse } from "@/types/model/observe";

export type HealthTimeRange = "15m" | "1d" | "7d" | "30d";

/** 获取服务健康总览（Landing Page） */
export function getObserveHealth(clusterId: string, timeRange?: HealthTimeRange) {
  return get<ObserveResponse<LandingPageResponse>>("/api/v2/observe/health", {
    cluster_id: clusterId,
    ...(timeRange ? { time_range: timeRange } : {}),
  });
}
