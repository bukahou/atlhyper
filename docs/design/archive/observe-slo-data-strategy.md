# SLO 信号数据策略分析

> 按页面视图 + API 端点分析 SLO 数据的存储与获取策略

---

## 一、页面结构与数据需求

### 页面: `/observe/slo/`

```
SLO 页面
├── 汇总卡片（总服务数、监控域名、平均可用性、P95、总 RPS、告警数）
├── 时间范围切换（1d / 7d / 30d）
├── 域名卡片列表
│   └── 每个域名可展开 4 个标签页
│       ├── Overview（概览：域名级聚合指标 + 服务级列表）
│       ├── Mesh（服务网格拓扑 + 上下游调用关系）
│       ├── Compare（当前 vs 上期对比）
│       └── Latency（延迟分布：bucket + method + status code）
└── 自动刷新（30s）
```

---

## 二、API 端点分析

### 2.1 域名列表（首屏加载）

| 项目 | 内容 |
|------|------|
| **端点** | `GET /api/v2/slo/domains/v2?cluster_id=&time_range=1d` |
| **Handler** | `slo.go::DomainsV2()` |
| **延迟** | <10ms |

**数据源拆解**:

| 数据 | 来源 | 存储位置 | 说明 |
|------|------|---------|------|
| IngressSLO 指标 | Agent 从 **ClickHouse** 聚合 | **Master 内存** (`OTelSnapshot`) | Agent 定期查 CH → 随快照上报 → Master 内存直读 |
| 域名列表 | Agent 采集 K8s IngressRoute CRD | **SQLite** (`slo_route_mapping`) | Agent 发现 → Master UPSERT → Handler 查询 |
| 路由映射 (domain → serviceKey) | 同上 | **SQLite** (`slo_route_mapping`) | 域名 → 服务的关联关系 |
| SLO 目标配置 | 用户手动设置 | **SQLite** (`slo_targets`) | 可用性目标、延迟阈值 |

**数据流**:

```
1. Handler 从 Master 内存获取 OTelSnapshot
2. 优先取 SLOWindows[timeRange].Current (d/w/m 预聚合)
   → 数据源头: Agent 从 ClickHouse 聚合，TTL 缓存后随快照上报
   → 若无数据，回退到 SLOIngress (5min 窗口)
3. 从 SQLite 查询 GetAllDomains() → 获取所有真实域名
4. 对每个域名:
   → SQLite: GetRouteMappingsByDomain(domain) → 获取 serviceKey 列表
   → 用 serviceKey 从步骤 2 的 IngressSLO 中匹配指标数据
   → 聚合为域名级指标
5. 返回响应
```

### 2.2 延迟分布（标签页切换）

| 项目 | 内容 |
|------|------|
| **端点** | `GET /api/v2/slo/domains/latency?domain=&time_range=1d` |
| **Handler** | `slo_latency.go::LatencyDistribution()` |
| **延迟** | <10ms |

| 数据 | 来源 | 存储位置 |
|------|------|---------|
| LatencyBuckets / Methods / StatusCodes | Agent 从 **ClickHouse** `otel_metrics_histogram` 聚合 | **Master 内存** (`SLOWindows[timeRange].Current` 中 IngressSLO 的子字段) |
| 路由映射 | K8s IngressRoute | **SQLite** |

### 2.3 服务网格拓扑（标签页切换）

| 项目 | 内容 |
|------|------|
| **端点** | `GET /api/v2/slo/mesh/topology?cluster_id=&time_range=1d` |
| **Handler** | `slo_mesh.go::MeshTopology()` |
| **延迟** | <10ms |

| 数据 | 来源 | 存储位置 |
|------|------|---------|
| MeshServices (Linkerd) | Agent 从 **ClickHouse** `mv_linkerd_response_total` 聚合 | **Master 内存** (`SLOWindows[timeRange].MeshServices`) |
| MeshEdges (Linkerd outbound) | Agent 从 **ClickHouse** 聚合 | **Master 内存** (`SLOWindows[timeRange].MeshEdges`) |

### 2.4 历史趋势（DomainHistory）

