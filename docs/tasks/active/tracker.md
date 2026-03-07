# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

---

## AI 多角色路由 + 功能实现

> 原设计文档:
> - [00-ai-role-definition.md](../../design/active/00-ai-role-definition.md)
> - [03-ai-role-routing-design.md](../../design/active/03-ai-role-routing-design.md)
> - [02-ai-reports-storage-design.md](../../design/active/02-ai-reports-storage-design.md)
> - [04-ai-background-analysis-design.md](../../design/active/04-ai-background-analysis-design.md)

### Phase 1: 数据模型 + 迁移 `00` `03`

> 设计文档: [00-ai-role-definition](../../design/active/00-ai-role-definition.md) + [03-ai-role-routing-design](../../design/active/03-ai-role-routing-design.md)

- `ai/role.go` — 角色常量（RoleBackground / RoleChat / RoleAnalysis）
- `database/types.go` — AIProvider 新增 Roles 字段; AIProviderModel 新增 ContextWindow 字段
- `database/sqlite/migrations.go` — ai_providers 新增 roles + context_window_override 列; ai_provider_models 新增 context_window 列
- `database/sqlite/ai.go` — CRUD 适配新字段
- `config/defaults.go` — initDefaultAIModels 填充各模型 context_window 默认值
- 测试: 迁移 + CRUD 验证

### Phase 2: 上下文管理器 `03`

> 设计文档: [03-ai-role-routing-design](../../design/active/03-ai-role-routing-design.md) — 「上下文管理器」章节

- `ai/context.go` — ContextManager: FitMessages + estimateTokens + estimateMessageTokens
- `ai/chat.go` — chatLoop 集成 ContextManager（发送前裁剪 + 前端提示）
- `ai/chat.go` — Tool 结果截断按 context_window 动态调整（toolResultMaxLen）
- 测试: 裁剪策略单元测试（8K/32K/无限制场景）

### Phase 3: 角色路由逻辑 `03`

> 设计文档: [03-ai-role-routing-design](../../design/active/03-ai-role-routing-design.md) — 「角色路由解析」章节

- `ai/role.go` — loadAIConfigForRole 解析函数（角色→Provider→配置）
- `database/interfaces.go` — AIProviderRepository 扩展（UpdateRoles, FindByRole）
- `database/sqlite/ai.go` — UpdateRoles / FindByRole 实现
- `ai/service.go` — aiServiceImpl 适配角色路由
- `master.go` — llmFactory 适配
- `aiops/ai/enhancer.go` — Prompt 截断感知 context_window
- 测试: 路由解析 + 向后兼容（无角色分配时退回全局 Provider）

### Phase 4: API + 前端 `03`

> 设计文档: [03-ai-role-routing-design](../../design/active/03-ai-role-routing-design.md) — 「API 设计」+「前端 UI 设计」章节

- `gateway/handler/admin/ai_provider.go` — Provider API 支持 roles / context_window
- `gateway/routes.go` — 路由注册
- 前端 Provider 卡片改造（角色标签 + context_window 显示 + 角色分配交互）
- i18n（zh.ts / ja.ts）

### Phase 5: 报告存储 `02`

> 设计文档: [02-ai-reports-storage-design](../../design/active/02-ai-reports-storage-design.md)

- `database/types.go` — AIReport struct
- `database/interfaces.go` — AIReportRepository 接口
- `database/sqlite/migrations.go` — ai_reports 表 + 索引
- `database/sqlite/ai_report.go` — Repository 实现
- 测试: CRUD + 按事件/集群查询

### Phase 6: background 自动触发 `04`

> 设计文档: [04-ai-background-analysis-design](../../design/active/04-ai-background-analysis-design.md) — 「一、background」章节

**行为链**: Engine 事件总线发布 → Enhancer 订阅入队 → 优先级队列单 worker 按风险排序执行 → 写 ai_reports + 同步 incidents.summary

- 解耦: 事件总线（Engine 发布，Enhancer 订阅，零耦合）
- 执行: 优先级队列 + 单 worker（按 RiskScore 降序，不无限并发）
- 去重: 同 IncidentID 替换队列中旧任务 + 执行中 needRerun 标记 + 完成后 30s 冷却
- 失败: 重试 1 次 + 熔断（连续 ≥3 次失败暂停 5min→10min→20min，上限 1h）
- 定时巡检报告（后续扩展）

### Phase 7: analysis 深度分析 `04`

> 设计文档: [04-ai-background-analysis-design](../../design/active/04-ai-background-analysis-design.md) — 「二、analysis」章节

**行为链**: severity=critical 自动触发 → 共享优先级队列（analysis 优先级高于 background）→ 复用 chatLoop 多轮 Tool Calling（最大 8 轮，AI 自主决定是否继续）→ 每轮写 ai_reports + 事件打标签

- 触发: 事件严重度 >= auto_trigger_min_severity 自动升级（DB 可配置，默认 critical；与 background 共享队列，analysis 优先级更高）
- 路由: loadAIConfigForRole("analysis")（Phase 3 提供）
- 执行: 复用 chatLoop 的 Tool Calling 基础设施，最大 8 轮，AI 返回 `{"continue": false}` 停止
- 记录: 每轮立即写 DB（思考过程 + Tool 调用 + 结果），中途崩溃不丢步骤
- 产出: 写入 ai_reports（Phase 5 提供），事件打上"已深度分析"标签
- 失败: 与 background 共享熔断器

**待讨论**: system prompt 详细内容设计
