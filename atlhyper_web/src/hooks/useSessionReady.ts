import { useAuthStore } from "@/store/authStore";

/**
 * 权限是否已加载完。
 *
 * 加载完之前一律视为无权限，所以需要权限的查询必须等它为真再启用，
 * 页面也要等它为真再决定显示真实数据还是演示数据。
 */
export function useSessionReady(): boolean {
  return useAuthStore((state) => state.meLoaded);
}
