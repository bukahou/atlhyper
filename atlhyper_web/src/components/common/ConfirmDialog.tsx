"use client";

import { X, AlertTriangle, Loader2 } from "lucide-react";

interface ConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  loading?: boolean;
  variant?: "danger" | "warning" | "info";
}

export function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = "确认",
  cancelText = "取消",
  loading = false,
  variant = "warning",
}: ConfirmDialogProps) {
  if (!isOpen) return null;

  const variantStyles = {
    danger: {
      icon: "text-red-500",
      button: "bg-red-500 hover:bg-red-600",
    },
    warning: {
      icon: "text-yellow-500",
      button: "bg-yellow-500 hover:bg-yellow-600",
    },
    info: {
      icon: "text-blue-500",
      button: "bg-blue-500 hover:bg-blue-600",
    },
  };

  const styles = variantStyles[variant];

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/50 z-[60]" onClick={onClose} />

      {/* Dialog */}
      <div className="fixed inset-0 flex items-center justify-center z-[60] p-4">
        <div
          className="bg-card rounded-xl shadow-2xl w-full max-w-md"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="flex items-center justify-between p-4 border-b border-[var(--border-color)]">
            <div className="flex items-center gap-3">
              <AlertTriangle className={`w-5 h-5 ${styles.icon}`} />
              <h2 className="text-lg font-semibold text-default">{title}</h2>
            </div>
            <button onClick={onClose} className="p-2 rounded-lg hover-bg" disabled={loading}>
              <X className="w-4 h-4 text-muted" />
            </button>
          </div>

          {/* Content */}
          <div className="p-4">
            <p className="text-secondary">{message}</p>
          </div>

          {/* Footer */}
          <div className="flex items-center justify-end gap-3 p-4 border-t border-[var(--border-color)]">
            <button
              onClick={onClose}
              disabled={loading}
              className="px-4 py-2 text-sm font-medium text-muted hover:text-default rounded-lg hover-bg transition-colors disabled:opacity-50"
            >
              {cancelText}
            </button>
            <button
              onClick={onConfirm}
              disabled={loading}
              className={`px-4 py-2 text-sm font-medium text-white rounded-lg transition-colors disabled:opacity-50 flex items-center gap-2 ${styles.button}`}
            >
              {loading && <Loader2 className="w-4 h-4 animate-spin" />}
              {confirmText}
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
