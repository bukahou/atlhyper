# 节点指标数据策略分析

> 按页面视图 + API 端点分析 Metrics 数据的存储与获取策略

---

## 一、页面结构与数据需求

### 页面: `/observe/metrics/`

```
Metrics 页面
├── 集群概览卡片（节点总数、平均 CPU、平均内存、最高温度、告警数）
├── 集群趋势图（ClusterOverviewChart）
│   └── 多节点多指标（CPU/Mem/Disk/Temp）时序可视化
│   └── 窗口切换: 1h / 6h / 1d / 7d
├── 节点列表
│   └── 每个节点可展开详情
│       ├── ResourceChart（CPU/Mem/Disk/Temp 四条趋势线）
│       ├── CPU / Memory / Disk / Network 详情卡
│       ├── Temperature / PSI / TCP 详情卡
│       └── System / VMStat 详情卡
└── 自动刷新（10s）
```

**关键特征**: 趋势图是核心交互，需要大量时序数据。

---

## 二、API 端点分析

### 2.1 集群汇总 + 节点列表（首屏加载）

| 项目 | 内容 |
|------|------|
| **端点** | `GET /api/v2/observe/metrics/summary` + `GET /api/v2/observe/metrics/nodes` |
| **Handler** | `observe.go::MetricsSummary()` + `observe.go::MetricsNodes()` |
| **数据源** | `OTelSnapshot.MetricsSummary` + `OTelSnapshot.MetricsNodes` |
| **响应** | 汇总统计 + 全部节点快照（CPU/Mem/Disk/Net/Temp/PSI/TCP/...） |
| **延迟** | <10ms（内存直读） |

**数据特征**: 这是**当前时刻的快照**，不是时序数据。每个节点约 40+ 字段。

### 2.2 节点历史时序（趋势图 — 核心需求）

| 项目 | 内容 |
|------|------|
| **端点** | `GET /api/v2/node-metrics/{name}/history?hours=N` |
| **Handler** | `node_metrics.go::getHistory()` |
| **三层路由** | 见下表 |

| 时间范围 | 数据源 | 粒度 | 点数 | 延迟 |
|---------|--------|------|------|------|
| ≤15min | **OTelTimeline Ring Buffer** | 10s | ~90 | <100ms |
| ≤60min | **Concentrator 预聚合** (`NodeMetricsSeries`) | 1min | ~60 | <50ms |
| >60min | **Command → Agent → ClickHouse** | 1min/5min/15min | 60-360 | 2-5s |

### 2.3 集群趋势图（ClusterOverviewChart）

| 窗口 | 粒度 | 数据源 |
|------|------|--------|
| 1h | 1min | Concentrator 预聚合 |
| 6h | 5min | ClickHouse |
| 1d | 15min | ClickHouse |
| 7d | 1h | ClickHouse |

---

## 三、当前存储现状

### OTelSnapshot 中的 Metrics 数据

```go
OTelSnapshot {
    // 当前快照
    MetricsSummary   *metrics.Summary         // 集群汇总（6 个数值）
    MetricsNodes     []metrics.NodeMetrics    // 节点列表（每节点 40+ 字段）

    // 预聚合时序（Concentrator 生成，1min 粒度 × 60 点）
    NodeMetricsSeries []NodeMetricsSeries {
        NodeName string
        Points   []NodeMetricsPoint           // 每点 25 个字段
    }
}
```

### Ring Buffer 存储（90 份）

每 10 秒一份完整 OTelSnapshot → 90 份中每份都包含：
- `MetricsSummary` — 6 个数值
- `MetricsNodes[]` — N 个节点 × 40+ 字段
- `NodeMetricsSeries[]` — N 个节点 × 60 个点 × 25 个字段

### 内存开销估算（6 节点集群）

```
单份 NodeMetricsSeries:
  6 节点 × 60 点 × 25 字段 × 8 bytes ≈ 72 KB

90 份 Ring Buffer 中的 NodeMetricsSeries:
  72 KB × 90 ≈ 6.3 MB  ← 这部分冗余最大

90 份 Ring Buffer 中的 MetricsNodes:
  6 节点 × 40 字段 × 8 bytes × 90 ≈ 170 KB
```

