# Node Metrics - Agent 端设计文档

> 总览文档: `docs/design/node-metrics-design.md`
> 相关文档: `docs/design/atlhyper-metrics-v2.md`（采集器）

---

## 1. 概述

### 1.1 职责

Agent 端负责：
- 接收集群内所有节点的 `atlhyper_metrics_v2` 推送的指标
- 按节点名存储指标数据（每个节点保留最新一条）
- 将所有节点的指标聚合到 ClusterSnapshot 中一起上报给 Master

### 1.2 分层架构

采用 SDK → Repository → Service 分层架构，与 Agent 现有设计保持一致：

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      atlhyper_agent_v2 (单实例)                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         SDK (通信层)                                 │   │
│  │                                                                     │   │
│  │   sdk/common/server.go + routes.go  — 通用 HTTP 服务器和路由注册     │   │
│  │   sdk/metrics/receiver.go           — Metrics 模块接收端点          │   │
│  │                                                                     │   │
│  │   POST /metrics/node  →  解析 JSON  →  调用 repository.Save()       │   │
│  │   (接收各节点 Metrics DaemonSet 的上报)                               │   │
│  └───────────────────────────────────────┬─────────────────────────────┘   │
│                                          │                                  │
│                                          ▼                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Repository (数据层)                              │   │
│  │                                                                     │   │
│  │   repository/metrics_repository.go                                  │   │
│  │   ┌─────────────────────────────────────────────────────────────┐  │   │
│  │   │  data map[nodeName]*NodeMetricsSnapshot  // 按节点名存储     │  │   │
│  │   │                                                             │  │   │
│  │   │  Save(snapshot)      // 按节点名覆盖                         │  │   │
│  │   │  Get(nodeName)       // 获取指定节点                         │  │   │
│  │   │  GetAll() map[...]   // 获取所有节点                         │  │   │
│  │   └─────────────────────────────────────────────────────────────┘  │   │
│  └───────────────────────────────────────┬─────────────────────────────┘   │
│                                          │                                  │
│                                          ▼                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Service (业务层)                                │   │
│  │                                                                     │   │
│  │   service/snapshot_service.go                                       │   │
│  │   BuildSnapshot() {                                                 │   │
│  │       nodes := nodeRepo.GetAll()                                    │   │
│  │       pods  := podRepo.GetAll()                                     │   │
│  │       metrics := metricsRepo.GetAll()  // 获取所有节点的 Metrics     │   │
│  │       return ClusterSnapshot{ NodeMetrics: metrics, ... }           │   │
│  │   }                                                                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                          │                                  │
│                                          ▼                                  │
│                                   RESTful 上报到 Master                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.3 项目结构

```
atlhyper_agent_v2/
├── cmd/
│   └── main.go
├── config/
│   └── ...
├── sdk/                                    # SDK 层（通信）
│   ├── common/                             # [新增] 通用功能
│   │   ├── server.go                       # [新增] HTTP 服务器
│   │   └── routes.go                       # [新增] 路由注册
│   └── metrics/                            # [新增] Metrics 模块
│       └── receiver.go                     # [新增] 接收 Metrics 上报
├── repository/                             # Repository 层（数据）
│   ├── node_repository.go
│   ├── pod_repository.go
│   └── metrics_repository.go               # [新增] NodeMetrics 存储
├── service/                                # Service 层（业务）
│   ├── interfaces.go
│   └── snapshot_service.go                 # [修改] 聚合 NodeMetrics
├── gateway/                                # Gateway 层（对外 API）
│   └── ...
└── agent.go                                # [修改] 初始化 metricsRepo
```

