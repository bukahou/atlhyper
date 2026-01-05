"use client";

import { useEffect, useState, useCallback } from "react";
import { CheckCircle, XCircle, X } from "lucide-react";

interface ToastMessage {
  id: number;
  type: "success" | "error";
  message: string;
}

let toastId = 0;
let addToastFn: ((type: "success" | "error", message: string) => void) | null = null;

// 全局 toast 调用函数
export const toast = {
  success: (message: string) => addToastFn?.("success", message),
  error: (message: string) => addToastFn?.("error", message),
};

export function ToastContainer() {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const addToast = useCallback((type: "success" | "error", message: string) => {
    const id = ++toastId;
    setToasts((prev) => [...prev, { id, type, message }]);

    // 3秒后自动移除
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 3000);
  }, []);

  const removeToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  // 注册全局函数
  useEffect(() => {
    addToastFn = addToast;
    return () => {
      addToastFn = null;
    };
  }, [addToast]);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed top-4 right-4 z-[100] space-y-2">
      {toasts.map((t) => (
        <div
          key={t.id}
          className={`flex items-center gap-3 px-4 py-3 rounded-lg shadow-lg border ${
            t.type === "success"
              ? "bg-green-50 dark:bg-green-900/30 border-green-200 dark:border-green-800 text-green-800 dark:text-green-300"
              : "bg-red-50 dark:bg-red-900/30 border-red-200 dark:border-red-800 text-red-800 dark:text-red-300"
          }`}
        >
          {t.type === "success" ? (
            <CheckCircle className="w-5 h-5 flex-shrink-0" />
          ) : (
            <XCircle className="w-5 h-5 flex-shrink-0" />
          )}
          <span className="text-sm">{t.message}</span>
          <button
            onClick={() => removeToast(t.id)}
            className="ml-2 p-1 hover:bg-black/10 dark:hover:bg-white/10 rounded"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      ))}
    </div>
  );
}
