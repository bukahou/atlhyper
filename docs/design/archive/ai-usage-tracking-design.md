# AI 使用追踪与安全兜底设计

> **状态**: active
> **目标**: 闭合 AI 调用的记录、预算扣减、自动触发安全兜底三个断环

---

## 背景

当前 AI 子系统存在三个核心断环：

1. **预算扣减未接入** — `IncrementUsage()` SQL 已实现但无人调用，导致日预算永远不会耗尽
2. **调用记录不完整** — `saveReport()` 未填充 token/provider/duration 元数据；Chat 无独立调用记录
3. **自动触发安全兜底缺失** — background/analysis 自动触发未检查日预算，无并发上限

---

## 现有安全防护

### Chat（用户主动触发，不需要额外限制）

| 防护层 | 机制 | 位置 |
|--------|------|------|
| **认证** | JWT Token + `AuthRequired` 中间件 | `routes.go` — `r.mux` |
| **角色分配** | 严格模式：chat 角色未分配则前端拦截 | `role.go` + `hooks.ts` |
| **日预算** | `checkBudget()` 检查限额 | `role.go:64` |
| **全局超时** | `chatTimeout = 3min` | `chat.go:26` |
| **Tool 轮数上限** | `maxToolRounds = 5` | `chat.go:24` |

> Chat 防护链完整：认证 → 角色分配 → 日/月预算 → 超时 → 轮数上限。
> Phase 1 接通预算扣减后，预算检查即生效。

### Background/Analysis（自动触发，需要额外防护）

| 防护层 | 机制 | 状态 |
|--------|------|------|
| 严重度阈值 | `autoTriggerMinSeverity` 过滤低优先级事件 | ✅ 已有 |
| 去重（5 分钟） | 同一事件 5 分钟内不重复触发 | ✅ 已有 |
| 冷却（60 秒） | Enhancer 全局 60s 冷却期 | ✅ 已有 |
| 日/月预算前置检查 | 触发前检查 budget 余额 | ❌ 缺失 |
| 并发上限 | 同时进行的分析数量限制 | ❌ 缺失 |

---

## 预算模型重设计

### 问题

现有 `ai_role_budget` 表只有 `daily_token_limit` / `daily_call_limit` 单字段，存在以下缺陷：

1. **无 input/output 区分** — 无法分别限制输入和输出 Token（成本差异大）
2. **无月度维度** — 日限额可以每天刷满，月度累积失控
3. **无默认值** — 表初始为空，不配就等于无限额

### 新表结构

直接修改建表 SQL（`migrations.go`），删库重建：

```sql
CREATE TABLE IF NOT EXISTS ai_role_budget (
    role TEXT PRIMARY KEY,

    -- 日限额
    daily_input_token_limit INTEGER DEFAULT 0,
    daily_output_token_limit INTEGER DEFAULT 0,
    daily_call_limit INTEGER DEFAULT 0,

    -- 日消耗
    daily_input_tokens_used INTEGER DEFAULT 0,
    daily_output_tokens_used INTEGER DEFAULT 0,
    daily_calls_used INTEGER DEFAULT 0,
    daily_reset_at TEXT,

    -- 月限额
    monthly_input_token_limit INTEGER DEFAULT 0,
    monthly_output_token_limit INTEGER DEFAULT 0,
    monthly_call_limit INTEGER DEFAULT 0,

    -- 月消耗
    monthly_input_tokens_used INTEGER DEFAULT 0,
    monthly_output_tokens_used INTEGER DEFAULT 0,
    monthly_calls_used INTEGER DEFAULT 0,
    monthly_reset_at TEXT,

    -- 配置
    fallback_provider_id INTEGER,
    auto_trigger_min_severity TEXT DEFAULT 'critical',
    updated_at TEXT NOT NULL
)
```

> `0 = 无限制` 的语义保留，管理员可以手动设为 0 解除限制。

### 默认种子数据

首次建表后插入（`migrations.go`），提供开箱即用的保守限额：

| 角色 | 日输入 | 日输出 | 日调用 | 月输入 | 月输出 | 月调用 | 自动触发 |
|------|--------|--------|--------|--------|--------|--------|---------|
| background | 400,000 | 100,000 | 50 | 4,000,000 | 1,000,000 | 500 | low |
| chat | 800,000 | 200,000 | 100 | 8,000,000 | 2,000,000 | 1,000 | — |
| analysis | 1,600,000 | 400,000 | 20 | 8,000,000 | 2,000,000 | 100 | critical |

