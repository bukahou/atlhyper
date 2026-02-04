# Node Metrics 硬件指标监控 - 总体设计

## 1. 概述

实现 K8s 集群节点级别的硬件指标监控，包括：
- CPU 使用率、负载、每核使用率
- 内存使用率、Swap、缓存
- 磁盘使用率、I/O 速率、IOPS
- 网络流量、包数、错误/丢包
- 温度（CPU、传感器）
- Top 进程列表

### 1.1 技术方案

采用 **DataHub + SQLite 混合存储** 方案：

| 项目 | 说明 |
|------|------|
| **DataHub** | 保持信封结构，存储完整 ClusterSnapshot |
| **SQLite 实时** | 每节点一条，覆盖更新，展示当前详细状态 |
| **SQLite 趋势** | 5 分钟采样，保留 30 天，用于历史趋势图 |
| **服务重启** | 历史数据持久化，不丢失 |

### 1.2 设计文档索引

| 组件 | 设计文档 | 说明 |
|------|----------|------|
| **Metrics 采集器** | [atlhyper-metrics-v2.md](atlhyper-metrics-v2.md) | DaemonSet，读取 procfs/sysfs |
| **Agent 端** | [node-metrics-agent.md](node-metrics-agent.md) | 接收、存储、聚合到快照 |
| **Master 端** | [node-metrics-master.md](node-metrics-master.md) | DataHub、Query、API |
| **任务追踪** | [node-metrics-tasks.md](../tasks/node-metrics-tasks.md) | 开发进度 |

---

## 2. 整体架构

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                   K8s Cluster                                    │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │  K8s Node 1                         K8s Node 2          K8s Node N      │   │
│   │  ┌───────────────────┐              ┌─────────────┐     ┌─────────────┐ │   │
│   │  │atlhyper_metrics_v2│              │   metrics   │     │   metrics   │ │   │
│   │  │   (DaemonSet)     │              │ (DaemonSet) │ ... │ (DaemonSet) │ │   │
│   │  │                   │              │             │     │             │ │   │
│   │  │ 读取 /proc /sys   │              │             │     │             │ │   │
│   │  │ CPU/Mem/Disk/Net  │              │             │     │             │ │   │
│   │  │ Temp/Process      │              │             │     │             │ │   │
│   │  └─────────┬─────────┘              └──────┬──────┘     └──────┬──────┘ │   │
│   │            │                               │                   │        │   │
│   └────────────┼───────────────────────────────┼───────────────────┼────────┘   │
│                │ HTTP POST                     │                   │            │
│                │ /metrics/node                 │                   │            │
│                ▼                               ▼                   ▼            │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │                         atlhyper_agent (单实例)                          │   │
│   │                                                                         │   │
│   │  ┌─────────────────────────────────────────────────────────────────┐   │   │
│   │  │  Gateway                                                         │   │   │
│   │  │  POST /metrics/node  ─►  接收各节点上报的 NodeMetricsSnapshot    │   │   │
│   │  └───────────────────────────────────────┬─────────────────────────┘   │   │
│   │                                          │                             │   │
│   │  ┌───────────────────────────────────────▼─────────────────────────┐   │   │
│   │  │  Repository                                                      │   │   │
│   │  │  map[nodeName]*NodeMetricsSnapshot  (每个节点一条，覆盖式存储)    │   │   │
│   │  └───────────────────────────────────────┬─────────────────────────┘   │   │
│   │                                          │                             │   │
│   │  ┌───────────────────────────────────────▼─────────────────────────┐   │   │
│   │  │  Service                                                         │   │   │
│   │  │  BuildSnapshot() ─► 聚合所有节点的 NodeMetrics 到 ClusterSnapshot │   │   │
│   │  └───────────────────────────────────────┬─────────────────────────┘   │   │
│   │                                          │                             │   │
│   └──────────────────────────────────────────┼─────────────────────────────┘   │
│                                              │                                  │
│                                              │ RESTful                          │
│                                              │ ClusterSnapshot + NodeMetrics    │
│                                              ▼                                  │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │                        atlhyper_master (单实例)                          │   │
│   │                                                                         │   │
│   │  ┌─────────────┐     ┌─────────────┐     ┌───────────────────────────┐ │   │
│   │  │  AgentSDK   │────►│  Processor  │────►│        DataHub            │ │   │
│   │  │  (接收)     │     │             │     │  (信封结构，存完整 Snapshot) │ │   │
│   │  └─────────────┘     │             │     └───────────────────────────┘ │   │
│   │                      │             │                                   │   │
│   │                      │             │     ┌───────────────────────────┐ │   │
│   │                      │             │────►│         SQLite            │ │   │
│   │                      │             │     │  latest: 实时数据（覆盖）   │ │   │
│   │                      │ (提取Metrics)│     │  history: 趋势（5min,30天）│ │   │
│   │                      └─────────────┘     └─────────────┬─────────────┘ │   │
│   │                                                        │               │   │
│   │  ┌─────────────────────────────────────────────────────▼─────────────┐ │   │
│   │  │  Service/Query                                                     │ │   │
│   │  │  实时数据: 从 SQLite latest 表读取                                  │ │   │
│   │  │  趋势数据: 从 SQLite history 表读取                                 │ │   │
│   │  └─────────────────────────────────────────────────────┬─────────────┘ │   │
│   │                                                        │               │   │
│   │  ┌─────────────────────────────────────────────────────▼─────────────┐ │   │
│   │  │  Gateway                                                           │ │   │
│   │  │  GET /api/v2/clusters/{id}/node-metrics                           │ │   │
│   │  │  GET /api/v2/clusters/{id}/node-metrics/{nodeName}                │ │   │
│   │  │  GET /api/v2/clusters/{id}/node-metrics/{nodeName}/history        │ │   │
│   │  └─────────────────────────────────────────────────────┬─────────────┘ │   │
│   │                                                        │               │   │
│   └────────────────────────────────────────────────────────┼───────────────┘   │
│                                                            │                    │
└────────────────────────────────────────────────────────────┼────────────────────┘
                                                             │ HTTP
                ▼
