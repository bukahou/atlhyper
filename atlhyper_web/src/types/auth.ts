/**
 * 认证相关类型定义
 * 基于后端实际响应格式定义
 */

// 登录请求
export interface LoginRequest {
  username: string;
  password: string;
}

// 用户信息（登录响应中的 user 字段）
export interface UserInfo {
  id: number;
  username: string;
  displayName: string;
  role: number; // 1=viewer, 2=operator, 3=admin
}

// 登录响应
export interface LoginResponse {
  cluster_ids: string[];
  token: string;
  user: UserInfo;
}

// 用户列表项（匹配后端 UserDTO JSON 格式）
export interface UserListItem {
  id: number;
  username: string;
  displayName: string;
  email: string;
  role: number;
  createdAt: string;
  lastLogin: string | null;
}

// 审计日志项（匹配后端 AuditLogDTO JSON 格式）
export interface AuditLogItem {
  id: number;
  userId: number;
  username: string;
  role: number;
  action: string;
  success: boolean;
  ip: string;
  method: string;
  status: number;
  timestamp: string;
}

// 认证状态
export interface AuthState {
  token: string | null;
  user: UserInfo | null;
  clusterIds: string[];
  isAuthenticated: boolean;
}

// 角色常量
export const UserRole = {
  VIEWER: 1,
  OPERATOR: 2,
  ADMIN: 3,
} as const;

export type UserRoleType = typeof UserRole[keyof typeof UserRole];
