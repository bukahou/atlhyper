"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { X } from "lucide-react";
import { registerUserMutation } from "@/queries/admin";
import { toast } from "@/components/common";
import { UserRole } from "@/types/auth";
import type { useI18n } from "@/i18n/context";

interface AddUserModalProps {
  onClose: () => void;
  t: ReturnType<typeof useI18n>["t"];
}

/** 新增用户弹窗。由页面在打开时挂载：每次打开都是一张空表单 */
export function AddUserModal({ onClose, t }: AddUserModalProps) {
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
  // 成功后用户列表由工厂负责失效重取
  const register = useMutation(registerUserMutation());
  const loading = register.isPending;
  const error = register.isError
    ? register.error instanceof Error
      ? register.error.message
      : t.common.loadFailed
    : "";

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    register.mutate(
      {
        username: form.username,
        password: form.password,
        displayName: form.displayName,
        email: form.email,
        role: form.role,
      },
      {
        onSuccess: () => {
          toast.success(t.common.success);
          onClose();
        },
      },
    );
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-end sm:items-center justify-center z-50 p-0 sm:p-4">
      <div className="bg-card rounded-t-xl sm:rounded-xl border border-[var(--border-color)] p-4 sm:p-6 w-full sm:max-w-md max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-base sm:text-lg font-semibold text-default">{t.users.addUser}</h3>
          <button onClick={onClose} className="p-1.5 hover-bg rounded">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-3 sm:space-y-4">
          <div>
            <label className="block text-sm font-medium text-muted mb-1">{t.users.username} *</label>
            <input
              type="text"
              required
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              className="w-full px-3 py-2.5 sm:py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none text-base sm:text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-muted mb-1">{t.common.password} *</label>
            <input
              type="password"
              required
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              className="w-full px-3 py-2.5 sm:py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none text-base sm:text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-muted mb-1">{t.users.displayName}</label>
            <input
              type="text"
              value={form.displayName}
              onChange={(e) => setForm({ ...form, displayName: e.target.value })}
              className="w-full px-3 py-2.5 sm:py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none text-base sm:text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-muted mb-1">{t.users.email}</label>
            <input
              type="email"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
              className="w-full px-3 py-2.5 sm:py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none text-base sm:text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-muted mb-1">{t.users.role}</label>
            <select
              value={form.role}
              onChange={(e) => setForm({ ...form, role: Number(e.target.value) })}
              className="w-full px-3 py-2.5 sm:py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:ring-2 focus:ring-primary outline-none text-base sm:text-sm"
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
              className="flex-1 px-4 py-2.5 sm:py-2 border border-[var(--border-color)] rounded-lg hover-bg text-sm"
            >
              {t.common.cancel}
            </button>
            <button
              type="submit"
              disabled={loading}
              className="flex-1 px-4 py-2.5 sm:py-2 bg-primary hover:bg-primary-hover text-white rounded-lg disabled:opacity-50 text-sm"
            >
              {loading ? t.common.loading : t.common.add}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