┌───────────────────────────────────────────────────────────────────────────────┐
│                              atlhyper_web                                      │
│                           /system/metrics                                      │
│                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │  概览卡片: 节点数 | CPU 均值 | 内存均值 | 最高温度 | 最高磁盘 | 告警节点  │  │
│  ├─────────────────────────────────────────────────────────────────────────┤  │
│  │  节点列表 (可滚动)                                                       │  │
│  │  ┌─────────────────────────────────────────────────────────────────┐   │  │
│  │  │ Node: k8s-worker-01                                             │   │  │
│  │  │ CPU | Memory | Disk | Network | Temperature | Processes         │   │  │
│  │  └─────────────────────────────────────────────────────────────────┘   │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. 组件职责

### 3.1 atlhyper_metrics_v2

| 项目 | 说明 |
|------|------|
| **部署方式** | DaemonSet（每节点一个 Pod） |
| **数据来源** | /proc, /sys（挂载宿主机目录） |
| **采集间隔** | 5 秒 |
| **输出** | HTTP POST 到 Agent（通过 Service 或 ClusterIP 访问） |

详细设计：[atlhyper-metrics-v2.md](atlhyper-metrics-v2.md)

### 3.2 atlhyper_agent

| 项目 | 说明 |
|------|------|
| **部署方式** | 单实例（Deployment，1 副本） |
| **接收接口** | POST /metrics/node |
| **存储方式** | Repository 层，按节点名存储（每节点保留最新一条） |
| **上报方式** | 聚合所有节点的 NodeMetrics 到 ClusterSnapshot，RESTful 上报 Master |

详细设计：[node-metrics-agent.md](node-metrics-agent.md)

### 3.3 atlhyper_master

| 项目 | 说明 |
|------|------|
| **DataHub** | 保持信封结构，存储完整 ClusterSnapshot |
| **SQLite** | 实时数据（覆盖更新）+ 趋势数据（5 分钟采样，30 天） |
| **数据处理** | Service/Query 层，适配 Web 需求 |
| **API** | Gateway 调用 Service 接口 |

详细设计：[node-metrics-master.md](node-metrics-master.md)

### 3.4 atlhyper_web

| 项目 | 说明 |
|------|------|
| **页面** | /system/metrics |
| **刷新** | 5 秒自动刷新 |
| **状态** | 前端 UI + Mock 已完成 |

---

## 4. 数据流