---

## 四、问题分析

### 4.1 Ring Buffer 中的冗余

| 字段 | 90 份是否必要 | 分析 |
|------|-------------|------|
| `MetricsSummary` | **不需要** | 汇总统计，只需最新 1 份 |
| `MetricsNodes[]` | **有用但冗余** | Ring Buffer 中按节点提取趋势用，但每份都是完整快照 |
| `NodeMetricsSeries[]` | **完全冗余** | 每份都含完整 60min 预聚合，90 份几乎相同 |

### 4.2 趋势图的真正需求

趋势图需要的是**按节点聚合的时序点**，而不是 90 份完整 OTelSnapshot。

当前 `getHistory()` 对 ≤15min 的处理方式:

```go
// node_metrics.go::buildNodeHistoryFromTimeline()
// 从 90 份 OTelSnapshot 中，逐个提取目标节点的 MetricsNodes 数据
// 每份取 1 个节点的快照值 → 组成 10s 粒度的时序
for _, entry := range timeline {
    for _, node := range entry.Snapshot.MetricsNodes {
        if node.NodeName == targetNode {
            appendPoint(entry.Timestamp, node.CPU.UsagePct)
        }
    }
}
```

**问题**: 为了提取 1 个节点的 1 个值，要遍历整份 OTelSnapshot（包含 SLO/APM/Logs 等所有数据）。

### 4.3 Concentrator 与 Ring Buffer 的重叠

- **Concentrator** (`NodeMetricsSeries`): 1min 粒度 × 60 点，覆盖最近 1h
- **Ring Buffer**: 10s 粒度 × 90 条，覆盖最近 15min
- **重叠区间**: 0-15min 同时被两者覆盖

---

## 五、优化方案讨论

### 方案 A: Ring Buffer 只存轻量指标切片

```go
// 新的轻量时间线条目（替代完整 OTelSnapshot）
type MetricsTimelineEntry struct {
    Timestamp time.Time
    Nodes     []NodeMetricsSlim  // 每节点只存关键指标
}

type NodeMetricsSlim struct {
    NodeName string
    CPUPct   float64
    MemPct   float64
    DiskPct  float64
    TempC    float64
    // ... 趋势图需要的 ~10 个关键字段
}
```

**优点**: Ring Buffer 从 ~6MB 降到 ~100KB
**缺点**: 需要改 Ring Buffer 结构，影响所有消费者

### 方案 B: 独立的 Metrics 时间线

```go
// 与 OTelRing 分离，独立维护 Metrics 时间线
type MetricsTimeline struct {
    entries  []MetricsTimelineEntry  // 15min / 10s = 90 条
    // 只存节点指标快照，不存 SLO/APM/Logs
}
```

**优点**: 不影响现有 OTelRing 结构
**缺点**: 新增存储结构

### 方案 C: 扩展 Concentrator 覆盖 15min

当前 Concentrator 已覆盖 60min（1min 粒度），如果前端趋势图可以接受 1min 粒度，
则 ≤15min 的请求可以直接从 Concentrator 获取，**不再需要 Ring Buffer 中的 Metrics 数据**。

```
当前三层:
  ≤15min → Ring Buffer (10s)
  ≤60min → Concentrator (1min)
  >60min → ClickHouse

优化后两层:
  ≤60min → Concentrator (1min)      ← 合并前两层
  >60min → ClickHouse
```

**优点**: 最简单，无需新增结构
**缺点**: 15min 内的粒度从 10s 降为 1min（趋势图差异不大）

---

## 六、结论

### Metrics 数据特征总结

| 特征 | 结论 |
|------|------|
| **核心需求** | 时序趋势图（大量数据点） |
| **首屏加载** | 当前快照（MetricsSummary + MetricsNodes）— 只需最新 1 份 |
| **趋势图** | 需要时间线数据，但不需要 90 份完整 OTelSnapshot |
| **自动刷新** | 10s 间隔，需要最新快照 |
| **长时间范围** | >1h 走 ClickHouse（Command 机制） |
| **跨信号关联** | 无（Metrics 独立） |