### 1.4 数据流

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              K8s Cluster                                     │
│                                                                             │
│  Node 1              Node 2              Node N                              │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐                    │
│  │   metrics   │     │   metrics   │     │   metrics   │                    │
│  │ (DaemonSet) │     │ (DaemonSet) │ ... │ (DaemonSet) │                    │
│  └──────┬──────┘     └──────┬──────┘     └──────┬──────┘                    │
│         │                   │                   │                            │
│         │ POST /metrics/node                    │                            │
│         └───────────────────┼───────────────────┘                            │
│                             ▼                                                │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    atlhyper_agent (单实例)                             │  │
│  │                                                                       │  │
│  │  SDK ─► Repository ─► Service ─► ClusterSnapshot                      │  │
│  │        (按节点名存储)   (聚合所有节点)   + NodeMetrics                   │  │
│  └────────────────────────────────────┬──────────────────────────────────┘  │
│                                       │                                      │
│                                       │ RESTful                              │
│                                       ▼                                      │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                         atlhyper_master                                │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. SDK 层 - 接收接口

### 2.1 HTTP 端点

```
POST /metrics/node
Content-Type: application/json

{
  "node_name": "k8s-worker-01",
  "timestamp": "2024-01-20T10:30:00Z",
  "cpu": { ... },
  "memory": { ... },
  "disks": [ ... ],
  "networks": [ ... ],
  "temperature": { ... },
  "top_processes": [ ... ]
}
```

### 2.2 Handler 实现

```go
// atlhyper_agent_v2/sdk/metrics/receiver.go  [新增]

package metrics

import (
    "github.com/gin-gonic/gin"
    "AtlHyper/model_v2"
    "AtlHyper/atlhyper_agent_v2/repository"
)

// Handler Metrics 模块 Handler
type Handler struct {
    metricsRepo *repository.MetricsRepository
}

// NewHandler 创建 Handler
func NewHandler(metricsRepo *repository.MetricsRepository) *Handler {
    return &Handler{metricsRepo: metricsRepo}
}

// HandleNodeMetrics 接收 atlhyper_metrics 推送的节点指标
func (h *Handler) HandleNodeMetrics(c *gin.Context) {
    var snapshot model_v2.NodeMetricsSnapshot
    if err := c.ShouldBindJSON(&snapshot); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 验证节点名
    if snapshot.NodeName == "" {
        c.JSON(400, gin.H{"error": "node_name is required"})
        return
    }

    // 存入 Repository（覆盖式）
    h.metricsRepo.Save(&snapshot)

    c.JSON(200, gin.H{"status": "ok"})
}
```

### 2.3 路由注册

```go
// atlhyper_agent_v2/sdk/common/routes.go  [新增]

package common

import (
    "AtlHyper/atlhyper_agent_v2/sdk/metrics"
)

func (s *Server) setupRoutes() {
    // Node Metrics 接收
    metricsHandler := metrics.NewHandler(s.metricsRepo)
    s.router.POST("/metrics/node", metricsHandler.HandleNodeMetrics)

    // ... 后续可扩展其他模块的路由 ...
}
```

### 2.4 Server 实现

```go
// atlhyper_agent_v2/sdk/common/server.go  [新增]

package common

import (
    "github.com/gin-gonic/gin"
    "AtlHyper/atlhyper_agent_v2/repository"
)

// Server SDK 通用服务器
// 负责统一管理 HTTP 服务和路由注册
type Server struct {
    router      *gin.Engine
    metricsRepo *repository.MetricsRepository
    // ... 后续可扩展其他 repository ...
}

func NewServer(metricsRepo *repository.MetricsRepository) *Server {
    s := &Server{
        router:      gin.Default(),
        metricsRepo: metricsRepo,
    }
    s.setupRoutes()
    return s
}

func (s *Server) Run(addr string) error {
    return s.router.Run(addr)
}
```

---

## 3. Repository 层 - 数据存储

### 3.1 MetricsRepository 结构

