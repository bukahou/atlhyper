"use client";

import { useMutation } from "@tanstack/react-query";
import { X } from "lucide-react";
import { deleteUserMutation } from "@/queries/admin";
import { toast } from "@/components/common";
import type { UserListItem } from "@/types/auth";
import type { useI18n } from "@/i18n/context";

interface DeleteUserModalProps {
  user: UserListItem;
  onClose: () => void;
  t: ReturnType<typeof useI18n>["t"];
}

/** 删除用户确认弹窗。由页面在选中要删的用户时挂载 */
export function DeleteUserModal({ user, onClose, t }: DeleteUserModalProps) {
  // 成功后用户列表由工厂负责失效重取
  const remove = useMutation(deleteUserMutation());
  const loading = remove.isPending;
  const error = remove.isError ? (remove.error instanceof Error ? remove.error.message : t.common.loadFailed) : "";

  const handleDelete = () => {
    remove.mutate(user.id, {
      onSuccess: () => {
        toast.success(t.common.success);
        onClose();
      },
    });
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-end sm:items-center justify-center z-50 p-0 sm:p-4">
      <div className="bg-card rounded-t-xl sm:rounded-xl border border-[var(--border-color)] p-4 sm:p-6 w-full sm:max-w-sm">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-base sm:text-lg font-semibold text-default">{t.users.deleteConfirmTitle}</h3>
          <button onClick={onClose} className="p-1.5 hover-bg rounded">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        <div className="space-y-3 sm:space-y-4">
          <p className="text-secondary text-sm sm:text-base">
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
              className="flex-1 px-4 py-2.5 sm:py-2 border border-[var(--border-color)] rounded-lg hover-bg text-sm"
            >
              {t.common.cancel}
            </button>
            <button
              onClick={handleDelete}
              disabled={loading}
              className="flex-1 px-4 py-2.5 sm:py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg disabled:opacity-50 text-sm"
            >
              {loading ? t.common.loading : t.common.delete}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
