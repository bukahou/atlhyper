/**
 * 图表提示框用的 HTML 拼接工具
 *
 * ECharts / G6 的提示框只接受 HTML 字符串。页面内容优先交给 React 渲染；
 * 只有这类「库只收字符串」的地方才拼 HTML，并且必须用这里的 safeHtml 模板 ——
 * 模板里写死的部分原样输出，所有插值按文本转义。
 */

/** 已经转义、可以直接当 HTML 使用的片段 */
export class SafeHtml {
  constructor(private readonly html: string) {}

  toString(): string {
    return this.html;
  }
}

const ESCAPES: Record<string, string> = {
  "&": "&amp;",
  "<": "&lt;",
  ">": "&gt;",
  '"': "&quot;",
  "'": "&#39;",
  "`": "&#96;",
};

/** 把文本转义成可安全放进 HTML 正文或带引号属性值里的形式 */
export function escapeHtml(text: string): string {
  return text.replace(/[&<>"'`]/g, (char) => ESCAPES[char]);
}

function renderValue(value: unknown): string {
  if (value instanceof SafeHtml) return value.toString();
  if (value === null || value === undefined || value === false) return "";
  return escapeHtml(String(value));
}

/**
 * HTML 模板：静态部分原样输出，插值转义。
 * 插值是 SafeHtml 时不重复转义，所以模板可以嵌套。
 */
export function safeHtml(strings: TemplateStringsArray, ...values: unknown[]): SafeHtml {
  let html = strings[0];
  for (let i = 0; i < values.length; i++) {
    html += renderValue(values[i]) + strings[i + 1];
  }
  return new SafeHtml(html);
}

/**
 * 把图表库自己生成的 HTML（如 ECharts 的 marker 圆点）标记为可直接使用。
 * ⛔ 只用于库生成的片段，不得用于任何来自接口或集群的数据。
 */
export function trustedHtml(html: string): SafeHtml {
  return new SafeHtml(html);
}

/** 用分隔片段连接多个 HTML 片段 */
export function joinHtml(parts: SafeHtml[], separator: SafeHtml): SafeHtml {
  return new SafeHtml(parts.map((part) => part.toString()).join(separator.toString()));
}