### Ring Buffer 中 Metrics 的处理建议

1. **`MetricsSummary`**: 不需要 90 份，只需最新 1 份
2. **`MetricsNodes[]`**: 趋势图需要历史值，但可以用 Concentrator 替代
3. **`NodeMetricsSeries[]`**: 完全不需要 90 份（每份都是独立完整的 60min 时序）

### 推荐方案

**方案 C（扩展 Concentrator）** 最为简洁：

- 前端趋势图 1min 粒度 vs 10s 粒度，视觉差异极小
- 无需新增数据结构
- Ring Buffer 中不再需要存储 Metrics 相关字段
- Concentrator 已经在 Agent 侧预聚合好，Master 直读即可

---

## 七、文件结构分析

### 7.1 当前 Metrics 文件分布

```
=== 前端 ===

atlhyper_web/src/
├── app/observe/metrics/
│   ├── page.tsx                              # Metrics 主页面（集群概览 + 节点列表）
│   └── components/
│       ├── ClusterOverviewChart.tsx           # 集群概览趋势图
│       ├── ResourceChart.tsx                  # 节点资源趋势图
│       ├── NodeSelector.tsx                   # 节点选择器
│       ├── CPUCard.tsx                        # CPU 详情卡
│       ├── MemoryCard.tsx                     # 内存详情卡
│       ├── DiskCard.tsx                       # 磁盘详情卡
│       ├── NetworkCard.tsx                    # 网络详情卡
│       ├── GPUCard.tsx                        # GPU 详情卡
│       ├── TemperatureCard.tsx                # 温度详情卡
│       ├── PSICard.tsx                        # 压力信息卡
│       ├── TCPCard.tsx                        # TCP 统计卡
│       ├── VMStatCard.tsx                     # 虚拟内存统计卡
│       ├── SystemResourcesCard.tsx            # 系统资源卡
│       ├── ProcessTable.tsx                   # 进程表
│       └── index.ts                           # 组件导出
├── types/node-metrics.ts                      # Metrics TypeScript 类型定义
├── api/node-metrics.ts                        # ✅ 独立 API 文件（节点指标专用）
├── api/observe.ts                             # ⚠️ 混合：MetricsSummary/MetricsNodes API 也在此
├── datasource/metrics.ts                      # Metrics 数据源适配
├── mock/metrics/
│   ├── index.ts                               # Mock 导出
│   ├── data.ts                                # Mock 数据
│   └── queries.ts                             # Mock 查询
└── config/data-source.ts                      # 数据源开关

=== Master 后端 ===

atlhyper_master_v2/
├── gateway/handler/
│   ├── node_metrics.go                        # ✅ 独立 Handler（节点指标历史 3 层路由）
│   ├── observe_metrics.go                     # Observe Metrics Handler（Summary/Nodes/Series）
│   ├── observe_timeline.go                    # 时序辅助函数（buildNodeMetricsSeries）
│   └── observe.go                             # ⚠️ 共用基础：TTL 缓存 + executeQuery()
├── gateway/routes.go                          # 路由注册
├── model/node_metrics.go                      # API 响应模型
├── model/convert/
│   ├── node_metrics.go                        # model_v3 → API 响应转换
│   └── node_metrics_test.go                   # 转换测试
└── service/query/
    ├── otel.go                                # ⚠️ 混合：GetOTelSnapshot()
    └── metrics_format.go                      # Metrics 格式化工具

=== Agent 后端 ===

atlhyper_agent_v2/
├── sdk/impl/k8s/metrics.go                   # K8s Metrics API 客户端
├── repository/
│   ├── interfaces.go                          # MetricsQueryRepository 接口定义
│   └── ch/
│       ├── query/metrics.go                   # ClickHouse 节点指标查询实现
│       └── dashboard.go                       # ⚠️ 混合：OTelDashboardRepository
└── service/snapshot/
    └── snapshot.go                            # ⚠️ 混合：getOTelSnapshot() 采集 4 信号域

=== 共享模型 ===

model_v2/node_metrics.go                       # Agent 上报格式（snake_case）
model_v3/metrics/node_metrics.go               # Master API 格式（camelCase）
model_v3/cluster/snapshot.go                   # ⚠️ 混合：OTelSnapshot 含 MetricsSummary/MetricsNodes/NodeMetricsSeries
```

