"use client";

import { useState, useEffect } from "react";
import { X, Shield, User, Eye } from "lucide-react";
import { updateUserRole } from "@/api/auth";
import { toast } from "@/components/common";
import type { UserListItem } from "@/types/auth";
import { UserRole } from "@/types/auth";
import type { useI18n } from "@/i18n/context";

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

interface EditRoleModalProps {
  user: UserListItem | null;
  onClose: () => void;
  onSuccess: () => void;
  t: ReturnType<typeof useI18n>["t"];
}

export function EditRoleModal({ user, onClose, onSuccess, t }: EditRoleModalProps) {
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
    <div className="fixed inset-0 bg-black/50 flex items-end sm:items-center justify-center z-50 p-0 sm:p-4">
      <div className="bg-card rounded-t-xl sm:rounded-xl border border-[var(--border-color)] p-4 sm:p-6 w-full sm:max-w-sm">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-base sm:text-lg font-semibold text-default">{t.users.changeRole}</h3>
          <button onClick={onClose} className="p-1.5 hover-bg rounded">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-3 sm:space-y-4">
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
                  className={`flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors active:scale-[0.98] ${
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
              className="flex-1 px-4 py-2.5 sm:py-2 border border-[var(--border-color)] rounded-lg hover-bg text-sm"
            >
              {t.common.cancel}
            </button>
            <button
              type="submit"
              disabled={loading || !roleChanged}
              className="flex-1 px-4 py-2.5 sm:py-2 bg-primary hover:bg-primary-hover text-white rounded-lg disabled:opacity-50 text-sm"
            >
              {loading ? t.common.loading : t.common.confirm}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