> 输入:输出 ≈ 4:1，因为输入包含 system prompt + 历史消息 + tool results，远大于输出。

### Token 计算方式

Token 数来自 Provider API 响应，不自行计算：

| Provider | 来源 | 映射 |
|----------|------|------|
| OpenAI | `usage.prompt_tokens` / `usage.completion_tokens` | → `InputTokens` / `OutputTokens` |
| Anthropic | `usage.input_tokens` / `usage.output_tokens` | → `InputTokens` / `OutputTokens` |
| Gemini | `usageMetadata.promptTokenCount` / `candidatesTokenCount` | → `InputTokens` / `OutputTokens` |
| Ollama | 最终 chunk `prompt_eval_count` / `eval_count` | → `InputTokens` / `OutputTokens` |

各 LLM 客户端已统一解析为 `llm.Usage{InputTokens, OutputTokens}` 返回。

### 预算检查逻辑

`checkBudget()` 改为分别检查 input 和 output：

```go
func checkBudget(budget *AIRoleBudget) bool {
    // 跨日/跨月重置（调用方负责）

    // 日限额检查（input 和 output 分别检查）
    if budget.DailyInputTokenLimit > 0 && budget.DailyInputTokensUsed >= budget.DailyInputTokenLimit {
        return false
    }
    if budget.DailyOutputTokenLimit > 0 && budget.DailyOutputTokensUsed >= budget.DailyOutputTokenLimit {
        return false
    }
    if budget.DailyCallLimit > 0 && budget.DailyCallsUsed >= budget.DailyCallLimit {
        return false
    }

    // 月限额检查
    if budget.MonthlyInputTokenLimit > 0 && budget.MonthlyInputTokensUsed >= budget.MonthlyInputTokenLimit {
        return false
    }
    if budget.MonthlyOutputTokenLimit > 0 && budget.MonthlyOutputTokensUsed >= budget.MonthlyOutputTokenLimit {
        return false
    }
    if budget.MonthlyCallLimit > 0 && budget.MonthlyCallsUsed >= budget.MonthlyCallLimit {
        return false
    }

    return true
}
```

### IncrementUsage 改为双维度

```go
// IncrementUsage SQL — 同时扣减日和月
func IncrementUsage(role string, inputTokens, outputTokens int) (string, []any) {
    return `UPDATE ai_role_budget SET
        daily_input_tokens_used = daily_input_tokens_used + ?,
        daily_output_tokens_used = daily_output_tokens_used + ?,
        daily_calls_used = daily_calls_used + 1,
        monthly_input_tokens_used = monthly_input_tokens_used + ?,
        monthly_output_tokens_used = monthly_output_tokens_used + ?,
        monthly_calls_used = monthly_calls_used + 1,
        updated_at = ?
    WHERE role = ?`,
    []any{inputTokens, outputTokens, inputTokens, outputTokens, time.Now().Format(time.RFC3339), role}
}
```

### 重置逻辑

```go
// ResetDailyUsage — 只重置日消耗
func ResetDailyUsage(role string) (string, []any)

// ResetMonthlyUsage — 重置月消耗（新增）
func ResetMonthlyUsage(role string) (string, []any)
```

跨日/跨月判断在 `loadAIConfigForRole` 中 `checkBudget()` 前执行：

```go
if needsDailyReset(budget)  { budgetRepo.ResetDailyUsage(ctx, role) }
if needsMonthlyReset(budget) { budgetRepo.ResetMonthlyUsage(ctx, role) }
```

---

## 设计方案

### Phase 1: 预算模型重构 + 扣减闭环（后端）

**目标**: 重构预算表结构 + 每次 AI 调用完成后扣减角色预算 + 更新 Provider 统计

#### 1.1 数据库层变更

- `database/types.go` — `AIRoleBudget` 结构体拆分 input/output + 新增月度字段
- `database/interfaces.go` — `AIRoleBudgetRepository.IncrementUsage` 签名改为 `(role, inputTokens, outputTokens int)`，新增 `ResetMonthlyUsage`
- `database/sqlite/migrations.go` — 建表 SQL 重写 + 种子数据
- `database/sqlite/ai_role_budget.go` — 所有 SQL 适配新字段
- `database/repo/ai_role_budget.go` — 适配新签名

