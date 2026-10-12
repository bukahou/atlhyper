import { describe, it, expect } from "vitest";
import { parseAuthCallbackFragment } from "./auth-callback";

describe("parseAuthCallbackFragment", () => {
  it("成功：取出 token 与 next", () => {
    expect(parseAuthCallbackFragment("#token=abc.def&next=%2Fobserve%2Flogs")).toEqual({
      kind: "success", token: "abc.def", next: "/observe/logs",
    });
  });

  it("没有 next 或 next 不安全 → 回首页", () => {
    expect(parseAuthCallbackFragment("#token=t")).toEqual({ kind: "success", token: "t", next: "/" });
    for (const bad of ["//evil.example.com", "/\\evil.example.com", "https://evil.example.com", "javascript:alert(1)"]) {
      const r = parseAuthCallbackFragment(`#token=t&next=${encodeURIComponent(bad)}`);
      expect(r).toEqual({ kind: "success", token: "t", next: "/" });
    }
  });

  it("失败：已知原因码原样返回，未知一律 application", () => {
    expect(parseAuthCallbackFragment("#error=disabled")).toEqual({ kind: "error", reason: "disabled" });
    expect(parseAuthCallbackFragment("#error=%3Cscript%3E")).toEqual({ kind: "error", reason: "application" });
  });

  it("既无 token 也无 error → application", () => {
    expect(parseAuthCallbackFragment("")).toEqual({ kind: "error", reason: "application" });
  });
});
