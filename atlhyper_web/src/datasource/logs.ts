/**
 * Logs 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 */

import { getDataSourceMode } from "@/config/data-source";
import { mockQueryLogs } from "@/mock/logs";
import type { MockLogQueryParams } from "@/mock/logs";

export type { MockLogQueryParams } from "@/mock/logs";

export function queryLogs(params?: MockLogQueryParams) {
  if (getDataSourceMode("logs") === "mock") return mockQueryLogs(params);
  // TODO: 真实 API 就绪后替换
  return mockQueryLogs(params);
}
