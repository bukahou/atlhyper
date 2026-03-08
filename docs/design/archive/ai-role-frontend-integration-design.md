# AI 角色路由前端集成 + 后端 API 补全 — 设计文档

> 创建日期: 2026-03-07
> 前置依赖: AI 多角色路由后端实装（已完成）

## 背景

AI 多角色路由后端已完成 7 个 Phase 的实装（角色定义、上下文管理、角色路由、报告存储、后台触发、深度分析），但存在两层问题：

1. **后端 API 缺口** — 数据模型和业务逻辑已实现，但部分功能未暴露 HTTP API
2. **前端完全缺失** — 角色分配、报告展示、深度分析过程等 UI 均未实现

本文档覆盖后端 API 补全 + 前端全量对接。

---

## Phase A: 后端 API 补全

### A1: AI 报告查询 API

**目标**: 暴露 `AIReport` 表的查询能力，前端可获取自动/手动分析的历史报告。

#### 接口定义

```
GET /api/v2/aiops/ai/reports?incident_id={id}
  权限: Operator
  查询参数: incident_id (必填)
  返回: { "message": "获取成功", "data": AIReportResponse[] }

GET /api/v2/aiops/ai/reports/{report_id}
  权限: Operator
  返回: { "message": "获取成功", "data": AIReportDetailResponse }
```

#### 数据结构

```go
// AIReportResponse 报告列表项
type AIReportResponse struct {
    ID               int64   `json:"id"`
    IncidentID       string  `json:"incidentId"`
    ClusterID        string  `json:"clusterId"`
    Role             string  `json:"role"`              // background / analysis
    Trigger          string  `json:"trigger"`            // incident_created / state_changed / manual / auto_escalation
    Summary          string  `json:"summary"`
    ProviderName     string  `json:"providerName"`
    Model            string  `json:"model"`
    InputTokens      int     `json:"inputTokens"`
    OutputTokens     int     `json:"outputTokens"`
    DurationMs       int64   `json:"durationMs"`
    CreatedAt        string  `json:"createdAt"`
}

// AIReportDetailResponse 报告详情（含完整分析内容）
type AIReportDetailResponse struct {
    AIReportResponse                                     // 嵌入列表项字段
    RootCauseAnalysis    string   `json:"rootCauseAnalysis"`
    Recommendations      string   `json:"recommendations"`       // JSON string
    SimilarIncidents     string   `json:"similarIncidents"`      // JSON string
    InvestigationSteps   string   `json:"investigationSteps"`    // JSON string (analysis 专用)
    EvidenceChain        string   `json:"evidenceChain"`         // JSON string (analysis 专用)
}
```

#### 文件变更

| 文件 | 变更 |
|------|------|
| `gateway/handler/aiops/aiops_ai.go` | 新增 `ReportsHandler` (GET 列表) + `ReportDetailHandler` (GET 详情) |
| `gateway/routes.go` | 注册 `/api/v2/aiops/ai/reports` (operator) + `/api/v2/aiops/ai/reports/` (operator) |
| `service/interfaces.go` | `QueryAIOps` 新增 `ListAIReports(ctx, incidentID) + GetAIReport(ctx, id)` |
| `service/query/aiops.go` | 实现 `ListAIReports` + `GetAIReport` |

### A2: 角色预算管理 API

**目标**: 暴露 `AIRoleBudget` 表的查询和更新能力。

#### 接口定义

```
GET /api/v2/ai/budgets
  权限: Operator
  返回: { "message": "获取成功", "data": AIRoleBudgetResponse[] }

PUT /api/v2/ai/budgets/{role}
  权限: Admin (审计)
  请求体: AIRoleBudgetUpdateRequest
  返回: { "message": "更新成功", "data": AIRoleBudgetResponse }
```

#### 数据结构

