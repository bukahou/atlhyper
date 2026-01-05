# AtlHyper 系统架构文档

> 本文档详细描述 AtlHyper 的技术架构、模块设计和数据流。

---

## 系统总览

AtlHyper 是一个轻量级 Kubernetes 集群可观测与控制平台，采用 **四层架构**。

### 核心四层架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Layer 1: Web UI                            │
│                                                                     │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │                   atlhyper_web (Next.js)                     │   │
│   │                 负责 Master 数据展示和交互                    │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                                │                                    │
│                                │ HTTP API                           │
│                                ▼                                    │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                          Layer 2: Master                            │
│                                                                     │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │                  atlhyper_master (Go/Gin)                    │   │
│   │          数据集中存储、API 服务、告警通知、命令下发            │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                                ▲                                    │
│                                │ HTTP Push (gzip)                   │
│                                │                                    │
└────────────────────────────────┼────────────────────────────────────┘
                                 │
┌────────────────────────────────┼────────────────────────────────────┐
│                                │          Kubernetes 集群           │
│                                ▼                                    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                      Layer 3: Agent                          │    │
│  │                                                             │    │
│  │   ┌─────────────────────────────────────────────────────┐   │    │
│  │   │               atlhyper_agent (Go)                    │   │    │
│  │   │                                                     │   │    │
│  │   │   ┌─────────────────────────────────────────────┐   │   │    │
│  │   │   │           internal/ (内部模块)               │   │   │    │
│  │   │   │   ┌────────────┐    ┌────────────┐          │   │   │    │
│  │   │   │   │ readonly/  │    │  watcher/  │          │   │   │    │
│  │   │   │   │ (只读查询) │    │ (事件监控) │          │   │   │    │
│  │   │   │   └─────┬──────┘    └─────┬──────┘          │   │   │    │
│  │   │   │         └────────┬────────┘                 │   │   │    │
│  │   │   │                  ▼                          │   │   │    │
│  │   │   │          Kubernetes API Server              │   │   │    │
│  │   │   └─────────────────────────────────────────────┘   │   │    │
│  │   │                                                     │   │    │
│  │   │   ┌─────────────────────────────────────────────┐   │   │    │
│  │   │   │           agent_store (数据缓存)             │   │   │    │
│  │   │   └─────────────────────────────────────────────┘   │   │    │
│  │   │                        ▲                            │   │    │
│  │   └────────────────────────┼────────────────────────────┘   │    │
│  │                            │ HTTP Push                      │    │
│  └────────────────────────────┼────────────────────────────────┘    │
│                               │                                     │
│  ┌────────────────────────────┴────────────────────────────────┐    │
│  │                      Layer 4: Metrics                        │    │
│  │                                                             │    │
│  │   ┌─────────────────────────────────────────────────────┐   │    │
│  │   │            atlhyper_metrics (DaemonSet)              │   │    │
│  │   │       节点级指标采集 (CPU/Memory/Disk/Network/Temp)   │   │    │
│  │   └─────────────────────────────────────────────────────┘   │    │
│  │                                                             │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 数据流方向

```
WebUI ←── Master ←── Agent ←── Metrics
  │         │          │
  │         │          └── 采集节点指标，推送给 Agent
  │         │
  │         └── 存储数据，提供 API，接收 Agent 推送
  │
  └── 展示数据，用户交互，调用 Master API
```

### 层次职责

| 层级 | 模块 | 职责 | 部署位置 |
|------|------|------|----------|
| Layer 1 | WebUI | Master 数据展示和交互 | 外部 |
| Layer 2 | Master | 数据集中存储、API、告警、命令下发 | 外部 |
| Layer 3 | Agent | 集群数据采集、推送、命令执行 | 集群内 |
| Layer 4 | Metrics | 节点系统指标采集 | 集群内 (DaemonSet) |

---

## 扩展模块（设计阶段）

> **注意**：以下模块尚在设计阶段，可能随时被删除或修改，暂不纳入正式架构。

| 模块 | 状态 | 说明 |
|------|------|------|
| atlhyper_adapter | 设计中 | 第三方监控系统适配器 (Prometheus/Zabbix 等) |

---

## 模块架构

### 1. atlhyper_master（主控中心）

