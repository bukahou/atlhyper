// atlhyper_web/src/api/ai-provider.ts
// AI Provider Management API

import { get, post, put, del } from "./request";

// ============================================================
// Types
// ============================================================

export interface AIProvider {
  id: number;
  name: string;
  provider: string;
  model: string;
  description: string;
  api_key_masked: string;
  api_key_set: boolean;
  is_active: boolean;
  status: string;
  total_requests: number;
  total_tokens: number;
  total_cost: number;
  last_used_at?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface ActiveConfig {
  enabled: boolean;
  provider_id: number | null;
  tool_timeout: number;
}

export interface ProviderModelInfo {
  provider: string;
  name: string;
  models: string[];
}

export interface ProviderListResponse {
  providers: AIProvider[];
  active_config: ActiveConfig;
  models: ProviderModelInfo[];
}

export interface ProviderCreateRequest {
  name: string;
  provider: string;
  api_key: string;
  model: string;
  description?: string;
}

export interface ProviderUpdateRequest {
  name?: string;
  provider?: string;
  api_key?: string;
  model?: string;
  description?: string;
}

export interface ActiveConfigUpdateRequest {
  enabled?: boolean;
  provider_id?: number;
  tool_timeout?: number;
}

// ============================================================
// API Functions
// ============================================================

// プロバイダー一覧取得
export function listProviders() {
  return get<ProviderListResponse>("/api/v2/ai/providers");
}

// プロバイダー作成
export function createProvider(data: ProviderCreateRequest) {
  return post<AIProvider>("/api/v2/ai/providers", data);
}

// プロバイダー取得
export function getProvider(id: number) {
  return get<AIProvider>(`/api/v2/ai/providers/${id}`);
}

// プロバイダー更新
export function updateProvider(id: number, data: ProviderUpdateRequest) {
  return put<AIProvider>(`/api/v2/ai/providers/${id}`, data);
}

// プロバイダー削除
export function deleteProvider(id: number) {
  return del<{ status: string }>(`/api/v2/ai/providers/${id}`);
}

// アクティブ設定取得
export function getActiveConfig() {
  return get<ActiveConfig>("/api/v2/ai/active");
}

// アクティブ設定更新
export function updateActiveConfig(data: ActiveConfigUpdateRequest) {
  return put<ActiveConfig>("/api/v2/ai/active", data);
}

// ============================================================
// Mock Data (Guest用)
// ============================================================

export const mockProviderList: ProviderListResponse = {
  providers: [
    {
      id: 1,
      name: "Gemini 本番",
      provider: "gemini",
      model: "gemini-2.0-flash",
      description: "本番環境用",
      api_key_masked: "AIza****1234",
      api_key_set: true,
      is_active: true,
      status: "healthy",
      total_requests: 1234,
      total_tokens: 456789,
      total_cost: 12.34,
      last_used_at: "2026-01-26T10:30:00Z",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-26T10:30:00Z",
    },
    {
      id: 2,
      name: "OpenAI 予備",
      provider: "openai",
      model: "gpt-4o",
      description: "バックアップ用",
      api_key_masked: "sk-****5678",
      api_key_set: true,
      is_active: false,
      status: "unknown",
      total_requests: 0,
      total_tokens: 0,
      total_cost: 0,
      created_at: "2026-01-15T00:00:00Z",
      updated_at: "2026-01-15T00:00:00Z",
    },
  ],
  active_config: {
    enabled: true,
    provider_id: 1,
    tool_timeout: 30,
  },
  models: [
    {
      provider: "gemini",
      name: "Google Gemini",
      models: ["gemini-2.0-flash", "gemini-2.0-flash-thinking-exp", "gemini-1.5-pro"],
    },
    {
      provider: "openai",
      name: "OpenAI",
      models: ["gpt-4o", "gpt-4o-mini", "gpt-4-turbo"],
    },
    {
      provider: "anthropic",
      name: "Anthropic Claude",
      models: [
        "claude-sonnet-4-20250514",
        "claude-opus-4-5-20251101",
        "claude-3-5-sonnet-20241022",
        "claude-3-5-haiku-20241022",
        "claude-3-opus-20240229",
        "claude-3-haiku-20240307",
      ],
    },
  ],
};
