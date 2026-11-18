import { describe, expect, it } from "vitest";
import { resolveLanguage } from "./language-preference";

describe("resolveLanguage", () => {
  it("手动选过的语言优先于浏览器语言", () => {
    expect(resolveLanguage("zh", "ja-JP")).toBe("zh");
    expect(resolveLanguage("ja", "zh-CN")).toBe("ja");
  });

  it("没选过时跟随浏览器：日语用日文，其余用中文", () => {
    expect(resolveLanguage(null, "ja")).toBe("ja");
    expect(resolveLanguage(null, "JA-jp")).toBe("ja");
    expect(resolveLanguage(null, "en-US")).toBe("zh");
    expect(resolveLanguage(null, "")).toBe("zh");
  });

  it("存着不认识的值时当作没选过", () => {
    expect(resolveLanguage("fr", "ja-JP")).toBe("ja");
    expect(resolveLanguage("undefined", "en-US")).toBe("zh");
  });
});
