import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const datasource = vi.hoisted(() => ({
  getEventOverview: vi.fn(),
  getEventsByResource: vi.fn(),
  getConfigMapList: vi.fn(),
  getSecretList: vi.fn(),
}));
vi.mock("@/datasource/cluster", () => datasource);
vi.mock("@/config/data-source", () => ({ getDataSourceMode: () => "api" }));

import {
  configMapListQuery,
  eventOverviewQuery,
  resourceEventsQuery,
  secretListQuery,
} from "./cluster";

beforeEach(() => {
  for (const reader of Object.values(datasource)) reader.mockReset();
});

describe("eventOverviewQuery", () => {
  it("没有选定集群时不启用", () => {
    expect(eventOverviewQuery("").enabled).toBe(false);
    expect(eventOverviewQuery("c1").enabled).toBe(true);
  });

  it("按集群取数，把取消信号交给数据源，返回解包后的数据", async () => {
    const overview = { rows: [], cards: {} };
    datasource.getEventOverview.mockResolvedValue(overview);

    const data = await new QueryClient().fetchQuery(eventOverviewQuery("c1"));

    expect(data).toBe(overview);
    expect(datasource.getEventOverview).toHaveBeenCalledTimes(1);
    const [params, signal] = datasource.getEventOverview.mock.calls[0];
    expect(params).toEqual({ ClusterID: "c1" });
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("带轮询间隔", () => {
    expect(eventOverviewQuery("c1").refetchInterval).toBeGreaterThan(0);
  });
});

describe("resourceEventsQuery", () => {
  const pod = { kind: "Pod", namespace: "ns1", name: "web-1" };

  it("没有集群或资源名为空时不启用", () => {
    expect(resourceEventsQuery("", pod).enabled).toBe(false);
    expect(resourceEventsQuery("c1", { ...pod, name: "" }).enabled).toBe(false);
    expect(resourceEventsQuery("c1", pod).enabled).toBe(true);
  });

  it("按资源取事件，把取消信号交给数据源", async () => {
    datasource.getEventsByResource.mockResolvedValue([]);

    await new QueryClient().fetchQuery(resourceEventsQuery("c1", pod));

    const [params, signal] = datasource.getEventsByResource.mock.calls[0];
    expect(params).toEqual({ cluster_id: "c1", kind: "Pod", namespace: "ns1", name: "web-1" });
    expect(signal).toBeInstanceOf(AbortSignal);
  });
});

describe("configMapListQuery / secretListQuery", () => {
  it("没有集群或 namespace 为空时不启用", () => {
    expect(configMapListQuery("", "ns1").enabled).toBe(false);
    expect(configMapListQuery("c1", "").enabled).toBe(false);
    expect(configMapListQuery("c1", "ns1").enabled).toBe(true);
  });

  it("Secret 列表还要求有 secret.read 权限", () => {
    expect(secretListQuery("c1", "ns1", false).enabled).toBe(false);
    expect(secretListQuery("", "ns1", true).enabled).toBe(false);
    expect(secretListQuery("c1", "ns1", true).enabled).toBe(true);
  });

  it("按 namespace 取数，把取消信号交给数据源", async () => {
    datasource.getConfigMapList.mockResolvedValue([]);
    datasource.getSecretList.mockResolvedValue([]);
    const client = new QueryClient();

    await client.fetchQuery(configMapListQuery("c1", "ns1"));
    await client.fetchQuery(secretListQuery("c1", "ns1", true));

    for (const reader of [datasource.getConfigMapList, datasource.getSecretList]) {
      const [params, signal] = reader.mock.calls[0];
      expect(params).toEqual({ cluster_id: "c1", namespace: "ns1" });
      expect(signal).toBeInstanceOf(AbortSignal);
    }
  });

  it("ConfigMap 与 Secret 的键不同", () => {
    expect(configMapListQuery("c1", "ns1").queryKey).not.toEqual(secretListQuery("c1", "ns1", true).queryKey);
  });
});