```go
// AIRoleBudgetResponse 角色预算响应
type AIRoleBudgetResponse struct {
    Role                    string `json:"role"`                    // background / chat / analysis
    DailyTokenLimit         int64  `json:"dailyTokenLimit"`         // 0 = 无限
    DailyCallLimit          int    `json:"dailyCallLimit"`          // 0 = 无限
    DailyTokensUsed         int64  `json:"dailyTokensUsed"`
    DailyCallsUsed          int    `json:"dailyCallsUsed"`
    DailyResetAt            string `json:"dailyResetAt"`
    AutoTriggerMinSeverity  string `json:"autoTriggerMinSeverity"`  // critical/high/medium/low/off
    FallbackProviderID      *int64 `json:"fallbackProviderId"`
}

// AIRoleBudgetUpdateRequest 预算更新请求
type AIRoleBudgetUpdateRequest struct {
    DailyTokenLimit        *int64  `json:"dailyTokenLimit,omitempty"`
    DailyCallLimit         *int    `json:"dailyCallLimit,omitempty"`
    AutoTriggerMinSeverity *string `json:"autoTriggerMinSeverity,omitempty"`
    FallbackProviderID     *int64  `json:"fallbackProviderId,omitempty"`
}
```

#### 文件变更

| 文件 | 变更 |
|------|------|
| `gateway/handler/admin/ai_provider.go` | 新增 `BudgetsHandler` (GET 列表) + `BudgetHandler` (PUT 单个角色) |
| `gateway/routes.go` | 注册 `/api/v2/ai/budgets` (operator 可读) + `/api/v2/ai/budgets/` (admin 审计) |
| `service/interfaces.go` | `OpsAdmin` 新增 `UpdateAIRoleBudget(ctx, role, req)` |
| `service/operations/admin.go` | 实现 `UpdateAIRoleBudget` |

### A3: 深度分析手动触发 API

**目标**: 允许用户对指定事件手动触发深度分析。

#### 接口定义

```
POST /api/v2/aiops/ai/analyze
  权限: Operator (审计)
  请求体: { "incidentId": "string" }
  返回: { "message": "分析已提交", "reportId": int64 }  // 异步执行
```

#### 文件变更

| 文件 | 变更 |
|------|------|
| `gateway/handler/aiops/aiops_ai.go` | 新增 `AnalyzeHandler` (POST) |
| `gateway/routes.go` | 注册 `/api/v2/aiops/ai/analyze` (operator 审计) |
| `aiops/ai/enhancer.go` | 新增 `TriggerAnalysis(incidentID)` 公开方法 |

### A4: 事件摘要 API 增强 — 优先返回已有报告

**目标**: `POST /api/v2/aiops/ai/summarize` 优先查询已有报告，无报告时再调 LLM 生成。

#### 行为变更

```
请求 → 查询 AIReport 表（incident_id + role=background）
  → 有报告 → 直接返回（附加 reportId + fromCache: true）
  → 无报告 → 调 LLM 生成 → 保存 → 返回（fromCache: false）
```

#### 响应增强

```go
// SummarizeResponse 新增字段
type SummarizeResponse struct {
    // ... 现有字段 ...
    ReportID  int64 `json:"reportId,omitempty"`  // 报告 ID（可用于查询详情）
    FromCache bool  `json:"fromCache"`           // 是否来自已有报告
}
```

#### 文件变更

| 文件 | 变更 |
|------|------|
| `aiops/ai/enhancer.go` | `Summarize` 方法增加 reportRepo 查询逻辑 |

---

## Phase B: 前端 — AI 设置页增强

### B1: API 类型补全

#### `api/ai-provider.ts` 类型更新

```typescript
export interface AIProvider {
  // ... 现有字段 ...
  roles: string[];                // 新增: 角色列表 ["background", "chat", "analysis"]
  contextWindowOverride: number;  // 新增: 上下文窗口覆盖
  baseUrl?: string;               // 新增: 自定义 API 地址 (Ollama 用)
}
```

#### 新增 API 函数