```go
// atlhyper_agent_v2/repository/metrics_repository.go  [新增]

package repository

import (
    "sync"
    "AtlHyper/model_v2"
)

// MetricsRepository 节点指标存储
// 按节点名存储，每个节点保留最新一条数据
type MetricsRepository struct {
    mu   sync.RWMutex
    data map[string]*model_v2.NodeMetricsSnapshot // nodeName -> snapshot
}

// NewMetricsRepository 创建存储
func NewMetricsRepository() *MetricsRepository {
    return &MetricsRepository{
        data: make(map[string]*model_v2.NodeMetricsSnapshot),
    }
}

// Save 保存节点指标（按节点名覆盖）
// 同一节点的新数据会覆盖旧数据
func (r *MetricsRepository) Save(snapshot *model_v2.NodeMetricsSnapshot) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.data[snapshot.NodeName] = snapshot
}

// Get 获取指定节点的指标
func (r *MetricsRepository) Get(nodeName string) *model_v2.NodeMetricsSnapshot {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.data[nodeName]
}

// GetAll 获取所有节点的指标
func (r *MetricsRepository) GetAll() map[string]*model_v2.NodeMetricsSnapshot {
    r.mu.RLock()
    defer r.mu.RUnlock()

    result := make(map[string]*model_v2.NodeMetricsSnapshot, len(r.data))
    for k, v := range r.data {
        result[k] = v
    }
    return result
}

// Clear 清空数据
func (r *MetricsRepository) Clear() {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.data = make(map[string]*model_v2.NodeMetricsSnapshot)
}

// Count 返回节点数量
func (r *MetricsRepository) Count() int {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return len(r.data)
}
```

### 3.2 存储策略

| 特性 | 说明 |
|------|------|
| **存储方式** | 内存存储，按节点名索引 |
| **更新方式** | 按节点名覆盖，每个节点只保留最新一条 |
| **并发安全** | sync.RWMutex 保护 |
| **过期机制** | 无，依赖 metrics 持续推送 |
| **持久化** | 无，Agent 重启后数据丢失（可接受） |

---

## 4. Service 层 - 快照构建

### 4.1 ClusterSnapshot 扩展

```go
// model_v2/snapshot.go  [修改]

type ClusterSnapshot struct {
    // ... 现有字段 ...
    ClusterID   string                 `json:"cluster_id"`
    Timestamp   time.Time              `json:"timestamp"`
    Nodes       []NodeInfo             `json:"nodes"`
    Pods        []PodInfo              `json:"pods"`
    // ...

    // Node Metrics [新增]
    NodeMetrics map[string]*NodeMetricsSnapshot `json:"node_metrics,omitempty"`
}
```

### 4.2 SnapshotService 修改

```go
// atlhyper_agent_v2/service/snapshot_service.go  [修改]

type SnapshotService struct {
    nodeRepo    *repository.NodeRepository
    podRepo     *repository.PodRepository
    metricsRepo *repository.MetricsRepository  // [新增]
    // ... 其他 repository ...
}

func NewSnapshotService(
    nodeRepo *repository.NodeRepository,
    podRepo *repository.PodRepository,
    metricsRepo *repository.MetricsRepository,  // [新增]
) *SnapshotService {
    return &SnapshotService{
        nodeRepo:    nodeRepo,
        podRepo:     podRepo,
        metricsRepo: metricsRepo,
    }
}

func (s *SnapshotService) BuildSnapshot() *model_v2.ClusterSnapshot {
    snapshot := &model_v2.ClusterSnapshot{
        ClusterID: s.clusterID,
        Timestamp: time.Now(),
        Nodes:     s.nodeRepo.GetAll(),
        Pods:      s.podRepo.GetAll(),
        // ... 其他数据 ...
    }

    // [新增] 获取所有节点的 NodeMetrics
    snapshot.NodeMetrics = s.metricsRepo.GetAll()

    return snapshot
}
```

---

## 5. 初始化

### 5.1 依赖注入

