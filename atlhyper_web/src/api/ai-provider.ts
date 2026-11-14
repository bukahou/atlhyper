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

export interface AISettings {
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
  settings: AISettings;
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
  autoTriggerMode: string; // "auto" | "manual" | "schedule"
  scheduleStartTime?: string; // HH:MM
  scheduleEndTime?: string; // HH:MM
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
  autoTriggerMode?: string;
  scheduleStartTime?: string;
  scheduleEndTime?: string;
  fallbackProviderId?: number | null;
}

export interface AISettingsUpdateRequest {
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
export function listProviders(signal?: AbortSignal) {
  return get<ProviderListResponse>("/api/v2/ai/providers", undefined, signal);
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

// AI 設定取得
export function getAISettings(signal?: AbortSignal) {
  return get<AISettings>("/api/v2/ai/settings", undefined, signal);
}

// AI 設定更新
export function updateAISettings(data: AISettingsUpdateRequest) {
  return put<AISettings>("/api/v2/ai/settings", data);
}

// 角色分配
export function updateProviderRoles(id: number, roles: string[]) {
  return put<{ message: string; roles: string[] }>(
    `/api/v2/ai/providers/${id}/roles`,
    { roles }
  );
}

// 角色总览
export function getRolesOverview(signal?: AbortSignal) {
  return get<{ message: string; data: RoleOverview[] }>("/api/v2/ai/roles", undefined, signal);
}

// 角色预算列表
export function getBudgets(signal?: AbortSignal) {
  return get<{ message: string; data: RoleBudget[] }>("/api/v2/ai/budgets", undefined, signal);
}

// 角色预算更新
export function updateBudget(role: string, data: BudgetUpdateRequest) {
  return put<{ message: string; role: string }>(
    `/api/v2/ai/budgets/${encodeURIComponent(role)}`,
    data
  );
}

// AI Reports (调用历史)
export function getAIReports(params?: { role?: string; limit?: number; offset?: number }, signal?: AbortSignal) {
  const query = new URLSearchParams();
  if (params?.role) query.set("role", params.role);
  if (params?.limit) query.set("limit", String(params.limit));
  if (params?.offset) query.set("offset", String(params.offset));
  const qs = query.toString();
  return get<{ message: string; data: AIReportItem[]; total: number }>(
    `/api/v2/ai/reports${qs ? `?${qs}` : ""}`,
    undefined,
    signal,
  );
}
