"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { TriangleAlert } from "lucide-react";
import { StatusPage } from "@/components/common";
import { useI18n } from "@/i18n/context";

/**
 * 页面渲染抛错时的兜底。
 *
 * 请求失败不会走到这里（由 QueryView 就地显示）；这里接的是组件渲染本身的错误，
 * 没有它，任何一个组件抛错都会让整页白屏。
 */
export default function PageError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  const router = useRouter();
  const { t } = useI18n();

  useEffect(() => {
    console.error("[PageError]", error);
  }, [error]);

  return (
    <div className="h-screen flex flex-col">
      <div className="flex-1 min-h-0">
        <StatusPage
          icon={TriangleAlert}
          title={t.common.pageError}
          description={t.common.pageErrorDesc}
          action={{ label: t.common.retry, onClick: reset }}
        />
      </div>
      <div className="pb-10 text-center">
        <button onClick={() => router.push("/overview")} className="text-sm text-muted hover:text-default underline">
          {t.common.backToHome}
        </button>
      </div>
    </div>
  );
}
