# AI 工具增强 — OTel 查询能力与事件上下文丰富 — 已完成

> 设计文档: [ai-tools-enhancement-design.md](../../design/archive/ai-tools-enhancement-design.md)
> 实现方式: Phase 1-3 一次性提交

## Phase 1: Command 路径工具 — ✅

- `ai/prompts/tools.go` — `toolsJSON` 追加 `query_traces` 和 `query_logs` 定义
- `ai/tool.go` — 新增 `TruncateToolResult()` / `truncateLogBodies()` 辅助函数
- `ai/prompts/chat.go` — `chatRole` 新增 `[可观测性查询工具]` + `[工具组合使用建议]`
- `ai/prompts/analysis.go` — `analysisSystem` 新增调查策略和信号关联指引
- `master.go` — 注册 `query_traces` / `query_logs` ToolHandler

## Phase 2: 内存直读工具 — ✅

- `ai/prompts/tools.go` — `toolsJSON` 追加 `query_slo` 和 `get_entity_detail` 定义
- `ai/tool.go` — 新增 `BuildEntityKey()` / `SimplifyEntityDetail()` 辅助函数
- `master.go` — 注册 `query_slo` / `get_entity_detail` ToolHandler

## Phase 3: 事件上下文丰富 — ✅

- `ai/prompts/background.go` — `IncidentPromptContext` 新增 3 个 OTel 字段
- `ai/prompts/background.go` — `BuildBackgroundPrompt()` 拼接 OTel 上下文
- `aiops/enricher/context_builder.go` — 新增 `buildOTelContext()` + 辅助函数
- `aiops/enricher/enricher.go` — 新增 `SetStore()`，`summarizeCore()` 读取 OTelSnapshot 丰富上下文

## 增强后工具全景（4 → 8 个）

| # | 工具 | 数据来源 | 执行路径 |
|---|------|---------|---------|
| 1 | `query_cluster` | K8s API | Command → Agent |
| 2 | `analyze_incident` | AIOps + OTel | 内存 + LLM |
| 3 | `get_cluster_risk` | AIOps Scorer | 内存直读 |
| 4 | `get_recent_incidents` | AIOps Store | 内存直读 |
| 5 | `query_traces` | ClickHouse | Command → Agent |
| 6 | `query_logs` | ClickHouse | Command → Agent |
| 7 | `query_slo` | OTelSnapshot | 内存直读 |
| 8 | `get_entity_detail` | AIOps Engine | 内存直读 |
