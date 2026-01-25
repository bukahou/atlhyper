# Agent V2 架构设计文档

> 状态: ✅ 已实施完成
> 创建时间: 2025-01
> 完成时间: 2025-01-17
> 备注: 已按此设计完成开发，归档备份

---

## 一、设计背景

### 1.1 参考架构

参考 Elastic APM 架构模式:

```
Elastic 架构                        AtlHyper 架构
─────────────                       ─────────────
APM Agent (多种)                    Collector (未来扩展)
      │                                   │
      ▼                                   ▼
APM Server (汇聚)                   Agent (汇聚 + K8s 操作)
      │                                   │
      ▼                                   ▼
Elasticsearch (存储/查询)           Master (存储/查询/告警)
      │                                   │
      ▼                                   ▼
Kibana (可视化)                     Web (可视化 + 操作)
```

### 1.2 设计原则

1. **封装** - 隐藏各层级的实现细节
2. **解耦** - 模块间通过接口交互，不依赖具体实现
3. **扩展性** - 统一数据接口，方便新增功能
4. **单向依赖** - 上层只依赖下层接口，下层不知道上层存在

### 1.3 当前阶段

- 暂不考虑第三方模块接入 (如 Metrics Collector)
- 专注 Agent 核心功能: K8s 数据采集 + 指令执行

---

## 二、整体架构

### 2.1 系统架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Kubernetes 集群                                    │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                            Agent                                     │   │
│   │                                                                      │   │
│   │   职责:                                                              │   │
│   │     1. 定时拉取 K8s 资源，推送到 Master                              │   │
│   │     2. 长轮询接收 Master 指令，执行 K8s 操作                         │   │
│   │                                                                      │   │
│   │   不做:                                                              │   │
│   │     × 异常检测 (由 Master 处理)                                      │   │
│   │     × 数据过滤 (由 Master 处理)                                      │   │
│   │     × 告警判断 (由 Master 处理)                                      │   │
│   │                                                                      │   │
│   └──────────────────────────────────────────────────────────────────────┘   │
│                              │                                              │
└──────────────────────────────┼──────────────────────────────────────────────┘
                               │
                               │  Agent 主动发起所有请求
                               │  • POST /snapshot  (推送数据)
                               │  • GET  /commands  (拉取指令)
                               │  • POST /result    (上报结果)
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Master                                          │
│                                                                             │
│   • 接收 Agent 推送的 ClusterSnapshot                                       │
│   • 存储到内存 (实时查询)                                                   │
│   • Event 处理 → 告警                                                      │
│   • 下发指令给 Agent 执行                                                   │
│   • 提供 API 给 Web                                                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                               Web                                            │
│                                                                             │
│   • 仪表盘 (集群概览)                                                       │
│   • 资源列表 (Pod/Node/Service/Deploy...)                                   │
│   • 资源详情                                                                │
│   • 事件日志                                                                │
│   • 操作中心 (扩缩容、重启等)                                               │
│   • (未来) AI 诊断助手                                                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 通信方向

```
Agent 主动发起所有请求，Master 被动响应:

Collector ──────────────────→ Agent        (未来)
           POST /ingest         │
                                │
                                │ Agent 主动
                                │
                 ┌──────────────┼──────────────┐
                 │              │              │
                 ▼              ▼              ▼
           POST /snapshot  GET /commands  POST /result
           (推送数据)      (拉取指令)     (上报结果)
                 │              │              │
                 └──────────────┼──────────────┘
                                │
                                ▼
                             Master
                                │
                                │ Master 被动响应
                                ▼
                              Web
```

---

## 三、Agent 分层架构

### 3.1 分层概览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│                          ┌──────────────────┐                               │
│                          │    Scheduler     │  调度层 (顶层)                │
│                          │                  │                               │
│                          │ • PushScheduler  │  定时/循环触发                │
│                          │ • CommandScheduler│ 生命周期管理                 │
│                          └────────┬─────────┘                               │
│                                   │                                         │
│                                   │ 依赖接口                                │
│                                   ▼                                         │
│                          ┌──────────────────┐                               │
│                          │    Service       │  服务层                       │
│                          │                  │                               │
│                          │ • SnapshotService│  业务逻辑处理                 │
│                          │ • ExecuteService │  数据组装转换                 │
│                          └────────┬─────────┘                               │
│                                   │                                         │
│                                   │ 依赖接口                                │
│                                   ▼                                         │
│                          ┌──────────────────┐                               │
│                          │   Repository     │  仓库层                       │
│                          │                  │                               │
│                          │ • PodRepository  │  数据访问封装                 │
│                          │ • NodeRepository │  模型转换                     │
│                          │ • ...            │                               │
│                          └────────┬─────────┘                               │
│                                   │                                         │
│                                   │ 依赖接口                                │
│                                   ▼                                         │
│                     ┌─────────────┴─────────────┐                           │
│                     │                           │                           │
│                     ▼                           ▼                           │
│            ┌──────────────┐            ┌──────────────┐                     │
│            │     SDK      │            │   Gateway    │  连接层             │
│            │              │            │              │                     │
│            │  K8sClient   │            │ MasterGateway│                     │
│            └──────────────┘            └──────────────┘                     │
│                     │                           │                           │
│                     ▼                           ▼                           │
│               K8s API Server               Master                           │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 层级职责

