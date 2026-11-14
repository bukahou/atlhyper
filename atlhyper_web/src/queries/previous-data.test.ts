import { describe, expect, it } from "vitest";
import { isQueryOfCluster } from "./previous-data";

describe("isQueryOfCluster", () => {
  it("只有同一个集群的上一份结果才留着用", () => {
    const previous = { queryKey: ["aiops", "incidents", { clusterId: "c1", state: "" }] };
    expect(isQueryOfCluster(previous, "c1")).toBe(true);
    expect(isQueryOfCluster(previous, "c2")).toBe(false);
  });

  it("没有上一份、或键里没有集群时不保留", () => {
    expect(isQueryOfCluster(undefined, "c1")).toBe(false);
    expect(isQueryOfCluster({ queryKey: ["admin", "users"] }, "c1")).toBe(false);
  });
});
