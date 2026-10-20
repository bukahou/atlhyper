/**
 * Pod 日志搜索高亮
 *
 * 把日志切成「普通 / 命中」片段，由组件渲染成文本节点和 <mark>，转义交给 React。
 * ⛔ 不要拼 HTML 字符串再插进页面：日志内容来自容器，等同于任何人都能写入的文本。
 */

export interface LogSegment {
  text: string;
  match: boolean;
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

/** 大小写不敏感、按字面匹配；搜索词只有空白时不高亮 */
export function splitLogHighlight(text: string, query: string): LogSegment[] {
  if (!query.trim()) return [{ text, match: false }];

  const segments: LogSegment[] = [];
  let last = 0;
  for (const m of text.matchAll(new RegExp(escapeRegExp(query), "gi"))) {
    const start = m.index ?? 0;
    if (start > last) segments.push({ text: text.slice(last, start), match: false });
    segments.push({ text: m[0], match: true });
    last = start + m[0].length;
  }
  if (last < text.length || segments.length === 0) {
    segments.push({ text: text.slice(last), match: false });
  }
  return segments;
}
