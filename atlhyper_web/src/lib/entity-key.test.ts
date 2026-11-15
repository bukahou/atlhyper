import { describe, expect, it } from "vitest";
import { incidentSubject, parseEntityKey } from "./entity-key";

describe("parseEntityKey", () => {
  it.each([
    ["shop/pod/api-7b5d4f8c9-x2k4j", { namespace: "shop", type: "pod", name: "api-7b5d4f8c9-x2k4j" }],
    ["shop/service/api", { namespace: "shop", type: "service", name: "api" }],
    ["shop/ingress/web", { namespace: "shop", type: "ingress", name: "web" }],
    ["shop/deployment/api", { namespace: "shop", type: "deployment", name: "api" }],
    ["shop/statefulset/orders-db", { namespace: "shop", type: "statefulset", name: "orders-db" }],
    ["kube-system/daemonset/node-exporter", { namespace: "kube-system", type: "daemonset", name: "node-exporter" }],
    ["shop/job/migrate", { namespace: "shop", type: "job", name: "migrate" }],
    ["shop/cronjob/report", { namespace: "shop", type: "cronjob", name: "report" }],
  ])("命名空间级的键 %s", (entityKey, expected) => {
    expect(parseEntityKey(entityKey)).toEqual(expected);
  });

  it("节点的键带集群前缀，没有命名空间", () => {
    expect(parseEntityKey("cluster/prod/node/worker-1")).toEqual({ namespace: "", type: "node", name: "worker-1" });
  });

  it("名称里带斜杠时整段算名称", () => {
    expect(parseEntityKey("shop/ingress/web/extra")).toEqual({ namespace: "shop", type: "ingress", name: "web/extra" });
  });

  it("只有两段：类型 + 名称", () => {
    expect(parseEntityKey("node/worker-1")).toEqual({ namespace: "", type: "node", name: "worker-1" });
  });

  it("认不出格式：整串当名称，类型 unknown", () => {
    expect(parseEntityKey("something")).toEqual({ namespace: "", type: "unknown", name: "something" });
    expect(parseEntityKey("")).toEqual({ namespace: "", type: "unknown", name: "" });
  });
});

describe("incidentSubject", () => {
  it("工作负载主体：类型取后端给的 subjectType，命名空间与名称从主体键里拆", () => {
    expect(
      incidentSubject({
        subject: "shop/deployment/api",
        subjectType: "deployment",
        rootCause: "shop/pod/api-7b5d4f8c9-x2k4j",
      }),
    ).toEqual({ key: "shop/deployment/api", namespace: "shop", type: "deployment", name: "api" });
  });

  it("主体就是实体自身（节点、服务、裸 Pod）", () => {
    expect(
      incidentSubject({
        subject: "cluster/prod/node/worker-1",
        subjectType: "node",
        rootCause: "cluster/prod/node/worker-1",
      }),
    ).toEqual({ key: "cluster/prod/node/worker-1", namespace: "", type: "node", name: "worker-1" });
  });

  it("后端没给 subjectType 时用键里拆出来的类型", () => {
    expect(incidentSubject({ subject: "shop/cronjob/report", subjectType: "", rootCause: "" }).type).toBe("cronjob");
  });

  it("主体键为空时退回到根因实体，类型也按根因实体算", () => {
    expect(incidentSubject({ subject: "", subjectType: "deployment", rootCause: "shop/pod/api-x2k4j" })).toEqual({
      key: "shop/pod/api-x2k4j",
      namespace: "shop",
      type: "pod",
      name: "api-x2k4j",
    });
  });
});
