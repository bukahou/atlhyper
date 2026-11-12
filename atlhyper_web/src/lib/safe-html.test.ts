import { describe, expect, it } from "vitest";
import { escapeHtml, joinHtml, safeHtml, trustedHtml } from "./safe-html";

describe("escapeHtml", () => {
  it("转义会被 HTML 解析的字符", () => {
    expect(escapeHtml(`<img src=x onerror="a('b')">&`)).toBe(
      "&lt;img src=x onerror=&quot;a(&#39;b&#39;)&quot;&gt;&amp;"
    );
  });

  it("普通文本原样返回", () => {
    expect(escapeHtml("GET /api/v1/users 200")).toBe("GET /api/v1/users 200");
  });
});

describe("safeHtml", () => {
  it("模板里的静态部分原样保留，插值一律转义", () => {
    const name = "<script>alert(1)</script>";
    expect(safeHtml`<b>${name}</b>`.toString()).toBe("<b>&lt;script&gt;alert(1)&lt;/script&gt;</b>");
  });

  it("插值在属性里也不能逃出引号", () => {
    const color = `red" onmouseover="x`;
    expect(safeHtml`<span style="color:${color}">●</span>`.toString()).toBe(
      `<span style="color:red&quot; onmouseover=&quot;x">●</span>`
    );
  });

  it("嵌套的 safeHtml 片段不重复转义", () => {
    const inner = safeHtml`<i>${"a<b"}</i>`;
    expect(safeHtml`<div>${inner}</div>`.toString()).toBe("<div><i>a&lt;b</i></div>");
  });

  it("数字按文本输出，null / undefined / false 输出空串", () => {
    expect(safeHtml`${1.5}|${null}|${undefined}|${false}|${0}`.toString()).toBe("1.5||||0");
  });

  it("trustedHtml 标记的片段原样输出", () => {
    const marker = trustedHtml(`<span class="dot"></span>`);
    expect(safeHtml`${marker} ${"a&b"}`.toString()).toBe(`<span class="dot"></span> a&amp;b`);
  });
});

describe("joinHtml", () => {
  it("用分隔片段连接，各片段保持已转义状态", () => {
    const lines = ["a<1", "b>2"].map((text) => safeHtml`${text}`);
    expect(joinHtml(lines, safeHtml`<br/>`).toString()).toBe("a&lt;1<br/>b&gt;2");
  });

  it("空数组得到空串", () => {
    expect(joinHtml([], safeHtml`<br/>`).toString()).toBe("");
  });
});
