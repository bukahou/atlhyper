/**
 * 外部身份登录回调：解析 URL 片段
 *
 * 后端回调成功后 302 到 /auth/callback#token=<jwt>&next=<应用内路径>，
 * 失败时 /auth/callback#error=<原因码>。片段不发往服务器、不进访问日志。
 */

export const AUTH_CALLBACK_REASONS = [
  "cancelled", "state", "upstream", "disabled", "rejected", "application",
] as const;
export type AuthCallbackReason = (typeof AUTH_CALLBACK_REASONS)[number];

export type AuthCallbackResult =
  | { kind: "success"; token: string; next: string }
  | { kind: "error"; reason: AuthCallbackReason };

// 只放行本站绝对路径（拦 //host 与 /\host 这类协议相对写法）
function safeNext(next: string | null): string {
  if (!next || !next.startsWith("/") || next.startsWith("//") || next.startsWith("/\\")) return "/";
  return next;
}

export function parseAuthCallbackFragment(hash: string): AuthCallbackResult {
  const params = new URLSearchParams(hash.replace(/^#/, ""));
  const token = params.get("token");
  if (token) return { kind: "success", token, next: safeNext(params.get("next")) };

  const reason = params.get("error") ?? "";
  return {
    kind: "error",
    reason: (AUTH_CALLBACK_REASONS as readonly string[]).includes(reason)
      ? (reason as AuthCallbackReason)
      : "application",
  };
}
