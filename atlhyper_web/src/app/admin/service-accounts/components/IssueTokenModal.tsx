"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { ServiceAccountView } from "@/api/service-accounts";
import { Modal } from "@/components/common";
import {
  API_TOKEN_NAME_MAX_CHARS,
  parseExpiryDays,
  serviceAccountFailure,
  tokenRequestProblem,
} from "@/lib/service-account-input";
import { issueAPITokenMutation } from "@/queries/service-accounts";
import type { CommonTranslations, ServiceAccountsTranslations } from "@/types/i18n";

interface IssueTokenModalProps {
  account: ServiceAccountView;
  /** 有效期的上限与默认值（天），后端下发的业务参数 */
  maxExpiresInDays: number;
  defaultExpiresInDays: number;
  /** 接收令牌明文：由页面放进一次性显示的弹窗，⛔ 不进缓存 */
  onIssued: (token: string) => void;
  onClose: () => void;
  t: ServiceAccountsTranslations;
  commonT: CommonTranslations;
}

type ExpiryMode = "never" | "days";

const INPUT_CLASS =
  "w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default text-sm focus:ring-2 focus:ring-primary outline-none disabled:opacity-60";

/** 签发访问令牌：名称（用途）+ 有效期（不过期或若干天） */
export function IssueTokenModal({
  account,
  maxExpiresInDays,
  defaultExpiresInDays,
  onIssued,
  onClose,
  t,
  commonT,
}: IssueTokenModalProps) {
  const [name, setName] = useState("");
  const [expiryMode, setExpiryMode] = useState<ExpiryMode>("days");
  const [daysText, setDaysText] = useState(String(defaultExpiresInDays));
  const [submitAttempted, setSubmitAttempted] = useState(false);
  const [serverError, setServerError] = useState("");
  const issue = useMutation(issueAPITokenMutation());

  const expiresInDays = expiryMode === "never" ? null : parseExpiryDays(daysText);
  const problem = tokenRequestProblem(name, expiresInDays, maxExpiresInDays);
  const fill = (text: string) =>
    text.replace("{max}", String(problem === "expiryRange" ? maxExpiresInDays : API_TOKEN_NAME_MAX_CHARS));
  const problemText = problem ? fill(t.errors[problem]) : "";

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitAttempted(true);
    setServerError("");
    if (problem) return;

    issue.mutate(
      {
        accountId: account.id,
        request: expiresInDays === null ? { name: name.trim() } : { name: name.trim(), expiresInDays },
        // 明文由工厂直接交到页面，不经过 mutation 的返回值（那会留在缓存里）
        deliverToken: onIssued,
      },
      {
        onSuccess: onClose,
        onError: (error) => setServerError(t.errors[serviceAccountFailure(error, "issue")]),
      },
    );
  };

  return (
    <Modal isOpen onClose={onClose} title={t.issueTitle} size="sm">
      <form onSubmit={handleSubmit} className="p-6 space-y-4" noValidate>
        <div className="text-sm text-muted">
          {t.issueFor}: <span className="font-mono text-default">{account.name}</span>
          {account.displayName && <span className="text-default"> ({account.displayName})</span>}
        </div>

        <div>
          <label htmlFor="api-token-name" className="block text-sm font-medium text-muted mb-1">
            {t.tokenName} *
          </label>
          <input
            id="api-token-name"
            type="text"
            value={name}
            onChange={(event) => setName(event.target.value)}
            maxLength={API_TOKEN_NAME_MAX_CHARS}
            placeholder={t.tokenNameHint.replace("{max}", String(API_TOKEN_NAME_MAX_CHARS))}
            autoComplete="off"
            className={INPUT_CLASS}
          />
        </div>

        <fieldset>
          <legend className="block text-sm font-medium text-muted mb-2">{t.expiry}</legend>
          <div className="flex flex-wrap items-center gap-4">
            <label className="flex items-center gap-2 text-sm text-default">
              <input
                type="radio"
                name="api-token-expiry"
                checked={expiryMode === "days"}
                onChange={() => setExpiryMode("days")}
              />
              {t.expiryDays}
            </label>
            <input
              type="text"
              inputMode="numeric"
              value={daysText}
              onChange={(event) => setDaysText(event.target.value)}
              disabled={expiryMode !== "days"}
              aria-label={t.expiryDays}
              className={`${INPUT_CLASS} w-24`}
            />
            <label className="flex items-center gap-2 text-sm text-default">
              <input
                type="radio"
                name="api-token-expiry"
                checked={expiryMode === "never"}
                onChange={() => setExpiryMode("never")}
              />
              {t.expiryNever}
            </label>
          </div>
          <p className="text-xs text-muted mt-1">{t.expiryDaysHint.replace("{max}", String(maxExpiresInDays))}</p>
        </fieldset>

        {submitAttempted && problemText && <p className="text-xs text-red-500">{problemText}</p>}

        {serverError && (
          <div className="p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg text-sm" role="alert">
            {serverError}
          </div>
        )}

        <div className="flex gap-3 pt-2">
          <button
            type="button"
            onClick={onClose}
            className="flex-1 px-4 py-2 border border-[var(--border-color)] rounded-lg hover-bg text-sm text-default"
          >
            {commonT.cancel}
          </button>
          <button
            type="submit"
            disabled={issue.isPending}
            className="flex-1 px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg disabled:opacity-50 text-sm"
          >
            {issue.isPending ? t.issuing : t.issueToken}
          </button>
        </div>
      </form>
    </Modal>
  );
}
