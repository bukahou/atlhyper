# Node Metrics - Master 端设计文档

> 总览文档: `docs/design/node-metrics-design.md`

---

## 1. 概述

### 1.1 职责

Master 端负责：
- 接收 Agent 上报的 ClusterSnapshot（含 NodeMetrics）
- DataHub 保持信封结构不变，存储完整 Snapshot
- 提取 NodeMetrics 存入 SQLite（实时 + 历史）
- 提供 API 供前端查询

### 1.2 存储策略

| 类型 | 存储位置 | 更新方式 | 频率 | 保留时长 | 用途 |
|------|----------|----------|------|----------|------|
| **Snapshot** | DataHub（内存） | 覆盖 | 5s | 最新 1 条 | 完整集群状态 |
| **实时指标** | SQLite | 覆盖 | 5s | 最新 1 条/节点 | 当前详细状态 |
| **趋势数据** | SQLite | 追加 | 5min 采样 | 30 天 | 历史趋势图 |

### 1.3 分层架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            atlhyper_master_v2                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Gateway (通信层)                             │   │
│  │                                                                     │   │
│  │   handler/node_metrics.go                                           │   │
│  │   GET /api/v2/clusters/{id}/node-metrics                            │   │
│  │   GET /api/v2/clusters/{id}/node-metrics/{nodeName}                 │   │
│  │   GET /api/v2/clusters/{id}/node-metrics/{nodeName}/history         │   │
│  │                                                                     │   │
│  │   直接使用 database.NodeMetricsRepository 查询                       │   │
│  └───────────────────────────────────────┬─────────────────────────────┘   │
│                                          │                                  │
│                                          ▼                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Database (持久化层)                             │   │
│  │                                                                     │   │
│  │   database/interfaces.go     — NodeMetricsRepository 接口           │   │
│  │   database/repo/node_metrics.go   — Repository 实现                 │   │
│  │   database/sqlite/node_metrics.go — Dialect 实现（SQL 生成）         │   │
│  │                                                                     │   │
│  │   表结构：                                                           │   │
│  │   node_metrics_latest   — 实时数据，每节点一行，覆盖更新              │   │
│  │   node_metrics_history  — 趋势数据，5 分钟采样，30 天保留             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                          ▲                                  │
│                                          │ 写入                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                   Service/Sync (同步层)                              │   │
│  │                                                                     │   │
│  │   service/sync/metrics_persist.go  [新增]                           │   │
│  │   MetricsPersistService（类似 EventPersistService）                  │   │
│  │   - Sync(clusterID): 从 DataHub 读取 → 写入 SQLite                  │   │
│  │   - cleanupLoop(): 定期清理 30 天过期数据                            │   │
│  └───────────────────────────────────────┬─────────────────────────────┘   │
│                                          ▲                                  │
│                                          │ 回调触发                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                       Processor (处理层)                             │   │
│  │                                                                     │   │
│  │   processor/processor.go (微调)                                     │   │
│  │   1. 接收 Snapshot 存入 DataHub（信封结构不变）                       │   │
│  │   2. 触发 onSnapshotReceived 回调                                   │   │
│  │      → 现有: EventPersistService.Sync()                             │   │
│  │      → 新增: MetricsPersistService.Sync()                           │   │
│  └───────────────────────────────────────┬─────────────────────────────┘   │
│                                          ▲                                  │
│                                          │                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        DataHub (内存存储)                            │   │
│  │                                                                     │   │
│  │   保持信封结构不变                                                    │   │
│  │   map[clusterID]*ClusterSnapshot（含 NodeMetrics）                   │   │
│  │   覆盖式存储，只保留最新                                              │   │
│  └───────────────────────────────────────┬─────────────────────────────┘   │
│                                          ▲                                  │
│                                          │                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                       AgentSDK (接收层)                              │   │
│  │                                                                     │   │
│  │   接收 Agent RESTful 上报的 ClusterSnapshot                          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                          ▲                                  │
└──────────────────────────────────────────┼──────────────────────────────────┘
                                           │
                                      Agent 上报
                                   ClusterSnapshot
                                   (含 NodeMetrics)
