"use client";

/**
 * 全局提示条：渲染、堆叠、自动消失与读屏播报（aria-live）交给 sonner。
 * 调用方只用这里的 toast.success / toast.error，不直接依赖 sonner。
 */
import { Toaster, toast as sonnerToast } from "sonner";
import { useI18n } from "@/i18n/context";
import { useTheme } from "@/theme/context";

export const toast = {
  success: (message: string) => {
    sonnerToast.success(message);
  },
  error: (message: string) => {
    sonnerToast.error(message);
  },
};

/** 在 Layout 里挂一次 */
export function ToastContainer() {
  const { resolvedTheme } = useTheme();
  const { t } = useI18n();
  return (
    <Toaster
      position="top-right"
      theme={resolvedTheme}
      richColors
      closeButton
      containerAriaLabel={t.common.notificationsRegion}
      toastOptions={{ closeButtonAriaLabel: t.common.close }}
    />
  );
}