**部署位置：** 集群外部（Docker Compose 推荐）

**核心职责：**
- 接收 Agent 推送的数据
- 集中存储和管理所有集群数据
- 提供 REST API 给 UI
- 协调告警通知（Slack/Email/Webhook）
- 与 AIService 协同进行智能诊断
- 下发控制命令给 Agent

**目录结构：**
```
atlhyper_master/
├── master_store/         # 数据存储核心
│   ├── hub.go           # 全局 Hub 实例
│   ├── types.go         # EnvelopeRecord 定义
│   ├── bootstrap.go     # 初始化
│   └── cleanup.go       # TTL 清理
├── interfaces/           # 统一接口层
│   ├── datasource/      # 数据源接口
│   ├── ui_interfaces/   # UI 接口
│   └── test_interfaces/ # 测试接口
├── ingest/              # 数据接收
│   └── receivers/       # 各类型接收器
├── db/                  # 数据库层
│   ├── sqlite/          # SQLite 管理
│   └── repository/      # 数据访问对象
├── control/             # 命令控制
├── client/              # 通知客户端
├── server/              # HTTP 服务器
└── aiservice/           # AI 诊断集成
```

### 2. atlhyper_agent（集群代理）

**部署位置：** Kubernetes 集群内部

**核心职责：**
- 监控和采集 Pod、Node、Service、Deployment、Event
- 集成 Metrics 模块获取节点指标
- 向 Master 推送数据（gzip 压缩）
- 接收并执行 Master 下发的控制命令

**目录结构：**
```
atlhyper_agent/
├── agent_store/          # 节点指标缓存
│   ├── hub.go           # 全局 Store 实例
│   ├── types.go         # Store 结构定义
│   ├── bootstrap.go     # 初始化
│   └── cleanup.go       # TTL 清理
├── interfaces/           # 数据查询接口
│   ├── cluster/         # 集群资源查询
│   ├── operations/      # 操作分派
│   └── data_api/        # 数据 API
├── internal/             # 内部模块
│   ├── readonly/        # 资源只读观察
│   ├── watcher/         # 事件监控
│   ├── diagnosis/       # 诊断系统
│   ├── operator/        # 操作执行
│   └── deployer/        # 部署管理
└── external/             # 外部交互
    ├── push/            # 数据推送
    ├── control/         # 命令控制
    └── ingest/          # HTTP Server
```

### 3. atlhyper_metrics（指标采集器）

**部署位置：** Kubernetes 集群内部（DaemonSet）

**核心职责：**
- 采集节点级系统指标
- 轻量级设计，适配边缘环境

**采集内容：**
| 指标类型 | 说明 | 采样频率 |
|----------|------|----------|
| CPU | 使用率、Load、TopK 进程 | 1-3 秒 |
| Memory | 总量、已用、可用、使用率 | 实时 |
| Disk | 各挂载点使用情况 | 实时 |
| Network | 网卡速率、流量 | 实时 |
| Temperature | CPU/GPU/NVMe 温度 | 实时 |

**目录结构：**
```
atlhyper_metrics/
├── collect/             # 指标采集
│   ├── cpu.go          # CPU 指标
│   ├── memory.go       # 内存指标
│   ├── disk.go         # 磁盘指标
│   ├── network.go      # 网络指标
│   ├── temperature.go  # 温度指标
│   └── procutil.go     # /proc 工具
├── config/              # 配置管理
├── push/                # 指标推送
└── internal/            # 内部处理
```

### 4. atlhyper_aiservice（AI 诊断服务）

**部署位置：** 可独立部署

**核心职责：**
- 基于 LLM 进行多阶段智能诊断
- 生成根因分析和修复建议

**诊断流水线：**
```
Stage1: 初步分析
    │
    ├── 输入: 事件日志
    ├── 输出: 初步诊断 + needResources 清单
    │
    ▼
Stage2: 上下文获取
    │
    ├── 解析 needResources
    ├── 向 Master 请求相关资源数据
    │
    ▼
Stage3: 最终诊断
    │
    ├── 输入: 事件 + 上下文资源
    ├── 输出: RootCause + Runbook + 报告
```

