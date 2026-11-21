/**
 * Commands API
 *
 * 命令历史查询接口
 */

import { get } from "./request";

/** 写操作（重启 / 扩缩容 / 改镜像 / 封锁节点）下发后的响应：指令已入队，由 Agent 稍后执行 */
export interface CommandResponse {
  message: string;
  commandId: string;
  status: string;
}

/** 指令状态（GET /api/v2/commands/{id}；结果只带成败与原因，不带输出） */
export interface CommandStatus {
  commandId: string;
  status: "pending" | "running" | "success" | "failed" | "timeout";
  result?: { success: boolean; error?: string };
}

/** 查询一条指令的当前状态 */
export function getCommandStatus(commandId: string, signal?: AbortSignal) {
  return get<CommandStatus>(`/api/v2/commands/${encodeURIComponent(commandId)}`, undefined, signal);
}

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
export function getCommandHistory(params: CommandQueryParams = {}, signal?: AbortSignal) {
  return get<CommandListResponse>(
    "/api/v2/commands/history",
    {
      clusterId: params.clusterId || "",
      source: params.source || "",
      status: params.status || "",
      action: params.action || "",
      search: params.search || "",
      limit: params.limit || 20,
      offset: params.offset || 0,
    },
    signal,
  );
}
