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
  email: string;
  role: number; // 0=member, 1=viewer, 2=operator, 3=admin
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
  status: number; // 1=Active, 0=Disabled
  createdAt: string;
  federated?: boolean; // 经外部身份（OIDC）登录建立
}

// 审计日志项（匹配后端 AuditLogResponse JSON 格式）
export interface AuditLogItem {
  id: number;
  timestamp: string;
  userId: number;
  username: string;
  role: number;
  source: string;        // web / api / ai
  action: string;        // login / create / update / delete / execute / read
  resource: string;      // user / pod / deployment / node / configmap / secret / command / notify
  method: string;        // GET / POST / PUT / DELETE
  requestSummary?: string;
  status: number;        // HTTP 状态码
  success: boolean;
  errorMessage?: string;
  ip: string;
  durationMs: number;
}

// 认证状态
export interface AuthState {
  token: string | null;
  user: UserInfo | null;
  clusterIds: string[];
  isAuthenticated: boolean;
}

// 角色常量（0 = 成员：与访客权限完全相同）
export const UserRole = {
  MEMBER: 0,
  VIEWER: 1,
  OPERATOR: 2,
  ADMIN: 3,
} as const;

export type UserRoleType = typeof UserRole[keyof typeof UserRole];

// 权限（与后端 middleware/permission.go 一一对应，由 /api/v2/auth/me 下发）
export type Permission =
  | "cluster.read" | "events.read" | "metrics.read" | "slo.read"
  | "apm.summary.read" | "logs.stats.read" | "aiops.read"
  | "workload.env.read" | "logs.read" | "traces.read" | "commands.read"
  | "deploy.history.read" | "github.status.read" | "ai.chat"
  | "secret.read" | "pod.logs.read" | "cluster.operate" | "audit.read"
  | "notify.manage" | "ai.analyze" | "ai.settings.read"
  | "user.manage" | "ai.settings.write" | "github.manage" | "deploy.manage";

// 当前主体（GET /api/v2/auth/me）：访客与身份无效者 user = null、level = 0
export interface AuthMe {
  user: UserInfo | null;
  level: number;
  permissions: Permission[];
}

// 外部登录方式（GET /api/v2/auth/providers；未启用时为空列表）
export interface AuthProvider {
  id: string;
  displayName: string;
}
