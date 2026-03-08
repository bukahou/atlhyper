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
  baseUrl?: string;
  description: string;
  apiKeyMasked: string;
  apiKeySet: boolean;
  isActive: boolean;
  roles: string[];
  contextWindowOverride: number;
  status: string;
  totalRequests: number;
  totalTokens: number;
  totalCost: number;
  lastUsedAt?: string;
  lastError?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ActiveConfig {
  enabled: boolean;
  providerId: number | null;
  toolTimeout: number;
  chatReady: boolean;
}

export interface ProviderModelInfo {
  provider: string;
  name: string;
  models: string[];
}

export interface ProviderListResponse {
  providers: AIProvider[];
  activeConfig: ActiveConfig;
  models: ProviderModelInfo[];
}

export interface ProviderCreateRequest {
  name: string;
  provider: string;
  apiKey: string;
  model: string;
  baseUrl?: string;
  description?: string;
}

export interface ProviderUpdateRequest {
  name?: string;
  provider?: string;
  apiKey?: string;
  model?: string;
  baseUrl?: string;
  description?: string;
}

export interface RoleOverview {
  role: string;
  roleName: string;
  provider: {
    id: number;
    name: string;
    model: string;
    contextWindow: number;
  } | null;
}

export interface RoleBudget {
  role: string;
  // 日限额
  dailyInputTokenLimit: number;
  dailyOutputTokenLimit: number;
  dailyCallLimit: number;
  // 日消耗
  dailyInputTokensUsed: number;
  dailyOutputTokensUsed: number;
  dailyCallsUsed: number;
  dailyResetAt?: string;
  // 月限额
  monthlyInputTokenLimit: number;
  monthlyOutputTokenLimit: number;
  monthlyCallLimit: number;
  // 月消耗
  monthlyInputTokensUsed: number;
  monthlyOutputTokensUsed: number;
  monthlyCallsUsed: number;
  monthlyResetAt?: string;
  // 配置
  autoTriggerMinSeverity: string;
  fallbackProviderId: number | null;
}

export interface BudgetUpdateRequest {
  dailyInputTokenLimit?: number;
  dailyOutputTokenLimit?: number;
  dailyCallLimit?: number;
  monthlyInputTokenLimit?: number;
  monthlyOutputTokenLimit?: number;
  monthlyCallLimit?: number;
  autoTriggerMinSeverity?: string;
  fallbackProviderId?: number | null;
}

export interface ActiveConfigUpdateRequest {
  enabled?: boolean;
  providerId?: number;
  toolTimeout?: number;
}

export interface AIReportItem {
  id: number;
  incidentId: string;
  clusterId: string;
  role: string;
  trigger: string;
  summary: string;
  providerName: string;
  model: string;
  inputTokens: number;
  outputTokens: number;
  durationMs: number;
  createdAt: string;
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

// 角色分配
export function updateProviderRoles(id: number, roles: string[]) {
  return put<{ message: string; roles: string[] }>(
    `/api/v2/ai/providers/${id}/roles`,
    { roles }
  );
}

// 角色总览
export function getRolesOverview() {
  return get<{ message: string; data: RoleOverview[] }>("/api/v2/ai/roles");
}

// 角色预算列表
export function getBudgets() {
  return get<{ message: string; data: RoleBudget[] }>("/api/v2/ai/budgets");
}

// 角色预算更新
export function updateBudget(role: string, data: BudgetUpdateRequest) {
  return put<{ message: string; role: string }>(
    `/api/v2/ai/budgets/${encodeURIComponent(role)}`,
    data
  );
}

// AI Reports (调用历史)
export function getAIReports(params?: { role?: string; limit?: number; offset?: number }) {
  const query = new URLSearchParams();
  if (params?.role) query.set("role", params.role);
  if (params?.limit) query.set("limit", String(params.limit));
  if (params?.offset) query.set("offset", String(params.offset));
  const qs = query.toString();
  return get<{ message: string; data: AIReportItem[]; total: number }>(
    `/api/v2/ai/reports${qs ? `?${qs}` : ""}`
  );
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
      apiKeyMasked: "AIza****1234",
      apiKeySet: true,
      isActive: true,
      roles: ["background", "chat"],
      contextWindowOverride: 0,
      status: "healthy",
      totalRequests: 1234,
      totalTokens: 456789,
      totalCost: 12.34,
      lastUsedAt: "2026-01-26T10:30:00Z",
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-26T10:30:00Z",
    },
    {
      id: 2,
      name: "OpenAI 予備",
      provider: "openai",
      model: "gpt-4o",
      description: "バックアップ用",
      apiKeyMasked: "sk-****5678",
      apiKeySet: true,
      isActive: false,
      roles: [],
      contextWindowOverride: 0,
      status: "unknown",
      totalRequests: 0,
      totalTokens: 0,
      totalCost: 0,
      createdAt: "2026-01-15T00:00:00Z",
      updatedAt: "2026-01-15T00:00:00Z",
    },
  ],
  activeConfig: {
    enabled: true,
    providerId: 1,
    toolTimeout: 30,
    chatReady: true,
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