| 项目 | 内容 |
|------|------|
| **端点** | `GET /api/v2/slo/domains/history?domain=&time_range=1d` |
| **Handler** | `slo.go::DomainHistory()` |
| **延迟** | <10ms |

| 数据 | 来源 | 存储位置 |
|------|------|---------|
| 时序数据点 (按 1h/6h/24h bucket) | Agent 从 **ClickHouse** `GetIngressSLOHistory()` 聚合 | **Master 内存** (`SLOWindows[timeRange].History`) |

### 2.5 数据源头总结

```
全部 API 的数据读取路径:

  前端 → Master Handler → Master 内存 (OTelSnapshot + SQLite)
                              ↑
                          Agent 定期上报
                              ↑
                Agent 从 ClickHouse 聚合（原始 OTel 指标）
                              ↑
                   Traefik / Linkerd → OTel Collector → ClickHouse

没有任何 SLO 端点直接查询 ClickHouse。
全部通过 Agent 预聚合 → Master 内存直读。
```

---

## 三、域名匹配机制分析

### 3.1 当前架构

```
域名发现:
  K8s IngressRoute CRD → Agent 采集 → ClusterSnapshot 上报 → Master UPSERT SQLite

域名关联:
  SQLite slo_route_mapping 表:
  ┌──────────┬──────────────┬──────────────────┬───────────────┐
  │ domain   │ path_prefix  │ service_key      │ service_name  │
  ├──────────┼──────────────┼──────────────────┼───────────────┤
  │ api.x.io │ /            │ geass-gateway-80 │ geass-gateway │
  │ api.x.io │ /ws          │ geass-gateway-8080│ geass-gateway│
  │ cdn.x.io │ /            │ geass-media-80   │ geass-media   │
  └──────────┴──────────────┴──────────────────┴───────────────┘

查询时:
  GetAllDomains()            → ["api.x.io", "cdn.x.io"]
  GetRouteMappingsByDomain() → domain 下的 service_key 列表
  用 service_key 匹配 IngressSLO 指标
```

### 3.2 问题一：域名不再使用（幽灵域名）

**场景**: `old.example.com` 的 IngressRoute 已从 K8s 中删除

```
时间线:
  T0: IngressRoute 存在 → Agent 上报 → SQLite 写入 route_mapping
  T1: 用户删除 IngressRoute → K8s 中不再存在
  T2: Agent 下次采集 → 不再上报 old.example.com
  T3: 但 SQLite 中的 route_mapping 记录永久存在（无 TTL / 无清理）

结果:
  GetAllDomains() 仍返回 "old.example.com"
  GetRouteMappingsByDomain("old.example.com") 仍返回旧映射
  ingressMap[serviceKey] 无数据（ClickHouse 中该服务已无流量）
  → 页面展示: 域名出现在列表中，但所有指标为空 / null
```

**根因**: `slo_route_mapping` 表**只有 UPSERT，没有 DELETE**。一旦写入，永不清除。

**影响**:
- 已废弃的域名会一直显示在 SLO 页面中（虽然无数据）
- 随着时间推移，幽灵域名越来越多，列表噪音增大
- 用户无法区分"真正无数据"和"已废弃"

### 3.3 问题二：极低频域名可能不被显示

**场景**: `rare.example.com` 每小时仅 1 个请求

```
Agent 采集链路:

1. ClickHouse 中的原始数据:
   traefik_service_requests_total 是 counter 类型
   OTel Collector 每 15s 采集一次 → ClickHouse 有连续的数据点
   关键: counter 值持续上报（即使 0 增量），不会因低频而消失

2. Agent ListIngressSLO(since=5min) 查询:
   SELECT Attributes['service'] AS svc,
          (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) AS delta
   FROM otel_metrics_sum
   WHERE MetricName = 'traefik_service_requests_total'
     AND TimeUnix >= now() - INTERVAL 300 SECOND
   GROUP BY svc, code, method
   HAVING count() >= 2

   → delta = 0 或 1（极低频）
   → 服务仍会出现在结果中（delta=0 也满足 HAVING count>=2）
   → RPS ≈ 0.003

3. 但真正的问题在于 SLOWindows 缓存:
```

