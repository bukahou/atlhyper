/**
 * 管理类数据的取数与写操作描述（权限表、用户、审计、指令历史）
 *
 * 这些页面没有 datasource：工厂直接调用 api 层。没有权限时页面整页显示演示数据
 * （由页面自己取，不经过这里），所以查询只在有权限时启用 ——
 * 有没有权限由调用方用 useCan 得到后传进来，工厂本身不读权限状态。
 *
 * 集群注册表在 cluster-registry.ts。
 */
import { mutationOptions, queryOptions, type QueryClient } from "@tanstack/react-query";
import {
  deleteUser,
  getAuditLogs,
  getPermissionTable,
  getUserList,
  registerUser,
  updateUserRole,
  updateUserStatus,
} from "@/api/auth";
import { getCommandHistory } from "@/api/commands";
import { adminKeys, type CommandHistoryFilter } from "./keys";
import { isQueryOfCluster } from "./previous-data";
import { REFRESH } from "./refresh";

/** 权限表（后端唯一权限源）。不随时间变化，不轮询 */
export function permissionTableQuery() {
  return queryOptions({
    queryKey: adminKeys.permissionTable(),
    queryFn: async ({ signal }) => (await getPermissionTable(signal)).data.data,
    staleTime: Infinity,
  });
}

// ============================================================
// 用户
// ============================================================

/** 用户列表（需要 user.manage） */
export function userListQuery(canManageUsers: boolean) {
  return queryOptions({
    queryKey: adminKeys.users(),
    queryFn: async ({ signal }) => (await getUserList(signal)).data.data ?? [],
    enabled: canManageUsers,
  });
}

/** 用户变了（新增 / 改角色 / 启停 / 删除），让用户列表重新取数 */
function invalidateUserList(client: QueryClient): void {
  void client.invalidateQueries({ queryKey: adminKeys.users() });
}

export interface NewUser {
  username: string;
  password: string;
  displayName?: string;
  email?: string;
  role?: number;
}

/** 新增用户 */
export function registerUserMutation() {
  return mutationOptions({
    mutationFn: (newUser: NewUser) => registerUser(newUser),
    onSuccess: (_created, _newUser, _onMutateResult, context) => invalidateUserList(context.client),
  });
}

/** 修改用户角色（唯一的提权入口） */
export function updateUserRoleMutation() {
  return mutationOptions({
    mutationFn: (change: { userId: number; role: number }) => updateUserRole(change),
    onSuccess: (_updated, _change, _onMutateResult, context) => invalidateUserList(context.client),
  });
}

/** 启用 / 禁用用户 */
export function updateUserStatusMutation() {
  return mutationOptions({
    mutationFn: (change: { userId: number; status: number }) => updateUserStatus(change),
    onSuccess: (_updated, _change, _onMutateResult, context) => invalidateUserList(context.client),
  });
}

/** 删除用户 */
export function deleteUserMutation() {
  return mutationOptions({
    mutationFn: (userId: number) => deleteUser(userId),
    onSuccess: (_deleted, _userId, _onMutateResult, context) => invalidateUserList(context.client),
  });
}

// ============================================================
// 审计
// ============================================================

/** 审计日志（需要 audit.read）。时间、用户、结果的筛选在页面上对已加载的记录做 */
export function auditLogsQuery(canReadAudit: boolean) {
  return queryOptions({
    queryKey: adminKeys.auditLogs(),
    queryFn: async ({ signal }) => (await getAuditLogs(undefined, signal)).data.data ?? [],
    enabled: canReadAudit,
  });
}

// ============================================================
// 指令历史
// ============================================================

/**
 * 指令历史（内容级数据，需要 commands.read）。
 *
 * clusterId 为空串时后端返回全部集群的指令，所以这里不要求选定集群。
 * 翻页、换筛选条件时先留着上一份结果（同一个集群内）。
 */
export function commandHistoryQuery(clusterId: string, filter: CommandHistoryFilter, canReadCommands: boolean) {
  return queryOptions({
    queryKey: adminKeys.commandHistory(clusterId, filter),
    queryFn: async ({ signal }) => {
      const page = (await getCommandHistory({ clusterId, ...filter }, signal)).data;
      return { commands: page.commands ?? [], total: page.total ?? 0 };
    },
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    placeholderData: (previousPage, previousQuery) =>
      isQueryOfCluster(previousQuery, clusterId) ? previousPage : undefined,
    enabled: canReadCommands,
  });
}