```

### 1.4 数据流

> 遵循现有 `EventPersistService` 模式：Processor 通过回调触发 Service 同步

```
Agent 上报 ClusterSnapshot (每 5s)
         │
         ▼
    ┌──────────────────────────────┐
    │ processor/processor.go       │
    │                              │
    │  1. 存入 DataHub（不变）      │
    │  2. 触发 onSnapshotReceived  │
    └────┬─────────────────────────┘
         │
         ├─────────────────────┐
         ▼                     ▼
    ┌─────────┐     ┌────────────────────────────┐
    │ DataHub │     │ 回调触发                    │
    │ (内存)  │     │ - EventPersistService.Sync │
    └────┬────┘     │ - MetricsPersistService.Sync│ [新增]
         │          └────────────────────────────┘
         │                     │
         │                     ▼
         │          ┌────────────────────────────┐
         │          │ MetricsPersistService      │
         │          │                            │
         └─────────►│ 1. 从 DataHub 读取 NodeMetrics │
                    │ 2. UpsertLatest (每次)      │
                    │ 3. InsertHistory (5分钟采样) │
                    │ 4. cleanupLoop (30天清理)   │
                    └────────────┬───────────────┘
                                 │
                                 ▼
                            ┌─────────┐
                            │ SQLite  │
                            │ latest表 │
                            │ history表│
                            └────┬────┘
                                 │
                                 ▼
                   ┌─────────────────────────────┐
                   │ Gateway Handler             │
                   │ 使用 NodeMetricsRepository  │
                   │ 查询 latest / history 表    │
                   └─────────────────────────────┘
                                 │
                                 ▼
                               Web
```

### 1.5 项目结构

```
atlhyper_master_v2/
├── cmd/
│   └── main.go
├── config/
│   └── ...
├── datahub/                                # DataHub 层（内存）- 不变
│   ├── interfaces.go
│   └── memory/
│       └── store.go                        # 保持信封结构不变
├── database/                               # Database 层 - 遵循现有模式
│   ├── interfaces.go                       # [修改] 添加 NodeMetrics 相关接口
│   ├── sqlite/                             # Dialect 实现（SQL 生成）
│   │   ├── ...                             # 现有文件不变
│   │   └── node_metrics.go                 # [新增] NodeMetricsDialect 实现
│   └── repo/                               # Repository 实现
│       ├── ...                             # 现有文件不变
│       └── node_metrics.go                 # [新增] NodeMetricsRepository 实现
├── processor/                              # Processor 层
│   └── processor.go                        # [微调] 添加回调
├── service/                                # Service 层
│   ├── interfaces.go                       # 不变（定义 Query/Ops 接口）
│   ├── factory.go                          # 不变
│   ├── query/                              # DataHub 数据读取（供 Gateway Handler 使用）
│   │   └── impl.go                         # 不变（注入 datahub.Store）
│   └── sync/                               # 持久化服务（直接注入 datahub.Store）
│       ├── event_persist.go                # 现有 - Event 持久化
│       └── metrics_persist.go              # [新增] NodeMetrics 持久化
├── gateway/                                # Gateway 层
│   ├── routes.go                           # [修改] 注册路由
│   └── handler/
│       ├── ...
│       └── node_metrics.go                 # [新增] API Handler
└── master.go                               # [修改] 初始化 MetricsPersistService

model_v2/
└── node_metrics.go                         # [新增] 共用数据模型
```

**说明**：
- 遵循现有 `EventPersistService` 模式，新增 `MetricsPersistService`
- Processor 通过回调触发，与 DB 层解耦
- Gateway Handler 直接使用 `database.NodeMetricsRepository` 查询（SQLite 数据不走 service/query）

---

## 2. 数据库设计

### 2.1 表结构

```sql
-- 实时数据：每个节点一行，覆盖更新
-- 存储完整快照 JSON，用于展示当前详细状态
CREATE TABLE node_metrics_latest (
    cluster_id   TEXT NOT NULL,
    node_name    TEXT NOT NULL,
    timestamp    DATETIME NOT NULL,

    -- 完整快照 JSON
    snapshot     TEXT NOT NULL,

    PRIMARY KEY (cluster_id, node_name)
);

-- 趋势数据：5 分钟一条，用于趋势图
-- 只存储关键指标，节省空间
CREATE TABLE node_metrics_history (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id   TEXT NOT NULL,
    node_name    TEXT NOT NULL,
    timestamp    DATETIME NOT NULL,

    -- 关键指标
    cpu_usage    REAL,          -- CPU 使用率 %
    mem_usage    REAL,          -- 内存使用率 %
    disk_usage   REAL,          -- 主磁盘使用率 %
    net_rx_bytes REAL,          -- 网络接收 bytes/s
    net_tx_bytes REAL,          -- 网络发送 bytes/s
    temperature  REAL,          -- CPU 温度

    UNIQUE (cluster_id, node_name, timestamp)
);

