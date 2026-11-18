import { describe, expect, it } from "vitest";
import { clusterDataAvailable } from "./availability";

describe("clusterDataAvailable", () => {
  it("真实接口：选定了集群才能取", () => {
    expect(clusterDataAvailable("c1", "api")).toBe(true);
    expect(clusterDataAvailable("", "api")).toBe(false);
  });

  it("演示数据：没有集群也能取", () => {
    expect(clusterDataAvailable("", "mock")).toBe(true);
    expect(clusterDataAvailable("c1", "mock")).toBe(true);
  });
});
