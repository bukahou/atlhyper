"use client";

import { useState, useEffect, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { PageHeader, LoadingSpinner, DemoBanner } from "@/components/common";
import { useCan } from "@/hooks/useCan";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { getUserList, updateUserStatus } from "@/api/auth";
import { toast } from "@/components/common";
import { Plus } from "lucide-react";
import type { UserListItem } from "@/types/auth";

import { AddUserModal, DeleteUserModal, EditRoleModal, UserRow } from "./components";
import { MOCK_USERS } from "./components/mock-data";

export default function UsersPage() {
  const { t } = useI18n();
  const meLoaded = useAuthStore((st) => st.meLoaded);
  const can = useCan();
  const requireAuth = useRequireAuth();
  const [users, setUsers] = useState<UserListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [editUser, setEditUser] = useState<UserListItem | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<UserListItem | null>(null);

  // 用户管理是纯管理功能页：无 user.manage 时整页展示演示数据
  const isAdmin = can("user.manage");
  const isDemo = meLoaded && !isAdmin;

  const fetchUsers = useCallback(async () => {
    if (!meLoaded) return;
    if (!isAdmin) {
      setUsers(MOCK_USERS);
      setError("");
      setLoading(false);
      return;
    }
    try {
      const res = await getUserList();
      setUsers(res.data.data || []);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [t.common.loadFailed, meLoaded, isAdmin]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const handleAddUser = () => {
    requireAuth(() => setAddModalOpen(true), "user.manage");
  };

  const handleEditRole = (user: UserListItem) => {
    requireAuth(() => setEditUser(user), "user.manage");
  };

  const handleDeleteUser = (user: UserListItem) => {
    requireAuth(() => setDeleteTarget(user), "user.manage");
  };

  const handleToggleStatus = async (user: UserListItem) => {
    if (!requireAuth(() => {}, "user.manage")) return;

    const newStatus = user.status === 1 ? 0 : 1;
    try {
      await updateUserStatus({ userId: user.id, status: newStatus });
      toast.success(newStatus === 1 ? t.users.enable : t.users.disable);
      fetchUsers();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t.common.loadFailed);
    }
  };

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
          {loading ? (
            <div className="py-12">
              <LoadingSpinner />
            </div>
          ) : error ? (
            <div className="text-center py-12 text-red-500">{error}</div>
          ) : (
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
                {users.length === 0 && (
                  <div className="px-4 py-12 text-center text-muted">
                    {t.common.noData}
                  </div>
                )}
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
          )}
        </div>
      </div>

      {/* Modals */}
      <AddUserModal
        isOpen={addModalOpen}
        onClose={() => setAddModalOpen(false)}
        onSuccess={fetchUsers}
        t={t}
      />
      <EditRoleModal
        user={editUser}
        onClose={() => setEditUser(null)}
        onSuccess={fetchUsers}
        t={t}
      />
      <DeleteUserModal
        user={deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onSuccess={fetchUsers}
        t={t}
      />
    </Layout>
  );
}
