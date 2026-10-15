/**
 * 用户管理演示数据（无 user.manage 权限时整页展示，均为虚构）
 */

import { UserRole, type UserListItem } from "@/types/auth";

export const MOCK_USERS: UserListItem[] = [
  { id: 1, username: "admin", displayName: "Demo Admin", email: "admin@example.com", role: UserRole.ADMIN, status: 1, createdAt: "2026-01-05T09:00:00Z" },
  { id: 2, username: "ops-lead", displayName: "Ops Lead", email: "ops@example.com", role: UserRole.OPERATOR, status: 1, createdAt: "2026-02-11T10:30:00Z" },
  { id: 3, username: "sre-viewer", displayName: "SRE Viewer", email: "viewer@example.com", role: UserRole.VIEWER, status: 1, createdAt: "2026-03-20T08:15:00Z" },
  { id: 4, username: "new-member", displayName: "New Member", email: "member@example.com", role: UserRole.MEMBER, status: 1, createdAt: "2026-09-01T12:00:00Z" },
  { id: 5, username: "former-intern", displayName: "Former Intern", email: "intern@example.com", role: UserRole.VIEWER, status: 0, createdAt: "2026-04-02T07:45:00Z" },
];
