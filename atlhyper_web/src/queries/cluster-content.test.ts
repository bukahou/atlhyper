import { QueryClient, QueryObserver } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const datasource = vi.hoisted(() => ({
  getPodLogs: vi.fn(),
  getConfigMapData: vi.fn(),
  getSecretData: vi.fn(),
}));
vi.mock("@/datasource/cluster", () => datasource);
vi.mock("@/config/data-source", () => ({ getDataSourceMode: () => "api" }));

import { configMapDataQuery, podLogsQuery, secretDataQuery } from "./cluster-content";

beforeEach(() => {
  for (const reader of Object.values(datasource)) reader.mockReset();
});

describe("podLogsQuery", () => {
  const stream = { namespace: "ns1", podName: "web-1", containerName: "app", tailLines: 100 };

  it("查看器关着（没有日志流）或没有集群时不启用", () => {
    expect(podLogsQuery("c1", null).enabled).toBe(false);
    expect(podLogsQuery("", stream).enabled).toBe(false);
    expect(podLogsQuery("c1", { ...stream, podName: "" }).enabled).toBe(false);
    expect(podLogsQuery("c1", stream).enabled).toBe(true);
  });

  it("按容器与行数取日志，把取消信号交给数据源", async () => {
    datasource.getPodLogs.mockResolvedValue("line-1\nline-2");

    const logs = await new QueryClient().fetchQuery(podLogsQuery("c1", stream));

    expect(logs).toBe("line-1\nline-2");
    const [params, signal] = datasource.getPodLogs.mock.calls[0];
    expect(params).toEqual({ ClusterID: "c1", Namespace: "ns1", Pod: "web-1", Container: "app", TailLines: 100 });
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("没有指定容器时不传容器名", async () => {
    datasource.getPodLogs.mockResolvedValue("");

    await new QueryClient().fetchQuery(podLogsQuery("c1", { ...stream, containerName: "" }));

    expect(datasource.getPodLogs.mock.calls[0][0].Container).toBeUndefined();
  });

  it("每次取日志都是一条下发给 Agent 的指令：不自动重试、切回标签页不重取", () => {
    const options = podLogsQuery("c1", stream);
    expect(options.retry).toBe(false);
    expect(options.refetchOnWindowFocus).toBe(false);
  });

  it("同一个容器换行数时先沿用上一份日志；换了容器就不沿用", async () => {
    datasource.getPodLogs.mockResolvedValueOnce("旧日志").mockReturnValue(new Promise(() => {}));
    const client = new QueryClient();
    const observer = new QueryObserver(client, podLogsQuery("c1", stream));
    const unsubscribe = observer.subscribe(() => {});
    await vi.waitFor(() => expect(observer.getCurrentResult().data).toBe("旧日志"));

    observer.setOptions(podLogsQuery("c1", { ...stream, tailLines: 500 }));
    expect(observer.getCurrentResult().data).toBe("旧日志");
    expect(observer.getCurrentResult().isPlaceholderData).toBe(true);

    observer.setOptions(podLogsQuery("c1", { ...stream, podName: "web-2", tailLines: 500 }));
    expect(observer.getCurrentResult().data).toBeUndefined();

    unsubscribe();
    client.clear();
  });
});

// 两个工厂的键类型不同，测试只关心它们共有的这几项
interface ContentQueryOptions {
  enabled?: boolean;
  staleTime?: unknown;
  gcTime?: unknown;
  retry?: unknown;
  refetchOnWindowFocus?: unknown;
}
type ContentQueryFactory = (clusterId: string, target: { namespace: string; name: string }) => ContentQueryOptions;

describe.each([
  { title: "configMapDataQuery", factory: configMapDataQuery as ContentQueryFactory, reader: datasource.getConfigMapData },
  { title: "secretDataQuery", factory: secretDataQuery as ContentQueryFactory, reader: datasource.getSecretData },
])("$title", ({ factory, reader }) => {
  const target = { namespace: "ns1", name: "app-config" };

  it("没有集群或目标不完整时不启用", () => {
    expect(factory("", target).enabled).toBe(false);
    expect(factory("c1", { namespace: "ns1", name: "" }).enabled).toBe(false);
    expect(factory("c1", target).enabled).toBe(true);
  });

  it("取这一个对象的内容，把取消信号交给数据源", async () => {
    reader.mockResolvedValue({ key: "value" });

    const content = await new QueryClient().fetchQuery(
      factory("c1", target) as Parameters<QueryClient["fetchQuery"]>[0],
    );

    expect(content).toEqual({ key: "value" });
    const [params, signal] = reader.mock.calls[0];
    expect(params).toEqual({ ClusterID: "c1", Namespace: "ns1", Name: "app-config" });
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("内容经指令通道读取：取到后不自动重取、不重试，没人显示时立刻从缓存丢弃", () => {
    const options = factory("c1", target);
    expect(options.staleTime).toBe(Infinity);
    expect(options.retry).toBe(false);
    expect(options.refetchOnWindowFocus).toBe(false);
    expect(options.gcTime).toBe(0);
  });
});
