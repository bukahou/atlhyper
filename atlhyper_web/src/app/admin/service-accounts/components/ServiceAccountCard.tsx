"use client";

import { Ban, KeyRound, Pencil, Trash2 } from "lucide-react";
import type { APITokenStatus, APITokenView, ServiceAccountView } from "@/api/service-accounts";
import { StatusBadge } from "@/components/common";
import { formatTimestamp } from "@/lib/format";
import type { RoleInfo } from "@/app/admin/roles/roles";
import type { CommonTranslations, ServiceAccountsTranslations } from "@/types/i18n";

interface ServiceAccountCardProps {
  account: ServiceAccountView;
  /** 等级 → 名称与说明（沿用角色权限页的文案） */
  roleOf: (level: number) => RoleInfo | undefined;
  /** 每个账号未吊销令牌的上限（后端下发），只用于提示文案 */
  maxActiveTokens: number;
  /** 演示数据：操作按钮照常显示但不可用 */
  readOnly: boolean;
  onEdit: (account: ServiceAccountView) => void;
  onIssueToken: (account: ServiceAccountView) => void;
  onDelete: (account: ServiceAccountView) => void;
  onRevokeToken: (account: ServiceAccountView, token: APITokenView) => void;
  t: ServiceAccountsTranslations;
  commonT: CommonTranslations;
}

const TOKEN_STATUS_BADGE: Record<APITokenStatus, "success" | "warning" | "default"> = {
  active: "success",
  expired: "warning",
  revoked: "default",
};

const ACTION_BUTTON_CLASS =
  "p-2 rounded-lg hover-bg text-muted hover:text-primary transition-colors disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:text-muted";

/** 一个服务账号：账号信息与操作在上，它的令牌列表在下（令牌状态由后端判定，这里只映射颜色与文案） */
export function ServiceAccountCard({
  account,
  roleOf,
  maxActiveTokens,
  readOnly,
  onEdit,
  onIssueToken,
  onDelete,
  onRevokeToken,
  t,
  commonT,
}: ServiceAccountCardProps) {
  const role = roleOf(account.role);
  const statusLabel: Record<APITokenStatus, string> = {
    active: t.statusActive,
    expired: t.statusExpired,
    revoked: t.statusRevoked,
  };
  const issueTitle = account.canIssueToken ? t.issueToken : t.tokenLimitReached.replace("{max}", String(maxActiveTokens));

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="flex flex-wrap items-start justify-between gap-3 p-4 border-b border-[var(--border-color)]">
        <div className="min-w-0 space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-sm font-semibold text-default">{account.name}</span>
            {account.displayName && <span className="text-sm text-muted">{account.displayName}</span>}
          </div>
          <div className="flex flex-wrap items-center gap-2 text-xs">
            <span className={`inline-flex px-2 py-0.5 rounded-full ${role?.bgColor ?? ""} ${role?.color ?? "text-muted"}`} title={role?.description}>
              {t.role}: {role?.name ?? account.role}
            </span>
            <StatusBadge status={account.enabled ? t.enabled : t.disabled} type={account.enabled ? "success" : "default"} />
            <span className="text-muted whitespace-nowrap">{t.createdAt.replace("{time}", formatTimestamp(account.createdAt))}</span>
            <span className="text-muted whitespace-nowrap">
              {t.lastUsed}: {account.lastUsedAt ? formatTimestamp(account.lastUsedAt) : t.neverUsed}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-1">
          <button type="button" onClick={() => onEdit(account)} disabled={readOnly} className={ACTION_BUTTON_CLASS}
            title={t.editAccount} aria-label={t.editAccount}>
            <Pencil className="w-4 h-4" />
          </button>
          <button type="button" onClick={() => onIssueToken(account)} disabled={readOnly || !account.canIssueToken}
            className={ACTION_BUTTON_CLASS} title={issueTitle} aria-label={t.issueToken}>
            <KeyRound className="w-4 h-4" />
          </button>
          <button type="button" onClick={() => onDelete(account)} disabled={readOnly}
            className={`${ACTION_BUTTON_CLASS} hover:text-red-500`} title={t.deleteAccount} aria-label={t.deleteAccount}>
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      {!account.canIssueToken && (
        <p className="px-4 pt-3 text-xs text-amber-600 dark:text-amber-400">{issueTitle}</p>
      )}

      <div className="p-4 overflow-x-auto">
        <p className="text-xs font-medium text-muted mb-2">{t.tokens}</p>
        {account.tokens.length === 0 ? (
          <p className="text-sm text-muted py-2">{t.noTokens}</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-muted">
                <th className="py-2 pr-4 font-medium">{t.tokenName}</th>
                <th className="py-2 pr-4 font-medium">{t.tokenPrefix}</th>
                <th className="py-2 pr-4 font-medium">{t.tokenStatus}</th>
                <th className="py-2 pr-4 font-medium">{t.tokenCreated}</th>
                <th className="py-2 pr-4 font-medium">{t.tokenExpires}</th>
                <th className="py-2 pr-4 font-medium">{t.tokenLastUsed}</th>
                <th className="py-2 font-medium">{commonT.action}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--border-color)]">
              {account.tokens.map((token) => (
                <tr key={token.id} className="align-top">
                  <td className="py-2 pr-4 text-default whitespace-nowrap">{token.name}</td>
                  <td className="py-2 pr-4">
                    <code className="font-mono text-default">{token.tokenPrefix}…</code>
                  </td>
                  <td className="py-2 pr-4 whitespace-nowrap">
                    <StatusBadge status={statusLabel[token.status]} type={TOKEN_STATUS_BADGE[token.status]} />
                    {token.revokedAt && (
                      <div className="text-xs text-muted mt-1">{t.tokenRevokedAt.replace("{time}", formatTimestamp(token.revokedAt))}</div>
                    )}
                  </td>
                  <td className="py-2 pr-4 text-muted whitespace-nowrap">
                    {formatTimestamp(token.createdAt)}
                    {token.createdBy && <div className="text-xs">{token.createdBy}</div>}
                  </td>
                  <td className="py-2 pr-4 text-muted whitespace-nowrap">
                    {token.expiresAt ? formatTimestamp(token.expiresAt) : t.tokenNeverExpires}
                  </td>
                  <td className="py-2 pr-4 text-muted whitespace-nowrap">
                    {token.lastUsedAt ? formatTimestamp(token.lastUsedAt) : t.neverUsed}
                  </td>
                  <td className="py-2">
                    {token.status !== "revoked" && (
                      <button type="button" onClick={() => onRevokeToken(account, token)} disabled={readOnly}
                        className={`${ACTION_BUTTON_CLASS} hover:text-red-500`} title={t.revokeToken} aria-label={t.revokeToken}>
                        <Ban className="w-4 h-4" />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