```typescript
// 角色分配
export function updateProviderRoles(id: number, roles: string[]) {
  return put<{ message: string; roles: string[] }>(
    `/api/v2/ai/providers/${id}/roles`, { roles }
  );
}

// 角色总览
export function getRolesOverview() {
  return get<{ message: string; data: RoleOverview[] }>("/api/v2/ai/roles");
}

// 角色预算
export function getBudgets() {
  return get<{ message: string; data: RoleBudget[] }>("/api/v2/ai/budgets");
}

export function updateBudget(role: string, data: BudgetUpdateRequest) {
  return put<{ message: string; data: RoleBudget }>(
    `/api/v2/ai/budgets/${encodeURIComponent(role)}`, data
  );
}
```

#### 新增类型

```typescript
export interface RoleOverview {
  role: string;        // background / chat / analysis
  roleName: string;    // 中文名称
  provider: {
    id: number;
    name: string;
    model: string;
    contextWindow: number;
  } | null;
}

export interface RoleBudget {
  role: string;
  dailyTokenLimit: number;
  dailyCallLimit: number;
  dailyTokensUsed: number;
  dailyCallsUsed: number;
  dailyResetAt: string;
  autoTriggerMinSeverity: string;
  fallbackProviderId: number | null;
}

export interface BudgetUpdateRequest {
  dailyTokenLimit?: number;
  dailyCallLimit?: number;
  autoTriggerMinSeverity?: string;
  fallbackProviderId?: number | null;
}
```

#### 文件变更

| 文件 | 变更 |
|------|------|
| `api/ai-provider.ts` | `AIProvider` 补字段 + 新增 API 函数 + 新增类型 + Mock 数据更新 |

### B2: ProviderCard 增强

**目标**: 卡片上显示角色标签 + 支持 Ollama。

#### UI 变更

- Provider 名称下方显示角色标签（彩色 badge）
  - `background` → 蓝色标签 "后台分析"
  - `chat` → 绿色标签 "交互对话"
  - `analysis` → 紫色标签 "深度分析"
- `providerColors` / `providerNames` 新增 `ollama` 映射
- 无角色时显示灰色 "未分配角色"

#### 文件变更

| 文件 | 变更 |
|------|------|
| `app/settings/ai/components/ProviderCard.tsx` | 角色标签渲染 + ollama 支持 |

### B3: ProviderModal 增强

**目标**: 编辑弹窗支持 baseUrl 字段和角色分配。

#### UI 变更

- 新增 `baseUrl` 输入框（当 provider 为 ollama 时显示，预填 `http://192.168.0.121:11434`）
- 新增角色复选框组（background / chat / analysis）
- 保存时：先调 `updateProvider` 更新基本信息，再调 `updateProviderRoles` 更新角色
- 角色互斥提示：若后端返回 409 Conflict，显示错误信息

#### 文件变更

| 文件 | 变更 |
|------|------|
| `app/settings/ai/components/ProviderModal.tsx` | baseUrl 字段 + 角色复选框 |
| `api/ai-provider.ts` | `ProviderCreateRequest` / `ProviderUpdateRequest` 新增 `base_url` |

### B4: 角色总览卡片

**目标**: 在 AI 设置页顶部展示三个角色的分配状态。

#### UI 设计

```
┌─────────────────────────────────────────────────────┐
│  角色分配总览                                        │
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │ 后台分析  │  │ 交互对话  │  │ 深度分析  │          │
│  │ Gemini   │  │ Gemini   │  │ 未分配    │          │
│  │ flash    │  │ flash    │  │          │          │
│  └──────────┘  └──────────┘  └──────────┘          │
└─────────────────────────────────────────────────────┘
```

- 已分配角色显示 Provider 名称 + 模型
- 未分配角色显示 "未分配" + 灰色样式

#### 文件变更

