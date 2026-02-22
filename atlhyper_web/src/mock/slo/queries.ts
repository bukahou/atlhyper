/**
 * SLO Mock — 查询函数
 *
 * 模拟 3 个读 API: getSLODomainsV2 / getSLODomainHistory / getSLOLatencyDistribution
 */

import type {
  DomainSLOListResponseV2,
  DomainSLOHistoryResponse,
  LatencyDistributionResponse,
} from "@/types/slo";
import {
  MOCK_DOMAINS,
  MOCK_SUMMARY,
  MOCK_PROVIDERS,
  generateHistory,
  generateLatencyDistribution,
} from "./data";

/** 模拟域名列表 (V2) */
export function mockGetSLODomainsV2(_timeRange?: string): DomainSLOListResponseV2 {
  return {
    domains: MOCK_DOMAINS,
    summary: MOCK_SUMMARY,
    providers: MOCK_PROVIDERS,
  };
}

/** 模拟域名历史数据 */
export function mockGetSLODomainHistory(
  host: string,
  timeRange?: string,
): DomainSLOHistoryResponse {
  return {
    host,
    history: generateHistory(host, timeRange ?? "1d"),
  };
}

/** 模拟延迟分布 */
export function mockGetSLOLatencyDistribution(
  domain: string,
  _timeRange?: string,
): LatencyDistributionResponse {
  const dist = generateLatencyDistribution(domain);
  return {
    domain,
    totalRequests: dist.total,
    p50LatencyMs: dist.p50,
    p95LatencyMs: dist.p95,
    p99LatencyMs: dist.p99,
    avgLatencyMs: dist.avg,
    buckets: dist.buckets,
    methods: dist.methods,
    statusCodes: dist.statusCodes,
  };
}
