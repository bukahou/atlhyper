/**
 * 系统配置 API
 *
 * 适配 Master V2 API
 */

import { get, post } from "./request";

// ============================================================
// Slack 配置类型
// ============================================================

// Slack 配置响应（Master V2 使用 snake_case）
export interface SlackConfig {
  id: number;
  name: string;
  enable: boolean;
  webhook: string;
  interval_sec: number;
  updated_at: string;
}

// Slack 配置更新请求
export interface SlackConfigUpdateRequest {
  enable?: boolean;
  webhook?: string;
  interval_sec?: number;
}

// ============================================================
// 响应类型
// ============================================================

interface SlackConfigResponse {
  message: string;
  data: SlackConfig;
}

interface OperationResponse {
  message: string;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 Slack 配置
 * GET /api/v2/config/notify/slack
 */
export function getSlackConfig() {
  return get<SlackConfigResponse>("/api/v2/config/notify/slack");
}

/**
 * 更新 Slack 配置（需要 Admin 权限）
 * POST /api/v2/config/notify/slack
 */
export function updateSlackConfig(data: SlackConfigUpdateRequest) {
  return post<OperationResponse>("/api/v2/config/notify/slack", data);
}

// ============================================================
// 兼容旧接口（后续移除）
// ============================================================

/** @deprecated 保留旧类型名称兼容 */
export interface SlackConfigResponse_Legacy {
  ID: number;
  Name: string;
  Enable: number;
  Webhook: string;
  IntervalSec: number;
  UpdatedAt: string;
}
