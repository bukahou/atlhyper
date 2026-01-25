# Claude 开发指南

> 本文档为 CLAUDE.md 的补充，提供实际开发中的具体指导。
> 架构规范请参考项目根目录 `CLAUDE.md`。

---

## 一、项目概述

### 1.1 AtlHyper 是什么

AtlHyper 是一个多集群 Kubernetes 监控和运维管理平台，采用 Master-Agent 架构。

### 1.2 核心组件

| 组件 | 目录 | 说明 |
|------|------|------|
| Master V2 | `atlhyper_master_v2/` | 中心管理端 (已实施) |
| Agent V2 | `atlhyper_agent_v2/` | 集群代理 (已实施) |
| Web | `atlhyper_web/` | React/Next.js 前端 (已实施) |
| Metrics | `atlhyper_metrics/` | 指标采集 (未来) |
| 旧版备份 | `_backup_v1/` | V1 备份 (禁止修改) |

### 1.3 技术栈

- **后端:** Go
- **前端:** React / Next.js
- **存储:** SQLite
- **消息队列:** 内存 MQ (DataHub)
- **通信:** HTTP/REST
- **容器:** Kubernetes

---

## 二、当前工作状态

### 2.1 已完成

- Master V2: 分层架构已实施
- Agent V2: 分层架构已实施
- Web: i18n 国际化完成 (中文/日文)
- 文档结构整理完成

### 2.2 进行中

- AI 系统设计 (见 `docs/design/ai-system.md`)

---

## 三、Master V2 开发指南

> 架构图和层级规则请参考 `CLAUDE.md`

### 3.1 新增 API 端点

1. `gateway/handler/` 新增 Handler
2. Handler 中只调用 Query (读) 或 Service (写)
3. `gateway/routes.go` 注册路由
4. 如需新数据源: 在 query/ 或 service/ 新增方法

### 3.2 新增业务功能

1. **读取功能:** `query/` 新增接口方法和实现
2. **写入功能:** `service/` 新增接口方法和实现
3. **Gateway 调用:** `gateway/handler/` 通过接口调用

### 3.3 新增数据存储

```
实时数据 (易失):
  → datahub/interfaces.go 新增接口方法
  → datahub/memory/ 实现

持久化数据:
  → database/repository/ 新增 Repository 接口
  → database/sqlite/impl/ 实现
```

### 3.4 初始化顺序

新增模块时，依赖注入必须按此顺序 (`master.go`):

```
1. DataHub       → 实时数据中心
2. Database      → 持久化数据库
3. Processor     → 数据处理 (依赖 DataHub)
4. Query         → 查询层 (依赖 DataHub, Database)
5. Service       → 服务层 (依赖 DataHub, Database)
6. AgentSDK      → Agent 通信 (依赖 Processor, DataHub)
7. Gateway       → Web API (依赖 Query, Service)
```

---

## 四、Agent V2 开发指南

### 4.1 架构分层

```
Scheduler 层 (定时调度)
    ↓
Service 层 (业务编排)
    ↓
Repository 层 (数据访问)
    ↓
SDK 层 (K8s 客户端) + Gateway 层 (Master 通信)
```

### 4.2 设计原则

- **Agent 薄，Master 厚:** Agent 只负责采集和执行
- **定时 List 拉取:** 统一使用定时 List，不用 Watch
- **无业务判断:** Agent 不做业务逻辑判断

### 4.3 通信方向

Agent 主动发起所有请求:
- `POST /snapshot` - 推送采集数据
- `GET /commands` - 拉取待执行指令
- `POST /result` - 上报执行结果

---

## 五、开发规范

### 5.1 接口规范

- 每个模块定义 `interfaces.go` 暴露接口
- 上层只依赖下层接口，不依赖实现
- 工厂函数返回接口类型: `func NewXxx(...) XxxInterface`
- 实现结构体不导出: `type xxxImpl struct{}`

### 5.2 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 包名 | 小写单词 | `service`, `repository` |
| 文件名 | 小写下划线 | `command_service.go` |
| 接口 | 大写开头 | `CommandService` |
| 实现 | 小写开头 | `commandServiceImpl` |
| 工厂函数 | `New` + 接口名 | `NewCommandService` |
| 接口文件 | 固定名称 | `interfaces.go` |

### 5.3 模型规范 (Agent)

- 所有资源模型内嵌 `CommonMeta`
- K8s 原生类型只在 SDK 内部使用，不暴露给上层
- 共用模型放在 `model_v2/`

### 5.4 代码组织

- 每个文件职责单一
- 测试文件与源文件同目录
- 不生成代码，先确认架构设计

---

## 六、关键设计决策

### 6.1 为什么用 DataHub

- 统一的实时数据中心
- 内存 MQ 实现指令队列
- 快照缓存避免频繁 DB 查询
- 各层通过 DataHub 接口解耦

### 6.2 为什么分 Query 和 Service

- Query: 只读操作，无副作用
- Service: 写入操作，有副作用
- 分离读写路径，便于扩展和优化

### 6.3 为什么用 SDK 封装 K8s

- 上层完全不感知 K8s client-go
- 未来可替换其他平台
- 便于测试 (mock)

---

## 七、参考文档

| 文档 | 路径 | 说明 |
|------|------|------|
| 项目指南 | `CLAUDE.md` | **架构规范和禁止行为** |
| 文档索引 | `docs/README.md` | 文档导航 |
| Master V2 架构 | `docs/architecture/master-v2.md` | Master 架构设计 |
| Agent V2 架构 | `docs/architecture/agent-v2.md` | Agent 架构设计 |
| AI 系统设计 | `docs/design/ai-system.md` | AI 功能设计方案 |
| 任务追踪 | `docs/tasks/tracker.md` | 开发任务进度 |

---

## 八、注意事项

1. **禁止修改 `_backup_v1/`** - 旧版本备份
2. **架构规范优先** - 参考 CLAUDE.md 中的禁止行为表
3. **共用 `go.mod`** - 使用根目录的 go.mod
4. **依赖注入** - 所有模块通过接口交互
5. **国际化** - Web 前端支持中文/日文
6. **先设计后编码** - 确认架构方案后再写代码
