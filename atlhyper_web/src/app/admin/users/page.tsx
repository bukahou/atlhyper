"use client";

import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner, DemoBanner, QueryView } from "@/components/common";
import { useCan } from "@/hooks/useCan";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useSessionReady } from "@/hooks/useSessionReady";
import { updateUserStatusMutation, userListQuery } from "@/queries/admin";
import { toast } from "@/components/common";
import { Plus } from "lucide-react";
import type { UserListItem } from "@/types/auth";

import { AddUserModal, DeleteUserModal, EditRoleModal, UserRow } from "./components";
import { MOCK_USERS } from "./components/mock-data";

export default function UsersPage() {
  const { t } = useI18n();
  const sessionReady = useSessionReady();
  const can = useCan();
  const requireAuth = useRequireAuth();
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [editUser, setEditUser] = useState<UserListItem | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<UserListItem | null>(null);

  // 用户管理是纯管理功能页：权限加载完才决定显示真实数据还是演示数据；无 user.manage 时整页展示演示数据
  const isAdmin = can("user.manage");
  const isDemo = sessionReady && !isAdmin;

  const usersQuery = useQuery(userListQuery(isAdmin));
  // 用户变了之后列表由工厂负责失效重取，这里只管提示
  const updateStatus = useMutation(updateUserStatusMutation());

  const handleAddUser = () => {
    requireAuth(() => setAddModalOpen(true), "user.manage");
  };

  const handleEditRole = (user: UserListItem) => {
    requireAuth(() => setEditUser(user), "user.manage");
  };

  const handleDeleteUser = (user: UserListItem) => {
    requireAuth(() => setDeleteTarget(user), "user.manage");
  };

  const handleToggleStatus = (user: UserListItem) => {
    if (!requireAuth(() => {}, "user.manage")) return;

    const newStatus = user.status === 1 ? 0 : 1;
    updateStatus.mutate(
      { userId: user.id, status: newStatus },
      {
        onSuccess: () => toast.success(newStatus === 1 ? t.users.enable : t.users.disable),
        onError: (err) => toast.error(err instanceof Error ? err.message : t.common.loadFailed),
      },
    );
  };

  const renderUsers = (users: UserListItem[]) => (
    <>
      {/* 移动端卡片视图 */}
      <div className="md:hidden divide-y divide-[var(--border-color)]">
        {users.map((user) => (
          <UserRow
            key={user.id}
            user={user}
            isAdmin={isAdmin}
            onEditRole={handleEditRole}
            onToggleStatus={handleToggleStatus}
            onDelete={handleDeleteUser}
            t={t}
            isMobile
          />
        ))}
        {users.length === 0 && <div className="px-4 py-12 text-center text-muted">{t.common.noData}</div>}
      </div>

      {/* 桌面端表格视图 */}
      <table className="w-full hidden md:table">
        <thead className="bg-[var(--background)]">
          <tr>
            <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.users.username}</th>
            <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.users.email}</th>
            <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.users.role}</th>
            <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.users.status}</th>
            <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.users.createdAt}</th>
            {isAdmin && (
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.common.action}</th>
            )}
          </tr>
        </thead>
        <tbody className="divide-y divide-[var(--border-color)]">
          {users.map((user) => (
            <UserRow
              key={user.id}
              user={user}
              isAdmin={isAdmin}
              onEditRole={handleEditRole}
              onToggleStatus={handleToggleStatus}
              onDelete={handleDeleteUser}
              t={t}
            />
          ))}
          {users.length === 0 && (
            <tr>
              <td colSpan={isAdmin ? 6 : 5} className="px-4 py-12 text-center text-muted">
                {t.common.noData}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </>
  );

  const addButton = isAdmin ? (
    <button
      onClick={handleAddUser}
      className="flex items-center gap-1.5 sm:gap-2 px-3 sm:px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg transition-colors text-sm"
    >
      <Plus className="w-4 h-4" />
      <span className="hidden sm:inline">{t.users.addUser}</span>
      <span className="sm:hidden">{t.common.add}</span>
    </button>
  ) : null;

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.users} description={t.users.pageDescription} actions={addButton} />

        {isDemo && <DemoBanner hint={t.common.demoModeHintUsers} />}

        {/* Users Table */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          {!sessionReady ? (
            <div className="py-12">
              <LoadingSpinner />
            </div>
          ) : isDemo ? (
            renderUsers(MOCK_USERS)
          ) : (
            <QueryView query={usersQuery}>{renderUsers}</QueryView>
          )}
        </div>
      </div>

      {/* Modals：打开时才挂载，表单与上一次的错误随关闭丢弃 */}
      {addModalOpen && <AddUserModal onClose={() => setAddModalOpen(false)} t={t} />}
      {editUser && <EditRoleModal key={editUser.id} user={editUser} onClose={() => setEditUser(null)} t={t} />}
      {deleteTarget && (
        <DeleteUserModal key={deleteTarget.id} user={deleteTarget} onClose={() => setDeleteTarget(null)} t={t} />
      )}
    </Layout>
  );
}