**缓存 miss 时的回退链**:

```
用户选择 time_range="1d":

路径 A（正常）: SLOWindows["1d"] 有缓存 → 使用 1d 聚合
  → 24h 内的所有请求都被统计 → 低频域名可见 ✓

路径 B（缓存 miss / 首次启动）: SLOWindows["1d"] 为空
  → 回退到 SLOIngress (5min 窗口)
  → 如果低频域名在最近 5min 内无请求 → delta=0 → RPS=0
  → 域名出现但指标全部为 0
  → 用户看到一个 "0 RPS, N/A 可用性" 的域名 → 困惑
```

**更严重的场景**: 如果 Traefik 完全没有为该服务报告过指标（例如刚配置但从未有请求）：

```
  → ClickHouse 中该 service 无 counter 数据
  → Agent ListIngressSLO() 结果中不包含该 service
  → SQLite 有 route_mapping（IngressRoute 存在）
  → ingressMap[serviceKey] 查无数据
  → 域名显示在列表中，但服务行显示 "无数据"
```

**总结**: 极低频域名**会显示**（SQLite 有映射），但指标数据**可能为空或为零**。
这不是"不被显示"的问题，而是"显示了但没有有意义的数据"的问题。

**解决方案**: 分两层处理。

**层 1: 消除缓存 miss 的窗口期**（根因修复）

Master 启动时主动预加载 SLOWindows 缓存，避免首次访问回退到 5min 窗口：

```go
// master.go 启动流程中追加
func (m *Master) warmupSLOWindows() {
    for _, timeRange := range []string{"1d", "7d", "30d"} {
        // 向每个在线 Agent 发送预加载 Command
        // Agent 查 ClickHouse → 填充 SLOWindows 缓存 → 随下次快照上报
    }
}
```

**效果**: Master 重启后 ~10s 内 SLOWindows 即可用，不再回退到 5min 窗口。

**层 2: 前端展示优化**（体验改善）

当域名有 SQLite 映射但指标全为 0/null 时，前端展示"无流量"状态而非困惑的空数据：

```typescript
// DomainCard.tsx 中
if (domain.totalRequests === 0 && domain.rps === 0) {
    return <StatusBadge status="inactive" text={t.noTraffic} />;
}
```

### 3.4 问题溯源：SQLite 路由映射的设计缺陷

| 问题 | 根因 | 影响 | 解决方案 |
|------|------|------|---------|
| 幽灵域名 | 无删除/过期机制 | 废弃域名永久停留 | 全量同步（见第六节） |
| 低频域名 | 5min 回退窗口太短 | 缓存 miss 时显示空指标 | 启动预加载 + 前端"无流量"状态 |
| 映射固化 | UPSERT 只更新已有记录 | 路由变更不会清除旧路径 | 全量同步（见第六节） |

---

## 四、当前存储现状

### OTelSnapshot 中的 SLO 数据

```go
OTelSnapshot {
    // 5min 基础快照（Agent 每次上报都更新，数据来自 ClickHouse 5min 窗口聚合）
    SLOIngress   []IngressSLO     // 最近 5min Traefik 聚合
    SLOServices  []ServiceSLO     // 最近 5min Linkerd 聚合
    SLOEdges     []ServiceEdge    // 服务间调用边
    SLOSummary   *SLOSummary      // 汇总统计

    // 多窗口预聚合（Agent 侧独立 TTL 缓存，过期后重新查 ClickHouse）
    SLOWindows map[string]*SLOWindowData {
        "1d":  { Current, Previous, History, MeshServices, MeshEdges }  // TTL 5min
        "7d":  { Current, Previous, History, MeshServices, MeshEdges }  // TTL 30min
        "30d": { Current, Previous, History, MeshServices, MeshEdges }  // TTL 2h
    }

    // Concentrator 时序（1min 粒度 × 60 点）
    SLOTimeSeries []SLOServiceTimeSeries
}
```

### Ring Buffer 存储（90 份）

每 10 秒一份完整 OTelSnapshot → 90 份中每份都包含上述 SLO 全部数据。

---

## 五、Ring Buffer 冗余分析

