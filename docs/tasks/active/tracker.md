# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

---

## AI 模块架构整理 + 提示词优化 — 待办

> 设计文档: [ai-role-prompts-optimization-design.md](../../design/active/ai-role-prompts-optimization-design.md)

- Phase 1: 提示词迁移 — 创建 `ai/prompts/` 子包，统一管理 3 角色提示词
- Phase 2: AI 接口扩展 — `ai.AIService` 新增 `Analyze()` + `Complete()` 方法
- Phase 3: aiops/ai → aiops/enricher 重命名 + 解耦 — 通过 `ai.AIService` 接口调用
