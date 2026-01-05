import { post } from "./request";

// Slack 配置响应类型
export interface SlackConfigResponse {
  ID: number;
  Name: string;
  Enable: number;       // 0 或 1
  Webhook: string;
  IntervalSec: number;
  UpdatedAt: string;
}

// Slack 配置更新请求
export interface SlackConfigUpdateRequest {
  enable?: number;
  webhook?: string;
  intervalSec?: number;
}

/**
 * 获取 Slack 配置
 */
export function getSlackConfig() {
  return post<SlackConfigResponse>("/uiapi/config/slack/get", {});
}

/**
 * 更新 Slack 配置（需要 Admin 权限）
 */
export function updateSlackConfig(data: SlackConfigUpdateRequest) {
  return post<SlackConfigResponse, SlackConfigUpdateRequest>("/uiapi/config/slack/update", data);
}