-- 索引：加速时间范围查询
CREATE INDEX idx_history_cluster_node_time
    ON node_metrics_history(cluster_id, node_name, timestamp);

-- 索引：加速清理过期数据
CREATE INDEX idx_history_timestamp
    ON node_metrics_history(timestamp);
```

### 2.2 数据量估算

| 表 | 每节点记录数 | 单条大小 | 10 节点 30 天 |
|-----|------------|---------|--------------|
| latest | 1 条 | ~2 KB | 20 KB |
| history | 8640 条 (30天×288条/天) | ~100 B | ~8.6 MB |

总计：10 节点 30 天约 **10 MB**，SQLite 轻松应对。

---

## 3. Database 层 - 遵循现有模式

> 遵循现有 `database/` 分层架构：
> - `interfaces.go` — 定义 Model、Repository 接口、Dialect 接口
> - `repo/*.go` — Repository 实现（业务逻辑，调用 Dialect）
> - `sqlite/*.go` — Dialect 实现（SQL 语句生成）

### 3.1 接口定义（添加到 interfaces.go）

```go
// database/interfaces.go  [修改 - 添加以下内容]

// ==================== NodeMetrics 数据模型 ====================

// NodeMetricsLatest 实时数据（每节点一行）
type NodeMetricsLatest struct {
    ClusterID string    `json:"cluster_id"`
    NodeName  string    `json:"node_name"`
    Timestamp time.Time `json:"timestamp"`
    Snapshot  string    `json:"snapshot"` // JSON 序列化的 NodeMetricsSnapshot
}

// NodeMetricsHistory 趋势数据（5 分钟采样）
type NodeMetricsHistory struct {
    ID          int64     `json:"id"`
    ClusterID   string    `json:"cluster_id"`
    NodeName    string    `json:"node_name"`
    Timestamp   time.Time `json:"timestamp"`
    CPUUsage    float64   `json:"cpu_usage"`
    MemUsage    float64   `json:"mem_usage"`
    DiskUsage   float64   `json:"disk_usage"`
    NetRxBytes  float64   `json:"net_rx_bytes"`
    NetTxBytes  float64   `json:"net_tx_bytes"`
    Temperature float64   `json:"temperature"`
}

// ==================== Repository 接口 ====================

// NodeMetricsRepository NodeMetrics 数据仓库接口
type NodeMetricsRepository interface {
    // 实时数据
    UpsertLatest(clusterID, nodeName string, snapshot *model_v2.NodeMetricsSnapshot) error
    GetLatest(clusterID, nodeName string) (*model_v2.NodeMetricsSnapshot, error)
    ListLatest(clusterID string) (map[string]*model_v2.NodeMetricsSnapshot, error)

    // 趋势数据
    InsertHistory(clusterID, nodeName string, point *model_v2.MetricsDataPoint) error
    GetHistory(clusterID, nodeName string, start, end time.Time) ([]model_v2.MetricsDataPoint, error)

    // 维护
    CleanupHistory(before time.Time) (int64, error)
}

// ==================== Dialect 接口 ====================

// NodeMetricsDialect NodeMetrics SQL 方言接口
type NodeMetricsDialect interface {
    // 表结构迁移
    CreateLatestTable() string
    CreateHistoryTable() string
    CreateHistoryIndexes() []string

    // 实时数据
    UpsertLatest() (query string)
    GetLatest() (query string)
    ListLatest() (query string)

    // 趋势数据
    InsertHistory() (query string)
    GetHistory() (query string)

    // 维护
    CleanupHistory() (query string)
}

// ==================== DB 扩展 ====================

// 在现有 DB 结构体中添加字段
type DB struct {
    // ... 现有字段 ...
    NodeMetrics NodeMetricsRepository  // [新增]
}
```

### 3.2 Dialect 实现（SQL 语句生成）

```go
// database/sqlite/node_metrics.go  [新增]

package sqlite

// NodeMetricsDialect SQLite 方言实现
type NodeMetricsDialect struct{}

func NewNodeMetricsDialect() *NodeMetricsDialect {
    return &NodeMetricsDialect{}
}

// ==================== 表结构 ====================

func (d *NodeMetricsDialect) CreateLatestTable() string {
    return `CREATE TABLE IF NOT EXISTS node_metrics_latest (
        cluster_id   TEXT NOT NULL,
        node_name    TEXT NOT NULL,
        timestamp    DATETIME NOT NULL,
        snapshot     TEXT NOT NULL,
        PRIMARY KEY (cluster_id, node_name)
    )`
}

func (d *NodeMetricsDialect) CreateHistoryTable() string {
    return `CREATE TABLE IF NOT EXISTS node_metrics_history (
        id           INTEGER PRIMARY KEY AUTOINCREMENT,
        cluster_id   TEXT NOT NULL,
        node_name    TEXT NOT NULL,
        timestamp    DATETIME NOT NULL,
        cpu_usage    REAL,
        mem_usage    REAL,
        disk_usage   REAL,
        net_rx_bytes REAL,
        net_tx_bytes REAL,
        temperature  REAL,
        UNIQUE (cluster_id, node_name, timestamp)
    )`
}

func (d *NodeMetricsDialect) CreateHistoryIndexes() []string {
    return []string{
        `CREATE INDEX IF NOT EXISTS idx_history_cluster_node_time
            ON node_metrics_history(cluster_id, node_name, timestamp)`,
        `CREATE INDEX IF NOT EXISTS idx_history_timestamp
            ON node_metrics_history(timestamp)`,
    }
}

// ==================== 实时数据 SQL ====================

func (d *NodeMetricsDialect) UpsertLatest() string {
    return `INSERT INTO node_metrics_latest (cluster_id, node_name, timestamp, snapshot)
        VALUES (?, ?, ?, ?)
        ON CONFLICT(cluster_id, node_name)
        DO UPDATE SET timestamp = excluded.timestamp, snapshot = excluded.snapshot`
}

func (d *NodeMetricsDialect) GetLatest() string {
    return `SELECT snapshot FROM node_metrics_latest WHERE cluster_id = ? AND node_name = ?`
}

func (d *NodeMetricsDialect) ListLatest() string {
    return `SELECT node_name, snapshot FROM node_metrics_latest WHERE cluster_id = ?`
}

// ==================== 趋势数据 SQL ====================

func (d *NodeMetricsDialect) InsertHistory() string {
    return `INSERT OR IGNORE INTO node_metrics_history
        (cluster_id, node_name, timestamp, cpu_usage, mem_usage, disk_usage, net_rx_bytes, net_tx_bytes, temperature)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
}

func (d *NodeMetricsDialect) GetHistory() string {
    return `SELECT timestamp, cpu_usage, mem_usage, disk_usage, net_rx_bytes, net_tx_bytes, temperature
        FROM node_metrics_history
        WHERE cluster_id = ? AND node_name = ? AND timestamp BETWEEN ? AND ?
        ORDER BY timestamp ASC`
}

// ==================== 维护 SQL ====================

func (d *NodeMetricsDialect) CleanupHistory() string {
    return `DELETE FROM node_metrics_history WHERE timestamp < ?`
}
```

### 3.3 Repository 实现（业务逻辑）

```go
// database/repo/node_metrics.go  [新增]

package repo

import (
    "database/sql"
    "encoding/json"
    "time"

    "AtlHyper/atlhyper_master_v2/database"
    "AtlHyper/atlhyper_master_v2/database/sqlite"
    "AtlHyper/model_v2"
)

type NodeMetricsRepo struct {
    db      *sql.DB
    dialect *sqlite.NodeMetricsDialect
}

func NewNodeMetricsRepo(db *sql.DB, dialect *sqlite.NodeMetricsDialect) *NodeMetricsRepo {
    return &NodeMetricsRepo{db: db, dialect: dialect}
}

// Migrate 执行表迁移
func (r *NodeMetricsRepo) Migrate() error {
    // 创建 latest 表
    if _, err := r.db.Exec(r.dialect.CreateLatestTable()); err != nil {
        return err
    }

    // 创建 history 表
    if _, err := r.db.Exec(r.dialect.CreateHistoryTable()); err != nil {
        return err
    }

    // 创建索引
    for _, idx := range r.dialect.CreateHistoryIndexes() {
        if _, err := r.db.Exec(idx); err != nil {
            return err
        }
    }

    return nil
}

// ==================== 实时数据 ====================

func (r *NodeMetricsRepo) UpsertLatest(clusterID, nodeName string, snapshot *model_v2.NodeMetricsSnapshot) error {
    data, err := json.Marshal(snapshot)
    if err != nil {
        return err
    }

    _, err = r.db.Exec(r.dialect.UpsertLatest(), clusterID, nodeName, snapshot.Timestamp, string(data))
    return err
}

func (r *NodeMetricsRepo) GetLatest(clusterID, nodeName string) (*model_v2.NodeMetricsSnapshot, error) {
    var data string
    err := r.db.QueryRow(r.dialect.GetLatest(), clusterID, nodeName).Scan(&data)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    var snapshot model_v2.NodeMetricsSnapshot
    if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
        return nil, err
    }
    return &snapshot, nil
}

func (r *NodeMetricsRepo) ListLatest(clusterID string) (map[string]*model_v2.NodeMetricsSnapshot, error) {
    rows, err := r.db.Query(r.dialect.ListLatest(), clusterID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    result := make(map[string]*model_v2.NodeMetricsSnapshot)
    for rows.Next() {
        var nodeName, data string
        if err := rows.Scan(&nodeName, &data); err != nil {
            return nil, err
        }

        var snapshot model_v2.NodeMetricsSnapshot
        if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
            continue
        }
        result[nodeName] = &snapshot
    }
    return result, nil
}

// ==================== 趋势数据 ====================

func (r *NodeMetricsRepo) InsertHistory(clusterID, nodeName string, point *model_v2.MetricsDataPoint) error {
    _, err := r.db.Exec(r.dialect.InsertHistory(),
        clusterID, nodeName, time.UnixMilli(point.Timestamp),
        point.CPUUsage, point.MemUsage, point.DiskUsage,
        point.NetRxBytesPS, point.NetTxBytesPS, point.Temperature)
    return err
}

func (r *NodeMetricsRepo) GetHistory(clusterID, nodeName string, start, end time.Time) ([]model_v2.MetricsDataPoint, error) {
    rows, err := r.db.Query(r.dialect.GetHistory(), clusterID, nodeName, start, end)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []model_v2.MetricsDataPoint
    for rows.Next() {
        var ts time.Time
        var p model_v2.MetricsDataPoint
        if err := rows.Scan(&ts, &p.CPUUsage, &p.MemUsage, &p.DiskUsage,
            &p.NetRxBytesPS, &p.NetTxBytesPS, &p.Temperature); err != nil {
            return nil, err
        }
        p.Timestamp = ts.UnixMilli()
        result = append(result, p)
    }
    return result, nil
}

// ==================== 维护 ====================

func (r *NodeMetricsRepo) CleanupHistory(before time.Time) (int64, error) {
    result, err := r.db.Exec(r.dialect.CleanupHistory(), before)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}
```

---

## 4. Service/Sync 层 - NodeMetrics 持久化

> 遵循现有 `EventPersistService` 模式，新增 `MetricsPersistService`。
> Processor 通过回调触发，与 DB 层完全解耦。

### 4.1 MetricsPersistService 实现

```go
// service/sync/metrics_persist.go  [新增]

package sync

import (
    "sync"
    "time"

    "AtlHyper/atlhyper_master_v2/database"
    "AtlHyper/atlhyper_master_v2/datahub"
    "AtlHyper/common/logger"
    "AtlHyper/model_v2"
)

var metricsLog = logger.Module("MetricsPersist")

// MetricsPersistService NodeMetrics 持久化服务
// 类似 EventPersistService，由快照到达时触发同步
type MetricsPersistService struct {
    store       datahub.Store
    metricsRepo database.NodeMetricsRepository

    // 配置
    sampleInterval  time.Duration  // 趋势采样间隔（5 分钟）
    retentionDays   int            // 历史数据保留天数（30 天）
    cleanupInterval time.Duration  // 清理任务间隔（1 小时）

    // 采样状态
    lastSample   map[string]time.Time  // clusterID:nodeName -> lastSampleTime
    lastSampleMu sync.RWMutex

    // 控制
    stopCh chan struct{}
    wg     sync.WaitGroup
}

// MetricsPersistConfig 配置
type MetricsPersistConfig struct {
    SampleInterval  time.Duration
    RetentionDays   int
    CleanupInterval time.Duration
}

// NewMetricsPersistService 创建服务
func NewMetricsPersistService(
    store datahub.Store,
    metricsRepo database.NodeMetricsRepository,
    cfg MetricsPersistConfig,
) *MetricsPersistService {
    return &MetricsPersistService{
        store:           store,
        metricsRepo:     metricsRepo,
        sampleInterval:  cfg.SampleInterval,
        retentionDays:   cfg.RetentionDays,
        cleanupInterval: cfg.CleanupInterval,
        lastSample:      make(map[string]time.Time),
        stopCh:          make(chan struct{}),
    }
}

// Start 启动服务
func (s *MetricsPersistService) Start() error {
    // 启动清理协程
    s.wg.Add(1)
    go s.cleanupLoop()

    metricsLog.Info("NodeMetrics 持久化服务已启动")
    return nil
}

// Stop 停止服务
func (s *MetricsPersistService) Stop() error {
    close(s.stopCh)
    s.wg.Wait()
    metricsLog.Info("NodeMetrics 持久化服务已停止")
    return nil
}

// Sync 同步指定集群的 NodeMetrics 到 SQLite
// 由快照到达时触发调用（通过 Processor 回调）
func (s *MetricsPersistService) Sync(clusterID string) error {
    // 1. 从 DataHub 获取当前集群的 Snapshot
    snapshot, err := s.store.GetSnapshot(clusterID)
    if err != nil {
        return err
    }
    if snapshot == nil || snapshot.NodeMetrics == nil {
        return nil
    }

    // 2. 遍历所有节点的 NodeMetrics
    for nodeName, metrics := range snapshot.NodeMetrics {
        // 2.1 更新实时数据（每次都更新）
        if err := s.metricsRepo.UpsertLatest(clusterID, nodeName, metrics); err != nil {
            metricsLog.Error("更新实时数据失败",
                "cluster", clusterID,
                "node", nodeName,
                "err", err,
            )
            continue
        }

        // 2.2 检查是否需要采样趋势数据
        if s.shouldSample(clusterID, nodeName) {
            point := s.snapshotToDataPoint(metrics)
            if err := s.metricsRepo.InsertHistory(clusterID, nodeName, point); err != nil {
                metricsLog.Error("插入趋势数据失败",
                    "cluster", clusterID,
                    "node", nodeName,
                    "err", err,
                )
            }
        }
    }

    metricsLog.Debug("NodeMetrics 同步完成",
        "cluster", clusterID,
        "nodes", len(snapshot.NodeMetrics),
    )
    return nil
}

// shouldSample 判断是否应该采样（5 分钟间隔）
func (s *MetricsPersistService) shouldSample(clusterID, nodeName string) bool {
    key := clusterID + ":" + nodeName
    now := time.Now()

    s.lastSampleMu.RLock()
    last, ok := s.lastSample[key]
    s.lastSampleMu.RUnlock()

    if !ok || now.Sub(last) >= s.sampleInterval {
        s.lastSampleMu.Lock()
        s.lastSample[key] = now
        s.lastSampleMu.Unlock()
        return true
    }
    return false
}

// snapshotToDataPoint 提取关键指标
func (s *MetricsPersistService) snapshotToDataPoint(m *model_v2.NodeMetricsSnapshot) *model_v2.MetricsDataPoint {
    var diskUsage float64
    if len(m.Disks) > 0 {
        diskUsage = m.Disks[0].UsagePercent
    }

    var netRx, netTx float64
    for _, n := range m.Networks {
        netRx += n.RxBytesPS
        netTx += n.TxBytesPS
    }

    return &model_v2.MetricsDataPoint{
        Timestamp:    m.Timestamp.UnixMilli(),
        CPUUsage:     m.CPU.UsagePercent,
        MemUsage:     m.Memory.UsagePercent,
        DiskUsage:    diskUsage,
        Temperature:  m.Temperature.CPUTemp,
        NetRxBytesPS: netRx,
        NetTxBytesPS: netTx,
    }
}

// cleanupLoop 定期清理过期趋势数据
func (s *MetricsPersistService) cleanupLoop() {
    defer s.wg.Done()

    ticker := time.NewTicker(s.cleanupInterval)
    defer ticker.Stop()

    for {
        select {
        case <-s.stopCh:
            return
        case <-ticker.C:
            s.cleanup()
        }
    }
}

// cleanup 执行清理
func (s *MetricsPersistService) cleanup() {
    cutoff := time.Now().AddDate(0, 0, -s.retentionDays)
    deleted, err := s.metricsRepo.CleanupHistory(cutoff)
    if err != nil {
        metricsLog.Error("清理过期数据失败", "err", err)
        return
    }
    if deleted > 0 {
        metricsLog.Info("已清理过期趋势数据", "deleted", deleted)
    }
}
```

### 4.2 Processor 回调配置

```go
// processor/processor.go  [微调]
// 只需在现有 onSnapshotReceived 回调中添加 MetricsPersistService.Sync 调用

// 现有代码已有回调机制：
// if p.onSnapshotReceived != nil {
//     p.onSnapshotReceived(clusterID)
// }

// 初始化时配置回调（在 master.go 中）：
processor := processor.New(processor.Config{
    Store: store,
    OnSnapshotReceived: func(clusterID string) {
        // 现有：Event 持久化
        eventPersistService.Sync(clusterID)
        // 新增：NodeMetrics 持久化
        metricsPersistService.Sync(clusterID)
    },
})
```

---

## 5. Gateway 层 - API Handler

> Gateway Handler 直接使用 `database.NodeMetricsRepository` 接口查询。
> NodeMetrics 数据存储在 SQLite，不走 service/query（service/query 用于读取 DataHub）。

### 5.1 Handler 实现

```go
// gateway/handler/node_metrics.go  [新增]

package handler

import (
    "strconv"
    "time"

    "github.com/gin-gonic/gin"

    "AtlHyper/atlhyper_master_v2/database"
    "AtlHyper/model_v2"
)

type NodeMetricsHandler struct {
    repo database.NodeMetricsRepository  // 直接使用 database 层接口
}

func NewNodeMetricsHandler(repo database.NodeMetricsRepository) *NodeMetricsHandler {
    return &NodeMetricsHandler{repo: repo}
}

// GetClusterNodeMetrics 获取集群所有节点指标
// GET /api/v2/clusters/{clusterId}/node-metrics
func (h *NodeMetricsHandler) GetClusterNodeMetrics(c *gin.Context) {
    clusterId := c.Param("clusterId")

    // 获取所有节点实时数据
    nodes, err := h.repo.ListLatest(clusterId)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // 计算汇总统计
    summary := h.calculateSummary(nodes)

    c.JSON(200, gin.H{
        "summary": summary,
        "nodes":   nodes,
    })
}

// GetNodeMetricsDetail 获取单节点详情
// GET /api/v2/clusters/{clusterId}/node-metrics/{nodeName}
func (h *NodeMetricsHandler) GetNodeMetricsDetail(c *gin.Context) {
    clusterId := c.Param("clusterId")
    nodeName := c.Param("nodeName")

    latest, err := h.repo.GetLatest(clusterId, nodeName)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    if latest == nil {
        c.JSON(404, gin.H{"error": "node not found"})
        return
    }

    c.JSON(200, latest)
}

// GetNodeMetricsHistory 获取节点趋势数据
// GET /api/v2/clusters/{clusterId}/node-metrics/{nodeName}/history?hours=24
func (h *NodeMetricsHandler) GetNodeMetricsHistory(c *gin.Context) {
    clusterId := c.Param("clusterId")
    nodeName := c.Param("nodeName")
    hoursStr := c.DefaultQuery("hours", "24")

    hours, _ := strconv.Atoi(hoursStr)
    if hours <= 0 || hours > 720 { // 最多 30 天
        hours = 24
    }

    end := time.Now()
    start := end.Add(-time.Duration(hours) * time.Hour)

    history, err := h.repo.GetHistory(clusterId, nodeName, start, end)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{
        "node_name": nodeName,
        "hours":     hours,
        "points":    len(history),
        "history":   history,
    })
}

// calculateSummary 计算集群汇总统计
func (h *NodeMetricsHandler) calculateSummary(nodes map[string]*model_v2.NodeMetricsSnapshot) *model_v2.ClusterMetricsSummary {
    summary := &model_v2.ClusterMetricsSummary{}
    var totalCPU, totalMem float64

    for nodeName, m := range nodes {
        summary.MetricsNodes++
        totalCPU += m.CPU.UsagePercent
        totalMem += m.Memory.UsagePercent
        summary.TotalCores += m.CPU.CoreCount
        summary.TotalMemBytes += m.Memory.TotalBytes
        summary.UsedMemBytes += m.Memory.UsedBytes

        // 最高温度
        if m.Temperature.CPUTemp > summary.MaxTemp {
            summary.MaxTemp = m.Temperature.CPUTemp
            summary.MaxTempNode = nodeName
        }

        // 最高磁盘使用率
        for _, disk := range m.Disks {
            if disk.UsagePercent > summary.MaxDisk {
                summary.MaxDisk = disk.UsagePercent
                summary.MaxDiskNode = nodeName
                summary.MaxDiskMount = disk.MountPoint
            }
        }

        // 告警节点
        if m.CPU.UsagePercent >= 80 || m.Memory.UsagePercent >= 80 || m.Temperature.CPUTemp >= 75 {
            summary.WarningNodes++
        }
    }

    if summary.MetricsNodes > 0 {
        summary.AvgCPU = totalCPU / float64(summary.MetricsNodes)
        summary.AvgMemory = totalMem / float64(summary.MetricsNodes)
    }

    return summary
}
```

### 5.2 路由注册

```go
// gateway/routes.go  [修改]

func SetupRoutes(r *gin.Engine, handlers *Handlers) {
    // ... 现有路由 ...

    // Node Metrics API [新增]
    clusters := r.Group("/api/v2/clusters/:clusterId")
    {
        clusters.GET("/node-metrics", handlers.NodeMetrics.GetClusterNodeMetrics)
        clusters.GET("/node-metrics/:nodeName", handlers.NodeMetrics.GetNodeMetricsDetail)
        clusters.GET("/node-metrics/:nodeName/history", handlers.NodeMetrics.GetNodeMetricsHistory)
    }
}
```

### 5.3 API 响应示例

**GET /api/v2/clusters/xxx/node-metrics**

```json
{
  "summary": {
    "metrics_nodes": 3,
    "avg_cpu": 45.2,
    "avg_memory": 62.5,
    "max_temp": 68.0,
    "max_temp_node": "k8s-worker-01",
    "max_disk": 70.0,
    "max_disk_node": "k8s-worker-01",
    "max_disk_mount": "/",
    "warning_nodes": 1,
    "total_cores": 24,
    "total_mem_bytes": 103079215104,
    "used_mem_bytes": 64424509440
  },
  "nodes": {
    "k8s-worker-01": { ... },
    "k8s-worker-02": { ... }
  }
}
```

**GET /api/v2/clusters/xxx/node-metrics/k8s-worker-01/history?hours=24**

```json
{
  "node_name": "k8s-worker-01",
  "hours": 24,
  "points": 288,
  "history": [
    {
      "timestamp": 1705660800000,
      "cpu_usage": 45.2,
      "mem_usage": 62.1,
      "disk_usage": 55.0,
      "temperature": 58.5,
      "net_rx_bytes_ps": 1024000,
      "net_tx_bytes_ps": 512000
    }
    // ... 288 个数据点 (24 小时 × 12 点/小时)
  ]
}
```

---

## 7. 文件清单

### 7.1 改动对照表

| 层级 | 文件 | 操作 | 说明 |
|------|------|------|------|
| **Database** | `database/interfaces.go` | 修改 | 添加 NodeMetrics Model/Repository/Dialect 接口 |
| **Database** | `database/sqlite/node_metrics.go` | **新增** | NodeMetricsDialect 实现（SQL 生成） |
| **Database** | `database/repo/node_metrics.go` | **新增** | NodeMetricsRepo 实现（业务逻辑） |
| **Service** | `service/sync/metrics_persist.go` | **新增** | MetricsPersistService（从 DataHub 读取 → SQLite） |
| **Processor** | `processor/processor.go` | 微调 | 回调中添加 MetricsPersistService.Sync |
| **Gateway** | `gateway/handler/node_metrics.go` | **新增** | API Handler（直接使用 database 层接口） |
| **Gateway** | `gateway/routes.go` | 修改 | 注册 node-metrics 路由 |
| **Main** | `master.go` | 修改 | 初始化并注入 NodeMetricsRepo |
| **Model** | `model_v2/node_metrics.go` | **新增** | 共用数据模型 |

---

## 8. 注意事项

### 8.1 DataHub 不变

DataHub 保持**信封结构**不变：
- 只存储最新的完整 ClusterSnapshot
- NodeMetrics 随 Snapshot 一起存储
- 不单独处理 NodeMetrics

### 8.2 SQLite 配置

```go
// config/types.go
type Config struct {
    // ...
    MetricsDBPath string  // SQLite 文件路径，默认 "./data/metrics.db"
}
```

### 8.3 采样策略

| 数据类型 | 更新频率 | 存储方式 |
|----------|----------|----------|
| 实时数据 | 5s（每次上报） | 覆盖更新 |
| 趋势数据 | 5min 采样 | 追加插入 |

### 8.4 数据保留

- **实时数据**：只保留最新 1 条/节点
- **趋势数据**：保留 30 天，每小时清理一次过期数据

### 8.5 并发安全

- SQLite 使用 WAL 模式，支持并发读写
- `lastSample` map 使用 `sync.RWMutex` 保护
