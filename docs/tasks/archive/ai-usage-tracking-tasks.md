# AI 使用追踪与安全兜底 — 已完成

> 原设计文档: [ai-usage-tracking-design.md](../../design/archive/ai-usage-tracking-design.md)
> 完成日期: 2026-03-08

## Phase 1: 预算模型重构 + 扣减闭环（后端 + 前端） ✅

- DB 表结构重构（input/output 拆分 + 月度维度）
- 种子数据（background/chat/analysis 默认限额）
- checkBudget 多维度检查 + 跨日/跨月重置
- RecordUsage 统一扣减（Chat + Background + Analysis）
- Gateway Handler 适配新字段
- 前端 BudgetConfigCard 进度条 + 编辑
- i18n 翻译扩展（zh + ja）

## Phase 2+4: Report 元数据补全 + 调用历史 API/前端 ✅

- LLMClientMeta 工厂返回 Provider 元数据（ProviderID/Name/Model）
- saveReport / saveAnalysisResult 填充 tokens/provider/duration
- saveAnalysisResult bug 修复：UpdateResult 保存全部字段（之前只存 investigation_steps）
- ListRecent API（GET /api/v2/ai/reports，支持 role 筛选 + 分页）
- UsageHistoryCard 前端组件（角色筛选 + 加载更多）

## Phase 3: 自动触发安全兜底（后端） ✅

- Background 预算前置检查（process() 中 AI 调用前检查 isBudgetAvailable）
- Analysis 预算前置检查（AnalysisConfig.BudgetRepo + RunAnalysis 入口检查）
- Analysis 预算前置检查（maybeTriggerAnalysis 中检查 analysis 角色预算）
- Background 并发信号量（maxConcurrentAnalysis=3，SummarizeBackground 入口 select）

## 关键文件变更

### 后端
- `database/types.go` — AIRoleBudget 拆分 input/output + 月度字段
- `database/interfaces.go` — IncrementUsage 新签名 + ResetMonthlyUsage + ListRecent + UpdateResult
- `database/sqlite/migrations.go` — 建表 SQL 重写 + 种子数据
- `database/sqlite/ai_role_budget.go` — SQL 适配
- `database/sqlite/ai_report.go` — SelectRecent/CountRecent/UpdateResult 实现
- `database/repo/ai_role_budget.go` — 适配新签名
- `database/repo/ai_report.go` — ListRecent/UpdateResult 实现
- `ai/role.go` — checkBudget 多维度 + RecordUsage + 跨日/跨月重置
- `ai/chat.go` — chatLoop 结束时 RecordUsage
- `aiops/ai/enhancer.go` — LLMClientMeta + concurrencySem + saveReport 元数据
- `aiops/ai/analysis.go` — BudgetRepo 前置检查 + saveAnalysisResult 完整更新
- `aiops/ai/background.go` — 预算前置检查 + isBudgetAvailable
- `service/interfaces.go` — ListRecentAIReports
- `service/query/admin.go` — 实现 ListRecentAIReports
- `gateway/handler/admin/ai_provider.go` — AIReportsHandler + budgetResponse 适配
- `gateway/routes.go` — /api/v2/ai/reports 路由
- `master.go` — LLMClientFactory 签名更新 + AnalysisConfig.BudgetRepo 注入

### 前端
- `api/ai-provider.ts` — AIReportItem 类型 + getAIReports()
- `types/i18n.ts` — 9 个新翻译键
- `i18n/locales/zh.ts` + `ja.ts` — 翻译
- `settings/ai/components/BudgetConfigCard.tsx` — 适配 input/output 拆分 + 月度
- `settings/ai/components/UsageHistoryCard.tsx` — 新增调用历史表格
- `settings/ai/page.tsx` — 集成 UsageHistoryCard
