/**
 * AI 设置页的演示数据（无 ai.settings.read 权限时整页展示，均为虚构）
 */
import type { AIReportItem, ProviderListResponse, RoleBudget, RoleOverview } from "@/api/ai-provider";

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
    {
      provider: "bedrock",
      name: "AWS Bedrock",
      models: ["qwen.qwen3-235b-a22b-2507-v1:0", "openai.gpt-oss-120b-1:0", "minimax.minimax-m2.5"],
    },
  ],
};