```
┌────────────────┐     ┌────────────────┐     ┌────────────────┐     ┌────────────┐
│    Metrics     │     │     Agent      │     │     Master     │     │    Web     │
│  (采集器)       │     │   (聚合)       │     │   (存储/API)   │     │  (展示)    │
└───────┬────────┘     └───────┬────────┘     └───────┬────────┘     └─────┬──────┘
        │                      │                      │                    │
        │ 读取 /proc /sys      │                      │                    │
        │ 组装 Snapshot        │                      │                    │
        │                      │                      │                    │
        │  POST /metrics/node  │                      │                    │
        │─────────────────────►│                      │                    │
        │                      │                      │                    │
        │                      │ Repository.Save()    │                    │
        │                      │ (覆盖式存储)          │                    │
        │                      │                      │                    │
        │                      │ RESTful 上报         │                    │
        │                      │ ClusterSnapshot      │                    │
        │                      │─────────────────────►│                    │
        │                      │                      │                    │
        │                      │                      │ Processor:         │
        │                      │                      │ 1. 存入 DataHub    │
        │                      │                      │ 2. 存入 SQLite     │
        │                      │                      │    - latest 覆盖   │
        │                      │                      │    - history 采样  │
        │                      │                      │                    │
        │                      │                      │     GET /node-metrics
        │                      │                      │◄───────────────────│
        │                      │                      │                    │
        │                      │                      │ Query 从 SQLite 读 │
        │                      │                      │ 返回 JSON          │
        │                      │                      │───────────────────►│
        │                      │                      │                    │
```

---

## 5. 数据模型

所有数据模型定义在 `model_v2/node_metrics.go`，三端共用：

| 模型 | 说明 | 使用场景 |
|------|------|----------|
| `NodeMetricsSnapshot` | 节点指标快照（完整数据） | Metrics→Agent→Master，SQLite latest 表 |
| `CPUMetrics` | CPU 指标 | 嵌套在 Snapshot |
| `MemoryMetrics` | 内存指标 | 嵌套在 Snapshot |
| `DiskMetrics` | 磁盘指标（数组） | 嵌套在 Snapshot |
| `NetworkMetrics` | 网络指标（数组） | 嵌套在 Snapshot |
| `TemperatureMetrics` | 温度指标 | 嵌套在 Snapshot |
| `ProcessMetrics` | 进程指标（数组） | 嵌套在 Snapshot |
| `MetricsDataPoint` | 历史数据点（简化版） | SQLite history 表，趋势图 |
| `ClusterMetricsSummary` | 集群汇总统计 | Master API 响应 |

---

## 6. API 设计

| Method | Path | 描述 | 响应 |
|--------|------|------|------|
| GET | `/api/v2/clusters/{clusterId}/node-metrics` | 集群所有节点实时指标 + 汇总 | `{ summary, nodes }` |
| GET | `/api/v2/clusters/{clusterId}/node-metrics/{nodeName}` | 单节点实时详情 | `NodeMetricsSnapshot` |
| GET | `/api/v2/clusters/{clusterId}/node-metrics/{nodeName}/history?hours=24` | 节点历史趋势（5min 采样） | `{ history: [] }` |

---

## 7. 分层架构总览

### 7.1 Metrics 端

```
Collector (采集层)
    │  读取 /proc /sys
    ▼
Aggregator (聚合层)
    │  组装 NodeMetricsSnapshot
    ▼
Pusher (推送层)
    │  HTTP POST 到 Agent
    ▼
Agent
```

### 7.2 Agent 端

```
Gateway (通信层)
    │  POST /metrics/node
    ▼
Repository (数据层)
    │  覆盖式存储
    ▼
Service (业务层)
    │  BuildSnapshot() 聚合到 ClusterSnapshot
    ▼
Master
```

### 7.3 Master 端

```
AgentSDK (接收层)
    │  接收 ClusterSnapshot
    ▼
Processor (处理层)
    │  1. 存入 DataHub（信封结构不变）
    │  2. 提取 NodeMetrics 存入 SQLite
    │     - 实时数据：覆盖更新（每次上报）
    │     - 趋势数据：追加插入（5 分钟采样）
    ▼
┌─────────────┬─────────────┐
│   DataHub   │   SQLite    │
│  (完整快照)  │ (实时+趋势)  │
└──────┬──────┴──────┬──────┘
       │             │
       ▼             ▼
Service/Query (查询层)
    │  实时数据：从 SQLite latest 表
    │  趋势数据：从 SQLite history 表
    ▼
Gateway (通信层)
    │  HTTP API
    ▼
Web
```

---

## 8. 后续扩展

### 8.1 告警集成

- 当指标超过阈值时触发告警
- 与现有告警系统集成

### 8.2 AI 分析

- 添加 AI Tool 支持查询节点指标
- 支持异常检测和容量预测

### 8.3 数据聚合（可选）

如果 30 天数据量过大，可以添加：
- 超过 7 天的数据按小时聚合
- 超过 30 天的数据按天聚合
- 减少存储空间
