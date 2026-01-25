# 任务追踪

## 进行中

### 通知系统实施

> 设计文档: `docs/design/notify-system.md` v1.0

#### P1: 核心组件

- [ ] 1.1 `notifier/alert.go` — Alert 结构体定义
- [ ] 1.2 `notifier/dedup.go` — 去重缓存 (10分钟 TTL)
- [ ] 1.3 `notifier/buffer.go` — 聚合缓冲 (30秒窗口)
- [ ] 1.4 `notifier/limiter.go` — 限流器 (5条/分钟)

#### P2: AlertManager

- [ ] 2.1 `notifier/manager.go` — AlertManager 主逻辑
- [ ] 2.2 `notifier/dispatch.go` — 分发到各渠道
- [ ] 2.3 初始化集成 (master.go 注入)

#### P3: 消息模板

- [ ] 3.1 `notifier/slack.go` — Slack BlockKit 聚合模板
- [ ] 3.2 `notifier/email.go` — Email HTML 聚合模板

#### P4: 触发点集成

- [ ] 4.1 `service/agent.go` — Agent 心跳超时检测
- [ ] 4.2 `handler/notify.go` — 测试发送接口 (真实发送)

#### P5: 验证

- [ ] 5.1 go build + go vet
- [ ] 5.2 手动测试: 配置 Slack → 测试发送
- [ ] 5.3 手动测试: Agent 离线 → 收到告警
- [ ] 5.4 Git commit

---

## 待办

### 前端通知配置页面

> 依赖: 通知系统实施完成

- [ ] Slack 配置表单
- [ ] Email 配置表单
- [ ] 测试发送按钮
- [ ] 渠道启用/禁用开关

---

## 已完成

### 2025-01-25: 工作台页面重构

- [x] 新增 `/workbench` 工作台首页
- [x] AI 对话移动到 `/workbench/ai`
- [x] 导航配置更新
- [x] i18n 翻译更新

### 2025-01-25: AI Chat 优化

- [x] InspectorPanel 重设计 (集群上下文 + 对话统计 + 能力边界)
- [x] ToolCallBlock 简化 (移除结果展示，保留 token 显示)
- [x] 数据截断修复 (list 操作返回表格摘要)
- [x] AI/Ops 命令分离 (Source 字段)
- [x] 系统提示词优化 (回复风格 + 调查优先)

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