### 5.1 Ring Buffer 中的冗余

| 字段 | 90 份是否必要 | 分析 |
|------|-------------|------|
| `SLOIngress` | **不需要** | 5min 聚合，90 份中的数据高度重复 |
| `SLOServices` | **不需要** | 同上 |
| `SLOEdges` | **不需要** | 拓扑变化极慢，只需最新 1 份 |
| `SLOSummary` | **不需要** | 汇总统计，只需最新 1 份 |
| `SLOWindows` | **不需要** | d/w/m 聚合数据，只需最新 1 份 |
| `SLOTimeSeries` | **需要讨论** | 趋势图需要，但每份都含完整 60 点是冗余 |

### 5.2 SLOWindows 的合理性

SLOWindows 本身的设计是合理的：
- **1d/7d/30d 三份聚合** — 各自独立 TTL，避免频繁查 ClickHouse
- **Current + Previous** — 支持对比视图
- **History** — 支持趋势图（按 bucket 聚合的时序点）
- **MeshServices + MeshEdges** — 支持拓扑图

**但这些数据不需要在 Ring Buffer 中保存 90 份。只需保留在 ClusterSnapshot.OTel 中的最新 1 份。**

---

## 六、域名匹配优化方案

### 方案：全量同步替代增量 UPSERT

**核心思路**: Agent 每次上报 IngressRoute 时，Master 做**全量覆盖**而非增量 UPSERT。

```
当前:
  Agent 上报 [A, B, C] → Master UPSERT A, B, C
  Agent 上报 [A, B]    → Master UPSERT A, B（C 仍留在 SQLite）

优化后:
  Agent 上报 [A, B, C] → Master 写入 A, B, C
  Agent 上报 [A, B]    → Master 删除该集群全部映射 → 重新写入 A, B（C 被清除）
```

**实现**:

```go
// 事务内执行: 先删后插
func SyncRouteMappings(clusterID string, mappings []SLORouteMapping) {
    tx.Exec("DELETE FROM slo_route_mapping WHERE cluster_id = ?", clusterID)
    for _, m := range mappings {
        tx.Exec("INSERT INTO slo_route_mapping (...) VALUES (...)", ...)
    }
}
```

**优点**:
- 自动清理已删除的 IngressRoute
- 不需要 TTL 或定时清理
- 与 Agent 采集周期同步（5min），延迟可接受

**注意**:
- 需要事务保护（防止删除和插入之间的查询返回空）
- Agent 如果离线，SQLite 中的数据保持不变（不会误删）

---

## 七、文件结构分析

### 7.1 当前 SLO 文件分布

```
=== 前端 ===

atlhyper_web/src/
├── app/observe/slo/
│   └── page.tsx                          # SLO 主页面
├── components/slo/
│   ├── common.tsx                        # 共用组件（StatusBadge, ErrorBudgetBar）
│   ├── DomainCard.tsx                    # 域名卡片
│   ├── OverviewTab.tsx                   # 概览标签页
│   ├── MeshTab.tsx                       # 服务网格标签页
│   ├── CompareTab.tsx                    # 对比标签页
│   ├── LatencyTab.tsx                    # 延迟分布标签页
│   └── SLOTargetModal.tsx               # SLO 目标配置模态框
├── api/slo.ts                            # SLO API 调用
├── datasource/slo.ts                     # 数据源切换（mock/api）
├── types/slo.ts                          # TypeScript 类型
├── mock/slo/
│   ├── index.ts                          # 导出入口
│   ├── data.ts                           # 模拟数据常量
│   └── queries.ts                        # 模拟查询函数
└── config/data-source.ts                 # 数据源配置（SLO 注册为 observe 子模块）

=== Master 后端 ===

atlhyper_master_v2/
├── slo/                                  # SLO 领域包
│   ├── interfaces.go                     # 接口定义
│   ├── calculator.go                     # 分位数/可用性计算
│   └── route_updater.go                  # 路由映射同步器
├── gateway/handler/
│   ├── slo.go                            # 主 SLO Handler（Domains/Targets/History）
│   ├── slo_latency.go                    # 延迟分布 Handler
│   ├── slo_mesh.go                       # 服务网格 Handler
│   └── observe.go                        # ⚠️ 混入了 5 个 SLO 端点（见下方分析）
├── service/query/
│   └── slo.go                            # SLO 查询服务
├── model/
│   └── slo.go                            # API 响应模型
├── database/
│   ├── repo/slo.go                       # SLO 仓库实现
│   └── sqlite/slo.go                     # SQLite SQL 生成
└── master.go                             # ⚠️ SLO RouteUpdater 初始化

=== Agent 后端 ===

atlhyper_agent_v2/
├── repository/ch/query/
│   └── slo.go                            # ClickHouse SLO 聚合查询（~800 行）
└── service/snapshot/
    └── snapshot.go                       # ⚠️ SLO 窗口缓存混在快照服务中

=== 共享模型 ===

model_v2/slo.go                           # Agent 上报的 SLO 快照模型
model_v3/slo/slo.go                       # 预聚合 SLO 模型（IngressSLO, ServiceSLO...）
```

