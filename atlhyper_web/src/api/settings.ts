/**
 * AI 配置 API
 *
 * 适配 Master V2 API
 */

import { get, put, post } from "./request";

// ============================================================
// 类型定义
// ============================================================

// AI 提供商信息
export interface ProviderInfo {
  id: string;
  name: string;
  models: string[];
}

// AI 配置响应
export interface AIConfigResponse {
  enabled: boolean;              // 用户设置的启用状态
  effective_enabled: boolean;    // 实际可用状态
  validation_errors: string[];   // 配置校验错误
  provider: string;              // 当前提供商
  api_key_masked: string;        // 脱敏后的 API Key
  api_key_set: boolean;          // 是否已设置 API Key
  model: string;                 // 当前模型
  tool_timeout: number;          // Tool 超时(秒)
  available_providers: ProviderInfo[]; // 可用提供商列表
  requires_restart: boolean;     // 修改后是否需要重启
}

// AI 配置更新请求
export interface AIConfigUpdateRequest {
  enabled?: boolean;
  provider?: string;
  api_key?: string;
  model?: string;
  tool_timeout?: number;
}

// 测试连接响应
export interface TestConnectionResponse {
  success: boolean;
  message: string;
  provider: string;
  model: string;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 AI 配置
 * GET /api/v2/settings/ai
 */
export function getAIConfig() {
  return get<AIConfigResponse>("/api/v2/settings/ai");
}

/**
 * 更新 AI 配置
 * PUT /api/v2/settings/ai
 */
export function updateAIConfig(data: AIConfigUpdateRequest) {
  return put<AIConfigResponse>("/api/v2/settings/ai/", data);
}

/**
 * 测试 AI 连接
 * POST /api/v2/settings/ai/test
 */
export function testAIConnection() {
  return post<TestConnectionResponse>("/api/v2/settings/ai/test");
}

// ============================================================
// Mock 数据（Guest 用户使用）
// ============================================================

export const mockAIConfig: AIConfigResponse = {
  enabled: true,
  effective_enabled: false,
  validation_errors: ["api_key 未配置"],
  provider: "gemini",
  api_key_masked: "",
  api_key_set: false,
  model: "gemini-2.0-flash",
  tool_timeout: 30,
  available_providers: [
    {
      id: "gemini",
      name: "Google Gemini",
      models: ["gemini-2.0-flash", "gemini-2.0-flash-thinking-exp", "gemini-1.5-pro", "gemini-1.5-flash"],
    },
    {
      id: "openai",
      name: "OpenAI",
      models: ["gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4"],
    },
    {
      id: "anthropic",
      name: "Anthropic Claude",
      models: ["claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022", "claude-3-opus-20240229"],
    },
  ],
  requires_restart: false,
};
