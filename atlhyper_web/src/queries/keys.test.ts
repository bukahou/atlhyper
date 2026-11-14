import { describe, expect, it } from "vitest";
import { adminKeys, apmKeys, clusterKeys, logsKeys, metricsKeys, observeKeys, sloKeys } from "./keys";

describe("clusterKeys", () => {
  it("不同集群的键不同：切换集群即切换缓存", () => {
    expect(clusterKeys.events("c1", "api")).not.toEqual(clusterKeys.events("c2", "api"));
    expect(clusterKeys.resourceList("pod", "c1", "api")).not.toEqual(clusterKeys.resourceList("pod", "c2", "api"));
  });

  it("真实数据与演示数据的键不同：两者永不共用缓存", () => {
    expect(clusterKeys.events("c1", "api")).not.toEqual(clusterKeys.events("c1", "mock"));
    expect(clusterKeys.resourceList("pod", "c1", "api")).not.toEqual(clusterKeys.resourceList("pod", "c1", "mock"));
  });

  it("键的形状固定为 [域, 资源, 范围]", () => {
    expect(clusterKeys.events("c1", "api")).toEqual(["cluster", "events", { clusterId: "c1", mode: "api" }]);
    expect(clusterKeys.resourceList("pod", "c1", "api")).toEqual(["cluster", "pod-list", { clusterId: "c1", mode: "api" }]);
    expect(clusterKeys.resourceDetail("pod", "c1", "api", { namespace: "ns1", name: "web-1" })).toEqual([
      "cluster",
      "pod-detail",
      { clusterId: "c1", mode: "api", namespace: "ns1", name: "web-1" },
    ]);
  });

  it("详情的键带 namespace 与 name：连续点不同行时各用各的缓存", () => {
    const first = clusterKeys.resourceDetail("pod", "c1", "api", { namespace: "ns1", name: "web-1" });
    expect(first).not.toEqual(clusterKeys.resourceDetail("pod", "c1", "api", { namespace: "ns1", name: "web-2" }));
    expect(first).not.toEqual(clusterKeys.resourceDetail("pod", "c1", "api", { namespace: "ns2", name: "web-1" }));
  });

  it("集群级资源没有 namespace，键里记为空串", () => {
    expect(clusterKeys.resourceDetail("node", "c1", "api", { name: "node-1" })).toEqual([
      "cluster",
      "node-detail",
      { clusterId: "c1", mode: "api", namespace: "", name: "node-1" },
    ]);
  });

  it("写操作的失效范围是该类资源全部集群的列表与详情（键的前缀）", () => {
    expect(clusterKeys.resourceScope("pod")).toEqual([
      ["cluster", "pod-list"],
      ["cluster", "pod-detail"],
    ]);
    expect(clusterKeys.resourceList("pod", "c1", "api").slice(0, 2)).toEqual(clusterKeys.resourceLists("pod"));
    expect(clusterKeys.resourceDetail("pod", "c1", "api", { name: "web-1" }).slice(0, 2)).toEqual(
      clusterKeys.resourceDetails("pod"),
    );
  });

  it("日志的键带容器与行数：换行数就是另一份数据", () => {
    const stream = { namespace: "ns1", podName: "web-1", containerName: "app", tailLines: 100 };
    expect(clusterKeys.podLogs("c1", "api", stream)).toEqual([
      "cluster",
      "pod-logs",
      { clusterId: "c1", mode: "api", namespace: "ns1", name: "web-1", container: "app", tailLines: 100 },
    ]);
    expect(clusterKeys.podLogs("c1", "api", stream)).not.toEqual(
      clusterKeys.podLogs("c1", "api", { ...stream, tailLines: 200 }),
    );
  });

  it("某个资源自己的事件按 kind / namespace / name 区分", () => {
    expect(clusterKeys.resourceEvents("c1", "api", { kind: "Pod", namespace: "ns1", name: "web-1" })).toEqual([
      "cluster",
      "resource-events",
      { clusterId: "c1", mode: "api", kind: "Pod", namespace: "ns1", name: "web-1" },
    ]);
  });
});

describe("adminKeys", () => {
  it("权限表不属于任何集群", () => {
    expect(adminKeys.permissionTable()).toEqual(["admin", "permission-table"]);
  });
});

describe("观测信号的键", () => {
  it("每个信号一个域，域的前缀能覆盖该信号的全部查询（手动刷新与写操作后的失效用）", () => {
    expect(metricsKeys.snapshot("c1", "api").slice(0, 1)).toEqual(metricsKeys.all);
    expect(metricsKeys.nodeHistory("c1", "api", "node-1", 1).slice(0, 1)).toEqual(metricsKeys.all);
    expect(logsKeys.histogram("c1", "api", { since: "1h" }).slice(0, 1)).toEqual(logsKeys.all);
    expect(apmKeys.traceDetail("c1", "api", "t1").slice(0, 1)).toEqual(apmKeys.all);
    expect(sloKeys.domains("c1", "api", "1h").slice(0, 1)).toEqual(sloKeys.all);
  });

  it("键的形状与集群资源一致：[域, 资源, { clusterId, mode, ...参数 }]", () => {
    expect(apmKeys.services("c1", "api", { since: "1h" })).toEqual([
      "apm",
      "services",
      { clusterId: "c1", mode: "api", since: "1h" },
    ]);
    expect(sloKeys.domainHistory("c1", "mock", "a.example.com", "6h")).toEqual([
      "slo",
      "domain-history",
      { clusterId: "c1", mode: "mock", host: "a.example.com", window: "6h" },
    ]);
  });

  it("真实数据与演示数据的键不同", () => {
    expect(metricsKeys.snapshot("c1", "api")).not.toEqual(metricsKeys.snapshot("c1", "mock"));
    expect(logsKeys.list("c1", "api", { since: "1h" })).not.toEqual(logsKeys.list("c1", "mock", { since: "1h" }));
  });

  it("信号新鲜度不属于某一个信号：四个观测页共用一份", () => {
    expect(observeKeys.signalFreshness("c1", "api")).toEqual([
      "observe",
      "signal-freshness",
      { clusterId: "c1", mode: "api" },
    ]);
  });
});
