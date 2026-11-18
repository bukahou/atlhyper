/**
 * SLO 数据源代理
 *
 * 根据中心配置自动切换 mock / api。约定同 datasource/cluster.ts：
 * - 读函数返回解包后的数据，mock 分支返回相同形态
 * - 读函数最后一个参数是可选的 AbortSignal，原样交给 api 层，用于取消请求
 */

import { getDataSourceMode } from "@/config/data-source";
import * as mock from "@/mock/slo";
import * as api from "@/api/slo";
import type {
  DomainSLOHistoryResponse,
  DomainSLOListResponseV2,
  LatencyDistributionResponse,
  SLODomainsParams,
  SLODomainHistoryParams,
  SLOLatencyParams,
} from "@/types/slo";

/** 域名 SLO 清单（按真实域名分组）+ 汇总 */
export async function getSLODomains(params: SLODomainsParams, signal?: AbortSignal): Promise<DomainSLOListResponseV2> {
  if (getDataSourceMode("slo") === "mock") return mock.mockGetSLODomainsV2(params.timeRange);
  return (await api.getSLODomainsV2(params, signal)).data;
}

/** 某个域名的 SLO 历史（预算与延迟的趋势图用） */
export async function getSLODomainHistory(
  params: SLODomainHistoryParams,
  signal?: AbortSignal,
): Promise<DomainSLOHistoryResponse> {
  if (getDataSourceMode("slo") === "mock") return mock.mockGetSLODomainHistory(params.host, params.timeRange);
  return (await api.getSLODomainHistory(params, signal)).data;
}

/** 某个域名的延迟分布（桶 + 方法分布 + 状态码分布） */
export async function getSLOLatencyDistribution(
  params: SLOLatencyParams,
  signal?: AbortSignal,
): Promise<LatencyDistributionResponse> {
  if (getDataSourceMode("slo") === "mock") return mock.mockGetSLOLatencyDistribution(params.domain, params.timeRange);
  return (await api.getSLOLatencyDistribution(params, signal)).data;
}

// ============================================================
// 写操作 —— 一律走真实接口（没有演示数据）
// ============================================================

export { upsertSLOTarget } from "@/api/slo";
