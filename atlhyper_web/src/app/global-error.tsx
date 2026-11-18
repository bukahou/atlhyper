"use client";

/**
 * 根布局本身抛错时的兜底。
 *
 * 它会替换整个根布局，所以拿不到主题与语言的 Provider：
 * 这里不用 i18n、不用主题变量，文案中日双语写死（唯一的例外）。
 */
export default function GlobalError({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <html lang="zh">
      <body
        style={{
          margin: 0,
          minHeight: "100vh",
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          gap: "16px",
          fontFamily: "system-ui, sans-serif",
          background: "#0f172a",
          color: "#e2e8f0",
        }}
      >
        <h2 style={{ margin: 0, fontSize: "20px" }}>页面出错了 / ページでエラーが発生しました</h2>
        <button
          onClick={reset}
          style={{
            padding: "10px 20px",
            borderRadius: "8px",
            border: "none",
            background: "#14b8a6",
            color: "#fff",
            fontSize: "14px",
            cursor: "pointer",
          }}
        >
          重试 / 再試行
        </button>
      </body>
    </html>
  );
}
