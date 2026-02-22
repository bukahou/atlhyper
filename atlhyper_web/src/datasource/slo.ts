/**
 * SLO 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 */

import { getDataSourceMode } from "@/config/data-source";
import * as mock from "@/mock/slo";
import * as api from "@/api/slo";
import type {
  SLODomainsParams,
  SLODomainHistoryParams,
  SLOLatencyParams,
} from "@/types/slo";

export async function getSLODomainsV2(params?: SLODomainsParams) {
  if (getDataSourceMode("slo") === "mock") {
    return { data: mock.mockGetSLODomainsV2(params?.timeRange) };
  }
  return api.getSLODomainsV2(params);
}

export async function getSLODomainHistory(params: SLODomainHistoryParams) {
  if (getDataSourceMode("slo") === "mock") {
    return { data: mock.mockGetSLODomainHistory(params.host, params.timeRange) };
  }
  return api.getSLODomainHistory(params);
}

export async function getSLOLatencyDistribution(params: SLOLatencyParams) {
  if (getDataSourceMode("slo") === "mock") {
    return { data: mock.mockGetSLOLatencyDistribution(params.domain, params.timeRange) };
  }
  return api.getSLOLatencyDistribution(params);
}
