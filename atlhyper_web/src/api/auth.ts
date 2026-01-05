import { post, get } from "./request";
import type { LoginRequest, LoginResponse, UserListItem, AuditLogItem } from "@/types/auth";

/**
 * 用户登录
 */
export function login(data: LoginRequest) {
  return post<LoginResponse, LoginRequest>("/uiapi/auth/login", data);
}

/**
 * 获取用户列表
 */
export function getUserList() {
  return get<UserListItem[]>("/uiapi/auth/user/list");
}

/**
 * 获取审计日志
 */
export function getAuditLogs() {
  return get<AuditLogItem[]>("/uiapi/auth/userauditlogs/list");
}

/**
 * 注册用户
 */
export function registerUser(data: { username: string; password: string; displayName?: string; email?: string }) {
  return post<unknown, typeof data>("/uiapi/auth/user/register", data);
}

/**
 * 更新用户角色
 */
export function updateUserRole(data: { userId: number; role: number }) {
  return post<unknown, { id: number; role: number }>("/uiapi/auth/user/update-role", {
    id: data.userId,
    role: data.role,
  });
}

/**
 * 删除用户
 */
export function deleteUser(id: number) {
  return post<unknown, { id: number }>("/uiapi/auth/user/delete", { id });
}