| 文件 | 变更 |
|------|------|
| `app/settings/ai/components/RoleOverviewCard.tsx` | 新建: 角色总览卡片组件 |
| `app/settings/ai/components/index.ts` | 导出 RoleOverviewCard |
| `app/settings/ai/page.tsx` | 集成 RoleOverviewCard（位于 GlobalSettingsCard 下方） |

### B5: i18n 补全

#### 新增翻译键

```typescript
// types/i18n.ts — AISettingsPageTranslations 新增
roleOverview: string;           // "角色分配总览"
roleBackground: string;         // "后台分析"
roleChat: string;               // "交互对话"
roleAnalysis: string;           // "深度分析"
roleUnassigned: string;         // "未分配"
roleAssignConflict: string;     // "该角色已被其他 Provider 持有"
baseUrl: string;                // "API 地址"
baseUrlPlaceholder: string;     // "http://localhost:11434"
baseUrlHint: string;            // "Ollama 等本地部署服务需要填写"
roles: string;                  // "角色"
rolesHint: string;              // "为该 Provider 分配 AI 角色"
rolesUpdated: string;           // "角色更新成功"
```

#### 文件变更

| 文件 | 变更 |
|------|------|
| `types/i18n.ts` | `AISettingsPageTranslations` 新增键 |
| `i18n/locales/zh.ts` | 中文翻译 |
| `i18n/locales/ja.ts` | 日文翻译 |

---

## Phase C: 前端 — 事件 AI 分析增强

### C1: AI 报告 API 对接

#### 新增 API 函数

```typescript
// api/aiops.ts 新增

export interface AIReport {
  id: number;
  incidentId: string;
  clusterId: string;
  role: string;               // background / analysis
  trigger: string;            // incident_created / state_changed / manual / auto_escalation
  summary: string;
  providerName: string;
  model: string;
  inputTokens: number;
  outputTokens: number;
  durationMs: number;
  createdAt: string;
}

export interface AIReportDetail extends AIReport {
  rootCauseAnalysis: string;
  recommendations: string;       // JSON string → parse to Recommendation[]
  similarIncidents: string;      // JSON string → parse to SimilarMatch[]
  investigationSteps: string;    // JSON string (analysis 专用)
  evidenceChain: string;         // JSON string (analysis 专用)
}

export function getAIReports(incidentId: string) {
  return get<{ message: string; data: AIReport[] }>(
    `/api/v2/aiops/ai/reports?incident_id=${encodeURIComponent(incidentId)}`
  );
}

export function getAIReportDetail(reportId: number) {
  return get<{ message: string; data: AIReportDetail }>(
    `/api/v2/aiops/ai/reports/${reportId}`
  );
}

export function triggerAnalysis(incidentId: string) {
  return post<{ message: string; reportId: number }>(
    "/api/v2/aiops/ai/analyze", { incidentId }
  );
}
```

#### 文件变更

| 文件 | 变更 |
|------|------|
| `api/aiops.ts` | 新增 AIReport 类型 + API 函数 |

### C2: 事件详情 AI 分析区块重构

**目标**: 事件详情弹窗中的 AI 分析区块改为「报告优先」模式。

#### 行为变更

```
打开事件详情
  → 自动查询 GET /aiops/ai/reports?incident_id=xxx
  → 有报告列表:
      → 显示报告卡片列表（按时间倒序）
      → 每张卡片: 角色标签 + 触发方式 + 摘要预览 + 时间 + Token 用量
      → 点击展开: 加载报告详情 → 显示完整分析
      → 底部: "重新分析" 按钮（调 summarizeIncident）
  → 无报告:
      → 显示 "生成 AI 分析" 按钮（调 summarizeIncident）
      → 分析完成后自动刷新报告列表
```

#### UI 设计