#### 1.2 预算扣减 + Provider 统计封装

`ai/role.go` 新增统一记录方法：

```go
// RecordUsage 记录 AI 调用消耗（预算扣减 + Provider 统计更新）
func (s *aiServiceImpl) RecordUsage(ctx context.Context, role string, providerID int64, inputTokens, outputTokens int) {
    // 1. 扣减角色预算（日 + 月同时扣减）
    if s.budgetRepo != nil {
        if err := s.budgetRepo.IncrementUsage(ctx, role, inputTokens, outputTokens); err != nil {
            log.Warn("扣减角色预算失败", "role", role, "err", err)
        }
    }
    // 2. 累加 Provider 统计
    totalTokens := int64(inputTokens + outputTokens)
    if err := s.providerRepo.IncrementUsage(ctx, providerID, 1, totalTokens, 0); err != nil {
        log.Warn("更新 Provider 统计失败", "provider", providerID, "err", err)
    }
}
```

> Provider 的 `IncrementUsage(id, requests, tokens, cost)` 已存在，cost 传 0（不跟踪费用）。

#### 1.3 接入三个调用点

**Chat** (`ai/chat.go` — `chatLoop` 两个结束点):

```go
s.RecordUsage(ctx, RoleChat, roleCfg.ProviderID, totalInputTokens, totalOutputTokens)
```

**Background** (`aiops/ai/enhancer.go` — `SummarizeBackground` 完成时)

**Analysis** (`aiops/ai/analysis.go` — `RunAnalysis` 完成时)

需要将 `RecordUsage` 作为回调函数传入 Enhancer，或让 Enhancer 持有必要的 repo 引用。

#### 1.4 预算检查 + 跨日/跨月重置

`ai/role.go` — `loadAIConfigForRole` 中，在 `checkBudget()` 前执行重置检查：

```go
if budget, _ := s.budgetRepo.Get(ctx, role); budget != nil {
    if needsDailyReset(budget) {
        s.budgetRepo.ResetDailyUsage(ctx, role)
        // 清零内存中的日消耗
    }
    if needsMonthlyReset(budget) {
        s.budgetRepo.ResetMonthlyUsage(ctx, role)
        // 清零内存中的月消耗
    }
    if !checkBudget(budget) {
        // 预算耗尽 → fallback 或报错
    }
}
```

---

### Phase 2: AI Report 元数据补全（后端）

**目标**: `saveReport()` 填充完整的调用元数据

#### 2.1 Enhancer.saveReport 补全

`aiops/ai/enhancer.go` — `saveReport()` 补充参数：

```go
func (e *Enhancer) saveReport(ctx context.Context, incident *database.AIOpsIncident,
    result *SummarizeResponse, trigger string,
    providerName, model string, inputTokens, outputTokens int, duration time.Duration) {

    report := &database.AIReport{
        // ... 现有字段不变 ...
        ProviderName: providerName,
        Model:        model,
        InputTokens:  inputTokens,
        OutputTokens: outputTokens,
        DurationMs:   duration.Milliseconds(),
    }
    // ...
}
```

调用方 `SummarizeBackground()` 传入 `roleCfg.ProviderName`, `roleCfg.Model`, token 统计, `time.Since(startTime)`。

#### 2.2 Analysis saveAnalysisResult 补全

`aiops/ai/analysis.go` — 同样补充 provider/model/tokens 字段。

---

### Phase 3: 自动触发安全兜底（后端）

**目标**: 防止 background/analysis 自动触发导致 AI API 成本失控

> Chat 由用户主动触发，天然受人类交互速度限制，不需要额外防护。
> 需要防护的是 background（事件触发）和 analysis（自动升级触发）。

#### 3.1 Background 预算前置检查

`aiops/ai/background.go` — `process()` 在 severity 检查之后、调用 AI 之前检查日/月预算：

```go
if bt.budgetRepo != nil {
    budget, err := bt.budgetRepo.Get(context.Background(), "background")
    if err == nil && budget != nil && !checkBudget(budget) {
        log.Warn("后台分析预算已用尽，跳过", "incident", evt.IncidentID)
        return
    }
}
```

#### 3.2 Analysis 预算前置检查