| 层级 | 职责 | 依赖 | 暴露 |
|------|------|------|------|
| **Scheduler** | 定时/循环触发、生命周期管理 | Service + Gateway 接口 | Start/Stop |
| **Service** | 业务逻辑处理、数据组装 | Repository 接口 | 服务接口 |
| **Repository** | 数据访问、模型转换 | SDK 接口 | 仓库接口 |
| **SDK** | K8s 客户端封装 | client-go (内部) | K8sClient 接口 |
| **Gateway** | Master 通信封装 | net/http (内部) | MasterGateway 接口 |

### 3.3 依赖规则

- 上层只依赖下层的**接口**，不依赖实现
- 下层不知道上层的存在
- 同层之间不互相依赖
- 所有层使用统一的 **model** 包

---

## 四、各层详细设计

### 4.1 SDK 层 (K8s 连接层)

**职责:**
- 封装 K8s client-go
- 只暴露接口，隐藏实现
- 连接管理

**对外暴露:**

```
K8sClient (接口)
│
│  资源读取
├── ListPods(namespace, opts)         → []k8s.Pod
├── GetPod(namespace, name)           → k8s.Pod
├── ListNodes(opts)                   → []k8s.Node
├── GetNode(name)                     → k8s.Node
├── ListDeployments(namespace, opts)  → []k8s.Deployment
├── ListEvents(namespace, opts)       → []k8s.Event
├── ListServices(namespace, opts)     → []k8s.Service
├── ... (其他资源)
│
│  资源写入
├── Create(obj)                       → error
├── Update(obj)                       → error
├── Delete(gvk, namespace, name)      → error
├── Patch(gvk, namespace, name, patch)→ error
│
│  特殊操作
├── GetPodLogs(namespace, name, opts) → string
├── UpdateScale(namespace, name, replicas) → error
└── Dynamic(request)                  → response  (AI 动态调用)
```

**内部隐藏:**
- client-go 的 kubernetes.Interface
- dynamic.Interface
- 所有 K8s 原生类型

### 4.2 Gateway 层 (Master 通信层)

**职责:**
- 封装与 Master 的 HTTP 通信
- 只暴露接口，隐藏 HTTP 细节

**对外暴露:**

```
MasterGateway (接口)
│
├── PushSnapshot(snapshot)   → error     // Agent POST 数据
├── PollCommands()           → []Command // Agent GET 指令 (长轮询)
└── ReportResult(id, result) → error     // Agent POST 结果
```

**内部隐藏:**
- HTTP 请求/响应处理
- JSON 序列化
- Gzip 压缩
- 错误重试

### 4.3 Repository 层 (仓库层)

**职责:**
- 定义数据访问接口
- 封装 SDK 调用
- K8s 原生类型 → 内部 model 类型转换

**接口定义:**

```
PodRepository (接口)
├── List(namespace, opts)    → []model.Pod
├── Get(namespace, name)     → model.Pod
├── Delete(namespace, name)  → error
└── GetLogs(namespace, name, opts) → string

NodeRepository (接口)
├── List(opts)               → []model.Node
└── Get(name)                → model.Node

DeploymentRepository (接口)
├── List(namespace, opts)    → []model.Deployment
├── Get(namespace, name)     → model.Deployment
├── Scale(namespace, name, replicas) → error
└── Restart(namespace, name) → error

EventRepository (接口)
└── List(namespace, opts)    → []model.Event

ServiceRepository (接口)
├── List(namespace, opts)    → []model.Service
└── Get(namespace, name)     → model.Service

GenericRepository (接口)
├── Create(kind, obj)        → error
├── Update(kind, obj)        → error
├── Delete(kind, ns, name)   → error
└── Execute(request)         → response  (AI 动态调用)

... (其他资源仓库)
```

### 4.4 Service 层 (服务层)

**职责:**
- 业务逻辑处理
- 数据组装和转换
- 多仓库协调