**目录结构：**
```
atlhyper_aiservice/
├── service/diagnose/    # 诊断管道
│   ├── pipeline.go      # 统一流程
│   ├── stage1_service.go
│   ├── stage2_service.go
│   ├── stage3_service.go
│   └── prompt/          # Prompt 模板
├── llm/                 # LLM 接口
│   ├── gemini_client.go
│   ├── llm_contract.go
│   └── llm_factory.go
├── handler/             # HTTP 处理器
├── client/              # 与 Master 通信
├── embedding/           # 向量嵌入（预留 RAG）
└── retriever/           # 知识检索（预留 RAG）
```

### 5. atlhyper_adapter（第三方适配器）

**部署位置：** 可独立部署

**核心职责：**
- 接收第三方监控系统数据
- 转换为标准化结构
- 推送给 Agent 或 Master

**支持系统：**
- Prometheus
- Zabbix
- Datadog
- Grafana

---

## 数据存储架构

### Agent Store（agent_store）

**功能：** 节点指标的内存缓存

**结构：**
```go
type Store struct {
    mu   sync.RWMutex
    data map[string]NodeMetricsSnapshot  // key = NodeName
}
```

**特点：**
- 每节点只保留最新快照
- RWMutex 并发保护
- TTL 10 分钟

**API：**
```go
// 写入
Put(node string, snap *NodeMetricsSnapshot)
PutSnapshot(snap *NodeMetricsSnapshot)

// 读取
GetAllLatestCopy() map[string]NodeMetricsSnapshot
Len() int
```

### Master Store（master_store）

**功能：** 全局 Envelope 池

**结构：**
```go
type Hub struct {
    mu  sync.RWMutex
    all []EnvelopeRecord
}

type EnvelopeRecord struct {
    Version     string
    ClusterID   string
    Source      string
    SentAtMs    int64
    EnqueuedAt  time.Time
    Payload     json.RawMessage
}
```

**特点：**
- 统一壳结构，不解析 Payload
- 支持批量写入
- ReplaceLatest 原子替换
- TTL 清理 + 容量限制

**API：**
```go
// 写入
Append(rec EnvelopeRecord)
AppendBatch(recs []EnvelopeRecord)
AppendEnvelope(env Envelope)
ReplaceLatest(env Envelope) int

// 读取
Snapshot() []EnvelopeRecord
Len() int
```

### SQLite 数据库

**用途：** 长期数据持久化

**存储内容：**
- 事件日志历史
- 用户信息
- 配置数据

**位置：** `db/repository/`

---

## 通信协议

### Envelope 协议

**统一上报壳：**
```go
type Envelope struct {
    Version     string          // 协议版本 (v1)
    ClusterID   string          // 集群 ID
    Source      string          // 数据源标识
    TimestampMs int64           // 发送时刻（毫秒）
    Payload     json.RawMessage // 原始载荷
}
```

**Source 标识：**
| Source | 说明 |
|--------|------|
| events | Kubernetes 事件日志 |
| metrics_snapshot | 节点指标快照 |
| pod_list_snapshot | Pod 列表快照 |
| node_list_snapshot | Node 列表快照 |
| service_list_snapshot | Service 列表快照 |
| namespace_list_snapshot | Namespace 列表快照 |
| ingress_list_snapshot | Ingress 列表快照 |
| deployment_list_snapshot | Deployment 列表快照 |
| configmap_list_snapshot | ConfigMap 列表快照 |

### Ingest 端点

**Master 接收端点：**
| 端点 | 方法 | 说明 |
|------|------|------|
| /ingest/events/v1/eventlog | POST | 事件日志 |
| /ingest/metrics/snapshot | POST | 节点指标快照 |
| /ingest/podlist | POST | Pod 列表 |
| /ingest/nodelist | POST | Node 列表 |
| /ingest/servicelist | POST | Service 列表 |
| /ingest/namespacelist | POST | Namespace 列表 |
| /ingest/ingresslist | POST | Ingress 列表 |
| /ingest/deploymentlist | POST | Deployment 列表 |
| /ingest/configmaplist | POST | ConfigMap 列表 |

### Control 协议

