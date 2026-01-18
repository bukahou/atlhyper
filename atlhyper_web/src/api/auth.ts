/**
 * 用户认证 API
 *
 * 适配 Master V2 API
 */

import { post, get } from "./request";
import type { LoginRequest, LoginResponse, UserListItem, AuditLogItem } from "@/types/auth";

// ============================================================
// Master V2 响应类型
// ============================================================

// 登录响应
interface LoginApiResponse {
  message: string;
  data: LoginResponse;
}

// 用户列表响应
interface UserListApiResponse {
  message: string;
  data: UserListItem[];
}

// 审计日志响应
interface AuditLogsApiResponse {
  message: string;
  data: AuditLogItem[];
  total: number;
}

// 通用操作响应
interface OperationResponse {
  message: string;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 用户登录
 * POST /api/v2/user/login
 */
export function login(data: LoginRequest) {
  return post<LoginApiResponse>("/api/v2/user/login", data);
}

/**
 * 获取用户列表（需要 Admin 权限）
 * GET /api/v2/user/list
 */
export function getUserList() {
  return get<UserListApiResponse>("/api/v2/user/list");
}

/**
 * 获取审计日志（需要 Admin 权限）
 * GET /api/v2/audit/logs
 */
export function getAuditLogs(params?: { user_id?: number; source?: string; action?: string; since?: string; until?: string; limit?: number; offset?: number }) {
  return get<AuditLogsApiResponse>("/api/v2/audit/logs", params);
}

/**
 * 注册用户（需要 Admin 权限）
 * POST /api/v2/user/register
 */
export function registerUser(data: { username: string; password: string; displayName?: string; email?: string; role?: number }) {
  return post<OperationResponse>("/api/v2/user/register", {
    username: data.username,
    password: data.password,
    display_name: data.displayName,
    email: data.email,
    role: data.role,
  });
}

/**
 * 更新用户角色（需要 Admin 权限）
 * POST /api/v2/user/update-role
 */
export function updateUserRole(data: { userId: number; role: number }) {
  return post<OperationResponse>("/api/v2/user/update-role", {
    user_id: data.userId,
    role: data.role,
  });
}

/**
 * 更新用户状态（需要 Admin 权限）
 * POST /api/v2/user/update-status
 */
export function updateUserStatus(data: { userId: number; status: number }) {
  return post<OperationResponse>("/api/v2/user/update-status", {
    user_id: data.userId,
    status: data.status,
  });
}

/**
 * 删除用户（需要 Admin 权限）
 * POST /api/v2/user/delete
 */
export function deleteUser(id: number) {
  return post<OperationResponse>("/api/v2/user/delete", { user_id: id });
}
