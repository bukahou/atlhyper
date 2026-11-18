import { QueryClient, QueryObserver } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const datasource = vi.hoisted(() => ({
  queryLogs: vi.fn(),
  queryLogHistogram: vi.fn(),
}));
const dataSourceMode = vi.hoisted(() => ({ current: "api" }));
vi.mock("@/datasource/logs", () => datasource);
vi.mock("@/config/data-source", () => ({ getDataSourceMode: () => dataSourceMode.current }));

import { toTimeRangeParams } from "@/lib/time-range";
import { logHistogramQuery, logListQuery, resolveLogListRange, traceLogsQuery } from "./observe-logs";
import { REFRESH } from "./refresh";

const filter = { search: "timeout", services: ["gateway"], severities: ["ERROR"], scopes: [] };
const lastHour = toTimeRangeParams({ mode: "preset", preset: "1h" });
const firstPage = { ...filter, ...lastHour, limit: 50, offset: 0 };

beforeEach(() => {
  for (const reader of Object.values(datasource)) reader.mockReset();
  dataSourceMode.current = "api";
});

describe("logListQuery", () => {
  it("没有集群时不启用", () => {
    expect(logListQuery("", firstPage, true).enabled).toBe(false);
    expect(logListQuery("c1", firstPage, true).enabled).toBe(true);
  });

  it("日志正文需要 logs.read：没有权限（或权限还没加载完）时不启用", () => {
    expect(logListQuery("c1", firstPage, false).enabled).toBe(false);
    dataSourceMode.current = "mock";
    expect(logListQuery("", firstPage, false).enabled).toBe(false);
  });

  it("键里带集群、数据源与全部查询条件", () => {
    expect(logListQuery("c1", firstPage, true).queryKey).toEqual([
      "logs",
      "list",
      {
        clusterId: "c1",
        mode: "api",
        search: "timeout",
        services: ["gateway"],
        severities: ["ERROR"],
        scopes: [],
        since: "1h",
        limit: 50,
        offset: 0,
      },
    ]);
  });

  it("相对时间范围的键稳定：同一个范围两次生成的键相等，键里没有现算的时间戳", () => {
    const first = logListQuery("c1", { ...filter, ...toTimeRangeParams({ mode: "preset", preset: "1h" }) }, true).queryKey;
    const second = logListQuery("c1", { ...filter, ...toTimeRangeParams({ mode: "preset", preset: "1h" }) }, true).queryKey;
    expect(first).toEqual(second);
    expect(first[2]).not.toHaveProperty("startTime");
    expect(first[2]).not.toHaveProperty("endTime");
  });

  it("键里没有直方图的桶宽：列表请求不依赖直方图的返回", () => {
    expect(JSON.stringify(logListQuery("c1", firstPage, true).queryKey)).not.toContain("interval");
  });

  it("把查询条件与取消信号交给数据源", async () => {
    const page = { logs: [], total: 0, facets: { services: [], severities: [], scopes: [] } };
    datasource.queryLogs.mockResolvedValue(page);

    expect(await new QueryClient().fetchQuery(logListQuery("c1", firstPage, true))).toBe(page);

    const [clusterId, params, signal] = datasource.queryLogs.mock.calls[0];
    expect(clusterId).toBe("c1");
    expect(params).toEqual(firstPage);
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("15 秒轮询，保鲜时间等长", () => {
    const options = logListQuery("c1", firstPage, true);
    expect(options.refetchInterval).toBe(REFRESH.moderate);
    expect(options.staleTime).toBe(REFRESH.moderate);
  });

  it("翻页时先沿用上一页，表格不闪空", async () => {
    const pageOne = { logs: [{ body: "第一页" }], total: 80, facets: { services: [], severities: [], scopes: [] } };
    datasource.queryLogs.mockResolvedValueOnce(pageOne).mockReturnValue(new Promise(() => {}));
    const client = new QueryClient();
    const observer = new QueryObserver(client, logListQuery("c1", firstPage, true));
    const unsubscribe = observer.subscribe(() => {});
    await vi.waitFor(() => expect(observer.getCurrentResult().data).toBe(pageOne));

    observer.setOptions(logListQuery("c1", { ...firstPage, offset: 50 }, true));

    expect(observer.getCurrentResult().data).toBe(pageOne);
    expect(observer.getCurrentResult().isPlaceholderData).toBe(true);

    unsubscribe();
    client.clear();
  });
});

describe("logHistogramQuery", () => {
  const scope = { ...filter, ...lastHour };

  it("没有集群时不启用；日志计数不需要权限", () => {
    expect(logHistogramQuery("", scope).enabled).toBe(false);
    expect(logHistogramQuery("c1", scope).enabled).toBe(true);
  });

  it("键里带集群、数据源与筛选条件，和列表不是同一份", () => {
    expect(logHistogramQuery("c1", scope).queryKey).toEqual([
      "logs",
      "histogram",
      { clusterId: "c1", mode: "api", ...scope },
    ]);
    expect(logHistogramQuery("c1", scope).queryKey).toEqual(logHistogramQuery("c1", { ...scope }).queryKey);
  });

  it("把取消信号交给数据源", async () => {
    datasource.queryLogHistogram.mockResolvedValue({ buckets: [], intervalMs: 0 });

    await new QueryClient().fetchQuery(logHistogramQuery("c1", scope));

    const [clusterId, params, signal] = datasource.queryLogHistogram.mock.calls[0];
    expect(clusterId).toBe("c1");
    expect(params).toEqual(scope);
    expect(signal).toBeInstanceOf(AbortSignal);
  });
});

describe("traceLogsQuery", () => {
  it("没有集群、没有 traceId 或没有 logs.read 时不启用", () => {
    expect(traceLogsQuery("", "t1", 100, true).enabled).toBe(false);
    expect(traceLogsQuery("c1", "", 100, true).enabled).toBe(false);
    expect(traceLogsQuery("c1", "t1", 100, false).enabled).toBe(false);
    expect(traceLogsQuery("c1", "t1", 100, true).enabled).toBe(true);
  });

  it("按 traceId 取整条链路的日志，把取消信号交给数据源", async () => {
    datasource.queryLogs.mockResolvedValue({ logs: [{ body: "boom" }], total: 1, facets: {} });

    const logs = await new QueryClient().fetchQuery(traceLogsQuery("c1", "t1", 30, true));

    expect(logs).toEqual([{ body: "boom" }]);
    const [clusterId, params, signal] = datasource.queryLogs.mock.calls[0];
    expect(clusterId).toBe("c1");
    expect(params).toEqual({ traceId: "t1", limit: 30 });
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("键里带 traceId：切换链路时各用各的缓存", () => {
    expect(traceLogsQuery("c1", "t1", 100, true).queryKey).not.toEqual(traceLogsQuery("c1", "t2", 100, true).queryKey);
  });
});

describe("resolveLogListRange", () => {
  it("没有 brush 选区时就是页面的时间范围", () => {
    expect(resolveLogListRange({ since: "1h" }, null)).toEqual({ since: "1h" });
    const absolute = { startTime: "2026-01-01T00:00:00.000Z", endTime: "2026-01-01T01:00:00.000Z" };
    expect(resolveLogListRange(absolute, null)).toEqual(absolute);
  });

  it("有选区时用选区的起止时间；结束时间含最后一个桶的整段", () => {
    const start = Date.UTC(2026, 0, 1, 0, 0, 0);
    const lastBucket = Date.UTC(2026, 0, 1, 0, 10, 0);
    expect(
      resolveLogListRange({ since: "1h" }, { firstBucketStartMs: start, lastBucketStartMs: lastBucket, bucketIntervalMs: 120_000 }),
    ).toEqual({
      since: "1h",
      startTime: "2026-01-01T00:00:00.000Z",
      endTime: "2026-01-01T00:12:00.000Z",
    });
  });
});
