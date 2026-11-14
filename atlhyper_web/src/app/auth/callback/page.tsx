"use client";

/**
 * 外部身份登录回调页
 *
 * 后端完成 OIDC 换票后 302 到这里，token / 错误码在 URL 片段里：
 * 成功 → 存 token → 拉取当前主体与权限 → 清掉片段 → 跳回 next；失败 → 显示原因。
 */

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Loader2, AlertCircle } from "lucide-react";
import { useHydrated } from "@/hooks/useHydrated";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { parseAuthCallbackFragment, type AuthCallbackResult } from "@/lib/auth-callback";

export default function AuthCallbackPage() {
  const { t } = useI18n();
  const router = useRouter();
  const handledRef = useRef(false);
  // 片段只在浏览器里读得到：挂载时读一次并留在状态里（随后 effect 会把它从地址栏抹掉）
  const [callbackResult] = useState<AuthCallbackResult | null>(() =>
    typeof window === "undefined" ? null : parseAuthCallbackFragment(window.location.hash),
  );
  // 服务端渲染与水合那一帧读不到片段，统一显示「处理中」，水合之后才显示失败原因
  const hydrated = useHydrated();
  const reason = hydrated && callbackResult?.kind === "error" ? callbackResult.reason : null;

  useEffect(() => {
    if (handledRef.current || callbackResult === null) return;
    handledRef.current = true;

    // 片段里有 token：读完立刻从地址栏和历史记录里抹掉
    window.history.replaceState(null, "", window.location.pathname);

    if (callbackResult.kind === "error") return;
    const { setToken, refreshMe } = useAuthStore.getState();
    setToken(callbackResult.token);
    void refreshMe().then(() => router.replace(callbackResult.next));
  }, [router, callbackResult]);

  const lt = t.login;

  if (!reason) {
    return (
      <main className="min-h-screen bg-[var(--background)] flex flex-col items-center justify-center gap-3 px-4">
        <Loader2 className="w-6 h-6 animate-spin text-primary" />
        <p className="text-sm text-muted">{lt.callbackProcessing}</p>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-[var(--background)] flex items-center justify-center px-4">
      <div className="w-full max-w-md bg-card rounded-xl border border-[var(--border-color)] p-6 space-y-4 text-center">
        <AlertCircle className="w-8 h-8 mx-auto text-red-500" />
        <h1 className="text-lg font-semibold text-default">{lt.callbackFailed}</h1>
        <p className="text-sm text-muted">{lt.callbackErrors[reason]}</p>
        <div className="flex gap-3 justify-center pt-2">
          <button
            type="button"
            onClick={() => router.replace("/")}
            className="px-4 py-2 text-sm rounded-lg border border-[var(--border-color)] hover-bg text-default"
          >
            {lt.backHome}
          </button>
          <button
            type="button"
            onClick={() => {
              router.replace("/");
              useAuthStore.getState().openLoginDialog();
            }}
            className="px-4 py-2 text-sm rounded-lg bg-primary hover:bg-primary-hover text-white"
          >
            {lt.tryAgain}
          </button>
        </div>
      </div>
    </main>
  );
}
