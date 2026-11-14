import { describe, expect, it } from "vitest";
import { resolveViewState, type ViewStateInput } from "./query-view-state";

const base: ViewStateInput = {
  requireCluster: true,
  clusterListLoaded: true,
  hasCluster: true,
  hasData: false,
  isEmpty: false,
  isPending: false,
  isError: false,
  isRefetchError: false,
};

describe("resolveViewState", () => {
  it("集群列表还没加载完：加载中，而不是「无集群」", () => {
    expect(resolveViewState({ ...base, clusterListLoaded: false, hasCluster: false, isPending: true })).toBe("loading");
  });

  it("列表加载完仍没有集群：无集群，先于加载中判定", () => {
    expect(resolveViewState({ ...base, hasCluster: false, isPending: true })).toBe("no-cluster");
  });

  it("不要求集群的页面不会进入「无集群」", () => {
    expect(resolveViewState({ ...base, requireCluster: false, hasCluster: false, isPending: true })).toBe("loading");
  });

  it("没有集群但已有数据（演示数据不依赖集群）：照常显示", () => {
    expect(resolveViewState({ ...base, hasCluster: false, hasData: true })).toBe("ready");
  });

  it("首次加载还没有数据：加载中", () => {
    expect(resolveViewState({ ...base, isPending: true })).toBe("loading");
  });

  it("没有数据且请求失败：出错", () => {
    expect(resolveViewState({ ...base, isError: true })).toBe("error");
  });

  it("已有数据、后台刷新失败：继续显示数据并提示", () => {
    expect(resolveViewState({ ...base, hasData: true, isError: true, isRefetchError: true })).toBe("stale");
  });

  it("请求成功但数据为空：空状态", () => {
    expect(resolveViewState({ ...base, hasData: true, isEmpty: true })).toBe("empty");
  });

  it("有数据：就绪", () => {
    expect(resolveViewState({ ...base, hasData: true })).toBe("ready");
  });
});
