import { describe, expect, it } from "vitest";
import { historyChartState } from "./historyChartState";

const point = { timestamp: "2026-11-18T05:00:00Z", value: 1 };

describe("historyChartState", () => {
  it("任何一个节点有这个指标的数据就画图（即使别的节点请求失败）", () => {
    const historyMap = { "desk-one": { cpu: [point] } };
    expect(historyChartState(["desk-one", "desk-two"], "cpu", historyMap, new Set(["desk-two"]))).toBe("data");
  });

  it("都没有数据、且有节点的请求失败时显示加载失败", () => {
    expect(historyChartState(["desk-one", "desk-two"], "cpu", {}, new Set(["desk-two"]))).toBe("failed");
  });

  it("都没有数据、请求也都成功时显示暂无数据", () => {
    const historyMap = { "desk-one": { cpu: [] }, "desk-two": { cpu: [] } };
    expect(historyChartState(["desk-one", "desk-two"], "cpu", historyMap, new Set())).toBe("empty");
  });

  it("失败的节点不在这张图的名单里时不算", () => {
    expect(historyChartState(["desk-one"], "cpu", {}, new Set(["desk-two"]))).toBe("empty");
  });

  it("只看本图的指标：别的指标有数据不算", () => {
    const historyMap = { "desk-one": { memory: [point] } };
    expect(historyChartState(["desk-one"], "cpu", historyMap, new Set())).toBe("empty");
  });
});
