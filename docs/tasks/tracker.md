# 任务追踪

## 进行中

### AI WebChat 功能实施

> 设计文档: `docs/design/ai-system.md` v2.0
> 前置完成: MQ TopicAI、指令持久化、Redis 备选后端

#### P1: 数据库层

- [ ] 1.1 新增 DB 模型: `AIConversation` + `AIMessage` (database/interfaces.go)
- [ ] 1.2 新增 Dialect 接口: `AIConversationDialect` + `AIMessageDialect`
- [ ] 1.3 实现 SQLite Dialect (database/sqlite/ai.go)
- [ ] 1.4 实现 Repo (database/repo/ai_conversation.go + ai_message.go)
- [ ] 1.5 更新 DB 结构体 + repo.Init + Migration

#### P2: LLM 抽象层

- [ ] 2.1 定义 LLM 接口 (ai/llm/interfaces.go): LLMClient, Request, Chunk, ToolCall, ToolDefinition
- [ ] 2.2 实现 LLM 工厂 (ai/llm/factory.go): 根据 provider 创建
- [ ] 2.3 实现 Gemini Client (ai/llm/gemini/client.go): ChatStream + Function Calling
- [ ] 2.4 添加 Gemini SDK 依赖 (go.mod)

#### P3: AI Core

- [ ] 3.1 定义 AIService 接口 (ai/interfaces.go): 对外类型 + 接口
- [ ] 3.2 实现 Blacklist (ai/blacklist.go): Action/Namespace/Resource 校验
- [ ] 3.3 实现 Prompt 构建 (ai/prompt.go): embed 加载 + 拼接
- [ ] 3.4 编写提示词文件: security.txt, role.txt, tools.json
- [ ] 3.5 实现 Tool 执行器 (ai/tool.go): CommandService + WaitCommandResult
- [ ] 3.6 实现 AIService (ai/service.go): 工厂 + CRUD
- [ ] 3.7 实现 Chat 逻辑 (ai/chat.go): 多轮 Tool Calling 循环 + SSE channel

#### P4: Gateway 集成

- [ ] 4.1 实现 AI Handler (gateway/handler/ai.go): SSE 流 + CRUD 路由
- [ ] 4.2 注册 AI 路由 (gateway/routes.go)

#### P5: Config + 组装

- [ ] 5.1 新增 AIConfig (config/types.go + defaults.go + loader.go)
- [ ] 5.2 更新 master.go: 创建 LLMClient + AIService，注入 Gateway

#### P6: 验证

- [ ] 6.1 go build + go vet
- [ ] 6.2 手动测试: 创建对话 → 发送消息 → 验证 SSE 流
- [ ] 6.3 Git commit

---

## 待办

(无)

---

## 已完成

### 2025-01-24: AI 底层基础设施

- [x] MQ: 多 Topic 支持 (TopicOps + TopicAI)
- [x] MQ: Redis 后端实现 (mq/redis/bus.go)
- [x] DataHub: Redis 后端实现 (datahub/redis/store.go)
- [x] 工厂模式: config 切换 memory/redis
- [x] 指令持久化: CreateCommand → DB (status=pending)
- [x] 结果持久化: AckCommand → DB (status=success/failed, result, duration)
- [x] ActionDynamic: AI 专用灵活查询动作
- [x] Source="ai": 全链路审计追踪

(归档到 `archive/` 目录)