**List+Watch 模式：**
```go
// 命令集
type CommandSet struct {
    ClusterID string
    RV        uint64     // 版本号
    Commands  []Command
}

// 单条命令
type Command struct {
    ID     string
    Type   string            // PodRestart / UpdateImage / ...
    Target map[string]string
    Args   map[string]any
    Idem   string            // 幂等键
    Op     string            // add/update/cancel
}

// 执行结果
type AckResult struct {
    CommandID  string
    Status     string  // success/failed
    Message    string
    ErrorCode  string
    Attempt    int
}
```

**端点：**
| 端点 | 方法 | 说明 |
|------|------|------|
| /control/watch | POST | Agent 长轮询获取命令 |
| /control/ack | POST | Agent 回报执行结果 |
| /control/enqueue | POST | 管理端入队命令 |

---

## 数据流详解

### 1. 指标采集流

```
┌─────────────┐
│   /proc     │  系统文件
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│  atlhyper_metrics   │  采集 CPU/Memory/Disk/Network/Temperature
└──────┬──────────────┘
       │ HTTP Push
       ▼
┌─────────────────────┐
│  atlhyper_agent     │
│  (agent_store)      │  缓存节点最新快照
└──────┬──────────────┘
       │ HTTP Push (gzip)
       ▼
┌─────────────────────┐
│  atlhyper_master    │
│  (master_store)     │  存储 EnvelopeRecord
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│  interfaces/        │  统一数据接口
│  datasource         │
└─────────────────────┘
```

### 2. 事件监控流

```
┌──────────────────────┐
│  Kubernetes API      │
│  (Watch Events)      │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│  Agent Watcher       │  事件监控
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│  Event Buffer        │  事件缓冲
└──────┬───────────────┘
       │ 5秒周期
       ▼
┌──────────────────────┐
│  Push to Master      │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│  Master ingest       │  接收处理
└──────┬───────────────┘
       │
       ├───────────────────┐
       ▼                   ▼
┌──────────────┐    ┌──────────────┐
│ master_store │    │  SQLite DB   │  持久化
└──────────────┘    └──────────────┘
```

### 3. AI 诊断流

```
┌──────────────────────┐
│  Master 诊断请求      │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│  AIService Stage1    │  初步分析
│  (LLM 调用)          │
└──────┬───────────────┘
       │ needResources
       ▼
┌──────────────────────┐
│  AIService Stage2    │  获取上下文
│  (向 Master 请求)    │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│  AIService Stage3    │  最终诊断
│  (LLM 调用)          │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│  诊断报告返回         │
│  (RootCause/Runbook) │
└──────────────────────┘
```

### 4. 命令控制流

```
┌──────────────────────┐
│  Web UI 发起操作      │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│  Master /control/    │
│  enqueue             │  命令入队
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│  control/store       │  命令集管理
│  (CommandSet)        │
└──────────────────────┘
       │
       │◄─── Agent /control/watch (长轮询)
       │
       ▼
┌──────────────────────┐
│  Agent 执行命令       │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│  Agent /control/ack  │  回报结果
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│  Master 更新状态      │
└──────────────────────┘
```

---

## 安全与性能

### 安全设计

1. **网络隔离**
   - Agent 是集群唯一出口
   - 外部无法直接访问集群内部

2. **推送模式**
   - Master 不主动拉取
   - 降低攻击面

3. **请求限制**
   - 压缩前：4MiB
   - 解压后：8MiB

4. **幂等性**
   - 命令携带幂等键
   - 防止重复执行

### 性能优化

1. **Gzip 压缩**
   - 自动嗅探 gzip 魔数
   - 减少网络传输

2. **批量操作**
   - 批量写入减少锁竞争
   - 共用批次入池时间

3. **内存管理**
   - 值拷贝避免引用共享
   - GC 友好的清理策略

4. **长轮询**
   - 使用 RV 判断更新
   - 减少频繁请求

---

## 扩展规划

| 阶段 | 内容 |
|------|------|
| Phase 1 | Adapter 外部监控集成（Prometheus/Zabbix） |
| Phase 2 | AIService RAG/Embedding 知识增强 |
| Phase 3 | 多集群/多租户支持 |
| Phase 4 | 自愈引擎（Self-Healing） |

---

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0 | 2025-01-04 | 初始版本 |
