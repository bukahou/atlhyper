import { describe, expect, it } from "vitest";
import { splitLogHighlight } from "./log-highlight";

describe("splitLogHighlight", () => {
  it("没有搜索词：整段原样作为一个普通片段", () => {
    expect(splitLogHighlight("line1\nline2", "")).toEqual([{ text: "line1\nline2", match: false }]);
    expect(splitLogHighlight("line1", "   ")).toEqual([{ text: "line1", match: false }]);
  });

  it("命中片段单独切出，大小写不敏感，保留原文大小写", () => {
    expect(splitLogHighlight("Error: x error y", "error")).toEqual([
      { text: "Error", match: true },
      { text: ": x ", match: false },
      { text: "error", match: true },
      { text: " y", match: false },
    ]);
  });

  it("正则特殊字符按字面匹配", () => {
    expect(splitLogHighlight("a.b a+b (x)", "a.b")).toEqual([
      { text: "a.b", match: true },
      { text: " a+b (x)", match: false },
    ]);
    expect(splitLogHighlight("call(x) done", "(x)")).toEqual([
      { text: "call", match: false },
      { text: "(x)", match: true },
      { text: " done", match: false },
    ]);
  });

  it("日志里的 HTML 只是文本：片段原样保留，拼回去与原文一致", () => {
    const log = 'GET /<img src=x onerror="alert(1)"> 404';
    const segments = splitLogHighlight(log, "img");
    expect(segments.map((s) => s.text).join("")).toBe(log);
    expect(segments.find((s) => s.match)?.text).toBe("img");
  });

  it("没有命中：一个普通片段", () => {
    expect(splitLogHighlight("nothing here", "zzz")).toEqual([{ text: "nothing here", match: false }]);
  });
});
