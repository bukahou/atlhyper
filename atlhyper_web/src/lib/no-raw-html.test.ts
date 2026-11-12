import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import ts from "typescript";
import { describe, expect, it } from "vitest";

// 页面上显示的内容大量来自集群（日志、事件、资源字段、应用上报的名字），按 HTML 渲染任何一段都可能执行脚本。
// 一律交给 React 以文本渲染；确有需要时先在这里讨论，⛔ 不要直接加白名单绕过。
const RAW_HTML_PATTERNS = ["dangerouslySetInnerHTML", "insertAdjacentHTML", "document.write("];

// 图表库（ECharts / G6）的提示框只接受 HTML 字符串，是唯一允许拼 HTML 的地方，
// 且必须用 lib/safe-html 的 safeHtml 模板：插值自动转义。
// 下面的规则据此检查：代码里的字符串 / 模板只要含 HTML 标签，就必须是 safeHtml 模板本身。
const HTML_TAG = /<\/?(a|b|br|div|em|i|iframe|img|li|ol|p|script|span|strong|style|svg|table|td|th|tr|ul)(\s[^<>]*)?\/?>/i;
const SAFE_HTML_TAG = "safeHtml";
const HTML_STRING_EXEMPT = new Set(["lib/safe-html.ts"]);

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const full = path.join(dir, name);
    if (statSync(full).isDirectory()) return sourceFiles(full);
    return /\.(ts|tsx)$/.test(name) && !name.endsWith(".test.ts") ? [full] : [];
  });
}

function isSafeHtmlTemplate(node: ts.Node): boolean {
  const parent = node.parent;
  return ts.isTaggedTemplateExpression(parent) && parent.template === node && parent.tag.getText() === SAFE_HTML_TAG;
}

/** 返回文件里「含 HTML 标签、却不是 safeHtml 模板」的字符串位置 */
function untaggedHtmlStrings(file: string, text: string): string[] {
  const source = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const hits: string[] = [];
  const report = (node: ts.Node) => {
    const { line } = source.getLineAndCharacterOfPosition(node.getStart());
    hits.push(`${file}:${line + 1}`);
  };
  const visit = (node: ts.Node) => {
    if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
      if (HTML_TAG.test(node.text) && !isSafeHtmlTemplate(node)) report(node);
    } else if (ts.isTemplateExpression(node)) {
      const staticText = [node.head.text, ...node.templateSpans.map((span) => span.literal.text)].join("\u0000");
      if (HTML_TAG.test(staticText) && !isSafeHtmlTemplate(node)) report(node);
    }
    ts.forEachChild(node, visit);
  };
  visit(source);
  return hits;
}

// G6 提示框的 getContent 返回什么字符串都会被当成 HTML，连「直接返回一个名字」也是。
// 所以它的每个 return 只能是空串，或 safeHtml 模板的 .toString()。
const SAFE_RETURN = /^safeHtml`[\s\S]*`\.toString\(\)$/;

/** 返回文件里 getContent 回调中「不是 safeHtml 结果」的 return 位置 */
function unsafeTooltipReturns(file: string, text: string): string[] {
  const source = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const hits: string[] = [];
  const checkReturns = (node: ts.Node) => {
    if (ts.isReturnStatement(node) && node.expression) {
      const expression = node.expression.getText();
      if (expression !== '""' && !SAFE_RETURN.test(expression)) {
        const { line } = source.getLineAndCharacterOfPosition(node.getStart());
        hits.push(`${file}:${line + 1}`);
      }
    }
    ts.forEachChild(node, checkReturns);
  };
  const visit = (node: ts.Node) => {
    if (ts.isPropertyAssignment(node) && node.name.getText() === "getContent") {
      checkReturns(node.initializer);
      return;
    }
    ts.forEachChild(node, visit);
  };
  visit(source);
  return hits;
}

describe("前端不以 HTML 渲染内容", () => {
  const srcDir = path.resolve(__dirname, "..");

  it("src/ 下没有原始 HTML 注入写法", () => {
    const hits = sourceFiles(srcDir).flatMap((file) => {
      const text = readFileSync(file, "utf8");
      return RAW_HTML_PATTERNS.filter((p) => text.includes(p)).map((p) => `${path.relative(srcDir, file)}: ${p}`);
    });
    expect(hits).toEqual([]);
  });

  it("含 HTML 标签的字符串必须是 safeHtml 模板", () => {
    const hits = sourceFiles(srcDir).flatMap((file) => {
      const relative = path.relative(srcDir, file);
      if (HTML_STRING_EXEMPT.has(relative)) return [];
      return untaggedHtmlStrings(relative, readFileSync(file, "utf8"));
    });
    expect(hits).toEqual([]);
  });

  it("G6 提示框的 getContent 只返回 safeHtml 的结果", () => {
    const hits = sourceFiles(srcDir).flatMap((file) =>
      unsafeTooltipReturns(path.relative(srcDir, file), readFileSync(file, "utf8"))
    );
    expect(hits).toEqual([]);
  });

  it("检查装置有效：getContent 里直接返回名字会被发现", () => {
    const sample = [
      "const plugin = { type: 'tooltip', getContent: (_e, items) => {",
      '  if (!items.length) return "";',
      "  if (!items[0].data) return items[0].id;",
      "  return safeHtml`<b>${items[0].id}</b>`.toString();",
      "} };",
    ].join("\n");
    expect(unsafeTooltipReturns("sample.tsx", sample)).toEqual(["sample.tsx:3"]);
  });

  it("检查装置有效：未标记的 HTML 模板会被发现，safeHtml 模板不会", () => {
    const sample = [
      "const a = `<div>${x}</div>`;",
      'const b = lines.join("<br/>");',
      "const c = safeHtml`<div>${x}</div>`;",
      "const d = <div className=\"x\">text</div>;",
    ].join("\n");
    expect(untaggedHtmlStrings("sample.tsx", sample)).toEqual(["sample.tsx:1", "sample.tsx:2"]);
  });
});