### 7.2 耦合问题

#### 问题一：`observe.go` 混入 SLO 端点

`gateway/handler/observe.go` 是一个大杂烩 Handler，混入了 5 个 SLO 端点：

```go
// observe.go 中的 SLO 方法（不应该在这里）
SLOSummary()    // GET /api/v2/observe/slo/summary
SLOIngress()    // GET /api/v2/observe/slo/ingress
SLOServices()   // GET /api/v2/observe/slo/services
SLOEdges()      // GET /api/v2/observe/slo/edges
SLOTimeSeries() // GET /api/v2/observe/slo/timeseries
```

这些方法是 OTelSnapshot 直读端点，功能上与 `slo.go` 中的 Handler 重复。
它们存在于 `observe.go` 是因为早期设计将所有 OTel Dashboard 端点统一放在 observe Handler 中。

**影响**:
- SLO 逻辑分散在两个 Handler 文件中
- 修改 SLO 时需要同时关注 `slo.go` 和 `observe.go`
- 其他信号（APM/Logs/Metrics）的类似端点也在 `observe.go` 中，互相交织

#### 问题二：Agent 快照服务中的 SLO 缓存

`service/snapshot/snapshot.go` 是 Agent 的核心快照采集服务，其中混入了 SLO 特有的缓存逻辑：

```go
// snapshot.go 中的 SLO 缓存（与其他信号缓存混在一起）
type snapshotService struct {
    // ... 通用字段
    sloWindowCaches map[string]*sloWindowCache  // SLO 专用缓存
}

func (s *snapshotService) collectSLOWindows(...) { ... }  // SLO 窗口采集
func (s *snapshotService) fetchSLOWindow(...) { ... }     // 单窗口查询
```

**影响**:
- `snapshot.go` 文件越来越大（所有信号的采集逻辑都在里面）
- SLO 缓存策略与 APM/Logs/Metrics 的采集逻辑交织
- 修改 SLO 缓存 TTL 需要在通用快照服务中操作

### 7.3 目标文件结构

为保证模块隔离，SLO 相关代码应该满足以下原则：

1. **SLO 逻辑只出现在 SLO 命名的文件/目录中**
2. **非 SLO 文件不应包含 SLO 业务逻辑**（初始化注册除外）
3. **前端已经做到了**（`api/slo.ts`, `components/slo/`, `types/slo.ts`）
4. **后端需要整理的部分**：

```
需要从 observe.go 迁移出的 SLO 端点:
  observe.go::SLOSummary()    → slo.go 或新建 slo_observe.go
  observe.go::SLOIngress()    → 同上
  observe.go::SLOServices()   → 同上
  observe.go::SLOEdges()      → 同上
  observe.go::SLOTimeSeries() → 同上

需要从 snapshot.go 拆分出的 SLO 逻辑:
  snapshot.go::sloWindowCaches       → 独立的 SLO 采集模块
  snapshot.go::collectSLOWindows()   → 同上
  snapshot.go::fetchSLOWindow()      → 同上
```

### 7.4 理想文件结构（整理后）

