import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const overviewSource = vi.hoisted(() => ({ getClusterOverview: vi.fn() }));
const sloSource = vi.hoisted(() => ({ getSLODomains: vi.fn() }));
const dataSourceMode = vi.hoisted(() => ({ current: "api" }));
vi.mock("@/datasource/overview", () => overviewSource);
vi.mock("@/datasource/slo", () => sloSource);
vi.mock("@/config/data-source", () => ({ getDataSourceMode: () => dataSourceMode.current }));

import { clusterOverviewQuery, overviewSLODomainsQuery } from "./overview";
import { sloKeys } from "./keys";
import { REFRESH } from "./refresh";

beforeEach(() => {
  overviewSource.getClusterOverview.mockReset();
  sloSource.getSLODomains.mockReset();
  dataSourceMode.current = "api";
});

describe("clusterOverviewQuery", () => {
  it("没有选定集群时不启用；演示数据不依赖集群", () => {
    expect(clusterOverviewQuery("").enabled).toBe(false);
    expect(clusterOverviewQuery("c1").enabled).toBe(true);

    dataSourceMode.current = "mock";
    expect(clusterOverviewQuery("").enabled).toBe(true);
  });

  it("键里带集群与数据源：切换集群即换数据", () => {
    expect(clusterOverviewQuery("c1").queryKey).toEqual(["overview", "cluster", { clusterId: "c1", mode: "api" }]);
    expect(clusterOverviewQuery("c1").queryKey).not.toEqual(clusterOverviewQuery("c2").queryKey);
  });

  it("把集群与取消信号交给数据源", async () => {
    const overview = { clusterId: "c1" };
    overviewSource.getClusterOverview.mockResolvedValue(overview);

    expect(await new QueryClient().fetchQuery(clusterOverviewQuery("c1"))).toBe(overview);

    const [clusterId, signal] = overviewSource.getClusterOverview.mock.calls[0];
    expect(clusterId).toBe("c1");
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("10 秒轮询，保鲜时间等长", () => {
    const options = clusterOverviewQuery("c1");
    expect(options.refetchInterval).toBe(REFRESH.fast);
    expect(options.staleTime).toBe(REFRESH.fast);
  });
});

describe("overviewSLODomainsQuery", () => {
  it("没有选定集群时不启用", () => {
    expect(overviewSLODomainsQuery("").enabled).toBe(false);
    expect(overviewSLODomainsQuery("c1").enabled).toBe(true);
  });

  it("键挂在 SLO 域下：改了 SLO 目标，总览的卡片也会重新取数", () => {
    const key = overviewSLODomainsQuery("c1").queryKey;
    expect(key.slice(0, sloKeys.all.length)).toEqual([...sloKeys.all]);
    expect(key).not.toEqual(overviewSLODomainsQuery("c2").queryKey);
  });

  it("不指定窗口，把集群与取消信号交给数据源", async () => {
    sloSource.getSLODomains.mockResolvedValue({ domains: [], summary: null });

    await new QueryClient().fetchQuery(overviewSLODomainsQuery("c1"));

    const [params, signal] = sloSource.getSLODomains.mock.calls[0];
    expect(params).toEqual({ clusterId: "c1" });
    expect(signal).toBeInstanceOf(AbortSignal);
  });
});