```go
// atlhyper_agent_v2/agent.go  [修改]

import (
    "AtlHyper/atlhyper_agent_v2/repository"
    "AtlHyper/atlhyper_agent_v2/service"
    "AtlHyper/atlhyper_agent_v2/sdk/common"
)

type Agent struct {
    // Repository 层
    nodeRepo    *repository.NodeRepository
    podRepo     *repository.PodRepository
    metricsRepo *repository.MetricsRepository  // [新增]

    // Service 层
    snapshotService *service.SnapshotService

    // SDK 层
    server *common.Server  // [新增] 使用 sdk/common 包
}

func NewAgent(cfg *config.Config) *Agent {
    // 1. 初始化 Repository
    nodeRepo := repository.NewNodeRepository()
    podRepo := repository.NewPodRepository()
    metricsRepo := repository.NewMetricsRepository()  // [新增]

    // 2. 初始化 Service
    snapshotService := service.NewSnapshotService(nodeRepo, podRepo, metricsRepo)

    // 3. 初始化 SDK [新增]
    server := common.NewServer(metricsRepo)

    return &Agent{
        nodeRepo:        nodeRepo,
        podRepo:         podRepo,
        metricsRepo:     metricsRepo,
        snapshotService: snapshotService,
        server:          server,
    }
}
```

---

## 6. 文件清单

### 6.1 完整结构（标注新增/修改）

```
atlhyper_agent_v2/
├── cmd/
│   └── main.go
├── config/
│   └── ...
├── sdk/
│   ├── common/                         # [新增] 通用功能
│   │   ├── server.go                   # [新增] HTTP 服务器
│   │   └── routes.go                   # [新增] 路由注册
│   └── metrics/                        # [新增] Metrics 模块
│       └── receiver.go                 # [新增] 接收端点
├── repository/
│   ├── node_repository.go
│   ├── pod_repository.go
│   └── metrics_repository.go           # [新增] NodeMetrics 存储
├── service/
│   ├── interfaces.go
│   └── snapshot_service.go             # [修改] 聚合 NodeMetrics
├── gateway/
│   └── ...
└── agent.go                            # [修改] 初始化 metricsRepo

model_v2/
└── snapshot.go                         # [修改] 添加 NodeMetrics 字段
```

### 6.2 改动对照表

| 层级 | 文件 | 操作 | 说明 |
|------|------|------|------|
| SDK | `sdk/common/` | **新增** | 通用功能文件夹 |
| SDK | `sdk/common/server.go` | **新增** | HTTP 服务器 |
| SDK | `sdk/common/routes.go` | **新增** | 路由注册 |
| SDK | `sdk/metrics/` | **新增** | Metrics 模块文件夹 |
| SDK | `sdk/metrics/receiver.go` | **新增** | HTTP Handler |
| Repository | `repository/metrics_repository.go` | **新增** | 内存存储 |
| Service | `service/snapshot_service.go` | 修改 | 构建时获取 metrics |
| Model | `model_v2/snapshot.go` | 修改 | 添加 NodeMetrics 字段 |
| Main | `agent.go` | 修改 | 初始化依赖链 |

---

## 7. 注意事项

### 7.1 单实例多节点

Agent 采用单实例部署，接收集群内所有节点的 Metrics 上报：
- Repository 按节点名存储，每个节点一条记录
- ClusterSnapshot.NodeMetrics 包含所有已上报节点的数据
- 各节点的 Metrics DaemonSet 通过 Service/ClusterIP 访问 Agent

### 7.2 数据一致性

| 组件 | 间隔 |
|------|------|
| metrics 推送 | 5s |
| 快照上报 | 5s（可配置） |

两者间隔相同但不完全同步，可能存在几百毫秒的时间差，但对于监控数据来说影响不大。

### 7.3 错误处理

| 场景 | 处理方式 |
|------|----------|
| metrics 推送失败 | metrics 侧负责重试 |
| Repository 为空 | 快照中 NodeMetrics 为空 map |
| JSON 解析失败 | 返回 400 错误，不影响其他功能 |

### 7.4 与现有架构的兼容性

本设计遵循 Agent 现有的分层架构：
- **SDK 负责通信** — 接收外部请求，调用 Repository
- **Repository 负责数据存储** — 提供 Save/Get 接口
- **Service 负责业务逻辑** — 组合多个 Repository 构建快照

改动最小化，不影响现有的 Node/Pod 等数据流。
