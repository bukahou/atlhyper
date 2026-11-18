import { describe, expect, it } from "vitest";
import { isQueryOfCluster, isSameFilterOtherPage } from "./previous-data";

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

describe("isSameFilterOtherPage", () => {
  const firstPage = { status: "firing", limit: 20, offset: 0 };
  const previous = { queryKey: ["notify", "alerts", firstPage] };

  it("只差页码：留着上一页", () => {
    expect(isSameFilterOtherPage(previous, { status: "firing", limit: 20, offset: 20 })).toBe(true);
  });

  it("筛选条件变了：不留，否则新筛选下会先显示旧筛选的数据", () => {
    expect(isSameFilterOtherPage(previous, { status: "resolved", limit: 20, offset: 0 })).toBe(false);
  });

  it("字段顺序不同不影响判断", () => {
    const reordered = { queryKey: ["notify", "alerts", { offset: 0, limit: 20, status: "firing" }] };
    expect(isSameFilterOtherPage(reordered, { status: "firing", limit: 20, offset: 40 })).toBe(true);
  });

  it("没有上一份、或键里没有范围时不保留", () => {
    expect(isSameFilterOtherPage(undefined, firstPage)).toBe(false);
    expect(isSameFilterOtherPage({ queryKey: ["notify", "policy"] }, firstPage)).toBe(false);
  });
});