```
┌─────────────────────────────────────────────┐
│  AI 分析报告                          [重新分析] │
│                                             │
│  ┌─ 后台分析 · 事件创建触发 · 2min ago ────┐  │
│  │  摘要: Service A 延迟上升...            │  │
│  │  根因: Pod OOM 导致...                  │  │
│  │  建议: 1. 增加内存限制 2. ...           │  │
│  │  Gemini flash · 1.2k tokens · 3.4s     │  │
│  └──────────────────────────────────────────┘  │
│                                             │
│  ┌─ 深度分析 · 自动升级触发 · 1min ago ────┐  │
│  │  摘要: 经过 6 轮调查...                 │  │
│  │  [展开调查过程]                         │  │
│  │    Round 1: 查询 Pod 状态 → ...        │  │
│  │    Round 2: 查询日志 → ...             │  │
│  │    Round 3: 查询指标 → ...             │  │
│  │  根因: ...                              │  │
│  │  证据链: ...                            │  │
│  └──────────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

#### 组件拆分

| 组件 | 职责 |
|------|------|
| `AIAnalysisSection.tsx` | 容器: 加载报告列表 + 生成/重新分析按钮 |
| `AIReportCard.tsx` | 单个报告卡片: 摘要/根因/建议/相似事件 |
| `InvestigationTimeline.tsx` | 深度分析调查步骤时间线（analysis 报告专用） |

#### 文件变更

| 文件 | 变更 |
|------|------|
| `components/aiops/AIAnalysisSection.tsx` | 新建: AI 分析区块容器 |
| `components/aiops/AIReportCard.tsx` | 新建: 报告卡片 |
| `components/aiops/InvestigationTimeline.tsx` | 新建: 调查步骤时间线 |
| `components/aiops/index.ts` | 导出新组件 |
| `app/aiops/incidents/IncidentDetailModal.tsx` | 替换现有 AI 分析内联代码为 `<AIAnalysisSection>` |

### C3: 事件列表 AI 状态标记

**目标**: 事件列表中标记「已有 AI 分析」的事件。

#### 行为

- `SummarizeResponse` 新增 `fromCache` 字段 → 事件详情判断是否有已有报告
- 或者：事件列表 API 返回 `hasAIReport: boolean`（需后端配合）
- 简化方案：在事件详情弹窗打开时异步查询报告列表，不影响列表本身

#### 文件变更

| 文件 | 变更 |
|------|------|
| `app/aiops/incidents/IncidentDetailModal.tsx` | 使用 `<AIAnalysisSection>` 替代内联实现 |

### C4: 深度分析手动触发

**目标**: 事件详情中增加「深入调查」按钮。

#### 行为

- 按钮显示条件: 事件状态为 warning/incident（未恢复）
- 点击后调 `POST /aiops/ai/analyze`
- 显示 "分析已提交，正在进行中..." 提示
- 定时轮询报告列表（每 5 秒），直到新报告出现
- 新报告出现后自动展开显示

#### 文件变更

| 文件 | 变更 |
|------|------|
| `components/aiops/AIAnalysisSection.tsx` | 深入调查按钮 + 轮询逻辑 |

### C5: i18n 补全

#### 新增翻译键

```typescript
// types/i18n.ts — AIOpsTranslations.ai 新增
aiReports: string;              // "AI 分析报告"
noReports: string;              // "暂无分析报告"
generateAnalysis: string;       // "生成 AI 分析"
reanalyze: string;              // "重新分析"
triggerDeepAnalysis: string;    // "深入调查"
analysisSubmitted: string;      // "分析已提交"
analysisInProgress: string;     // "正在分析中..."
fromBackground: string;         // "后台自动分析"
fromAnalysis: string;           // "深度调查"
fromManual: string;             // "手动触发"
triggerIncidentCreated: string;  // "事件创建触发"
triggerStateChanged: string;     // "状态变更触发"
triggerManual: string;           // "手动触发"
investigationSteps: string;     // "调查过程"
evidenceChain: string;          // "证据链"
round: string;                  // "轮次"
toolCalls: string;              // "工具调用"
tokens: string;                 // "Token 用量"
duration: string;               // "耗时"
expandSteps: string;            // "展开调查过程"
collapseSteps: string;          // "收起调查过程"
reportMeta: string;             // "{provider} · {tokens} tokens · {duration}"
fromCache: string;              // "已有报告"
```

#### 文件变更

| 文件 | 变更 |
|------|------|
| `types/i18n.ts` | `AIOpsTranslations` 新增 AI 报告相关键 |
| `i18n/locales/zh.ts` | 中文翻译 |
| `i18n/locales/ja.ts` | 日文翻译 |

---

## Phase D: 前端 — 角色预算配置（增强功能）

### D1: 预算配置卡片

**目标**: AI 设置页底部展示角色预算配置。

#### UI 设计

```
┌─────────────────────────────────────────────────────┐
│  角色预算配置                                        │
│                                                     │
│  ┌── background ──────────────────────────────────┐  │
│  │  每日 Token 上限: [___50000___]  已用: 12,345  │  │
│  │  每日调用上限:    [___100_____]  已用: 23      │  │
│  │  自动触发阈值:    [medium  ▼]                  │  │
│  │  降级 Provider:   [Ollama 本地 ▼]              │  │
│  │                                    [保存]      │  │
│  └────────────────────────────────────────────────┘  │
│                                                     │
│  ┌── chat ────────────────────────────────────────┐  │
│  │  ...                                           │  │
│  └────────────────────────────────────────────────┘  │
│                                                     │
│  ┌── analysis ────────────────────────────────────┐  │
│  │  ...                                           │  │
│  └────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

