# Claude 开发指南

> 本文档供 Claude 在开发时参考，包含项目背景、开发规范和当前状态。
> 上下文切换时请先阅读本文档。

---

## 一、项目概述

### 1.1 AtlHyper 是什么

AtlHyper 是一个多集群 Kubernetes 监控和运维管理平台，采用 Master-Agent 架构。

### 1.2 核心组件

| 组件 | 目录 | 说明 |
|------|------|------|
| Master | `atlhyper_master/` | 中心管理端，接收数据、存储、告警、API |
| Agent (旧) | `atlhyper_agent/` | 集群代理，采集数据、执行指令 (待废弃) |
| Agent V2 | `atlhyper_agent_v2/` | 新版 Agent，重构中 |
| Web | `atlhyper_web/` | 前端界面 |
| Metrics | `atlhyper_metrics/` | 指标采集 (未来) |

### 1.3 技术栈

- 后端: Go
- 前端: Vue/React (待确认)
- 通信: HTTP/REST
- 容器: Kubernetes

---

## 二、当前工作状态

### 2.1 正在进行

**Agent V2 重构**

- 状态: 架构设计完成，待开发
- 设计文档: `docs/agent-v2-architecture.md`
- 任务进度: `docs/task-tracker.md`

### 2.2 架构要点

Agent V2 采用分层架构:

```
Scheduler 层 (调度)
    ↓
Service 层 (业务逻辑)
    ↓
Repository 层 (数据访问)
    ↓
SDK 层 + Gateway 层 (连接)
```

### 2.3 设计原则

1. **封装** - 隐藏各层级实现细节
2. **解耦** - 模块间通过接口交互
3. **扩展性** - 统一数据接口
4. **单向依赖** - 上层只依赖下层接口

---

## 三、目录结构

```
atlhyper/
├── atlhyper_agent/         # 旧 Agent (保留不动)
├── atlhyper_agent_v2/      # 新 Agent (开发中)
│   ├── model/              #   数据模型
│   ├── sdk/                #   K8s 连接层
│   ├── gateway/            #   Master 通信层
│   ├── repository/         #   仓库层
│   ├── service/            #   服务层
│   ├── scheduler/          #   调度层
│   └── config/             #   配置
│
├── atlhyper_master/        # Master
├── atlhyper_web/           # Web 前端
├── atlhyper_metrics/       # Metrics (未来)
│
├── cmd/                    # 入口文件
│   ├── atlhyper_agent/     #   旧 Agent 入口
│   ├── atlhyper_agent_v2/  #   新 Agent 入口
│   │   └── main.go
│   └── atlhyper_master/    #   Master 入口
│
├── model/                  # 共用模型
├── common/                 # 共用工具
├── docs/                   # 文档
│   ├── agent-v2-architecture.md  # Agent V2 架构设计
│   ├── claude-dev-guide.md       # 本文档
│   └── task-tracker.md           # 任务进度
│
├── go.mod                  # 根目录共用
└── go.sum
```

---

## 四、开发规范

### 4.1 分层规范

| 层级 | 职责 | 命名规范 |
|------|------|---------|
| Scheduler | 定时/循环触发、生命周期 | `xxx_scheduler.go` |
| Service | 业务逻辑处理 | `xxx_service.go` |
| Repository | 数据访问、模型转换 | `xxx_repository.go` |
| SDK | 底层客户端封装 | `interfaces.go`, `client.go` |
| Gateway | 外部通信封装 | `interfaces.go`, `xxx_gateway.go` |

### 4.2 接口规范

- 每层都要定义 `interfaces.go`
- 上层只依赖下层的接口，不依赖实现
- 构造函数返回接口类型，隐藏实现

### 4.3 模型规范

- 所有资源模型内嵌 `CommonMeta`
- CommonMeta 包含关联字段 (NodeName, PodName, OwnerKind...)
- K8s 原生类型只在 SDK 内部使用，不暴露给上层

### 4.4 命名规范

- 包名: 小写，单词
- 文件名: 小写，下划线分隔
- 接口名: 大写开头，描述能力 (如 `K8sClient`, `PodRepository`)
- 实现名: 小写开头，不导出 (如 `k8sClientImpl`, `podRepository`)

### 4.5 代码组织

- 不生成代码，只进行架构讨论时
- 代码放在对应层级的目录中
- 每个文件职责单一
- 测试文件与源文件同目录

---

## 五、关键设计决策

### 5.1 为什么重构 Agent

旧 Agent 问题:
- Event 匹配类型固定，不灵活
- Watcher 和 Snapshot 两套逻辑
- Agent 承担过多业务逻辑

新 Agent 原则:
- Agent 薄，Master 厚
- 统一使用定时 List 拉取
- Agent 只负责采集和执行，不做业务判断

### 5.2 为什么用 SDK 封装 K8s

- 上层完全不感知 K8s client-go
- 未来可替换其他平台
- 便于测试 (mock)

### 5.3 为什么分 Repository 和 Service

- Repository: 单一资源的 CRUD
- Service: 跨资源的业务编排
- 职责分离，便于测试和维护

### 5.4 通信方向

Agent 主动发起所有请求，Master 被动响应:
- POST /snapshot (推送数据)
- GET /commands (拉取指令)
- POST /result (上报结果)

---

## 六、参考文档

| 文档 | 路径 | 说明 |
|------|------|------|
| Agent V2 架构 | `docs/agent-v2-architecture.md` | 完整架构设计 |
| 任务进度 | `docs/task-tracker.md` | 开发任务跟踪 |
| Master 接口 | `atlhyper_master/repository/interfaces.go` | 参考设计 |

---

## 七、常用命令

```bash
# 项目根目录
cd /home/wuxiafeng/AtlHyper/GitHub/atlhyper

# 查看结构
ls -la

# 查看旧 Agent
ls -la atlhyper_agent/

# 查看 Master 接口设计 (参考)
cat atlhyper_master/repository/interfaces.go
```

---

## 八、注意事项

1. **不要修改旧 Agent** - `atlhyper_agent/` 保持不动
2. **新代码放 V2** - 所有新开发在 `atlhyper_agent_v2/`
3. **共用 go.mod** - 使用根目录的 go.mod
4. **入口在 cmd** - main.go 放在 `cmd/atlhyper_agent_v2/`
5. **及时更新任务** - 开发进度更新到 `docs/task-tracker.md`
6. **架构讨论不写代码** - 确认设计后再写代码
