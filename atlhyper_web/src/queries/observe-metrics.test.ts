import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const datasource = vi.hoisted(() => ({
  getClusterNodeMetrics: vi.fn(),
  getHardwareHealth: vi.fn(),
  getNodeMetricsHistory: vi.fn(),
  getSignalFreshness: vi.fn(),
}));
const dataSourceMode = vi.hoisted(() => ({ current: "api" }));
vi.mock("@/datasource/metrics", () => datasource);
vi.mock("@/config/data-source", () => ({ getDataSourceMode: () => dataSourceMode.current }));

import { signalFreshnessQuery } from "./observe-freshness";
import { hardwareHealthQuery, metricsSnapshotQuery, nodeMetricsHistoryQuery } from "./observe-metrics";
import { REFRESH } from "./refresh";

beforeEach(() => {
  for (const reader of Object.values(datasource)) reader.mockReset();
  dataSourceMode.current = "api";
});

describe("metricsSnapshotQuery / hardwareHealthQuery", () => {
  it("没有选定集群时不启用；演示数据不依赖集群", () => {
    expect(metricsSnapshotQuery("").enabled).toBe(false);
    expect(hardwareHealthQuery("").enabled).toBe(false);
    expect(metricsSnapshotQuery("c1").enabled).toBe(true);
    expect(hardwareHealthQuery("c1").enabled).toBe(true);

    dataSourceMode.current = "mock";
    expect(metricsSnapshotQuery("").enabled).toBe(true);
  });

  it("键里带集群与数据源，快照与硬件各是一份", () => {
    expect(metricsSnapshotQuery("c1").queryKey).toEqual(["metrics", "snapshot", { clusterId: "c1", mode: "api" }]);
    expect(hardwareHealthQuery("c1").queryKey).toEqual(["metrics", "hardware", { clusterId: "c1", mode: "api" }]);
  });

  it("把取消信号交给数据源，返回解包后的数据", async () => {
    const snapshot = { summary: {}, nodes: [] };
    datasource.getClusterNodeMetrics.mockResolvedValue(snapshot);
    datasource.getHardwareHealth.mockResolvedValue({ rows: [] });
    const client = new QueryClient();

    expect(await client.fetchQuery(metricsSnapshotQuery("c1"))).toBe(snapshot);
    await client.fetchQuery(hardwareHealthQuery("c1"));

    for (const reader of [datasource.getClusterNodeMetrics, datasource.getHardwareHealth]) {
      const [clusterId, signal] = reader.mock.calls[0];
      expect(clusterId).toBe("c1");
      expect(signal).toBeInstanceOf(AbortSignal);
    }
  });

  it("按 10 秒轮询，保鲜时间与轮询间隔等长", () => {
    for (const options of [metricsSnapshotQuery("c1"), hardwareHealthQuery("c1")]) {
      expect(options.refetchInterval).toBe(REFRESH.fast);
      expect(options.staleTime).toBe(REFRESH.fast);
    }
  });
});

describe("nodeMetricsHistoryQuery", () => {
  it("没有集群或节点名为空时不启用", () => {
    expect(nodeMetricsHistoryQuery("", "node-1", 1).enabled).toBe(false);
    expect(nodeMetricsHistoryQuery("c1", "", 1).enabled).toBe(false);
    expect(nodeMetricsHistoryQuery("c1", "node-1", 1).enabled).toBe(true);
  });

  it("每个「节点 + 时间范围」是一份独立的缓存", () => {
    const base = nodeMetricsHistoryQuery("c1", "node-1", 1).queryKey;
    expect(base).toEqual(["metrics", "node-history", { clusterId: "c1", mode: "api", nodeName: "node-1", hours: 1 }]);
    expect(base).not.toEqual(nodeMetricsHistoryQuery("c1", "node-2", 1).queryKey);
    expect(base).not.toEqual(nodeMetricsHistoryQuery("c1", "node-1", 24).queryKey);
    // 同一节点同一范围，两次生成的键相等：集群趋势图与节点展开面板共用一份
    expect(base).toEqual(nodeMetricsHistoryQuery("c1", "node-1", 1).queryKey);
  });

  it("按节点与小时数取数，把取消信号交给数据源", async () => {
    const history = { cpu: [] };
    datasource.getNodeMetricsHistory.mockResolvedValue(history);

    expect(await new QueryClient().fetchQuery(nodeMetricsHistoryQuery("c1", "node-1", 6))).toBe(history);

    const [clusterId, nodeName, hours, signal] = datasource.getNodeMetricsHistory.mock.calls[0];
    expect([clusterId, nodeName, hours]).toEqual(["c1", "node-1", 6]);
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("不跟着快照的 10 秒轮询走：趋势点是分钟级的，一分钟刷新一次", () => {
    const options = nodeMetricsHistoryQuery("c1", "node-1", 1);
    expect(options.refetchInterval).toBe(REFRESH.minute);
    expect(options.staleTime).toBe(REFRESH.minute);
  });
});

describe("signalFreshnessQuery", () => {
  it("没有选定集群时不启用", () => {
    expect(signalFreshnessQuery("").enabled).toBe(false);
    expect(signalFreshnessQuery("c1").enabled).toBe(true);
  });

  it("键里只有集群与数据源：四个观测页共用同一份", () => {
    expect(signalFreshnessQuery("c1").queryKey).toEqual([
      "observe",
      "signal-freshness",
      { clusterId: "c1", mode: "api" },
    ]);
  });

  it("把取消信号交给数据源", async () => {
    datasource.getSignalFreshness.mockResolvedValue({ signals: [], collectorHealthy: true });

    await new QueryClient().fetchQuery(signalFreshnessQuery("c1"));

    const [clusterId, signal] = datasource.getSignalFreshness.mock.calls[0];
    expect(clusterId).toBe("c1");
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("一分钟轮询一次，保鲜时间等长：一分钟内切换观测页不重复请求", () => {
    const options = signalFreshnessQuery("c1");
    expect(options.refetchInterval).toBe(REFRESH.minute);
    expect(options.staleTime).toBe(REFRESH.minute);
  });
});