#### 文件变更

| 文件 | 变更 |
|------|------|
| `app/settings/ai/components/BudgetConfigCard.tsx` | 新建: 预算配置卡片 |
| `app/settings/ai/components/index.ts` | 导出 BudgetConfigCard |
| `app/settings/ai/page.tsx` | 集成 BudgetConfigCard（位于 Provider 列表下方） |

### D2: 预算 i18n

#### 新增翻译键

```typescript
// types/i18n.ts — AISettingsPageTranslations 新增
budgetConfig: string;              // "角色预算配置"
dailyTokenLimit: string;           // "每日 Token 上限"
dailyCallLimit: string;            // "每日调用上限"
dailyTokensUsed: string;           // "已用 Token"
dailyCallsUsed: string;            // "已用调用"
autoTriggerMinSeverity: string;    // "自动触发阈值"
fallbackProvider: string;          // "降级 Provider"
severityCritical: string;          // "仅 Critical"
severityHigh: string;              // "High 及以上"
severityMedium: string;            // "Medium 及以上"
severityLow: string;               // "Low 及以上"
severityOff: string;               // "关闭"
unlimited: string;                 // "无限制"
budgetSaved: string;               // "预算配置已保存"
budgetSaveFailed: string;          // "预算配置保存失败"
```

---

## 文件变更总览

### 后端变更 (Phase A)

| 文件 | 类型 | 说明 |
|------|------|------|
| `gateway/handler/aiops/aiops_ai.go` | 修改 | 新增 ReportsHandler + ReportDetailHandler + AnalyzeHandler |
| `gateway/handler/admin/ai_provider.go` | 修改 | 新增 BudgetsHandler + BudgetHandler |
| `gateway/routes.go` | 修改 | 注册 4 条新路由 |
| `service/interfaces.go` | 修改 | QueryAIOps + OpsAdmin 新增方法签名 |
| `service/query/aiops.go` | 修改 | 实现 ListAIReports + GetAIReport |
| `service/operations/admin.go` | 修改 | 实现 UpdateAIRoleBudget |
| `aiops/ai/enhancer.go` | 修改 | Summarize 增加报告查询优先 + 新增 TriggerAnalysis |

### 前端变更 (Phase B + C + D)