`aiops/ai/analysis.go` — `RunAnalysis()` 入口检查 analysis 角色预算：

```go
if e.budgetRepo != nil {
    budget, _ := e.budgetRepo.Get(ctx, "analysis")
    if budget != nil && !checkBudget(budget) {
        return fmt.Errorf("analysis 角色预算已用尽")
    }
}
```

#### 3.3 Background 并发上限

`aiops/ai/enhancer.go` — 限制同时进行的 background 分析数量：

```go
type Enhancer struct {
    // ... 现有字段 ...
    concurrencySem chan struct{} // 并发信号量，限制同时进行的分析数
}

// 初始化时: concurrencySem: make(chan struct{}, 3) — 最多 3 个并发
```

`SummarizeBackground()` 入口：

```go
select {
case e.concurrencySem <- struct{}{}:
    defer func() { <-e.concurrencySem }()
default:
    return nil, fmt.Errorf("后台分析并发上限，跳过")
}
```

现有 60s 冷却 + 5 分钟去重 + 严重度阈值已提供基本保护，并发上限是最后一道防线。

---

### Phase 4: 调用历史 API + 前端（后端 + 前端）

**目标**: 让管理员能查看所有 AI 调用记录 + 预算配置编辑

#### 4.1 新增调用历史查询接口

`database/interfaces.go` — `AIReportRepository` 新增：

```go
// ListRecent 获取最近的 AI 报告（跨事件、跨角色）
ListRecent(ctx context.Context, limit, offset int) ([]*AIReport, int, error)
```

`service/interfaces.go` — `QueryAdmin` 新增：

```go
ListRecentAIReports(ctx context.Context, limit, offset int) ([]*database.AIReport, int, error)
```

`gateway/routes.go` — 新增路由（Operator 权限）：

```go
register("/api/v2/ai/reports", aiProviderH.AIReportsHandler)
```

返回格式：

```json
{
  "message": "获取成功",
  "data": [
    {
      "id": 1,
      "incidentId": "inc-xxx",
      "clusterId": "cluster-1",
      "role": "background",
      "trigger": "incident_created",
      "summary": "...",
      "providerName": "Ollama",
      "model": "qwen2.5:14b",
      "inputTokens": 2633,
      "outputTokens": 512,
      "durationMs": 8500,
      "createdAt": "2026-03-08T10:00:00Z"
    }
  ],
  "total": 42
}
```

#### 4.2 前端 BudgetConfigCard 改造

适配新字段结构，每个角色卡片显示：

- 日输入 Token 限额 / 已用（进度条）
- 日输出 Token 限额 / 已用（进度条）
- 日调用次数 限额 / 已用
- 月输入 Token 限额 / 已用（进度条）
- 月输出 Token 限额 / 已用（进度条）
- 月调用次数 限额 / 已用
- 自动触发最低严重度（下拉）
- 降级 Provider（下拉）

管理员可编辑所有限额字段并保存。

#### 4.3 前端调用历史

在 `/settings/ai` 页面新增「调用历史」Section：

- 显示最近 AI 调用记录列表（角色、Provider、Model、输入/输出 Token、耗时）
- 按角色筛选

前端文件变更：

| 文件 | 变更 |
|------|------|
| `api/ai-provider.ts` | 类型适配新字段 + 新增 `getAIReports()` |
| `types/i18n.ts` | 新增翻译键 |
| `i18n/locales/zh.ts` + `ja.ts` | 新增翻译 |
| `settings/ai/components/BudgetConfigCard.tsx` | 改造：适配 input/output 拆分 + 月度字段 |
| `settings/ai/components/UsageHistoryCard.tsx` | **新增** — 调用历史表格 |
| `settings/ai/page.tsx` | 加入 `UsageHistoryCard` |

---

## 文件变更清单

### Phase 1: 预算模型重构 + 扣减闭环

