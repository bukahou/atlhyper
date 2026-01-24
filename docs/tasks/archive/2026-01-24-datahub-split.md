# DataHub 拆分为 Store + MQ

- **完成日期**: 2026-01-24
- **提交**: 8c9b313

## 变更内容

将 `datahub/` 拆分为独立的 `datahub/`(Store) + `mq/`(CommandBus):

- `datahub/interfaces.go` → 纯 Store 接口 (快照/Agent/Event)
- `datahub/factory.go` → Store 工厂
- `datahub/memory/store.go` → MemoryStore 实现
- `mq/interfaces.go` → CommandBus 接口
- `mq/factory.go` → CommandBus 工厂
- `mq/memory/bus.go` → MemoryBus 实现

## 消费者依赖变更

| 消费者 | 变更前 | 变更后 |
|--------|--------|--------|
| Processor | datahub.DataHub | datahub.Store |
| QueryService | datahub.DataHub | datahub.Store + mq.CommandBus |
| CommandService | datahub.DataHub | mq.CommandBus |
| EventPersistService | datahub.DataHub | datahub.Store |
| AgentSDK | datahub.DataHub | mq.CommandBus |
| Gateway/OpsHandler | datahub.DataHub | mq.CommandBus |

## 同步更新

- CLAUDE.md: 任务管理规范 + Git 规范 + docs 引用 + 架构图更新
- docs/tasks/: 任务追踪结构建立
