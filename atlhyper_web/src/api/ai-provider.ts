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

// AI 設定取得
export function getAISettings() {
  return get<AISettings>("/api/v2/ai/settings");
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

// Mock: 角色总览
export const mockRolesOverview: RoleOverview[] = [
  {
    role: "background",
    roleName: "Background Analysis",
    provider: { id: 1, name: "Gemini 本番", model: "gemini-2.0-flash", contextWindow: 1048576 },
  },
  {
    role: "chat",
    roleName: "AI Chat",
    provider: { id: 1, name: "Gemini 本番", model: "gemini-2.0-flash", contextWindow: 1048576 },
  },
  {
    role: "analysis",
    roleName: "Deep Analysis",
    provider: null,
  },
];

// Mock: 角色预算
export const mockBudgets: RoleBudget[] = [
  {
    role: "background",
    dailyInputTokenLimit: 500000,
    dailyOutputTokenLimit: 100000,
    dailyCallLimit: 50,
    dailyInputTokensUsed: 234567,
    dailyOutputTokensUsed: 45678,
    dailyCallsUsed: 23,
    dailyResetAt: "2026-03-13T00:00:00+09:00",
    monthlyInputTokenLimit: 10000000,
    monthlyOutputTokenLimit: 2000000,
    monthlyCallLimit: 1000,
    monthlyInputTokensUsed: 4567890,
    monthlyOutputTokensUsed: 890123,
    monthlyCallsUsed: 456,
    monthlyResetAt: "2026-04-01T00:00:00+09:00",
    autoTriggerMinSeverity: "high",
    autoTriggerMode: "auto",
    fallbackProviderId: 2,
  },
  {
    role: "chat",
    dailyInputTokenLimit: 200000,
    dailyOutputTokenLimit: 50000,
    dailyCallLimit: 30,
    dailyInputTokensUsed: 87654,
    dailyOutputTokensUsed: 12345,
    dailyCallsUsed: 8,
    dailyResetAt: "2026-03-13T00:00:00+09:00",
    monthlyInputTokenLimit: 5000000,
    monthlyOutputTokenLimit: 1000000,
    monthlyCallLimit: 500,
    monthlyInputTokensUsed: 1234567,
    monthlyOutputTokensUsed: 234567,
    monthlyCallsUsed: 120,
    monthlyResetAt: "2026-04-01T00:00:00+09:00",
    autoTriggerMinSeverity: "off",
    autoTriggerMode: "manual",
    fallbackProviderId: null,
  },
  {
    role: "analysis",
    dailyInputTokenLimit: 1000000,
    dailyOutputTokenLimit: 200000,
    dailyCallLimit: 20,
    dailyInputTokensUsed: 0,
    dailyOutputTokensUsed: 0,
    dailyCallsUsed: 0,
    dailyResetAt: "2026-03-13T00:00:00+09:00",
    monthlyInputTokenLimit: 20000000,
    monthlyOutputTokenLimit: 4000000,
    monthlyCallLimit: 400,
    monthlyInputTokensUsed: 0,
    monthlyOutputTokensUsed: 0,
    monthlyCallsUsed: 0,
    monthlyResetAt: "2026-04-01T00:00:00+09:00",
    autoTriggerMinSeverity: "critical",
    autoTriggerMode: "schedule",
    scheduleStartTime: "02:00",
    scheduleEndTime: "06:00",
    fallbackProviderId: null,
  },
];

// Mock: 调用历史
export const mockAIReports: AIReportItem[] = [
  {
    id: 1, incidentId: "INC-2026-0312-001", clusterId: "ZGFX-X10A", role: "background",
    trigger: "metric_anomaly", summary: "Node desk-one CPU 使用率が95%を超え、Pod のメモリ不足が検出されました",
    providerName: "Gemini 本番", model: "gemini-2.0-flash",
    inputTokens: 12345, outputTokens: 3456, durationMs: 4200, createdAt: "2026-03-12T18:30:00Z",
  },
  {
    id: 2, incidentId: "INC-2026-0312-002", clusterId: "ZGFX-X10A", role: "background",
    trigger: "pod_restart", summary: "geass-auth Pod が5分以内に3回再起動、CrashLoopBackOff 状態",
    providerName: "Gemini 本番", model: "gemini-2.0-flash",
    inputTokens: 8900, outputTokens: 2100, durationMs: 3100, createdAt: "2026-03-12T17:45:00Z",
  },
  {
    id: 3, incidentId: "", clusterId: "ZGFX-X10A", role: "chat",
    trigger: "manual", summary: "ユーザーが geass-media の高レイテンシについて質問",
    providerName: "Gemini 本番", model: "gemini-2.0-flash",
    inputTokens: 5600, outputTokens: 1800, durationMs: 2800, createdAt: "2026-03-12T16:20:00Z",
  },
  {
    id: 4, incidentId: "INC-2026-0311-005", clusterId: "ZGFX-X10A", role: "background",
    trigger: "slo_breach", summary: "geass-gateway の SLO 目標 99.9% に対して可用性が 99.2% に低下",
    providerName: "Gemini 本番", model: "gemini-2.0-flash",
    inputTokens: 15678, outputTokens: 4567, durationMs: 5600, createdAt: "2026-03-11T23:10:00Z",
  },
  {
    id: 5, incidentId: "", clusterId: "ZGFX-X10A", role: "chat",
    trigger: "manual", summary: "クラスター全体のリソース使用状況の分析を依頼",
    providerName: "Gemini 本番", model: "gemini-2.0-flash",
    inputTokens: 23456, outputTokens: 6789, durationMs: 8200, createdAt: "2026-03-11T14:30:00Z",
  },
  {
    id: 6, incidentId: "INC-2026-0311-003", clusterId: "ZGFX-X10A", role: "background",
    trigger: "metric_anomaly", summary: "raspi-zero ノードのディスク使用率が90%超過、NFS マウントポイントの空き容量不足",
    providerName: "Gemini 本番", model: "gemini-2.0-flash",
    inputTokens: 9876, outputTokens: 2345, durationMs: 3400, createdAt: "2026-03-11T08:15:00Z",
  },
  {
    id: 7, incidentId: "INC-2026-0310-001", clusterId: "ZGFX-X10A", role: "background",
    trigger: "pod_oom", summary: "atlhyper-master Pod が OOMKilled、メモリリミット 512Mi に対して 498Mi 使用",
    providerName: "Gemini 本番", model: "gemini-2.0-flash",
    inputTokens: 11234, outputTokens: 3456, durationMs: 4100, createdAt: "2026-03-10T22:45:00Z",
  },
  {
    id: 8, incidentId: "", clusterId: "ZGFX-X10A", role: "chat",
    trigger: "manual", summary: "Elasticsearch のインデックスサイズ最適化について相談",
    providerName: "Gemini 本番", model: "gemini-2.0-flash",
    inputTokens: 18900, outputTokens: 5600, durationMs: 7300, createdAt: "2026-03-10T15:00:00Z",
  },
];

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
  settings: {
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
