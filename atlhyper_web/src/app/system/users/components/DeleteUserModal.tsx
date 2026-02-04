"use client";

import { useState } from "react";
import { X } from "lucide-react";
import { deleteUser } from "@/api/auth";
import { toast } from "@/components/common";
import type { UserListItem } from "@/types/auth";
import type { useI18n } from "@/i18n/context";

interface DeleteUserModalProps {
  user: UserListItem | null;
  onClose: () => void;
  onSuccess: () => void;
  t: ReturnType<typeof useI18n>["t"];
}

export function DeleteUserModal({ user, onClose, onSuccess, t }: DeleteUserModalProps) {
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