| 文件 | 操作 | 说明 |
|------|------|------|
| `database/types.go` | 修改 | `AIRoleBudget` 拆分 input/output + 新增月度字段 |
| `database/interfaces.go` | 修改 | `IncrementUsage` 签名改为 `(role, inputTokens, outputTokens)`，新增 `ResetMonthlyUsage` |
| `database/sqlite/migrations.go` | 修改 | 建表 SQL 重写 + 种子数据 |
| `database/sqlite/ai_role_budget.go` | 修改 | 所有 SQL 适配新字段 |
| `database/repo/ai_role_budget.go` | 修改 | 适配新签名 |
| `ai/role.go` | 修改 | `checkBudget` 改多维度 + `RecordUsage` + 跨日/跨月重置 |
| `ai/chat.go` | 修改 | `chatLoop` 结束时调用 `RecordUsage` |
| `aiops/ai/enhancer.go` | 修改 | `SummarizeBackground` 完成时扣减（需要 `RecordUsage` 回调或 repo 引用） |
| `aiops/ai/analysis.go` | 修改 | `RunAnalysis` 完成时扣减 |
| `gateway/handler/admin/ai_provider.go` | 修改 | `budgetResponse` struct 拆分 input/output + 月度字段；`BudgetHandler` 请求 struct 适配新字段 |

### Phase 2: Report 元数据补全

| 文件 | 操作 | 说明 |
|------|------|------|
| `aiops/ai/enhancer.go` | 修改 | `saveReport()` 补充 provider/model/tokens/duration |
| `aiops/ai/analysis.go` | 修改 | 保存报告时补充元数据 |

### Phase 3: 自动触发安全兜底

| 文件 | 操作 | 说明 |
|------|------|------|
| `aiops/ai/background.go` | 修改 | 触发前检查预算 |
| `aiops/ai/analysis.go` | 修改 | 入口检查预算 |
| `aiops/ai/enhancer.go` | 修改 | 新增并发信号量（最多 3 个同时分析） |

### Phase 4: 调用历史 API + 前端

| 文件 | 操作 | 说明 |
|------|------|------|
| `database/interfaces.go` | 修改 | `AIReportRepository` 新增 `ListRecent` |
| `database/sqlite/ai_report.go` | 修改 | 实现 `ListRecent` SQL |
| `database/repo/ai_report.go` | 修改 | 实现 `ListRecent` 方法 |
| `service/interfaces.go` | 修改 | `QueryAdmin` 新增 `ListRecentAIReports` |
| `service/query/admin.go` | 修改 | 实现查询 |
| `gateway/handler/admin/ai_provider.go` | 修改 | 新增 `AIReportsHandler`（调用历史查询端点） |
| `gateway/routes.go` | 修改 | 注册 `/api/v2/ai/reports` 路由 |
| `api/ai-provider.ts` | 修改 | 类型适配 + 新增 `getAIReports()` |
| `types/i18n.ts` | 修改 | 新增翻译键 |
| `i18n/locales/zh.ts` | 修改 | 新增中文翻译 |
| `i18n/locales/ja.ts` | 修改 | 新增日文翻译 |
| `settings/ai/components/BudgetConfigCard.tsx` | 修改 | 适配 input/output 拆分 + 月度字段 |
| `settings/ai/components/UsageHistoryCard.tsx` | **新增** | 调用历史表格 |
| `settings/ai/page.tsx` | 修改 | 集成 UsageHistoryCard |

---

## 验证方法

### Phase 1

1. 首次启动 → `ai_role_budget` 表有 3 条种子数据（background/chat/analysis）
2. 触发后台分析 → `daily_input_tokens_used` / `daily_output_tokens_used` / `monthly_*` 均增加
3. Chat 对话 → 同上
4. 检查 `ai_providers` 表 `total_requests`/`total_tokens` 增加
5. 设置低限额 → 预算耗尽后触发被拒绝
6. 跨日 → 日消耗重置，月消耗保留
7. 跨月 → 月消耗重置

### Phase 2

1. 触发 background/analysis 分析
2. 检查 `ai_reports` 表 `provider_name`/`model`/`input_tokens`/`output_tokens`/`duration_ms` 有值

### Phase 3

1. Background 预算耗尽 → 新事件不再触发自动分析
2. Analysis 预算耗尽 → 手动/自动触发深度分析被拒绝
3. 同时触发 4+ 个 background 分析 → 第 4 个被跳过（并发上限 3）

### Phase 4

1. `GET /api/v2/ai/reports?limit=20` 返回最近调用记录（含 inputTokens/outputTokens 分开）
2. 前端预算配置卡片显示 input/output 分别的进度条
3. 管理员可编辑日/月限额并保存
4. 前端调用历史列表正常展示