| 文件 | 类型 | 说明 |
|------|------|------|
| **API 层** | | |
| `api/ai-provider.ts` | 修改 | AIProvider 补字段 + 角色/预算 API + 类型 + Mock |
| `api/aiops.ts` | 修改 | AIReport 类型 + 报告查询/深度分析 API |
| **设置页** | | |
| `app/settings/ai/components/ProviderCard.tsx` | 修改 | 角色标签 + ollama 支持 |
| `app/settings/ai/components/ProviderModal.tsx` | 修改 | baseUrl 字段 + 角色复选框 |
| `app/settings/ai/components/RoleOverviewCard.tsx` | 新建 | 角色总览卡片 |
| `app/settings/ai/components/BudgetConfigCard.tsx` | 新建 | 预算配置卡片 |
| `app/settings/ai/components/index.ts` | 修改 | 导出新组件 |
| `app/settings/ai/page.tsx` | 修改 | 集成新卡片 |
| **事件页** | | |
| `components/aiops/AIAnalysisSection.tsx` | 新建 | AI 分析区块容器 |
| `components/aiops/AIReportCard.tsx` | 新建 | 报告卡片 |
| `components/aiops/InvestigationTimeline.tsx` | 新建 | 调查步骤时间线 |
| `components/aiops/index.ts` | 新建/修改 | 导出 |
| `app/aiops/incidents/IncidentDetailModal.tsx` | 修改 | 替换内联 AI 分析为组件 |
| **i18n** | | |
| `types/i18n.ts` | 修改 | 新增翻译键（AI 设置 + AIOps 报告） |
| `i18n/locales/zh.ts` | 修改 | 中文翻译 |
| `i18n/locales/ja.ts` | 修改 | 日文翻译 |

---

## 验证方法

### 后端验证

```bash
# 编译通过
cd /home/wuxiafeng/AtlHyper/GitHub/atlhyper && go build ./...

# 单元测试（如有）
go test ./atlhyper_master_v2/...
```

### 前端验证

```bash
# 编译通过
cd atlhyper_web && npm run build

# 类型检查
npx tsc --noEmit
```

### 功能验证清单

#### Phase A 后端
- [ ] `GET /api/v2/aiops/ai/reports?incident_id=xxx` 返回报告列表
- [ ] `GET /api/v2/aiops/ai/reports/{id}` 返回报告详情（含 investigationSteps）
- [ ] `GET /api/v2/ai/budgets` 返回三个角色的预算信息
- [ ] `PUT /api/v2/ai/budgets/background` 更新预算成功
- [ ] `POST /api/v2/aiops/ai/analyze` 提交深度分析任务
- [ ] `POST /api/v2/aiops/ai/summarize` 优先返回已有报告

#### Phase B 前端设置页
- [ ] ProviderCard 显示角色标签（background/chat/analysis）
- [ ] ProviderCard 支持 Ollama 样式
- [ ] ProviderModal 有 baseUrl 输入框（Ollama 时显示）
- [ ] ProviderModal 有角色复选框
- [ ] 角色互斥冲突时显示错误提示
- [ ] 角色总览卡片显示三角色分配状态
- [ ] Mock 数据包含 roles 字段

#### Phase C 前端事件分析
- [ ] 事件详情自动加载已有 AI 报告
- [ ] 无报告时显示「生成 AI 分析」按钮
- [ ] 有报告时显示报告卡片列表
- [ ] 报告卡片展示摘要、根因、建议、相似事件
- [ ] 深度分析报告展示调查步骤时间线
- [ ] 「深入调查」按钮触发深度分析
- [ ] 分析提交后轮询直到新报告出现

#### Phase D 预算配置
- [ ] 预算配置卡片显示三个角色的用量和限制
- [ ] 可编辑每日上限、触发阈值、降级 Provider
- [ ] 保存成功 Toast 提示

#### i18n
- [ ] 所有新增 UI 文本有中文翻译
- [ ] 所有新增 UI 文本有日文翻译
- [ ] 无硬编码文本
