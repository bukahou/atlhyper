"use client";

import { useState, useEffect, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { getUserList, registerUser, updateUserRole, deleteUser } from "@/api/auth";
import { toast } from "@/components/common";
import { Plus, Edit2, Trash2, Shield, User, Eye, X } from "lucide-react";
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
}: {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const [form, setForm] = useState({
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
      });
      toast.success("用户添加成功");
      onSuccess();
      onClose();
      setForm({ username: "", password: "", displayName: "", email: "", role: UserRole.VIEWER });
    } catch (err) {
      setError(err instanceof Error ? err.message : "注册失败");
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-default">添加用户</h3>
          <button onClick={onClose} className="p-1 hover-bg rounded">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-muted mb-1">用户名 *</label>
            <input
              type="text"
              required
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-muted mb-1">密码 *</label>
            <input
              type="password"
              required
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-muted mb-1">显示名称</label>
            <input
              type="text"
              value={form.displayName}
              onChange={(e) => setForm({ ...form, displayName: e.target.value })}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-muted mb-1">邮箱</label>
            <input
              type="email"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none"
            />
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
              取消
            </button>
            <button
              type="submit"
              disabled={loading}
              className="flex-1 px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg disabled:opacity-50"
            >
              {loading ? "添加中..." : "添加"}
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
}: {
  user: UserListItem | null;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleDelete = async () => {
    if (!user) return;

    setError("");
    setLoading(true);

    try {
      await deleteUser(user.ID);
      toast.success("用户删除成功");
      onSuccess();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除失败");
    } finally {
      setLoading(false);
    }
  };

  if (!user) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-6 w-full max-w-sm">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-default">确认删除</h3>
          <button onClick={onClose} className="p-1 hover-bg rounded">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        <div className="space-y-4">
          <p className="text-secondary">
            确定要删除用户 <span className="font-medium text-default">{user.Username}</span> 吗？
          </p>
          <p className="text-sm text-muted">
            此操作不可撤销，该用户的所有数据将被永久删除。
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
              取消
            </button>
            <button
              onClick={handleDelete}
              disabled={loading}
              className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg disabled:opacity-50"
            >
              {loading ? "删除中..." : "确认删除"}
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
}: {
  user: UserListItem | null;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const [role, setRole] = useState(user?.Role || UserRole.VIEWER);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (user) setRole(user.Role);
  }, [user]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!user) return;

    setError("");
    setLoading(true);

    try {
      await updateUserRole({ userId: user.ID, role });
      toast.success("角色更新成功");
      onSuccess();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "更新失败");
    } finally {
      setLoading(false);
    }
  };

  if (!user) return null;

  const roleChanged = role !== user.Role;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-6 w-full max-w-sm">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-default">编辑角色</h3>
          <button onClick={onClose} className="p-1 hover-bg rounded">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-muted mb-1">用户</label>
            <div className="text-default font-medium">{user.Username}</div>
          </div>

          <div>
            <label className="block text-sm font-medium text-muted mb-2">角色</label>
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

          {roleChanged && (
            <div className="p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 text-yellow-800 dark:text-yellow-300 rounded-lg text-sm">
              确认将 <span className="font-medium">{user.Username}</span> 的角色从 {roleConfig[user.Role as keyof typeof roleConfig]?.label} 更改为 {roleConfig[role as keyof typeof roleConfig]?.label}？
            </div>
          )}

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
              取消
            </button>
            <button
              type="submit"
              disabled={loading || !roleChanged}
              className="flex-1 px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg disabled:opacity-50"
            >
              {loading ? "保存中..." : "确认修改"}
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
      setError(err instanceof Error ? err.message : "加载失败");
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
      添加用户
    </button>
  ) : null;

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.users} description="用户账户管理" actions={addButton} />

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
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">用户</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">邮箱</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">角色</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">创建时间</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">最后登录</th>
                  {isAdmin && (
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">操作</th>
                  )}
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--border-color)]">
                {users.map((user) => {
                  const config = roleConfig[user.Role as keyof typeof roleConfig] || roleConfig[UserRole.VIEWER];
                  const RoleIcon = config.icon;
                  return (
                    <tr key={user.ID} className="hover:bg-[var(--background)]">
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
                            <RoleIcon className="w-4 h-4 text-primary" />
                          </div>
                          <div>
                            <div className="font-medium text-default">{user.Username}</div>
                            {user.DisplayName && (
                              <div className="text-xs text-muted">{user.DisplayName}</div>
                            )}
                          </div>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-sm text-secondary">
                        {user.Email || "-"}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${config.color}`}>
                          {config.label}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm text-secondary">
                        {formatDate(user.CreatedAt)}
                      </td>
                      <td className="px-4 py-3 text-sm text-secondary">
                        {formatDate(user.LastLogin)}
                      </td>
                      {isAdmin && (
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleEditRole(user)}
                              className="p-2 hover-bg rounded-lg"
                              title="编辑角色"
                            >
                              <Edit2 className="w-4 h-4 text-muted hover:text-primary" />
                            </button>
                            <button
                              onClick={() => handleDeleteUser(user)}
                              className="p-2 hover-bg rounded-lg"
                              title="删除用户"
                            >
                              <Trash2 className="w-4 h-4 text-muted hover:text-red-500" />
                            </button>
                          </div>
                        </td>
                      )}
                    </tr>
                  );
                })}
                {users.length === 0 && (
                  <tr>
                    <td colSpan={isAdmin ? 6 : 5} className="px-4 py-12 text-center text-muted">
                      暂无用户数据
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
      />
      <EditRoleModal
        user={editUser}
        onClose={() => setEditUser(null)}
        onSuccess={fetchUsers}
      />
      <DeleteUserModal
        user={deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onSuccess={fetchUsers}
      />
    </Layout>
  );
}
