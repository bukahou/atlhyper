import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const api = vi.hoisted(() => ({
  getClusterRisk: vi.fn(),
  getEntityRisks: vi.fn(),
  getEntityRiskDetail: vi.fn(),
  getEntityBaseline: vi.fn(),
  getGraph: vi.fn(),
  getGraphTrace: vi.fn(),
  getIncidents: vi.fn(),
  getIncidentStats: vi.fn(),
  getIncidentDetail: vi.fn(),
  getIncidentPatterns: vi.fn(),
}));
vi.mock("@/api/aiops", () => api);

import {
  clusterRiskQuery,
  dependencyGraphQuery,
  entityBaselineQuery,
  entityRiskDetailQuery,
  entityRisksQuery,
  graphTraceQuery,
  incidentDetailQuery,
  incidentPatternsQuery,
  incidentsQuery,
  incidentStatsQuery,
} from "./aiops";
import { REFRESH } from "./refresh";

beforeEach(() => {
  for (const reader of Object.values(api)) reader.mockReset();
});

describe("按集群取数的查询", () => {
  it("没有选定集群时一律不启用（页面显示「未接入集群」而不是永远转圈）", () => {
    expect(clusterRiskQuery("").enabled).toBe(false);
    expect(entityRisksQuery("", 20, REFRESH.slow).enabled).toBe(false);
    expect(dependencyGraphQuery("", REFRESH.fast).enabled).toBe(false);
    expect(incidentsQuery("", "").enabled).toBe(false);
    expect(incidentStatsQuery("").enabled).toBe(false);

    expect(clusterRiskQuery("c1").enabled).toBe(true);
    expect(entityRisksQuery("c1", 20, REFRESH.slow).enabled).toBe(true);
    expect(dependencyGraphQuery("c1", REFRESH.fast).enabled).toBe(true);
    expect(incidentsQuery("c1", "").enabled).toBe(true);
    expect(incidentStatsQuery("c1").enabled).toBe(true);
  });

  it("键里带集群：切换集群即换数据", () => {
    expect(clusterRiskQuery("c1").queryKey).toEqual(["aiops", "cluster-risk", { clusterId: "c1" }]);
    expect(clusterRiskQuery("c1").queryKey).not.toEqual(clusterRiskQuery("c2").queryKey);
    expect(dependencyGraphQuery("c1", false).queryKey).not.toEqual(dependencyGraphQuery("c2", false).queryKey);
    expect(incidentStatsQuery("c1").queryKey).not.toEqual(incidentStatsQuery("c2").queryKey);
  });

  it("把集群与取消信号交给接口", async () => {
    api.getClusterRisk.mockResolvedValue({ risk: 0 });
    api.getGraph.mockResolvedValue({ nodes: {}, edges: [] });
    api.getIncidentStats.mockResolvedValue({ totalIncidents: 0 });
    const client = new QueryClient();

    await client.fetchQuery(clusterRiskQuery("c1"));
    await client.fetchQuery(dependencyGraphQuery("c1", false));
    await client.fetchQuery(incidentStatsQuery("c1"));

    expect(api.getClusterRisk.mock.calls[0][0]).toBe("c1");
    expect(api.getClusterRisk.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
    expect(api.getGraph.mock.calls[0][0]).toBe("c1");
    expect(api.getGraph.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
    expect(api.getIncidentStats.mock.calls[0][0]).toBe("c1");
    expect(api.getIncidentStats.mock.calls[0][2]).toBeInstanceOf(AbortSignal);
  });
});

describe("entityRisksQuery", () => {
  it("条数进键：前 20 与前 100 各用各的缓存", () => {
    expect(entityRisksQuery("c1", 20, REFRESH.slow).queryKey).toEqual([
      "aiops",
      "entity-risks",
      { clusterId: "c1", limit: 20 },
    ]);
    expect(entityRisksQuery("c1", 20, REFRESH.slow).queryKey).not.toEqual(
      entityRisksQuery("c1", 100, REFRESH.slow).queryKey,
    );
  });

  it("按最终风险排序取前 limit 个，带取消信号", async () => {
    api.getEntityRisks.mockResolvedValue([]);

    await new QueryClient().fetchQuery(entityRisksQuery("c1", 50, REFRESH.slow));

    const [clusterId, sort, limit, signal] = api.getEntityRisks.mock.calls[0];
    expect([clusterId, sort, limit]).toEqual(["c1", "r_final", 50]);
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("轮询间隔由页面给，保鲜时间等长；暂停自动刷新时不轮询", () => {
    const polling = entityRisksQuery("c1", 100, REFRESH.fast);
    expect(polling.refetchInterval).toBe(REFRESH.fast);
    expect(polling.staleTime).toBe(REFRESH.fast);

    const paused = dependencyGraphQuery("c1", false);
    expect(paused.refetchInterval).toBe(false);
    expect(paused.staleTime).toBe(0);
  });
});

describe("incidentsQuery", () => {
  it("状态进键；空串表示不按状态过滤", async () => {
    api.getIncidents.mockResolvedValue([]);
    const client = new QueryClient();

    await client.fetchQuery(incidentsQuery("c1", ""));
    await client.fetchQuery(incidentsQuery("c1", "warning"));

    expect(incidentsQuery("c1", "").queryKey).not.toEqual(incidentsQuery("c1", "warning").queryKey);
    expect(api.getIncidents.mock.calls[0][0]).toEqual({ cluster: "c1", state: undefined, limit: 20 });
    expect(api.getIncidents.mock.calls[1][0]).toEqual({ cluster: "c1", state: "warning", limit: 20 });
    expect(api.getIncidents.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
  });

  it("30 秒轮询，保鲜时间等长", () => {
    expect(incidentsQuery("c1", "").refetchInterval).toBe(REFRESH.slow);
    expect(incidentsQuery("c1", "").staleTime).toBe(REFRESH.slow);
  });
});

describe("详情类查询", () => {
  it("目标不完整时不启用", () => {
    expect(entityRiskDetailQuery("c1", "").enabled).toBe(false);
    expect(entityRiskDetailQuery("", "ns/pod/a").enabled).toBe(false);
    expect(entityRiskDetailQuery("c1", "ns/pod/a").enabled).toBe(true);
    expect(entityBaselineQuery("c1", "").enabled).toBe(false);
    expect(graphTraceQuery("c1", "", "upstream").enabled).toBe(false);
    expect(incidentDetailQuery("").enabled).toBe(false);
    expect(incidentDetailQuery("inc-1").enabled).toBe(true);
    // 根因实体要等事件详情回来才知道
    expect(incidentPatternsQuery("c1", "").enabled).toBe(false);
    expect(incidentPatternsQuery("c1", "ns/pod/a").enabled).toBe(true);
  });

  it("键里带实体与方向：连续点不同节点、上下游各用各的缓存", () => {
    expect(entityRiskDetailQuery("c1", "a").queryKey).not.toEqual(entityRiskDetailQuery("c1", "b").queryKey);
    expect(graphTraceQuery("c1", "a", "upstream").queryKey).not.toEqual(
      graphTraceQuery("c1", "a", "downstream").queryKey,
    );
    expect(incidentDetailQuery("inc-1").queryKey).toEqual(["aiops", "incident-detail", { incidentId: "inc-1" }]);
  });

  it("把取消信号交给接口", async () => {
    api.getEntityRiskDetail.mockResolvedValue({});
    api.getEntityBaseline.mockResolvedValue({});
    api.getGraphTrace.mockResolvedValue({ nodes: [], edges: [], depth: 0 });
    api.getIncidentDetail.mockResolvedValue({});
    api.getIncidentPatterns.mockResolvedValue([]);
    const client = new QueryClient();

    await client.fetchQuery(entityRiskDetailQuery("c1", "a"));
    await client.fetchQuery(entityBaselineQuery("c1", "a"));
    await client.fetchQuery(graphTraceQuery("c1", "a", "upstream"));
    await client.fetchQuery(incidentDetailQuery("inc-1"));
    await client.fetchQuery(incidentPatternsQuery("c1", "a"));

    expect(api.getEntityRiskDetail.mock.calls[0].slice(0, 2)).toEqual(["c1", "a"]);
    expect(api.getEntityRiskDetail.mock.calls[0][2]).toBeInstanceOf(AbortSignal);
    expect(api.getEntityBaseline.mock.calls[0][2]).toBeInstanceOf(AbortSignal);
    expect(api.getGraphTrace.mock.calls[0].slice(0, 4)).toEqual(["c1", "a", "upstream", 3]);
    expect(api.getGraphTrace.mock.calls[0][4]).toBeInstanceOf(AbortSignal);
    expect(api.getIncidentDetail.mock.calls[0][0]).toBe("inc-1");
    expect(api.getIncidentDetail.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
    expect(api.getIncidentPatterns.mock.calls[0].slice(0, 2)).toEqual(["c1", "a"]);
    expect(api.getIncidentPatterns.mock.calls[0][3]).toBeInstanceOf(AbortSignal);
  });
});
