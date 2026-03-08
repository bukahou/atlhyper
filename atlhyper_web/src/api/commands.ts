/**
 * Commands API
 *
 * 命令历史查询接口
 */

import { get } from "./request";

// 命令历史记录
export interface CommandHistory {
  id: number;
  commandId: string;
  clusterId: string;
  source: string; // web / ai
  userId: number;
  action: string; // restart / scale / delete_pod / cordon / uncordon
  targetKind: string;
  targetNamespace: string;
  targetName: string;
  params: string; // JSON
  status: string; // pending / running / success / failed / timeout
  result: string; // JSON
  errorMessage: string;
  createdAt: string;
  startedAt: string | null;
  finishedAt: string | null;
  durationMs: number;
}

// 查询参数
export interface CommandQueryParams {
  clusterId?: string;
  source?: string;
  status?: string;
  action?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

// 响应格式
export interface CommandListResponse {
  commands: CommandHistory[];
  total: number;
}

/**
 * 获取命令历史列表
 */
export function getCommandHistory(params: CommandQueryParams = {}) {
  return get<CommandListResponse>("/api/v2/commands/history", {
    clusterId: params.clusterId || "",
    source: params.source || "",
    status: params.status || "",
    action: params.action || "",
    search: params.search || "",
    limit: params.limit || 20,
    offset: params.offset || 0,
  });
}

/**
 * 获取命令状态
 */
export function getCommandStatus(commandId: string) {
  return get<CommandHistory>(`/api/v2/commands/${commandId}`);
}
