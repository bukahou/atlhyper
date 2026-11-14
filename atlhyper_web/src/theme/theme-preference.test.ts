import { describe, expect, it } from "vitest";
import { parseStoredTheme, resolveTheme } from "./theme-preference";

describe("parseStoredTheme", () => {
  it("合法的值原样返回", () => {
    expect(parseStoredTheme("light")).toBe("light");
    expect(parseStoredTheme("dark")).toBe("dark");
    expect(parseStoredTheme("system")).toBe("system");
  });

  it("没选过或存着不认识的值时跟随系统", () => {
    expect(parseStoredTheme(null)).toBe("system");
    expect(parseStoredTheme("blue")).toBe("system");
  });
});

describe("resolveTheme", () => {
  it("明确选了亮或暗时不看系统", () => {
    expect(resolveTheme("light", true)).toBe("light");
    expect(resolveTheme("dark", false)).toBe("dark");
  });

  it("跟随系统时取系统当前的亮暗色", () => {
    expect(resolveTheme("system", true)).toBe("dark");
    expect(resolveTheme("system", false)).toBe("light");
  });
});
