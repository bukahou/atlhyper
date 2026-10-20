import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

// 页面上显示的内容大量来自集群（日志、事件、资源字段），按 HTML 渲染任何一段都可能执行脚本。
// 一律交给 React 以文本渲染；确有需要时先在这里讨论，⛔ 不要直接加白名单绕过。
const RAW_HTML_PATTERNS = ["dangerouslySetInnerHTML", "insertAdjacentHTML", "document.write("];

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const full = path.join(dir, name);
    if (statSync(full).isDirectory()) return sourceFiles(full);
    return /\.(ts|tsx)$/.test(name) && !name.endsWith(".test.ts") ? [full] : [];
  });
}

describe("前端不以 HTML 渲染内容", () => {
  it("src/ 下没有原始 HTML 注入写法", () => {
    const srcDir = path.resolve(__dirname, "..");
    const hits = sourceFiles(srcDir).flatMap((file) => {
      const text = readFileSync(file, "utf8");
      return RAW_HTML_PATTERNS.filter((p) => text.includes(p)).map((p) => `${path.relative(srcDir, file)}: ${p}`);
    });
    expect(hits).toEqual([]);
  });
});
