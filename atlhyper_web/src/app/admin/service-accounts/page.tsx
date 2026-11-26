"use client";

import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { IdCard, Plus } from "lucide-react";
import type { APITokenView, ServiceAccountLimits, ServiceAccountView } from "@/api/service-accounts";
import { ConfirmDialog, DemoBanner, LoadingSpinner, PageHeader, QueryView, toast } from "@/components/common";
import { Layout } from "@/components/layout/Layout";
import { useCan } from "@/hooks/useCan";
import { useSessionReady } from "@/hooks/useSessionReady";
import { useI18n } from "@/i18n/context";
import { serviceAccountFailure } from "@/lib/service-account-input";
import { MOCK_SERVICE_ACCOUNT_LIMITS, MOCK_SERVICE_ACCOUNTS } from "@/mock/service-accounts";
import { REFRESH } from "@/queries/refresh";
import {
  deleteServiceAccountMutation,
  revokeAPITokenMutation,
  serviceAccountsQuery,
  type ServiceAccountsData,
} from "@/queries/service-accounts";
import { getRoles } from "@/app/admin/roles/roles";
import { IssuedTokenDialog, type IssuedServiceAccountToken } from "./components/IssuedTokenDialog";
import { IssueTokenModal } from "./components/IssueTokenModal";
import { ServiceAccountCard } from "./components/ServiceAccountCard";
import { ServiceAccountModal, type ServiceAccountModalTarget } from "./components/ServiceAccountModal";

interface RevokeTarget {
  account: ServiceAccountView;
  token: APITokenView;
}

const DEMO_DATA: ServiceAccountsData = { accounts: MOCK_SERVICE_ACCOUNTS, limits: MOCK_SERVICE_ACCOUNT_LIMITS };

