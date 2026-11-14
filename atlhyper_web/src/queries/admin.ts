/**
 * 管理类数据的取数描述
 */
import { queryOptions } from "@tanstack/react-query";
import { getPermissionTable } from "@/api/auth";
import { adminKeys } from "./keys";

/** 权限表（后端唯一权限源）。不随时间变化，不轮询 */
export function permissionTableQuery() {
  return queryOptions({
    queryKey: adminKeys.permissionTable(),
    queryFn: async ({ signal }) => (await getPermissionTable(signal)).data.data,
    staleTime: Infinity,
  });
}