### 7.2 耦合问题

#### 问题一：两套 Metrics API 端点并存

Metrics 拥有两套独立的 API 路径和 Handler，返回相似数据：

| 端点 | Handler | 数据源 | 用途 |
|-----|---------|--------|------|
| `/api/v2/node-metrics/*` | `NodeMetricsHandler` | OTelSnapshot + Ring Buffer + Concentrator + ClickHouse | 节点指标历史（3 层路由） |
| `/api/v2/observe/metrics/*` | `ObserveHandler.MetricsSummary/MetricsNodes` | OTelSnapshot 直读 | Observe 页面首屏 |

**影响**:
- 两个 Handler 独立维护，但底层数据源相同
- `node_metrics.go` 有自己的 3 层路由逻辑（Ring Buffer → Concentrator → ClickHouse）
- `observe_metrics.go` 也有类似的 3 层降级逻辑
- 修改数据源时需要同步更新两处

**解决方案**: 保留两套 API，统一底层实现。

两套 API 存在的原因不同：
- `/api/v2/node-metrics/*` — 独立 Handler，由 cluster 详情页使用，需要 Convert 层转换
- `/api/v2/observe/metrics/*` — ObserveHandler 方法，由 Observe 页面使用，返回原始 model_v3 格式

**不合并的理由**: 两者的响应模型不同（`master/model.NodeMetricsSnapshot` vs `model_v3/metrics.NodeMetrics`），强行合并需要前端改动。

**实际可做的优化**: 将两者共用的 3 层数据源路由逻辑提取为共享函数：

```go
// gateway/handler/metrics_datasource.go（新增）

// resolveNodeSeries 统一 3 层数据源路由（Ring Buffer → Concentrator → Command）
// 由 node_metrics.go 和 observe_metrics.go 共同调用
func resolveNodeSeries(
    querySvc service.Query,
    ctx context.Context, clusterID, nodeName, metric string,
    minutes int,
) (interface{}, error) {
    // 层 1: Ring Buffer（≤15min）
    if minutes <= 15 { ... }
    // 层 2: Concentrator（≤60min）
    ...
    // 层 3: Command（>60min）
    ...
}
```

**效果**: 消除两个 Handler 中重复的 3 层路由逻辑，修改数据源只需改一处。

#### 问题二：`observe_timeline.go` 辅助函数归属不清

`observe_timeline.go` 包含 `buildNodeMetricsSeries()` 等辅助函数，专门为 Metrics 时序构建服务，但文件名暗示通用"时间线"用途。

**影响**:
- 新开发者难以判断这是 Metrics 专用还是通用辅助
- 实际上只被 `observe_metrics.go` 和 `node_metrics.go` 使用

**解决方案**: 重命名为 `observe_metrics_helpers.go`，明确归属。

```
当前: observe_timeline.go    → 暗示通用"时间线"
改为: observe_metrics_helpers.go → 明确是 Metrics 辅助函数

内含函数:
  buildNodeMetricsSeries()      — 从 Ring Buffer 构建节点时序
  filterNodePointsByMinutes()   — 按时间窗口过滤 Concentrator 数据
  extractNodeMetricPoints()     — 从预聚合点中提取指定指标
```

如果后续实施问题一的方案（提取 `metrics_datasource.go`），则这些辅助函数可以合并到该文件中。

#### 问题三：Agent `snapshot.go` 混合采集

同 APM/Logs — `getOTelSnapshot()` 中混合了 `GetMetricsSummary()` 和 `ListAllNodeMetrics()` 的采集调用。

