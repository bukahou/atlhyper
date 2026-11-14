import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const datasource = vi.hoisted(() => ({
  getAPMServices: vi.fn(),
  getTopology: vi.fn(),
  getOperations: vi.fn(),
  queryTraces: vi.fn(),
  getTraceDetail: vi.fn(),
  getServiceTimeSeries: vi.fn(),
  getHTTPStats: vi.fn(),
  getDBStats: vi.fn(),
  getCorrelations: vi.fn(),
}));
vi.mock("@/datasource/apm", () => datasource);

import { toTimeRangeParams } from "@/lib/time-range";
import {
  apmCorrelationsQuery,
  apmDbStatsQuery,
  apmHttpStatsQuery,
  apmOperationSamplesQuery,
  apmOperationsQuery,
  apmServiceSeriesQuery,
  apmServicesQuery,
  apmTopologyQuery,
  apmTraceDetailQuery,
  apmTracesQuery,
} from "./observe-apm";
import { REFRESH } from "./refresh";

const lastHour = toTimeRangeParams({ mode: "preset", preset: "1h" });
const absolute = toTimeRangeParams({ mode: "absolute", start: Date.UTC(2026, 0, 1), end: Date.UTC(2026, 0, 2) });

beforeEach(() => {
  for (const reader of Object.values(datasource)) reader.mockReset();
});