**接口定义:**

```
SnapshotService (接口)
└── BuildSnapshot()  → model.ClusterSnapshot

ExecuteService (接口)
└── Execute(command) → model.Result
```

**SnapshotService 职责:**
- 并行调用各 Repository 获取数据
- 组装 ClusterSnapshot
- 计算 Summary 统计
- 返回完整快照

**ExecuteService 职责:**
- 解析指令类型
- 调用对应 Repository 方法
- 封装返回结果

**支持的指令类型:**
- scale - 扩缩容
- restart - 重启 Deployment
- delete_pod - 删除 Pod
- get_logs - 获取日志
- dynamic - 动态 API 调用 (AI)

### 4.5 Scheduler 层 (调度层)

**职责:**
- 定时/循环触发
- 调用 Service 处理
- 调用 Gateway 通信
- 生命周期管理 (Start/Stop)

**组件:**

```
PushScheduler
├── Start(ctx) - 启动推送循环
└── Stop()     - 停止

流程:
1. 定时触发 (30s)
2. 调用 SnapshotService.BuildSnapshot()
3. 调用 Gateway.PushSnapshot()

CommandScheduler
├── Start(ctx) - 启动控制循环
└── Stop()     - 停止

流程:
1. 调用 Gateway.PollCommands() (长轮询)
2. 调用 ExecuteService.Execute()
3. 调用 Gateway.ReportResult()
4. 循环
```

---

## 五、数据模型设计

### 5.1 公共字段 (关联查询基础)

```
CommonMeta
├── UID           string    // 全局唯一 ID
├── Name          string    // 资源名称
├── Namespace     string    // 命名空间
├── Kind          string    // 资源类型
│
│  关联字段
├── NodeName      string    // 所在节点 (Pod, Event)
├── PodName       string    // 关联的 Pod (Event)
├── OwnerKind     string    // 所属资源类型
├── OwnerName     string    // 所属资源名称
│
│  标签
├── Labels        map[string]string
├── Annotations   map[string]string
│
│  时间
├── CreatedAt     time.Time
└── UpdatedAt     time.Time
```

### 5.2 资源模型

所有资源模型都内嵌 CommonMeta，支持关联查询:

- model.Pod
- model.Node
- model.Deployment
- model.Service
- model.Event
- model.ConfigMap
- model.Ingress
- model.StatefulSet
- model.DaemonSet
- model.Namespace
- ...

### 5.3 ClusterSnapshot

```
ClusterSnapshot
├── ClusterID     string
├── Version       uint64
├── FetchedAt     time.Time
│
│  资源数据
├── Pods          []Pod
├── Nodes         []Node
├── Deployments   []Deployment
├── Services      []Service
├── Events        []Event
├── ConfigMaps    []ConfigMap
├── Ingresses     []Ingress
├── Namespaces    []Namespace
├── ...
│
│  统计摘要
└── Summary       ClusterSummary
```

### 5.4 指令模型

```
Command
├── ID            string
├── ClusterID     string
├── Action        string    // scale, restart, delete_pod, get_logs, dynamic
├── Namespace     string
├── Name          string
├── Params        map[string]any
└── CreatedAt     time.Time

Result
├── CommandID     string
├── Success       bool
├── Data          any       // 返回数据 (如日志内容)
├── Error         string
└── ExecutedAt    time.Time
```

---

## 六、目录结构

### 6.1 项目位置

新 Agent 与现有 Agent 并列:

```
atlhyper/
├── atlhyper_agent/         # 旧 Agent (保留不动)
├── atlhyper_agent_v2/      # 新 Agent
├── atlhyper_master/
├── atlhyper_web/
├── cmd/
│   ├── atlhyper_agent/     # 旧 Agent 入口
│   ├── atlhyper_agent_v2/  # 新 Agent 入口
│   │   └── main.go
│   └── atlhyper_master/
├── model/                  # 共用模型 (可选)
├── common/                 # 共用工具
├── docs/
├── go.mod                  # 根目录共用
└── go.sum
```

### 6.2 新 Agent 内部结构

