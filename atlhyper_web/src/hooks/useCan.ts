/**
 * 权限判断 Hook（前端唯一的权限判断入口）
 *
 * 权限列表由后端 GET /api/v2/auth/me 下发；访客与新注册成员同为 0 级。
 * ⛔ 不要用「是否登录」或角色数字直接判断能否看 / 做某件事。
 *
 * @example
 * const can = useCan();
 * if (can("logs.read")) { ... }
 */

import { useCallback } from "react";
import { useAuthStore } from "@/store/authStore";
import type { Permission } from "@/types/auth";

export function useCan() {
  const permissions = useAuthStore((s) => s.permissions);
  return useCallback((perm: Permission) => permissions.includes(perm), [permissions]);
}

/**
 * 非组件场景（datasource 层）的即时权限判断：无权限时不发请求
 */
export function canNow(perm: Permission): boolean {
  return useAuthStore.getState().permissions.includes(perm);
}