export default function ServiceAccountsPage() {
  const { t } = useI18n();
  const accountsT = t.serviceAccounts;
  // 服务账号管理是纯管理功能页：权限加载完才决定显示真实数据还是演示数据；无 user.manage 时整页演示数据
  const sessionReady = useSessionReady();
  const canManage = useCan()("user.manage");
  const isDemo = sessionReady && !canManage;

  const accountsQuery = useQuery(serviceAccountsQuery(canManage));
  // 写操作成功后工厂让列表重新取数，这里只管提示与关弹窗
  const deleteAccount = useMutation(deleteServiceAccountMutation());
  const revokeToken = useMutation(revokeAPITokenMutation());

  const [modalTarget, setModalTarget] = useState<ServiceAccountModalTarget | null>(null);
  const [issueTarget, setIssueTarget] = useState<ServiceAccountView | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<ServiceAccountView | null>(null);
  const [revokeTarget, setRevokeTarget] = useState<RevokeTarget | null>(null);
  // 刚签发的令牌明文只放在这里：不进查询缓存、不落 localStorage、不打日志；关掉弹窗即置空丢弃
  const [issuedToken, setIssuedToken] = useState<IssuedServiceAccountToken | null>(null);

  // 等级的名称与说明沿用角色权限页的文案；可选的等级由后端下发
  const roles = getRoles(t.roles);
  const roleOf = (level: number) => roles.find((role) => role.level === level);
  const limits: ServiceAccountLimits | undefined = isDemo ? MOCK_SERVICE_ACCOUNT_LIMITS : accountsQuery.data?.limits;
  const assignableRoles = roles.filter((role) => limits?.assignableRoles.includes(role.level));

  const handleDeleteConfirm = () => {
    if (!deleteTarget) return;
    deleteAccount.mutate(deleteTarget.id, {
      onSuccess: () => toast.success(accountsT.deletedToast),
      onError: (error) => toast.error(accountsT.errors[serviceAccountFailure(error, "delete")]),
      onSettled: () => setDeleteTarget(null),
    });
  };

  const handleRevokeConfirm = () => {
    if (!revokeTarget) return;
    revokeToken.mutate(
      { accountId: revokeTarget.account.id, tokenId: revokeTarget.token.id },
      {
        onSuccess: () => toast.success(accountsT.revokedToast),
        onError: (error) => toast.error(accountsT.errors[serviceAccountFailure(error, "revoke")]),
        onSettled: () => setRevokeTarget(null),
      },
    );
  };

  const renderAccounts = (data: ServiceAccountsData) => (
    <div className="space-y-4">
      {data.accounts.map((account) => (
        <ServiceAccountCard
          key={account.id}
          account={account}
          roleOf={roleOf}
          maxActiveTokens={data.limits.maxActiveTokens}
          readOnly={isDemo}
          onEdit={(target) => setModalTarget({ mode: "edit", account: target })}
          onIssueToken={setIssueTarget}
          onDelete={setDeleteTarget}
          onRevokeToken={(target, token) => setRevokeTarget({ account: target, token })}
          t={accountsT}
          commonT={t.common}
        />
      ))}
    </div>
  );

  const createButton = (
    <button
      type="button"
      onClick={() => setModalTarget({ mode: "create" })}
      // 演示模式下写操作不可用
      disabled={!canManage || !limits}
      className="flex items-center gap-1.5 sm:gap-2 px-3 sm:px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg transition-colors text-sm disabled:opacity-40 disabled:cursor-not-allowed"
    >
      <Plus className="w-4 h-4" />
      <span className="hidden sm:inline">{accountsT.createAccount}</span>
      <span className="sm:hidden">{accountsT.create}</span>
    </button>
  );

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title={t.nav.serviceAccounts}
          description={accountsT.pageDescription}
          autoRefreshSeconds={REFRESH.slow / 1000}
          actions={createButton}
        />

        {isDemo && <DemoBanner hint={accountsT.demoHint} />}

        {!sessionReady ? (
          <div className="py-12">
            <LoadingSpinner />
          </div>
        ) : isDemo ? (
          renderAccounts(DEMO_DATA)
        ) : (
          <QueryView
            query={accountsQuery}
            isEmpty={(data) => data.accounts.length === 0}
            empty={
              <div className="bg-card rounded-xl border border-[var(--border-color)] py-12 text-center">
                <IdCard className="w-12 h-12 mx-auto mb-3 text-muted opacity-50" />
                <p className="text-default font-medium mb-1">{accountsT.empty}</p>
                <p className="text-sm text-muted">{accountsT.emptyHint}</p>
              </div>
            }
          >
            {renderAccounts}
          </QueryView>
        )}
      </div>

      {/* 新建 / 编辑弹窗：按目标整个重新挂载，表单初值直接取自目标 */}
      {modalTarget && (
        <ServiceAccountModal
          key={modalTarget.mode === "edit" ? `edit:${modalTarget.account.id}` : "create"}
          target={modalTarget}
          roles={assignableRoles}
          onClose={() => setModalTarget(null)}
          t={accountsT}
          commonT={t.common}
        />
      )}

      {issueTarget && limits && (
        <IssueTokenModal
          key={issueTarget.id}
          account={issueTarget}
          maxExpiresInDays={limits.maxExpiresInDays}
          defaultExpiresInDays={limits.defaultExpiresInDays}
          onIssued={(token) =>
            setIssuedToken({ accountName: issueTarget.name, accountDisplayName: issueTarget.displayName, token })
          }
          onClose={() => setIssueTarget(null)}
          t={accountsT}
          commonT={t.common}
        />
      )}

      {/* 删除确认：令牌随之删除并立即失效 */}
      <ConfirmDialog
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDeleteConfirm}
        title={accountsT.deleteConfirmTitle}
        message={accountsT.deleteConfirmMessage.replace("{name}", deleteTarget?.name ?? "")}
        confirmText={accountsT.deleteAccount}
        cancelText={t.common.cancel}
        loading={deleteAccount.isPending}
        variant="danger"
      />

      {/* 吊销确认：使用它的调用方立即被拒绝 */}
      <ConfirmDialog
        isOpen={revokeTarget !== null}
        onClose={() => setRevokeTarget(null)}
        onConfirm={handleRevokeConfirm}
        title={accountsT.revokeConfirmTitle}
        message={accountsT.revokeConfirmMessage
          .replace("{name}", revokeTarget?.token.name ?? "")
          .replace("{prefix}", revokeTarget?.token.tokenPrefix ?? "")}
        confirmText={accountsT.revokeToken}
        cancelText={t.common.cancel}
        loading={revokeToken.isPending}
        variant="danger"
      />

      {/* 令牌只显示这一次：关闭即从内存丢弃 */}
      {issuedToken && <IssuedTokenDialog issued={issuedToken} onClose={() => setIssuedToken(null)} t={accountsT} />}
    </Layout>
  );
}
