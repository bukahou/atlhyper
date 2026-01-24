# 任务追踪

## 进行中

### AI WebChat 功能实施

> 设计文档: `docs/design/ai-system.md` v2.0

#### P6: 验证

- [x] 6.1 go build + go vet
- [ ] 6.2 手动测试: 创建对话 → 发送消息 → 验证 SSE 流
- [x] 6.3 Git commit (P1-P5 各阶段已提交)

---

## 待办

(无)

---

## 已完成

### 2025-01-24: AI WebChat 功能 (P1-P5)

- [x] P1: 数据库层 (AIConversation + AIMessage 模型/Dialect/Repo/Migration)
- [x] P2: LLM 抽象层 (LLMClient 接口 + Gemini 流式实现)
- [x] P3: AI Core (AIService/Blacklist/Prompt/Tool执行器/Chat多轮循环)
- [x] P4: Gateway 集成 (SSE Handler + CRUD 路由)
- [x] P5: Config + 组装 (AIConfig + master.go 条件注入)

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