describe("服务列表 / 拓扑 / 操作统计", () => {
  const overviewFactories = [
    { name: "services", factory: apmServicesQuery, reader: datasource.getAPMServices },
    { name: "topology", factory: apmTopologyQuery, reader: datasource.getTopology },
    { name: "operations", factory: apmOperationsQuery, reader: datasource.getOperations },
  ];

  it.each(overviewFactories)("$name：没有集群时不启用", ({ factory }) => {
    expect(factory("", lastHour).enabled).toBe(false);
    expect(factory("c1", lastHour).enabled).toBe(true);
  });

  it.each(overviewFactories)("$name：键里带集群、数据源与时间范围", ({ name, factory }) => {
    expect(factory("c1", lastHour).queryKey).toEqual(["apm", name, { clusterId: "c1", mode: "api", since: "1h" }]);
    expect(factory("c1", absolute).queryKey).toEqual([
      "apm",
      name,
      { clusterId: "c1", mode: "api", startTime: "2026-01-01T00:00:00.000Z", endTime: "2026-01-02T00:00:00.000Z" },
    ]);
  });

  it.each(overviewFactories)("$name：相对时间范围的键稳定", ({ factory }) => {
    const first = factory("c1", toTimeRangeParams({ mode: "custom", value: 30, unit: "m" })).queryKey;
    const second = factory("c1", toTimeRangeParams({ mode: "custom", value: 30, unit: "m" })).queryKey;
    expect(first).toEqual(second);
  });

  it.each(overviewFactories)("$name：把时间范围与取消信号交给数据源", async ({ factory, reader }) => {
    reader.mockResolvedValue([]);

    await new QueryClient().fetchQuery(factory("c1", lastHour) as Parameters<QueryClient["fetchQuery"]>[0]);

    const [clusterId, range, signal] = reader.mock.calls[0];
    expect(clusterId).toBe("c1");
    expect(range).toEqual({ since: "1h" });
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it.each(overviewFactories)("$name：按默认档轮询，保鲜时间等长", ({ factory }) => {
    const options = factory("c1", lastHour);
    expect(options.refetchInterval).toBe(REFRESH.standard);
    expect(options.staleTime).toBe(REFRESH.standard);
  });

  it("数据源抛的错不被吞掉：查询以失败结束，由界面显示出错状态", async () => {
    datasource.getAPMServices.mockRejectedValue(new Error("clickhouse down"));

    await expect(
      new QueryClient().fetchQuery({ ...apmServicesQuery("c1", lastHour), retry: false }),
    ).rejects.toThrow("clickhouse down");
  });
});

describe("apmTracesQuery", () => {
  const filter = { service: "gateway", limit: 200 };

  it("链路列表需要 traces.read：没有集群或没有权限时不启用", () => {
    expect(apmTracesQuery("", filter, lastHour, true).enabled).toBe(false);
    expect(apmTracesQuery("c1", filter, lastHour, false).enabled).toBe(false);
    expect(apmTracesQuery("c1", filter, lastHour, true).enabled).toBe(true);
  });

  it("键里带筛选条件：按状态码筛出的链路与全部链路各是一份", () => {
    expect(apmTracesQuery("c1", filter, lastHour, true).queryKey).toEqual([
      "apm",
      "traces",
      { clusterId: "c1", mode: "api", service: "gateway", limit: 200, since: "1h" },
    ]);
    expect(apmTracesQuery("c1", filter, lastHour, true).queryKey).not.toEqual(
      apmTracesQuery("c1", { ...filter, statusCode: "500", method: "GET" }, lastHour, true).queryKey,
    );
  });

  it("把筛选条件、时间范围与取消信号交给数据源", async () => {
    datasource.queryTraces.mockResolvedValue({ traces: [], total: 0 });

    await new QueryClient().fetchQuery(apmTracesQuery("c1", filter, lastHour, true));

    const [clusterId, passedFilter, range, signal] = datasource.queryTraces.mock.calls[0];
    expect([clusterId, passedFilter, range]).toEqual(["c1", filter, { since: "1h" }]);
    expect(signal).toBeInstanceOf(AbortSignal);
  });
});

describe("apmOperationSamplesQuery", () => {
  const operation = { service: "gateway", operation: "GET /api/users" };

  it("没有集群、没有权限或没有指定操作时不启用", () => {
    expect(apmOperationSamplesQuery("", operation, lastHour, true).enabled).toBe(false);
    expect(apmOperationSamplesQuery("c1", operation, lastHour, false).enabled).toBe(false);
    expect(apmOperationSamplesQuery("c1", { service: "gateway", operation: "" }, lastHour, true).enabled).toBe(false);
    expect(apmOperationSamplesQuery("c1", operation, lastHour, true).enabled).toBe(true);
  });

  it("取这个操作的样本，返回链路数组", async () => {
    const traces = [{ traceId: "t1" }, { traceId: "t2" }];
    datasource.queryTraces.mockResolvedValue({ traces, total: 2 });

    expect(await new QueryClient().fetchQuery(apmOperationSamplesQuery("c1", operation, lastHour, true))).toBe(traces);

    const [clusterId, filter, range, signal] = datasource.queryTraces.mock.calls[0];
    expect([clusterId, range]).toEqual(["c1", { since: "1h" }]);
    expect(filter).toEqual({ service: "gateway", operation: "GET /api/users", limit: 200 });
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("样本序列进入时取一次就不再动：不轮询、不因切回标签页重取，离开即丢弃", () => {
    const options = apmOperationSamplesQuery("c1", operation, lastHour, true);
    expect(options.refetchInterval).toBeUndefined();
    expect(options.staleTime).toBe(Infinity);
    expect(options.gcTime).toBe(0);
  });
});

describe("apmTraceDetailQuery", () => {
  it("没有集群、没有 traceId 或没有 traces.read 时不启用", () => {
    expect(apmTraceDetailQuery("", "t1", true).enabled).toBe(false);
    expect(apmTraceDetailQuery("c1", "", true).enabled).toBe(false);
    expect(apmTraceDetailQuery("c1", "t1", false).enabled).toBe(false);
    expect(apmTraceDetailQuery("c1", "t1", true).enabled).toBe(true);
  });

  it("键里带 traceId：快速切换时显示的一定是最后点的那条", () => {
    expect(apmTraceDetailQuery("c1", "t1", true).queryKey).toEqual([
      "apm",
      "trace-detail",
      { clusterId: "c1", mode: "api", traceId: "t1" },
    ]);
    expect(apmTraceDetailQuery("c1", "t1", true).queryKey).not.toEqual(apmTraceDetailQuery("c1", "t2", true).queryKey);
  });

  it("按 traceId 取详情，把取消信号交给数据源", async () => {
    const detail = { traceId: "t1", spans: [] };
    datasource.getTraceDetail.mockResolvedValue(detail);

    expect(await new QueryClient().fetchQuery(apmTraceDetailQuery("c1", "t1", true))).toBe(detail);

    const [clusterId, traceId, signal] = datasource.getTraceDetail.mock.calls[0];
    expect([clusterId, traceId]).toEqual(["c1", "t1"]);
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("链路是历史数据：取到后不自动重取", () => {
    const options = apmTraceDetailQuery("c1", "t1", true);
    expect(options.refetchInterval).toBeUndefined();
    expect(options.staleTime).toBe(Infinity);
  });
});

describe("服务详情：趋势 / HTTP 状态码 / 数据库统计", () => {
  const serviceFactories = [
    { name: "service-series", factory: apmServiceSeriesQuery, reader: datasource.getServiceTimeSeries },
    { name: "http-stats", factory: apmHttpStatsQuery, reader: datasource.getHTTPStats },
    { name: "db-stats", factory: apmDbStatsQuery, reader: datasource.getDBStats },
  ];

  it.each(serviceFactories)("$name：没有集群或服务名为空时不启用", ({ factory }) => {
    expect(factory("", "gateway", lastHour).enabled).toBe(false);
    expect(factory("c1", "", lastHour).enabled).toBe(false);
    expect(factory("c1", "gateway", lastHour).enabled).toBe(true);
  });

  it.each(serviceFactories)("$name：键里带服务名与时间范围", ({ name, factory }) => {
    expect(factory("c1", "gateway", lastHour).queryKey).toEqual([
      "apm",
      name,
      { clusterId: "c1", mode: "api", serviceName: "gateway", since: "1h" },
    ]);
  });

  it.each(serviceFactories)("$name：把服务名、时间范围与取消信号交给数据源", async ({ factory, reader }) => {
    reader.mockResolvedValue([]);

    await new QueryClient().fetchQuery(factory("c1", "gateway", lastHour) as Parameters<QueryClient["fetchQuery"]>[0]);

    const [clusterId, serviceName, range, signal] = reader.mock.calls[0];
    expect([clusterId, serviceName, range]).toEqual(["c1", "gateway", { since: "1h" }]);
    expect(signal).toBeInstanceOf(AbortSignal);
  });
});

describe("apmCorrelationsQuery", () => {
  const subject = { serviceName: "gateway", correlationMode: "failure" as const };

  it("相关性分析读的是链路属性，需要 traces.read", () => {
    expect(apmCorrelationsQuery("", subject, lastHour, true).enabled).toBe(false);
    expect(apmCorrelationsQuery("c1", subject, lastHour, false).enabled).toBe(false);
    expect(apmCorrelationsQuery("c1", subject, lastHour, true).enabled).toBe(true);
  });

  it("键里带分析模式：失败与延迟两种模式各是一份", () => {
    expect(apmCorrelationsQuery("c1", subject, lastHour, true).queryKey).toEqual([
      "apm",
      "correlations",
      { clusterId: "c1", mode: "api", serviceName: "gateway", correlationMode: "failure", operation: "", since: "1h" },
    ]);
    expect(apmCorrelationsQuery("c1", subject, lastHour, true).queryKey).not.toEqual(
      apmCorrelationsQuery("c1", { ...subject, correlationMode: "latency" }, lastHour, true).queryKey,
    );
  });

  it("把分析对象、时间范围与取消信号交给数据源", async () => {
    datasource.getCorrelations.mockResolvedValue(null);

    await new QueryClient().fetchQuery(apmCorrelationsQuery("c1", subject, lastHour, true));

    const [clusterId, passedSubject, range, signal] = datasource.getCorrelations.mock.calls[0];
    expect([clusterId, passedSubject, range]).toEqual(["c1", subject, { since: "1h" }]);
    expect(signal).toBeInstanceOf(AbortSignal);
  });
});
