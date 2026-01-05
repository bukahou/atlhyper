# AtlHyper 开发规范指南

> 本文档定义了 AtlHyper 项目的核心设计理念和开发规范，所有开发者必须遵守。

---

## 目录

1. [项目概览](#项目概览)
2. [核心设计理念](#核心设计理念)
3. [架构规范](#架构规范)
4. [数据存储规范](#数据存储规范)
5. [接口调用规范](#接口调用规范)
6. [数据流规范](#数据流规范)
7. [模块详解](#模块详解)

---

## 项目概览

**AtlHyper** 是一个轻量级 Kubernetes 集群可观测与控制平台，采用 **Master-Agent** 分布式架构。

### 核心定位
- 实时监控 Node、Pod、Deployment 等资源
- 异常检测与告警通知
- AI 智能诊断与根因分析
- 集群操作控制

### 技术栈

**后端：**
- Go、Gin、Kubernetes Client
- SQLite（持久化）
- Gemini LLM（AI 诊断）

**前端（atlhyper_web）：**
- Next.js 16 + React 19
- TypeScript 5
- Tailwind CSS 4
- Zustand（状态管理）

---

## 核心设计理念

### 1. Agent 是集群中心

**Agent 是每个 Kubernetes 集群的唯一数据出口。**

- Agent 部署在 Kubernetes 集群内部
- 集群内所有数据（Pod、Node、Service、Metrics 等）只能通过 Agent 对外传输
- 集群内所有扩展（如 Metrics）必须上报给 Agent
- 保证集群内数据只在集群内流通，确保数据安全性

```
┌───────────────────────────────────────────────────────────────────────────┐
│                              四层架构总览                                  │
│                                                                           │
│   Layer 1: WebUI                                                          │
│   ┌─────────────────────────────────────────────────────────────────────┐ │
│   │  atlhyper_web (Next.js) - 负责 Master 数据展示和交互                 │ │
│   └───────────────────────────────┬─────────────────────────────────────┘ │
│                                   │ HTTP API                              │
│   Layer 2: Master                 ▼                                       │
│   ┌─────────────────────────────────────────────────────────────────────┐ │
│   │  atlhyper_master (Go) - 数据存储、API、告警、命令下发                 │ │
│   └───────────────────────────────▲─────────────────────────────────────┘ │
│                                   │ HTTP Push (gzip)                      │
└───────────────────────────────────┼───────────────────────────────────────┘
                                    │
┌───────────────────────────────────┼───────────────────────────────────────┐
│                     Kubernetes 集群                                        │
│                                   │                                       │
│   Layer 3: Agent                  │                                       │
│   ┌───────────────────────────────┴─────────────────────────────────────┐ │
│   │  atlhyper_agent (Go) - 集群数据采集、推送、命令执行                   │ │
│   │                                                                     │ │
│   │   ┌─────────────────────────────────────────────────────────────┐   │ │
│   │   │  internal/ (内部模块)                                        │   │ │
│   │   │   ├── readonly/ (只读查询 Pod/Node/Service)                  │   │ │
│   │   │   └── watcher/  (事件监控)                                   │   │ │
│   │   │              ↓                                               │   │ │
│   │   │      Kubernetes API Server                                   │   │ │
│   │   └─────────────────────────────────────────────────────────────┘   │ │
│   │                              ▲                                      │ │
│   └──────────────────────────────┼──────────────────────────────────────┘ │
│                                  │ HTTP Push                              │
│   Layer 4: Metrics               │                                       │
│   ┌──────────────────────────────┴──────────────────────────────────────┐ │
│   │  atlhyper_metrics (DaemonSet) - 节点指标采集                         │ │
│   │  (CPU / Memory / Disk / Network / Temperature)                      │ │
│   └─────────────────────────────────────────────────────────────────────┘ │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘
```

**四层架构说明：**
| 层级 | 模块 | 说明 |
|------|------|------|
| Layer 1 | WebUI | 负责 Master 数据展示和交互 |
| Layer 2 | Master | 数据存储、API 服务、告警通知、命令下发 |
| Layer 3 | Agent | 集群数据采集、推送、命令执行（含 readonly/watcher 内部模块） |
| Layer 4 | Metrics | Agent 的下位组件，节点系统指标采集 |

### 2. Master 是外部统一管理中心

**Master 部署在集群外部，可对应多个集群（多个 Agent）。**

- Master 集中管理所有 Agent 的数据
- 通过 HTTP 接收 Agent 推送的数据
- 提供统一的 API 接口给 UI 和 AIService
- 不主动拉取集群数据，只被动接收推送

### 3. 推送优于拉取（Push over Pull）

**所有外部扩展必须使用推送模式，而非主动拉取。**

- Agent → Master：推送模式
- Metrics → Agent：推送模式
- Adapter → Master：推送模式

**原因：**
- 保证没有下层应用也不会出错导致崩溃
- 从外部无法入侵到集群内部
- 解耦各模块，降低依赖

### 4. 数据唯一性保证

**数据存储必须保证数据唯一性，通过层层唯一 Key 进行数据清理。**

#### Agent Store（agent_store）
- 存储结构：`map[string]NodeMetricsSnapshot`
- 唯一 Key：`NodeName`（节点名）
- 每个节点只保留最新一条快照
- TTL 清理：默认 10 分钟过期

```go
type Store struct {
    mu   sync.RWMutex
    data map[string]NodeMetricsSnapshot  // key = NodeName
}
```

#### Master Store（master_store）
- 存储结构：`[]EnvelopeRecord`
- 唯一 Key：`ClusterID + Source`
- 对于快照类数据（pod_list、node_list 等），使用 `ReplaceLatest()` 原子替换
- TTL 清理：默认 24 小时，metrics_snapshot 15 分钟

```go
type EnvelopeRecord struct {
    ClusterID   string          // 集群 ID
    Source      string          // 数据源标识
    EnqueuedAt  time.Time       // 入池时间
    Payload     json.RawMessage // 原始载荷
}
```

### 5. 解耦与接口规范

**调用链路必须统一，入口统一，出口统一。**

---

## 架构规范

### 核心四层架构

```
┌───────────────────────────────────────────────────────────────┐
│  Layer 1: WebUI     │  atlhyper_web      │  数据展示和交互    │
├───────────────────────────────────────────────────────────────┤
│  Layer 2: Master    │  atlhyper_master   │  数据存储、API     │
├───────────────────────────────────────────────────────────────┤
│  Layer 3: Agent     │  atlhyper_agent    │  集群数据采集      │
├───────────────────────────────────────────────────────────────┤
│  Layer 4: Metrics   │  atlhyper_metrics  │  节点指标采集      │
└───────────────────────────────────────────────────────────────┘
```

**数据流向：** `WebUI ←── Master ←── Agent ←── Metrics`

### 模块层次

```
AtlHyper/
├── atlhyper_web          # Layer 1: 前端（外部部署）
├── atlhyper_master       # Layer 2: 主控中心（外部部署）
├── atlhyper_agent        # Layer 3: 集群内代理
├── atlhyper_metrics      # Layer 4: 节点级指标采集器（DaemonSet）
├── model/                # 共享数据模型（统一管理）
├── config/               # 配置文件
└── utils/                # 工具库
```

### 模块职责

| 层级 | 模块 | 职责 | 部署位置 |
|------|------|------|----------|
| Layer 1 | WebUI | Master 数据展示和交互 | 外部 |
| Layer 2 | Master | 数据集中存储、API 服务、告警通知、命令下发 | 外部 |
| Layer 3 | Agent | 集群数据采集、推送、命令执行 | 集群内 |
| Layer 4 | Metrics | 节点系统指标采集 | 集群内 (DaemonSet) |

### 扩展模块（设计阶段）

> **注意**：以下模块尚在设计阶段，可能随时被删除或修改，暂不纳入正式架构。

| 模块 | 状态 | 说明 |
|------|------|------|
| atlhyper_adapter | 设计中 | 第三方监控适配器 |

---

## 数据模型规范

### 统一管理原则

**所有数据模型必须存放在 `root/model/` 目录下。**

```
model/
├── pod/              # Pod 相关模型
├── node/             # Node 相关模型
├── deployment/       # Deployment 相关模型
├── service/          # Service 相关模型
├── metrics/          # 指标相关模型
├── event/            # 事件相关模型
├── envelope/         # Envelope 统一壳
├── ai/               # AI 相关模型
└── ...
```

### 规范要求

1. **强一致性**
   - 各模块引用同一数据模型，避免数据不一致
   - 禁止在各模块内部重复定义相同结构

2. **引用方式**
   ```go
   import (
       modelpod "AtlHyper/model/pod"
       modelnode "AtlHyper/model/node"
       nmetrics "AtlHyper/model/metrics"
   )
   ```

3. **修改规则**
   - 修改 model 需要考虑所有引用模块的影响
   - 重大变更需要更新版本号

---

## 配置管理规范

### 基本原则

**所有模块的配置都必须在 config 文件中定义，并设置环境变量和默认值。**

### 规范要求

1. **集中管理**
   - 配置不能分散在代码各处
   - 避免多处设置导致一处修改不生效

2. **环境变量支持**
   - 所有配置项必须支持环境变量覆盖
   - 便于不同环境部署

3. **默认值**
   - 所有配置项必须有合理的默认值
   - 保证无配置时也能正常运行

### 配置示例

```go
// config/config.go
type Config struct {
    MasterAddr string `env:"ATLHYPER_MASTER_ADDR" default:"http://localhost:8080"`
    PushInterval int  `env:"ATLHYPER_PUSH_INTERVAL" default:"5"`
    TTL int          `env:"ATLHYPER_TTL" default:"600"`
}
```

### 待定事项

> **TODO**: config 管理方式尚未确定
>
> 需要评估以下方案：
> - **方案 A**: 统一 config 文件管理（所有模块共用一个配置入口）
> - **方案 B**: 模块自治（每个模块独立管理自己的配置）
>
> 此事项需要在后续任务中讨论和决策。

---

## 数据存储规范

### Agent 数据存储池

**位置：** `atlhyper_agent/agent_store/`

**存储内容：** 节点指标快照（NodeMetricsSnapshot）

**特点：**
- 每个节点只保留最新一条快照
- 以 NodeName 为唯一 Key
- 使用 RWMutex 保证并发安全
- 定期清理过期数据（TTL：10 分钟）

**核心接口：**
```go
// 写入
func Put(node string, snap *NodeMetricsSnapshot)
func PutSnapshot(snap *NodeMetricsSnapshot)

// 读取
func GetAllLatestCopy() map[string]NodeMetricsSnapshot
func Len() int
```

### Master 数据存储池

**位置：** `atlhyper_master/master_store/`

**存储内容：** EnvelopeRecord（统一壳记录）

**特点：**
- 不直接存储业务结构，存储统一的 EnvelopeRecord
- 避免强耦合，Store 无需理解 Payload 内部格式
- 使用 RWMutex 保证并发安全
- 定期清理过期数据

**清理策略：**
- 通用 TTL：24 小时
- Metrics TTL：15 分钟
- 容量限制：最多 5 万条
- 清理周期：5 分钟

**核心接口：**
```go
// 写入
func Append(rec EnvelopeRecord)
func AppendBatch(recs []EnvelopeRecord)
func AppendEnvelope(env Envelope)
func ReplaceLatest(env Envelope) int  // 原子替换

// 读取
func Snapshot() []EnvelopeRecord
func Len() int
```

---

## 接口调用规范

### 统一接口层（interfaces）

**所有调用内部数据的函数操作都需要经过 interfaces 目录。**

#### Master interfaces 结构

```
atlhyper_master/interfaces/
├── datasource/          # 数据源接口（核心）
│   ├── interfaces.go    # Reader 接口定义
│   ├── hub_sources.go   # Hub 实现
│   └── hub_decode.go    # 解码工具
├── ui_interfaces/       # UI 接口
│   ├── pod/
│   ├── node/
│   ├── deployment/
│   └── ...
└── test_interfaces/     # 测试接口
```

#### 数据源接口（Reader）

```go
type Reader interface {
    // 事件
    GetK8sEventsRecent(ctx, clusterID, limit) ([]LogEvent, error)

    // 指标
    GetClusterMetricsLatest(ctx, clusterID) ([]NodeMetricsSnapshot, error)
    GetClusterMetricsRange(ctx, clusterID, since, until) ([]NodeMetricsSnapshot, error)

    // 资源列表
    GetPodListLatest(ctx, clusterID) ([]Pod, error)
    GetNodeListLatest(ctx, clusterID) ([]Node, error)
    GetServiceListLatest(ctx, clusterID) ([]Service, error)
    // ... 其他资源
}
```

#### 调用链路

```
┌─────────────┐
│  UI Handler │
└──────┬──────┘
       │
       ▼
┌─────────────────────────┐
│  interfaces/datasource  │  ◄── 统一入口
└──────┬──────────────────┘
       │
       ▼
┌─────────────────────────┐
│  master_store.Snapshot  │  ◄── 数据池出口
└─────────────────────────┘
```

### Agent interfaces 结构

```
atlhyper_agent/interfaces/
├── cluster/             # 集群资源查询
│   ├── pod.go
│   ├── node.go
│   └── ...
├── operations/          # 操作分派
│   └── dispatcher.go
└── data_api/            # 数据 API
```

---

## 数据流规范

### Agent → Master 数据流

**推送模式（Push）：**

```
Kubernetes API Server
        │
        ▼
Agent Internal (Watchers/ReadOnly)
        │
        ▼
agent_store (缓存)
        │
        ▼
External Push (HTTP + gzip)
        │
        ▼
Master /ingest/* Endpoints
        │
        ▼
master_store (存储)
```

**上报频率：**

| 数据类型 | 频率 | Source 标识 |
|----------|------|-------------|
| 事件日志 | 5 秒 | events |
| 指标快照 | 5 秒 | metrics_snapshot |
| Pod 列表 | 25 秒 | pod_list_snapshot |
| Node 列表 | 30 秒 | node_list_snapshot |
| Service 列表 | 35 秒 | service_list_snapshot |
| Namespace 列表 | 40 秒 | namespace_list_snapshot |
| Ingress 列表 | 45 秒 | ingress_list_snapshot |
| Deployment 列表 | 50 秒 | deployment_list_snapshot |
| ConfigMap 列表 | 55 秒 | configmap_list_snapshot |

### Metrics → Agent 数据流

```
┌─────────────────────────────────────────┐
│            Kubernetes Node              │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │       atlhyper_metrics          │   │
│  │  (CPU/Memory/Disk/Network/Temp) │   │
│  └─────────────┬───────────────────┘   │
│                │ HTTP Push              │
│                ▼                        │
│  ┌─────────────────────────────────┐   │
│  │       atlhyper_agent            │   │
│  │       (agent_store)             │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

### Master → Agent 命令流（Control）

**List+Watch 模式（类似 Kubernetes）：**

```
Master                          Agent
  │                               │
  │◄────── POST /control/watch ───│  (携带已知 RV)
  │                               │
  │  若有新版本，立即返回         │
  │  否则挂起等待（30s）          │
  │                               │
  │─────── CommandSet ───────────►│
  │                               │
  │                               │  (执行命令)
  │                               │
  │◄────── POST /control/ack ─────│  (回报结果)
  │                               │
```

---

## 模块详解

### Agent 模块

#### 子模块

| 目录 | 职责 |
|------|------|
| agent_store/ | 内存数据存储（节点指标缓存） |
| interfaces/ | 数据查询接口 |
| internal/readonly/ | 资源只读观察 |
| internal/watcher/ | 事件监控 |
| internal/operator/ | 操作执行 |
| external/push/ | 数据推送 |
| external/control/ | 命令控制 |
| external/ingest/ | HTTP Server（接收 Metrics） |

### Master 模块

#### 子模块

| 目录 | 职责 |
|------|------|
| master_store/ | 全局数据存储 |
| interfaces/ | 统一数据接口 |
| ingest/receivers/ | 数据接收处理器 |
| db/ | 数据库层 |
| control/ | 命令下发与执行控制 |
| client/ | 通知分发（Email/Slack） |
| server/ | HTTP 服务器 |

### 数据模型（model）

#### Envelope（统一上报壳）

```go
type Envelope struct {
    Version     string          // 协议版本
    ClusterID   string          // 集群 ID
    Source      string          // 数据源标识
    TimestampMs int64           // 发送时刻（毫秒）
    Payload     json.RawMessage // 原始载荷
}
```

#### Source 标识常量

```go
const (
    SourceK8sEvent              = "events"
    SourceMetricsSnapshot       = "metrics_snapshot"
    SourcePodListSnapshot       = "pod_list_snapshot"
    SourceNodeListSnapshot      = "node_list_snapshot"
    SourceServiceListSnapshot   = "service_list_snapshot"
    SourceNamespaceListSnapshot = "namespace_list_snapshot"
    SourceIngressListSnapshot   = "ingress_list_snapshot"
    SourceDeploymentListSnapshot = "deployment_list_snapshot"
    SourceConfigMapListSnapshot = "configmap_list_snapshot"
)
```

---

## 开发注意事项

### 必须遵守

1. **不要绕过 interfaces**
   - 所有数据访问必须通过 interfaces 目录
   - 禁止直接操作 store

2. **使用推送模式**
   - 外部扩展必须推送数据给 Agent/Master
   - 禁止 Agent/Master 主动拉取外部数据

3. **保证数据唯一性**
   - 使用唯一 Key 存储数据
   - 实现 TTL 清理机制

4. **Agent 是唯一出口**
   - 集群内数据只能通过 Agent 对外传输
   - 禁止其他模块直接对外通信

5. **并发安全**
   - Store 操作必须使用锁保护
   - 返回数据副本而非引用

### 建议

1. 使用 gzip 压缩网络传输
2. 批量操作减少锁竞争
3. 定期清理过期数据
4. 记录操作日志便于调试

---

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0 | 2025-01-04 | 初始版本 |
