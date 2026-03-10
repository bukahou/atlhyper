# AI 模块架构整理 + 提示词优化 — 已完成

> 设计文档: [ai-role-prompts-optimization-design.md](../../design/archive/ai-role-prompts-optimization-design.md)

## Phase 1: 提示词迁移 — ✅

- 创建 `ai/prompts/` 子包（security.go / chat.go / background.go / analysis.go / tools.go）
- 散落在 3 个文件的提示词统一迁移
- chat.go / service.go 改调 `prompts.BuildChatPrompt()` / `prompts.GetToolDefinitions()`
- aiops/ai 改调 `prompts.BuildBackgroundPrompt()` / `prompts.BuildAnalysisPrompt()`
- Commit: `7cf45f9`

## Phase 2: AI 接口扩展 — ✅

- `ai.AIService` 新增 `Analyze()` 多轮 Tool Calling 接口
- `ai.AIService` 新增 `Complete()` 单轮 LLM 调用接口
- `ai/analyze.go` 通用多轮 Tool Calling 循环实现
- `ai/service.go` 单轮 Complete 实现
- 类型定义: `AnalyzeRequest/Result`, `CompleteRequest/Result`, `AnalyzeStep`, `ToolCallRecord`
- Commit: `2d66d19`

## Phase 3: aiops/ai → aiops/enricher 解耦 — ✅

- 目录重命名 `aiops/ai/` → `aiops/enricher/`
- `Enhancer` → `Enricher`，持有 `ai.AIService` 接口替代 `LLMClientFactory`
- 删除 `LLMClientFactory` / `LLMClientMeta` / `RecordUsageFunc`（跳层依赖）
- 删除 `analysis.go`（逻辑已移至 `ai/analyze.go`）
- `master.go` 初始化从 ~60 行简化至 ~10 行
- 24 个测试全通过
- Commit: `fa8b52a`
