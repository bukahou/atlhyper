import { beforeEach, describe, expect, it, vi } from "vitest";

// 数据源层的约定：读函数返回解包后的数据、把取消信号原样交给 api 层，
// 演示数据分支返回相同形态。这里按「概览 / 列表 / 详情 / 指令通道」各取一个代表来验证。
// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升。
const { podApi, jobApi, eventApi, currentMode } = vi.hoisted(() => ({
  podApi: { getPodOverview: vi.fn(), getPodDetail: vi.fn(), getPodLogs: vi.fn() },
  jobApi: { getJobList: vi.fn() },
  eventApi: { getEventsByResource: vi.fn() },
  currentMode: { value: "api" },
}));
vi.mock("@/api/pod", () => podApi);
vi.mock("@/api/job", () => jobApi);
vi.mock("@/api/event", () => eventApi);
vi.mock("@/config/data-source", () => ({ getDataSourceMode: () => currentMode.value }));

import { getEventsByResource, getJobList, getPodDetail, getPodLogs, getPodOverview } from "./cluster";

const signal = new AbortController().signal;

beforeEach(() => {
  currentMode.value = "api";
  for (const api of [podApi, jobApi, eventApi]) {
    for (const reader of Object.values(api)) reader.mockReset();
  }
});

describe("真实接口分支", () => {
  it("概览：返回解包后的数据，取消信号交给 api 层", async () => {
    const overview = { cards: { running: 1 }, pods: [] };
    podApi.getPodOverview.mockResolvedValue({ data: { data: overview } });

    await expect(getPodOverview({ ClusterID: "c1" }, signal)).resolves.toBe(overview);
    expect(podApi.getPodOverview).toHaveBeenCalledWith({ ClusterID: "c1" }, signal);
  });

  it("列表：返回数组本身；后端没给数组时返回空数组", async () => {
    const jobs = [{ name: "backup" }];
    jobApi.getJobList.mockResolvedValueOnce({ data: { data: jobs } }).mockResolvedValueOnce({ data: { data: null } });

    await expect(getJobList({ cluster_id: "c1" }, signal)).resolves.toBe(jobs);
    await expect(getJobList({ cluster_id: "c1" }, signal)).resolves.toEqual([]);
    expect(jobApi.getJobList).toHaveBeenCalledWith({ cluster_id: "c1" }, signal);
  });

  it("详情：返回解包后的对象", async () => {
    const detail = { name: "web-1" };
    podApi.getPodDetail.mockResolvedValue({ data: { data: detail } });
    const params = { ClusterID: "c1", Namespace: "ns1", PodName: "web-1" };

    await expect(getPodDetail(params, signal)).resolves.toBe(detail);
    expect(podApi.getPodDetail).toHaveBeenCalledWith(params, signal);
  });

  it("日志：返回文本；没有日志时是空串，而不是界面文案", async () => {
    podApi.getPodLogs
      .mockResolvedValueOnce({ data: { data: { logs: "line-1" } } })
      .mockResolvedValueOnce({ data: { message: "ok" } });
    const params = { ClusterID: "c1", Namespace: "ns1", Pod: "web-1", TailLines: 100 };

    await expect(getPodLogs(params, signal)).resolves.toBe("line-1");
    await expect(getPodLogs(params, signal)).resolves.toBe("");
    expect(podApi.getPodLogs).toHaveBeenCalledWith(params, signal);
  });

  it("资源事件：返回事件数组", async () => {
    eventApi.getEventsByResource.mockResolvedValue({ data: {} });
    const params = { cluster_id: "c1", kind: "Pod", namespace: "ns1", name: "web-1" };

    await expect(getEventsByResource(params, signal)).resolves.toEqual([]);
    expect(eventApi.getEventsByResource).toHaveBeenCalledWith(params, signal);
  });
});

describe("演示数据分支", () => {
  beforeEach(() => {
    currentMode.value = "mock";
  });

  it("不调用真实接口，返回与真实分支相同的形态", async () => {
    const overview = await getPodOverview({ ClusterID: "c1" }, signal);
    const jobs = await getJobList({ cluster_id: "c1" }, signal);

    expect(Array.isArray(overview.pods)).toBe(true);
    expect(overview.cards).toBeTypeOf("object");
    expect(Array.isArray(jobs)).toBe(true);
    expect(podApi.getPodOverview).not.toHaveBeenCalled();
    expect(jobApi.getJobList).not.toHaveBeenCalled();
  });

  it("详情：演示数据里找不到时返回 null", async () => {
    const overview = await getPodOverview({ ClusterID: "c1" }, signal);
    const known = overview.pods[0];

    const found = await getPodDetail({ ClusterID: "c1", Namespace: known.namespace, PodName: known.name }, signal);
    const missing = await getPodDetail({ ClusterID: "c1", Namespace: "no-such-ns", PodName: "no-such-pod" }, signal);

    expect(found?.name).toBe(known.name);
    expect(missing).toBeNull();
  });
});
