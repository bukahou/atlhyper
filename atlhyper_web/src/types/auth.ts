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

// 用户列表项
export interface UserListItem {
  ID: number;
  Username: string;
  PasswordHash: string;
  DisplayName: string;
  Email: string;
  Role: number;
  CreatedAt: string;
  LastLogin: string | null;
}

// 审计日志项
export interface AuditLogItem {
  ID: number;
  UserID: number;
  Username: string;
  Role: number;
  Action: string;
  Success: boolean;
  IP: string;
  Method: string;
  Status: number;
  Timestamp: string;
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
