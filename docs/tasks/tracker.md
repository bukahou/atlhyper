# 任务追踪

## 进行中

### DataHub 拆分为 Store + MQ

- **状态**: 进行中
- **描述**: 将 `datahub/` 拆分为独立的 `datahub/`(Store) + `mq/`(CommandBus)
- **目标**: 解耦存储逻辑和消息队列逻辑，支持独立替换后端

#### 子任务

- [x] 更新 CLAUDE.md (任务管理/Git/docs/架构图)
- [x] 创建 docs/tasks/ 结构
- [ ] 创建 mq/ 包结构 (interfaces.go, factory.go, memory/bus.go)
- [ ] 从 datahub/memory/hub.go 提取 CommandBus 代码到 mq/
- [ ] 重写 datahub/interfaces.go 为纯 Store 接口
- [ ] 重写 datahub/memory/ 为纯 MemoryStore
- [ ] 创建 datahub/factory.go 和 mq/factory.go
- [ ] 更新所有消费者依赖
- [ ] 更新 master.go 初始化流程
- [ ] go build + go vet 验证
- [ ] 本地 git commit

---

## 待办

(无)

---

## 已完成

(归档到 `archive/` 目录)
