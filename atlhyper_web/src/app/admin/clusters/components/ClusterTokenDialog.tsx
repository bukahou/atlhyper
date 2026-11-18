"use client";

import { useState } from "react";
import { AlertTriangle, Check, Copy, KeyRound } from "lucide-react";
import type { ClusterRegistryTranslations } from "@/types/i18n";

/** 刚签发的令牌。只存在于页面的本地状态里，关闭弹窗即丢弃 */
export interface IssuedClusterToken {
  clusterId: string;
  clusterName: string;
  /** 令牌明文。⛔ 不得写进查询缓存、localStorage 或日志 */
  token: string;
}

interface ClusterTokenDialogProps {
  issued: IssuedClusterToken;
  onClose: () => void;
  t: ClusterRegistryTranslations;
}

type CopyState = "idle" | "copied" | "failed";

/**
 * 一次性显示刚签发的令牌。
 *
 * 不用通用的 Modal：它点遮罩或按 Esc 就会关，而这里一关令牌就再也看不到了，
 * 所以只留一个写明后果的按钮作为唯一的关闭方式。
 */
export function ClusterTokenDialog({ issued, onClose, t }: ClusterTokenDialogProps) {
  const [copyState, setCopyState] = useState<CopyState>("idle");

  const handleCopy = () => {
    // 剪贴板是用户明确要求的去处；除此之外令牌不离开这个组件
    navigator.clipboard.writeText(issued.token).then(
      () => setCopyState("copied"),
      () => setCopyState("failed"),
    );
  };

  return (
    <div className="fixed inset-0 bg-black/50 z-[60] flex items-center justify-center p-4">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="cluster-token-dialog-title"
        className="bg-card rounded-xl shadow-2xl w-full max-w-lg"
      >
        <div className="flex items-center gap-3 p-4 border-b border-[var(--border-color)]">
          <KeyRound className="w-5 h-5 text-primary" />
          <h2 id="cluster-token-dialog-title" className="text-lg font-semibold text-default">
            {t.tokenDialogTitle}
          </h2>
        </div>

        <div className="p-4 space-y-4">
          <div className="flex items-start gap-3 p-3 rounded-lg border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20">
            <AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
            <p className="text-sm font-semibold text-red-700 dark:text-red-300">{t.tokenOnceWarning}</p>
          </div>

          <div className="text-sm text-muted">
            {t.tokenFor}:{" "}
            <span className="font-mono text-default">{issued.clusterId}</span>
            {issued.clusterName && <span className="text-default"> ({issued.clusterName})</span>}
          </div>

          <div className="flex items-stretch gap-2">
            <code className="flex-1 min-w-0 px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] font-mono text-sm text-default break-all select-all">
              {issued.token}
            </code>
            <button
              type="button"
              onClick={handleCopy}
              className="flex items-center gap-1.5 px-3 py-2 text-sm rounded-lg border border-[var(--border-color)] hover-bg text-default flex-shrink-0"
            >
              {copyState === "copied" ? <Check className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4" />}
              {t.copyToken}
            </button>
          </div>
          <p className="text-xs min-h-4" role="status">
            {copyState === "copied" && <span className="text-green-600 dark:text-green-400">{t.tokenCopied}</span>}
            {copyState === "failed" && <span className="text-red-500">{t.tokenCopyFailed}</span>}
          </p>

          <p className="text-xs text-muted">{t.tokenUsageHint}</p>
        </div>

        <div className="flex justify-end p-4 border-t border-[var(--border-color)]">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-white rounded-lg bg-primary hover:bg-primary-hover transition-colors"
          >
            {t.tokenDialogClose}
          </button>
        </div>
      </div>
    </div>
  );
}
