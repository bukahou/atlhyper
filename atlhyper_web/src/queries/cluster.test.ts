import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const { getEventOverview } = vi.hoisted(() => ({ getEventOverview: vi.fn() }));
vi.mock("@/datasource/cluster", () => ({ getEventOverview }));
vi.mock("@/config/data-source", () => ({ getDataSourceMode: () => "api" }));

import { eventOverviewQuery } from "./cluster";

describe("eventOverviewQuery", () => {
  beforeEach(() => getEventOverview.mockReset());

  it("没有选定集群时不启用", () => {
    expect(eventOverviewQuery("").enabled).toBe(false);
    expect(eventOverviewQuery("c1").enabled).toBe(true);
  });

  it("按集群取数，把取消信号交给数据源，返回解包后的数据", async () => {
    const overview = { rows: [], cards: {} };
    getEventOverview.mockResolvedValue(overview);

    const data = await new QueryClient().fetchQuery(eventOverviewQuery("c1"));

    expect(data).toBe(overview);
    expect(getEventOverview).toHaveBeenCalledTimes(1);
    const [params, signal] = getEventOverview.mock.calls[0];
    expect(params).toEqual({ ClusterID: "c1" });
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("带轮询间隔", () => {
    expect(eventOverviewQuery("c1").refetchInterval).toBeGreaterThan(0);
  });
});
