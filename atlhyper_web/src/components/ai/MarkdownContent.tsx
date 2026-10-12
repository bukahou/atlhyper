"use client";

/**
 * AI 回答的 Markdown 渲染（GFM：表格 / 列表 / 代码块 / 删除线）
 *
 * react-markdown 默认不渲染原始 HTML（不启用 rehype-raw）—— 模型输出里的 HTML 会作为文本显示，不会执行。
 * 样式用 Tailwind 逐元素指定，跟随主题变量，亮暗两套主题都可读。
 */

import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";

const components: Components = {
  p: ({ children }) => <p className="my-1.5 leading-relaxed">{children}</p>,
  ul: ({ children }) => <ul className="my-1.5 ml-5 list-disc space-y-0.5">{children}</ul>,
  ol: ({ children }) => <ol className="my-1.5 ml-5 list-decimal space-y-0.5">{children}</ol>,
  h1: ({ children }) => <h3 className="mt-3 mb-1.5 text-base font-semibold">{children}</h3>,
  h2: ({ children }) => <h3 className="mt-3 mb-1.5 text-base font-semibold">{children}</h3>,
  h3: ({ children }) => <h4 className="mt-2.5 mb-1 text-sm font-semibold">{children}</h4>,
  strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
  a: ({ children, href }) => (
    <a href={href} target="_blank" rel="noopener noreferrer" className="text-primary underline underline-offset-2">
      {children}
    </a>
  ),
  blockquote: ({ children }) => (
    <blockquote className="my-2 border-l-2 border-[var(--border-color)] pl-3 text-muted">{children}</blockquote>
  ),
  code: ({ children, className }) =>
    className ? (
      <code className={`${className} font-mono text-xs`}>{children}</code>
    ) : (
      <code className="rounded bg-[var(--background)] px-1 py-0.5 font-mono text-xs">{children}</code>
    ),
  pre: ({ children }) => (
    <pre className="my-2 overflow-x-auto rounded-lg border border-[var(--border-color)] bg-[var(--background)] p-3 text-xs">
      {children}
    </pre>
  ),
  // 宽表格在自己的容器里横向滚动，不撑破气泡
  table: ({ children }) => (
    <div className="my-2 overflow-x-auto">
      <table className="min-w-full border-collapse text-xs">{children}</table>
    </div>
  ),
  thead: ({ children }) => <thead className="bg-[var(--background)]">{children}</thead>,
  th: ({ children }) => (
    <th className="border border-[var(--border-color)] px-2 py-1 text-left font-semibold whitespace-nowrap">{children}</th>
  ),
  td: ({ children }) => <td className="border border-[var(--border-color)] px-2 py-1 align-top">{children}</td>,
  hr: () => <hr className="my-3 border-[var(--border-color)]" />,
};

export function MarkdownContent({ content }: { content: string }) {
  return (
    <div className="break-words">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {content}
      </ReactMarkdown>
    </div>
  );
}
