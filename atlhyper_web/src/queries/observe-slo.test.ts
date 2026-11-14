import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const datasource = vi.hoisted(() => ({
  getSLODomains: vi.fn(),
  getSLODomainHistory: vi.fn(),
  getSLOLatencyDistribution: vi.fn(),
  upsertSLOTarget: vi.fn(),
}));
const dataSourceMode = vi.hoisted(() => ({ current: "api" }));
vi.mock("@/datasource/slo", () => datasource);
vi.mock("@/config/data-source", () => ({ getDataSourceMode: () => dataSourceMode.current }));

import { sloKeys } from "./keys";
import {
  sloDomainHistoryQuery,
  sloDomainsQuery,
  sloLatencyDistributionQuery,
  upsertSLOTargetMutation,
} from "./observe-slo";
import { REFRESH } from "./refresh";

beforeEach(() => {
  for (const reader of Object.values(datasource)) reader.mockReset();
  dataSourceMode.current = "api";
});

describe("sloDomainsQuery", () => {
  it("用传入的集群取数；没有选定集群时不启用，演示数据不依赖集群", () => {
    expect(sloDomainsQuery("", "1h").enabled).toBe(false);
    expect(sloDomainsQuery("c1", "1h").enabled).toBe(true);

    dataSourceMode.current = "mock";
    expect(sloDomainsQuery("", "1h").enabled).toBe(true);
  });

  it("键里带集群、数据源与窗口：换集群、换窗口都是另一份数据", () => {
    expect(sloDomainsQuery("c1", "1h").queryKey).toEqual(["slo", "domains", { clusterId: "c1", mode: "api", window: "1h" }]);
    expect(sloDomainsQuery("c1", "1h").queryKey).not.toEqual(sloDomainsQuery("c2", "1h").queryKey);
    expect(sloDomainsQuery("c1", "1h").queryKey).not.toEqual(sloDomainsQuery("c1", "6h").queryKey);
    expect(sloDomainsQuery("c1", "1h").queryKey).toEqual(sloDomainsQuery("c1", "1h").queryKey);
  });

  it("把集群、窗口与取消信号交给数据源", async () => {
    const list = { domains: [], summary: null };
    datasource.getSLODomains.mockResolvedValue(list);

    expect(await new QueryClient().fetchQuery(sloDomainsQuery("c1", "6h"))).toBe(list);

    const [params, signal] = datasource.getSLODomains.mock.calls[0];
    expect(params).toEqual({ clusterId: "c1", timeRange: "6h" });
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("30 秒轮询，保鲜时间等长", () => {
    const options = sloDomainsQuery("c1", "1h");
    expect(options.refetchInterval).toBe(REFRESH.slow);
    expect(options.staleTime).toBe(REFRESH.slow);
  });
});

describe("sloDomainHistoryQuery / sloLatencyDistributionQuery", () => {
  it("没有集群或域名为空时不启用", () => {
    expect(sloDomainHistoryQuery("", "a.example.com", "1h").enabled).toBe(false);
    expect(sloDomainHistoryQuery("c1", "", "1h").enabled).toBe(false);
    expect(sloDomainHistoryQuery("c1", "a.example.com", "1h").enabled).toBe(true);
    expect(sloLatencyDistributionQuery("", "a.example.com", "1h").enabled).toBe(false);
    expect(sloLatencyDistributionQuery("c1", "", "1h").enabled).toBe(false);
    expect(sloLatencyDistributionQuery("c1", "a.example.com", "1h").enabled).toBe(true);
  });

  it("键里带域名与窗口：换窗口时各用各的缓存", () => {
    expect(sloDomainHistoryQuery("c1", "a.example.com", "1h").queryKey).toEqual([
      "slo",
      "domain-history",
      { clusterId: "c1", mode: "api", host: "a.example.com", window: "1h" },
    ]);
    expect(sloLatencyDistributionQuery("c1", "a.example.com", "24h").queryKey).toEqual([
      "slo",
      "latency",
      { clusterId: "c1", mode: "api", domain: "a.example.com", window: "24h" },
    ]);
  });

  it("把取消信号交给数据源", async () => {
    datasource.getSLODomainHistory.mockResolvedValue({ host: "a.example.com", history: [] });
    datasource.getSLOLatencyDistribution.mockResolvedValue({ domain: "a.example.com", buckets: [] });
    const client = new QueryClient();

    await client.fetchQuery(sloDomainHistoryQuery("c1", "a.example.com", "1h"));
    await client.fetchQuery(sloLatencyDistributionQuery("c1", "a.example.com", "1h"));

    const [historyParams, historySignal] = datasource.getSLODomainHistory.mock.calls[0];
    expect(historyParams).toEqual({ clusterId: "c1", host: "a.example.com", timeRange: "1h" });
    expect(historySignal).toBeInstanceOf(AbortSignal);
    const [latencyParams, latencySignal] = datasource.getSLOLatencyDistribution.mock.calls[0];
    expect(latencyParams).toEqual({ clusterId: "c1", domain: "a.example.com", timeRange: "1h" });
    expect(latencySignal).toBeInstanceOf(AbortSignal);
  });
});

describe("upsertSLOTargetMutation", () => {
  const target = { host: "a.example.com", windowDays: 7, availabilityTarget: 99.9, p95LatencyTarget: 300 };

  it("保存到当前集群；成功后让全部 SLO 查询重新取数（目标变了，预算与状态都跟着变）", async () => {
    datasource.upsertSLOTarget.mockResolvedValue({ data: { status: "ok" } });
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");

    await client.getMutationCache().build(client, upsertSLOTargetMutation("c1")).execute(target);

    expect(datasource.upsertSLOTarget).toHaveBeenCalledWith({ clusterId: "c1", ...target });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: sloKeys.all });
  });

  it("保存失败时不刷新", async () => {
    datasource.upsertSLOTarget.mockRejectedValue(new Error("forbidden"));
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");

    await expect(
      client.getMutationCache().build(client, upsertSLOTargetMutation("c1")).execute(target),
    ).rejects.toThrow("forbidden");
    expect(invalidate).not.toHaveBeenCalled();
  });
});