**解决方案**: 见 APM 数据策略文档 6.2 问题二的方案 — 提取 `otel_collector.go`。Metrics 采集逻辑（2 个调用：`GetMetricsSummary` + `ListAllNodeMetrics`）跟随 `otel_collector.go` 即可，无需独立文件。

#### 问题四：`api/observe.ts` 与 `api/node-metrics.ts` 分裂

前端有两个 API 文件涉及 Metrics:
- `api/node-metrics.ts` — 独立的节点指标 API（历史时序）
- `api/observe.ts` — 混合文件中的 `getMetricsSummary()` + `getMetricsNodes()` + `getMetricsNodeSeries()`

**解决方案**: 将 `api/observe.ts` 中的 Metrics 函数合并到 `api/node-metrics.ts`：

```
迁出函数:
  getMetricsSummary()       → api/node-metrics.ts
  getMetricsNodes()         → api/node-metrics.ts
  getMetricsNode()          → api/node-metrics.ts
  getMetricsNodeSeries()    → api/node-metrics.ts

更新导入:
  datasource/metrics.ts 的 import 从 "@/api/observe" → "@/api/node-metrics"
  app/observe/metrics/page.tsx 的 import 同步更新
```

### 7.3 理想文件结构（整理后）

```
=== 前端（基本达标，API 层可整合） ===

atlhyper_web/src/
├── app/observe/metrics/
│   ├── page.tsx                               # 页面
│   └── components/*.tsx                       # 组件（15 个）
├── types/node-metrics.ts                      # 类型
├── api/node-metrics.ts                        # ← 统一为一个 API 文件（合并 observe.ts 中的 Metrics 部分）
├── datasource/metrics.ts                      # 数据源
└── mock/metrics/*.ts                          # Mock（3 个）

=== Master 后端（需统一 Handler） ===

atlhyper_master_v2/
├── gateway/handler/
│   ├── node_metrics.go                        # 保持（或合并到 observe_metrics.go）
│   ├── observe_metrics.go                     # Metrics Handler（含时序辅助函数）
│   └── observe.go                             # 共用基础
├── model/node_metrics.go                      # 响应模型
├── model/convert/node_metrics.go              # 转换
└── service/query/
    ├── otel.go                                # OTel 快照查询
    └── metrics_format.go                      # 格式化工具

=== Agent 后端（需拆分 snapshot.go） ===

atlhyper_agent_v2/
├── sdk/impl/k8s/metrics.go                   # SDK（独立 ✅）
├── repository/
│   ├── interfaces.go                          # 保持 MetricsQueryRepository 独立
│   └── ch/query/metrics.go                    # ClickHouse 实现（独立 ✅）
└── service/snapshot/
    ├── snapshot.go                            # 通用快照编排
    └── otel_collector.go                      # ← OTel 采集逻辑

=== 共享模型（已达标） ===

model_v2/node_metrics.go                       # Agent 上报格式
model_v3/metrics/node_metrics.go               # API 格式
```

### 7.4 整理检查清单

| 检查项 | 当前 | 目标 |
|--------|------|------|
| 前端 Metrics 页面/组件是否独立 | ✅ 已隔离 | 无需修改 |
| 前端 Metrics 类型是否独立 | ✅ `types/node-metrics.ts` 独立 | 无需修改 |
| 前端 API 调用是否独立 | ⚠️ 分裂在 `api/node-metrics.ts` + `api/observe.ts` | 可合并（优先级低） |
| Master Handler 是否独立 | ⚠️ 两套并存 (`node_metrics.go` + `observe_metrics.go`) | 评估合并可行性 |
| Master Model/Convert 是否独立 | ✅ 独立 | 无需修改 |
| Agent 查询层是否独立 | ✅ `ch/query/metrics.go` 独立 | 无需修改 |
| Agent 快照采集是否独立 | ❌ 混在 `snapshot.go` 中 | 拆分 OTel 采集逻辑 |
| 共享模型是否独立 | ✅ `model_v3/metrics/` 独立 | 无需修改 |
