# Agent V2 开发任务进度

> 最后更新: 2026-01-17

---

## 总体进度

| 阶段 | 状态 | 说明 |
|------|------|------|
| 架构设计 | ✅ 完成 | 见 `agent-v2-architecture.md` |
| 目录搭建 | ✅ 完成 | 项目结构已创建 |
| Model 层 | ✅ 完成 | 5 个文件，全部资源模型 |
| SDK 层 | ✅ 完成 | K8s 客户端封装 |
| Gateway 层 | ✅ 完成 | Master 通信封装 |
| Repository 层 | ✅ 完成 | 17 个资源仓库 |
| Service 层 | ✅ 完成 | Snapshot, Command, Resource, Operation |
| Scheduler 层 | ✅ 完成 | 统一调度器 |
| Config | ✅ 完成 | 三文件结构 (types/defaults/loader) |
| 入口和启动 | ✅ 完成 | main.go + agent.go |
| 代码注释 | ✅ 完成 | 所有文件添加中文注释 |
| 集成测试 | ⏳ 待开始 | 整体测试 |
| 部署 | ⏳ 待开始 | K8s 部署 |

状态说明: ✅ 完成 | 🔄 进行中 | ⏳ 待开始 | ❌ 阻塞

---

## 已完成文件清单

### Model 层 (5 文件)
- `model/common.go` - CommonMeta, ResourceRef, ResourceList, ResourceRequirements
- `model/resource.go` - Pod, Node, Deployment, Service, Event 等 16 种资源
- `model/snapshot.go` - ClusterSnapshot, ClusterSummary
- `model/command.go` - Command, Result, DynamicRequest/Response
- `model/options.go` - ListOptions, LogOptions, DeleteOptions 等

### SDK 层 (2 文件)
- `sdk/interfaces.go` - K8sClient 接口定义
- `sdk/impl.go` - client-go 实现 (unexported)
- `sdk/client.go` - NewK8sClient 构造函数

### Gateway 层 (2 文件)
- `gateway/interfaces.go` - MasterGateway 接口
- `gateway/master_gateway.go` - HTTP + Gzip 实现

### Repository 层 (17 文件)
- `repository/interfaces.go` - 所有仓库接口
- `repository/converter.go` - K8s → model 类型转换
- `repository/pod_repository.go`
- `repository/node_repository.go`
- `repository/deployment_repository.go`
- `repository/statefulset_repository.go`
- `repository/daemonset_repository.go`
- `repository/replicaset_repository.go`
- `repository/service_repository.go`
- `repository/ingress_repository.go`
- `repository/configmap_repository.go`
- `repository/secret_repository.go`
- `repository/namespace_repository.go`
- `repository/event_repository.go`
- `repository/job_repository.go`
- `repository/cronjob_repository.go`
- `repository/pv_repository.go`
- `repository/pvc_repository.go`
- `repository/generic_repository.go`

### Service 层 (4 文件)
- `service/interfaces.go` - SnapshotService, CommandService, ResourceService, OperationService
- `service/snapshot_service.go` - 并发采集 16 种资源
- `service/command_service.go` - scale, restart, delete, get_logs, dynamic
- `service/resource_service.go` - 单资源查询
- `service/operation_service.go` - 资源变更操作

### Scheduler 层 (2 文件)
- `scheduler/interfaces.go` - Scheduler 接口
- `scheduler/scheduler.go` - 快照循环 + 指令轮询 + 心跳

### Config 层 (3 文件)
- `config/types.go` - 配置结构体定义 (AppConfig, SchedulerConfig, TimeoutConfig 等)
- `config/defaults.go` - 默认值定义 (所有环境变量集中管理)
- `config/loader.go` - 配置加载函数 (LoadConfig)

### 入口 (2 文件)
- `atlhyper_agent_v2/agent.go` - Agent 结构体 + 依赖注入
- `cmd/atlhyper_agent_v2/main.go` - 启动入口

---

## 编译状态

```
✅ go build ./cmd/atlhyper_agent_v2/... 通过
📦 bin/atlhyper_agent_v2 (65MB)
```

---

## 环境变量配置

> 所有环境变量统一使用 `AGENT_` 前缀，定义见 `config/defaults.go`

### 基础配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `AGENT_CLUSTER_ID` | `default-cluster` | 集群唯一标识 |
| `AGENT_MASTER_URL` | `http://localhost:8080` | Master 服务地址 |
| `AGENT_KUBECONFIG` | (空=InCluster) | kubeconfig 文件路径 |

### 调度器配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `AGENT_SNAPSHOT_INTERVAL` | `30s` | 快照采集间隔 |
| `AGENT_COMMAND_POLL_INTERVAL` | `1s` | 指令轮询间隔 |
| `AGENT_HEARTBEAT_INTERVAL` | `15s` | 心跳发送间隔 |

### 超时配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `AGENT_TIMEOUT_HTTP_CLIENT` | `60s` | HTTP 客户端超时 (含长轮询) |
| `AGENT_TIMEOUT_SNAPSHOT_COLLECT` | `30s` | 快照采集操作超时 |
| `AGENT_TIMEOUT_COMMAND_POLL` | `60s` | 指令轮询操作超时 |
| `AGENT_TIMEOUT_HEARTBEAT` | `10s` | 心跳操作超时 |

---

## 阶段十一: 集成测试

### 任务列表

| 任务 | 状态 | 完成时间 | 备注 |
|------|------|---------|------|
| 本地 K8s 连接测试 | ⏳ | - | |
| 快照构建测试 | ⏳ | - | |
| Master 通信测试 | ⏳ | - | 需要 Mock 或真实 Master |
| 指令执行测试 | ⏳ | - | |

---

## 阶段十二: 部署

### 任务列表

| 任务 | 状态 | 完成时间 | 备注 |
|------|------|---------|------|
| Dockerfile | ⏳ | - | |
| K8s 部署 YAML | ⏳ | - | ServiceAccount, RBAC |
| 与旧 Agent 并行测试 | ⏳ | - | |
| 切换上线 | ⏳ | - | |

---

## 待确认问题

| 问题 | 状态 | 决定 |
|------|------|------|
| 新 Agent 项目名称 | ✅ 已确认 | `atlhyper_agent_v2` |
| model 包位置 | ✅ 已确认 | agent 内部 `atlhyper_agent_v2/model` |
| 是否需要增量推送 | ⏳ 待确认 | 当前全量推送 |
| Event 推送频率 | ⏳ 待确认 | 当前随快照 30s |

---

## 变更记录

| 日期 | 变更内容 |
|------|---------|
| 2026-01-18 | 前端国际化任务启动，详见 `docs/i18n-task-tracker.md` |
| 2025-01-17 | 创建任务跟踪文档 |
| 2025-01-17 | 架构设计完成 |
| 2025-01-17 | 基础框架实现完成 (Model → Scheduler 全部完成) |
| 2025-01-17 | 编译通过，生成二进制 65MB |
| 2025-01-17 | 重构 main.go，抽取 agent.go 启动器 |
| 2026-01-17 | 为所有代码文件添加中文注释 |
| 2026-01-17 | 重构 config 为三文件结构 (types/defaults/loader) |
| 2026-01-17 | 集中管理所有配置变量，统一 AGENT_ 前缀 |
| 2026-01-17 | 新增超时配置 (HTTP、快照、轮询、心跳) |
