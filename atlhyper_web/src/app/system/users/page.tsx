"use client";

import { useState, useEffect, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { getUserList, registerUser, updateUserRole, updateUserStatus, deleteUser } from "@/api/auth";
import { toast } from "@/components/common";
import { Plus, Edit2, Trash2, Shield, User, Eye, X, Power, PowerOff } from "lucide-react";
import type { UserListItem } from "@/types/auth";
import { UserRole } from "@/types/auth";

// 角色显示配置
const roleConfig = {
  [UserRole.ADMIN]: {
    label: "Admin",
    color: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
    icon: Shield,
  },
  [UserRole.OPERATOR]: {
    label: "Operator",
    color: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
    icon: User,
  },
  [UserRole.VIEWER]: {
    label: "Viewer",
    color: "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300",
    icon: Eye,
  },
};

// 添加用户弹窗
function AddUserModal({
  isOpen,
  onClose,
  onSuccess,
  t,
}: {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  t: ReturnType<typeof useI18n>["t"];
}) {
  const [form, setForm] = useState<{
    username: string;
    password: string;
    displayName: string;
    email: string;
    role: number;
  }>({
    username: "",
    password: "",
    displayName: "",
    email: "",
    role: UserRole.VIEWER,
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      await registerUser({
        username: form.username,
        password: form.password,
        displayName: form.displayName,
        email: form.email,
        role: form.role,
      });
      toast.success(t.common.success);
      onSuccess();
      onClose();
      setForm({ username: "", password: "", displayName: "", email: "", role: UserRole.VIEWER });
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-default">{t.users.addUser}</h3>
          <button onClick={onClose} className="p-1 hover-bg rounded">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-muted mb-1">{t.users.username} *</label>
            <input
              type="text"
              required
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-muted mb-1">{t.common.password} *</label>
            <input
              type="password"
              required
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-muted mb-1">{t.users.displayName}</label>
            <input
              type="text"
              value={form.displayName}
              onChange={(e) => setForm({ ...form, displayName: e.target.value })}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-muted mb-1">{t.users.email}</label>
            <input
              type="email"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-muted mb-1">{t.users.role}</label>
            <select
              value={form.role}
              onChange={(e) => setForm({ ...form, role: Number(e.target.value) })}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none"
            >
              <option value={UserRole.VIEWER}>{t.users.roleViewer}</option>
              <option value={UserRole.OPERATOR}>{t.users.roleOperator}</option>
              <option value={UserRole.ADMIN}>{t.users.roleAdmin}</option>
            </select>
          </div>

          {error && (
            <div className="p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 border border-[var(--border-color)] rounded-lg hover-bg"
            >
              {t.common.cancel}
            </button>
            <button
              type="submit"
              disabled={loading}
              className="flex-1 px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg disabled:opacity-50"
            >
              {loading ? t.common.loading : t.common.add}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// 删除用户确认弹窗
function DeleteUserModal({
  user,
  onClose,
  onSuccess,
  t,
}: {
  user: UserListItem | null;
  onClose: () => void;
  onSuccess: () => void;
  t: ReturnType<typeof useI18n>["t"];
}) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleDelete = async () => {
    if (!user) return;

    setError("");
    setLoading(true);

    try {
      await deleteUser(user.id);
      toast.success(t.common.success);
      onSuccess();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  };

  if (!user) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-6 w-full max-w-sm">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-default">{t.users.deleteConfirmTitle}</h3>
          <button onClick={onClose} className="p-1 hover-bg rounded">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        <div className="space-y-4">
          <p className="text-secondary">
            {t.users.deleteConfirmMessage.replace("{name}", user.username)}
          </p>

          {error && (
            <div className="p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 border border-[var(--border-color)] rounded-lg hover-bg"
            >
              {t.common.cancel}
            </button>
            <button
              onClick={handleDelete}
              disabled={loading}
              className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg disabled:opacity-50"
            >
              {loading ? t.common.loading : t.common.delete}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// 编辑角色弹窗
function EditRoleModal({
  user,
  onClose,
  onSuccess,
  t,
}: {
  user: UserListItem | null;
  onClose: () => void;
  onSuccess: () => void;
  t: ReturnType<typeof useI18n>["t"];
}) {
  const [role, setRole] = useState(user?.role || UserRole.VIEWER);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (user) setRole(user.role);
  }, [user]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!user) return;

    setError("");
    setLoading(true);

    try {
      await updateUserRole({ userId: user.id, role });
      toast.success(t.common.success);
      onSuccess();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  };

  if (!user) return null;

  const roleChanged = role !== user.role;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-6 w-full max-w-sm">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-default">{t.users.changeRole}</h3>
          <button onClick={onClose} className="p-1 hover-bg rounded">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-muted mb-1">{t.users.username}</label>
            <div className="text-default font-medium">{user.username}</div>
          </div>

          <div>
            <label className="block text-sm font-medium text-muted mb-2">{t.users.role}</label>
            <div className="space-y-2">
              {Object.entries(roleConfig).map(([value, config]) => (
                <label
                  key={value}
                  className={`flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${
                    role === Number(value)
                      ? "border-primary bg-primary/5"
                      : "border-[var(--border-color)] hover:bg-[var(--background)]"
                  }`}
                >
                  <input
                    type="radio"
                    name="role"
                    value={value}
                    checked={role === Number(value)}
                    onChange={() => setRole(Number(value))}
                    className="sr-only"
                  />
                  <config.icon className="w-4 h-4 text-muted" />
                  <span className="text-default">{config.label}</span>
                </label>
              ))}
            </div>
          </div>

          {error && (
            <div className="p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 border border-[var(--border-color)] rounded-lg hover-bg"
            >
              {t.common.cancel}
            </button>
            <button
              type="submit"
              disabled={loading || !roleChanged}
              className="flex-1 px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg disabled:opacity-50"
            >
              {loading ? t.common.loading : t.common.confirm}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default function UsersPage() {
  const { t } = useI18n();
  const { user: currentUser, isAuthenticated, openLoginDialog } = useAuthStore();
  const [users, setUsers] = useState<UserListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [editUser, setEditUser] = useState<UserListItem | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<UserListItem | null>(null);

  const isAdmin = currentUser?.role === UserRole.ADMIN;

  const fetchUsers = useCallback(async () => {
    try {
      const res = await getUserList();
      setUsers(res.data.data || []);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const handleAddUser = () => {
    if (!isAuthenticated) {
      openLoginDialog(() => setAddModalOpen(true));
    } else if (isAdmin) {
      setAddModalOpen(true);
    }
  };

  const handleEditRole = (user: UserListItem) => {
    if (!isAuthenticated) {
      openLoginDialog(() => setEditUser(user));
    } else if (isAdmin) {
      setEditUser(user);
    }
  };

  const handleDeleteUser = (user: UserListItem) => {
    if (!isAuthenticated) {
      openLoginDialog(() => setDeleteTarget(user));
    } else if (isAdmin) {
      setDeleteTarget(user);
    }
  };

  const handleToggleStatus = async (user: UserListItem) => {
    if (!isAuthenticated) {
      openLoginDialog(() => handleToggleStatus(user));
      return;
    }
    if (!isAdmin) return;

    const newStatus = user.status === 1 ? 0 : 1;
    try {
      await updateUserStatus({ userId: user.id, status: newStatus });
      toast.success(newStatus === 1 ? t.users.enable : t.users.disable);
      fetchUsers();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t.common.loadFailed);
    }
  };

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return "-";
    return new Date(dateStr).toLocaleString();
  };

  const addButton = isAdmin ? (
    <button
      onClick={handleAddUser}
      className="flex items-center gap-2 px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg transition-colors"
    >
      <Plus className="w-4 h-4" />
      {t.users.addUser}
    </button>
  ) : null;

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.users} description={t.users.pageDescription} actions={addButton} />

        {/* Users Table */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          {loading ? (
            <div className="py-12">
              <LoadingSpinner />
            </div>
          ) : error ? (
            <div className="text-center py-12 text-red-500">{error}</div>
          ) : (
            <table className="w-full">
              <thead className="bg-[var(--background)]">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.users.username}</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.users.email}</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.users.role}</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.users.status}</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.users.createdAt}</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.users.lastLogin}</th>
                  {isAdmin && (
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">{t.common.action}</th>
                  )}
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--border-color)]">
                {users.map((user) => {
                  const config = roleConfig[user.role as keyof typeof roleConfig] || roleConfig[UserRole.VIEWER];
                  const RoleIcon = config.icon;
                  return (
                    <tr key={user.id} className="hover:bg-[var(--background)]">
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
                            <RoleIcon className="w-4 h-4 text-primary" />
                          </div>
                          <div>
                            <div className="font-medium text-default">{user.username}</div>
                            {user.displayName && (
                              <div className="text-xs text-muted">{user.displayName}</div>
                            )}
                          </div>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-sm text-secondary">
                        {user.email || "-"}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${config.color}`}>
                          {config.label}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${
                          user.status === 1
                            ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                            : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
                        }`}>
                          {user.status === 1 ? t.users.statusActive : t.users.statusDisabled}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm text-secondary">
                        {formatDate(user.createdAt)}
                      </td>
                      <td className="px-4 py-3 text-sm text-secondary">
                        <div>{formatDate(user.lastLogin)}</div>
                        {user.lastLoginIP && (
                          <div className="text-xs text-muted">{user.lastLoginIP}</div>
                        )}
                      </td>
                      {isAdmin && (
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleEditRole(user)}
                              className="p-2 hover-bg rounded-lg"
                              title={t.users.changeRole}
                            >
                              <Edit2 className="w-4 h-4 text-muted hover:text-primary" />
                            </button>
                            {/* admin 用户不可禁用 */}
                            {user.username !== "admin" && (
                              <button
                                onClick={() => handleToggleStatus(user)}
                                className="p-2 hover-bg rounded-lg"
                                title={user.status === 1 ? t.users.disable : t.users.enable}
                              >
                                {user.status === 1 ? (
                                  <PowerOff className="w-4 h-4 text-muted hover:text-yellow-500" />
                                ) : (
                                  <Power className="w-4 h-4 text-muted hover:text-green-500" />
                                )}
                              </button>
                            )}
                            {/* admin 用户不可删除 */}
                            {user.username !== "admin" && (
                              <button
                                onClick={() => handleDeleteUser(user)}
                                className="p-2 hover-bg rounded-lg"
                                title={t.users.deleteUser}
                              >
                                <Trash2 className="w-4 h-4 text-muted hover:text-red-500" />
                              </button>
                            )}
                          </div>
                        </td>
                      )}
                    </tr>
                  );
                })}
                {users.length === 0 && (
                  <tr>
                    <td colSpan={isAdmin ? 7 : 6} className="px-4 py-12 text-center text-muted">
                      {t.common.noData}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
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