```
atlhyper_agent_v2/
│
├── model/                          # 内部数据模型
│   ├── common.go                   #   公共字段 (CommonMeta)
│   ├── resource.go                 #   Pod, Node, Deployment...
│   ├── snapshot.go                 #   ClusterSnapshot
│   ├── command.go                  #   Command, Result
│   └── options.go                  #   ListOptions, LogOptions
│
├── sdk/                            # SDK 层 (K8s 连接)
│   ├── interfaces.go               #   K8sClient 接口 (导出)
│   ├── client.go                   #   NewK8sClient 构造 (导出)
│   └── internal/                   #   内部实现 (不导出)
│       └── impl.go                 #     client-go 封装
│
├── gateway/                        # Gateway 层 (Master 通信)
│   ├── interfaces.go               #   MasterGateway 接口
│   └── master_gateway.go           #   实现
│
├── repository/                     # Repository 层 (数据访问)
│   ├── interfaces.go               #   所有仓库接口定义
│   ├── pod_repository.go           #   PodRepository 实现
│   ├── node_repository.go          #   NodeRepository 实现
│   ├── deployment_repository.go    #   DeploymentRepository 实现
│   ├── event_repository.go         #   EventRepository 实现
│   ├── service_repository.go       #   ServiceRepository 实现
│   ├── generic_repository.go       #   GenericRepository 实现
│   ├── converter.go                #   K8s → model 转换
│   └── ...
│
├── service/                        # Service 层 (业务逻辑)
│   ├── interfaces.go               #   服务接口定义
│   ├── snapshot_service.go         #   SnapshotService 实现
│   └── execute_service.go          #   ExecuteService 实现
│
├── scheduler/                      # Scheduler 层 (调度)
│   ├── push_scheduler.go           #   推送调度器
│   └── command_scheduler.go        #   指令调度器
│
└── config/                         # 配置
    └── config.go
```

### 6.3 入口文件

```
cmd/atlhyper_agent_v2/main.go

启动流程:
1. 加载配置
2. 创建 SDK 层 (K8sClient)
3. 创建 Gateway 层 (MasterGateway)
4. 创建 Repository 层 (注入 SDK)
5. 创建 Service 层 (注入 Repository)
6. 创建 Scheduler 层 (注入 Service + Gateway)
7. 启动 Scheduler
8. 等待退出信号
```

---

## 七、关联查询设计

### 7.1 关联字段

通过 CommonMeta 中的字段实现资源关联:

| 字段 | 用途 | 示例 |
|------|------|------|
| NodeName | Node ↔ Pod | 查找某节点上的所有 Pod |
| PodName | Pod ↔ Event | 查找某 Pod 的所有事件 |
| OwnerKind + OwnerName | Pod ↔ Deployment | 查找某 Deployment 的所有 Pod |
| Labels + Selector | Deployment ↔ Pod ↔ Service | 通过标签关联 |

### 7.2 关联关系图

```
Node
  │
  │ NodeName
  ├──────────→ Pod ←──────────── Event (InvolvedObject.Kind=Pod)
  │              │                  │
  │              │ OwnerKind        │ InvolvedObject
  │              │ OwnerName        │
  │              ▼                  │
  │          ReplicaSet ←───────────┤
  │              │                  │
  │              │ OwnerKind        │
  │              │ OwnerName        │
  │              ▼                  │
  │          Deployment ←───────────┘
  │              │
  │              │ Selector
  │              ▼
  │          Service (通过 Selector 关联 Pod)
  │              │
  │              │ backend
  │              ▼
  │          Ingress
  │
  └──────────→ Event (InvolvedObject.Kind=Node)
```

---

## 八、扩展性设计

### 8.1 新增资源类型

1. model/ 新增资源结构 (内嵌 CommonMeta)
2. SDK 接口新增 List/Get 方法
3. SDK 内部实现
4. Repository 新增接口和实现
5. SnapshotService 纳入快照

上层代码无需修改。

### 8.2 替换数据源

1. 新增适配器实现 K8sClient 接口
2. 配置切换适配器

Repository、Service、Scheduler 完全不变。

### 8.3 新增指令类型

1. model/command.go 新增指令定义
2. Repository 新增对应方法
3. ExecuteService 新增 case 分发

Gateway、Scheduler 不变。

### 8.4 未来扩展: 第三方模块接入

预留扩展点:

```
atlhyper_agent_v2/
├── receiver/                       # 接收层 (未来)
│   └── metrics_receiver.go         #   接收 Metrics
│
├── cache/                          # 缓存层 (未来)
│   └── metrics_cache.go            #   Metrics 缓存
```

---

## 九、待定事项

- [ ] 新 Agent 项目名称确认 (atlhyper_agent_v2?)
- [ ] 是否需要支持增量推送
- [ ] Event 推送频率是否需要单独设置
- [ ] 是否需要支持资源过滤 (只推送部分 namespace)
- [ ] model 包是放在 agent 内部还是根目录共用
- [ ] 旧 Agent 的迁移计划

---

## 十、参考

- Elastic APM 架构
- Master 现有 interfaces.go 设计
- 当前 Agent 架构分析
