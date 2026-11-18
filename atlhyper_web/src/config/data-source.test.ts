import { describe, expect, it } from "vitest";
import { MODULE_REGISTRY, resolveDataSourceMode } from "./data-source";

describe("resolveDataSourceMode", () => {
  it("保存过的选择优先，否则用注册表里的默认值", () => {
    expect(resolveDataSourceMode("slo", {})).toBe("api");
    expect(resolveDataSourceMode("slo", { slo: "mock" })).toBe("mock");
  });

  it("没有演示数据的模块恒为真实接口，哪怕保存的是 mock", () => {
    expect(resolveDataSourceMode("apm", { apm: "mock" })).toBe("api");
    expect(resolveDataSourceMode("audit", { audit: "mock" })).toBe("api");
  });

  it("注册表里没有的模块按真实接口处理", () => {
    expect(resolveDataSourceMode("unknown-module", {})).toBe("api");
  });
});

describe("MODULE_REGISTRY", () => {
  it("APM 没有演示数据，不标成可切换", () => {
    expect(MODULE_REGISTRY.find((mod) => mod.key === "apm")?.hasMock).toBe(false);
  });
});