```
=== 前端（已达标，无需修改） ===

atlhyper_web/src/
├── app/observe/slo/page.tsx              # 页面
├── components/slo/*.tsx                  # 组件（7 个）
├── api/slo.ts                            # API
├── datasource/slo.ts                     # 数据源
├── types/slo.ts                          # 类型
└── mock/slo/*.ts                         # Mock（3 个）

=== Master 后端（需整理 observe.go） ===

atlhyper_master_v2/
├── slo/                                  # SLO 领域包（不变）
│   ├── interfaces.go
│   ├── calculator.go
│   └── route_updater.go
├── gateway/handler/
│   ├── slo.go                            # SLO 主 Handler
│   ├── slo_latency.go                    # 延迟分布
│   ├── slo_mesh.go                       # 服务网格
│   └── slo_observe.go                    # ← 新增：从 observe.go 迁出的 5 个直读端点
├── service/query/slo.go                  # 查询服务
├── model/slo.go                          # 响应模型
└── database/{repo,sqlite}/slo.go         # 数据库

=== Agent 后端（需拆分 snapshot.go） ===

atlhyper_agent_v2/
├── repository/ch/query/slo.go            # ClickHouse 查询（不变）
└── service/snapshot/
    ├── snapshot.go                       # 通用快照编排（调用各信号采集器）
    └── slo_collector.go                  # ← 新增：SLO 窗口采集 + 缓存逻辑

=== 共享模型（不变） ===

model_v2/slo.go
model_v3/slo/slo.go
```

### 7.5 整理检查清单

| 检查项 | 当前 | 目标 |
|--------|------|------|
| 前端 SLO 代码是否独立 | ✅ 已隔离 | 无需修改 |
| Master Handler SLO 是否独立 | ❌ observe.go 混入 5 个端点 | 迁出到 `slo_observe.go` |
| Master Service/Model/DB 是否独立 | ✅ 已隔离 | 无需修改 |
| Agent 查询层是否独立 | ✅ `slo.go` 独立 | 无需修改 |
| Agent 快照采集是否独立 | ❌ 混在 `snapshot.go` 中 | 拆分到 `slo_collector.go` |
| 共享模型是否独立 | ✅ `model_v3/slo/` 独立 | 无需修改 |

---

## 八、存储优化结论

### SLO 存储策略

```
当前:
  ClusterSnapshot.OTel (最新 1 份) → 包含 SLOWindows
  OTelRing[90] → 每份都包含 SLOWindows  ← 完全冗余

优化后:
  ClusterSnapshot.OTel (最新 1 份) → 包含 SLOWindows（不变）
  OTelRing[90] → 不存 SLO 数据（或只存标量摘要）
```

### SLO 数据特征总结

| 特征 | 结论 |
|------|------|
| **数据源头** | Agent 从 ClickHouse 聚合 → 随快照上报 → Master 内存直读 |
| **时间粒度** | d/w/m 级别，无需秒级时间线 |
| **数据更新频率** | Agent TTL 缓存（5min/30min/2h），非实时 |
| **页面首屏** | Master 内存直读 `SLOWindows[timeRange]` |
| **详细查询** | 无（所有视图都是聚合数据） |
| **跨信号关联** | 无（SLO 独立） |
| **Ring Buffer 需求** | **不需要** — 只需最新 1 份 |
| **域名匹配** | SQLite 路由映射需改为全量同步，解决幽灵域名问题 |

### 需要保留的存储

```
最新 1 份 OTelSnapshot 中:
├── SLOWindows["1d"]  → Current + Previous + History + Mesh
├── SLOWindows["7d"]  → Current + Previous + History + Mesh
├── SLOWindows["30d"] → Current + Previous + History + Mesh
├── SLOIngress        → 5min 快照（SLOWindows 不可用时的回退）
├── SLOServices       → 5min 快照
└── SLOEdges          → 5min 快照

SQLite:
├── slo_route_mapping → 域名 ↔ serviceKey 映射（改为全量同步）
└── slo_targets       → SLO 目标配置（用户手动设置）
```

**SLO 是四个信号中存储需求最简单的 — 全部聚合数据，无原始条目，无实时查询。**
