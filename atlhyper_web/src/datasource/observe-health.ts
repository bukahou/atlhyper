/**
 * Observe Landing Page 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 */

import { getDataSourceMode } from "@/config/data-source";
import { mockGetObserveHealth } from "@/mock/observe";
import * as observeApi from "@/api/observe-health";
import type { LandingPageResponse } from "@/types/model/observe";

type TimeRange = "15m" | "1d" | "7d" | "30d";

export async function getObserveHealth(
  clusterId?: string,
  timeRange?: TimeRange,
): Promise<LandingPageResponse> {
  if (getDataSourceMode("observeHealth") === "mock" || !clusterId) {
    return mockGetObserveHealth();
  }
  const res = await observeApi.getObserveHealth(clusterId, timeRange);
  return res.data.data;
}
