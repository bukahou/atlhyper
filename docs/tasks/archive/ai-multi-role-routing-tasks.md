# AI 多角色路由 + 功能实现 — 完成记录

> 完成日期: 2026-03-07
> 设计文档:
> - [00-ai-role-definition.md](../../design/active/00-ai-role-definition.md)
> - [02-ai-reports-storage-design.md](../../design/active/02-ai-reports-storage-design.md)
> - [03-ai-role-routing-design.md](../../design/active/03-ai-role-routing-design.md)
> - [04-ai-background-analysis-design.md](../../design/active/04-ai-background-analysis-design.md)

## Phase 1: 数据模型 + 迁移 ✅

- `ai/role.go` — 角色常量 (RoleBackground / RoleChat / RoleAnalysis) + ValidRoles + IsValidRole
- `database/types.go` — AIProvider.Roles + AIProvider.ContextWindowOverride; AIProviderModel.ContextWindow; AIRoleBudget; AIReport
- `database/interfaces.go` — AIProviderRepository.UpdateRoles/FindByRole; AIRoleBudgetRepository; AIReportRepository; 对应 Dialect 接口
- `database/sqlite/migrations.go` — ai_providers 新增 roles + context_window_override; ai_provider_models 新增 context_window; ai_role_budget 表; ai_reports 表
- `database/sqlite/ai.go` — ScanRow 适配新字段
- `database/sqlite/ai_role_budget.go` — AIRoleBudgetDialect 实现
- `database/sqlite/ai_report.go` — AIReportDialect 实现
- `database/repo/ai_provider.go` — UpdateRoles + FindByRole 实现
- `database/repo/ai_role_budget.go` — AIRoleBudgetRepository 实现
- `database/repo/ai_report.go` — AIReportRepository 实现

## Phase 2: 上下文管理器 ✅

- `ai/context.go` — ContextManager: FitMessages (贪心后向填充) + estimateTokens (1 token ≈ 2.5 chars) + toolResultMaxLen (按 context_window 动态调整)
- `ai/chat.go` — chatLoop 集成 ContextManager（发送前裁剪 + 前端截断提示 + Tool 结果动态截断）

## Phase 3: 角色路由逻辑 ✅

- `ai/role.go` — RoleConfig (wraps llm.Config + ContextWindow + ProviderID/Name); loadAIConfigForRole (角色→Provider→预算检查→降级链); loadAIConfigFallback; EffectiveContextWindow; checkBudget; MaxPromptCharsForContext
- `ai/service.go` — aiServiceImpl 新增 modelRepo + budgetRepo 字段
- `ai/chat.go` — loadAIConfig 简化为调用 loadAIConfigForRole(RoleChat); chatLoop 使用 RoleConfig
- `master.go` — llmFactory 返回 (LLMClient, contextWindow, error); 搜索 background 角色 Provider 优先
- `aiops/ai/enhancer.go` — LLMClientFactory 签名变更; buildPromptWithTruncation 接收 contextWindow
- `aiops/ai/enhancer_test.go` — 所有 mock 适配新签名

## Phase 4: API + 路由 ✅

- `gateway/handler/admin/ai_provider.go` — ProviderResponse 新增 Roles + ContextWindowOverride; ProviderRolesHandler (PUT /providers/{id}/roles, 角色验证+互斥检查); RolesOverviewHandler (GET /roles); ProviderHandler 委派 /roles 子路径
- `gateway/routes.go` — 注册 /api/v2/ai/roles (operator 可读)
- `service/interfaces.go` — OpsAdmin 新增 UpdateAIProviderRoles
- `service/operations/admin.go` — UpdateAIProviderRoles 实现

## Phase 5: 报告存储集成 ✅

- `aiops/ai/enhancer.go` — NewEnhancer 接收 reportRepo; summarizeCore 提取共用核心; Summarize/SummarizeBackground 分别保存不同 trigger; saveReport 持久化报告
- `master.go` — 传递 db.AIReport 给 NewEnhancer
- `aiops/ai/enhancer_test.go` — 所有 NewEnhancer 调用适配新签名 (nil reportRepo)

## Phase 6: background 自动触发 ✅

- `aiops/ai/background.go` — backgroundTrigger: channel-based worker + 5分钟去重 + 严重度阈值检查 (从 ai_role_budget 读取) + Submit/Stop
- `aiops/ai/enhancer.go` — EnableBackgroundTrigger + NotifyIncidentEvent + Stop
- `aiops/core/engine.go` — IncidentNotifyFunc 类型; SetIncidentNotify 方法; OnWarningCreated/OnStateEscalated 中调用通知
- `aiops/interfaces.go` — Engine 接口新增 SetIncidentNotify
- `master.go` — 启用后台触发器 + 连接 Engine 通知

## Phase 7: analysis 深度分析 ✅

- `aiops/ai/analysis.go` — RunAnalysis: 多轮 Tool Calling 循环 (最大 8 轮); AnalysisConfig (LLMFactory + ToolExecuteFunc + ToolDefs + repos); InvestigationStep/InvestigationTool (每轮记录); 每轮写 DB 防崩溃丢失; 分析完成后保存最终报告
- `aiops/ai/background.go` — SetAnalysisConfig; maybeTriggerAnalysis (检查 analysis 角色 auto_trigger_min_severity)
- `ai/interfaces.go` — AIService 新增 GetToolExecuteFunc + GetToolDefs
- `ai/service.go` — 实现 GetToolExecuteFunc + GetToolDefs
- `master.go` — 配置 AnalysisConfig (复用 AI Service 的 Tool 基础设施)
