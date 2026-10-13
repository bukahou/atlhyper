/**
 * 用户认证 API
 *
 * 适配 Master V2 API
 */

import { post, get } from "./request";
import type { LoginRequest, LoginResponse, UserListItem, AuditLogItem, AuthMe, AuthProvider, PermissionTable } from "@/types/auth";

// ============================================================
// Master V2 响应类型
// ============================================================

// 登录响应
interface LoginApiResponse {
  message: string;
  data: LoginResponse;
}

// 当前主体响应
interface AuthMeApiResponse {
  message: string;
  data: AuthMe;
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
 * 获取当前主体与权限（无需登录；访客得到 0 级权限）
 * GET /api/v2/auth/me
 */
export function getMe() {
  return get<AuthMeApiResponse>("/api/v2/auth/me");
}

/**
 * 完整权限表（无需登录；角色说明页用）
 * GET /api/v2/auth/permissions
 */
export function getPermissionTable() {
  return get<{ message: string; data: PermissionTable }>("/api/v2/auth/permissions");
}

/**
 * 外部登录方式（无需登录；未启用时为空列表）
 * GET /api/v2/auth/providers
 */
export function getAuthProviders() {
  return get<{ message: string; data: AuthProvider[] }>("/api/v2/auth/providers");
}

/**
 * 外部登录发起地址（整页跳转，不走 axios）：next 为登录成功后回到的应用内路径
 */
export function oidcStartURL(next: string) {
  return `/api/v2/auth/oidc/start?next=${encodeURIComponent(next)}`;
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
export function getAuditLogs(params?: { userId?: number; source?: string; action?: string; since?: string; until?: string; limit?: number; offset?: number }) {
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
    displayName: data.displayName,
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
    userId: data.userId,
    role: data.role,
  });
}

/**
 * 更新用户状态（需要 Admin 权限）
 * POST /api/v2/user/update-status
 */
export function updateUserStatus(data: { userId: number; status: number }) {
  return post<OperationResponse>("/api/v2/user/update-status", {
    userId: data.userId,
    status: data.status,
  });
}

/**
 * 删除用户（需要 Admin 权限）
 * POST /api/v2/user/delete
 */
export function deleteUser(id: number) {
  return post<OperationResponse>("/api/v2/user/delete", { userId: id });
}
