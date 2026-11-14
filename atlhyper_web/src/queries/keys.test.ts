import { describe, expect, it } from "vitest";
import { adminKeys, clusterKeys } from "./keys";

describe("clusterKeys", () => {
  it("不同集群的键不同：切换集群即切换缓存", () => {
    expect(clusterKeys.events("c1", "api")).not.toEqual(clusterKeys.events("c2", "api"));
  });

  it("真实数据与演示数据的键不同：两者永不共用缓存", () => {
    expect(clusterKeys.events("c1", "api")).not.toEqual(clusterKeys.events("c1", "mock"));
  });

  it("键的形状固定为 [域, 资源, 范围]", () => {
    expect(clusterKeys.events("c1", "api")).toEqual(["cluster", "events", { clusterId: "c1", mode: "api" }]);
  });
});

describe("adminKeys", () => {
  it("权限表不属于任何集群", () => {
    expect(adminKeys.permissionTable()).toEqual(["admin", "permission-table"]);
  });
});
