# 日志数据策略分析

> 按页面视图 + API 端点分析 Logs 数据的存储与获取策略

---

## 一、页面结构与数据需求

### 页面: `/observe/logs/`

```
Logs 页面
├── LogToolbar（搜索框 + 分页信息）
├── Filter Pills（激活的筛选条件展示）
├── LogHistogram（日志量直方图，支持 brush 选择时间范围）
├── LogFacets（左侧面板: 服务/严重级别/Scope 多选）
├── LogList（中央日志条目列表，50 条/页）
└── LogDetailDrawer（右侧抽屉: 日志全部字段 + Trace 关联链接）
```

**关键特征**:
- 页面打开瞬间需要展示**默认 15 分钟的全部日志**
- 支持全文搜索、多维筛选、Trace 关联
- 是唯一需要**原始条目**（非聚合）的信号

---

## 二、API 端点分析

### 2.1 日志查询（唯一端点，承载所有场景）

| 项目 | 内容 |
|------|------|
| **端点** | `POST /api/v2/observe/logs/query` |
| **Handler** | `observe.go::LogsQuery()` |
| **双路径** | 快速路径（内存）+ Command 路径（ClickHouse） |

#### 快速路径（默认 15min，无搜索）

| 条件 | 数据源 | 延迟 |
|------|--------|------|
| 无 search query | `OTelSnapshot.RecentLogs` (内存) | <10ms |
| 无 traceId/spanId | 直接从 500 条缓存中过滤 | |
| 时间范围 ≤15min | 支持 service/severity/scope 筛选 | |

```
请求: { cluster_id, limit: 50, offset: 0 }
  → Master 读 OTelSnapshot.RecentLogs (≤500 条)
  → 应用 service/severity/scope 筛选
  → 从全量过滤结果计算 Facet 计数（不受分页影响）
  → 从全量过滤结果生成 Histogram（不受分页影响）
  → 分页截取 offset/limit
  → 返回 { logs, total, facets, histogram }
```

#### Command 路径（搜索 / Trace 关联 / 自定义时间）

| 条件 | 数据源 | 延迟 |
|------|--------|------|
| 有 search query | ClickHouse `otel_logs` 表 | 100-500ms |
| 有 traceId/spanId | ClickHouse（**无时间窗口限制**） | 100-500ms |
| 自定义时间 >15min | ClickHouse | 100-500ms |

```
请求: { cluster_id, query: "error", limit: 50 }
  → Master 创建 Command(action="query_logs")
  → Agent 收到 Command → 查询 ClickHouse
  → Agent 并行执行 3 条 SQL:
     1. 主查询（过滤 + 分页）
     2. 计数查询（总条数）
     3. Facet 查询（按时间窗口，忽略搜索条件）
  → 返回结果给 Master → 转发给前端
```

---

## 三、跨信号关联

### Trace ↔ Log 关联（强关联）

#### 从 Log 到 Trace

```
LogDetailDrawer 中:
  → 如果日志条目有 traceId → 显示 "查看 Trace" 链接
  → 点击 → 跳转 /observe/apm?trace={traceId}
```

#### 从 Trace 到 Log（APM 页面发起）

```
APM TraceWaterfall 中:
  → 用户点击 Span → 查看关联日志
  → 跳转 /observe/logs?traceId={traceId}&spanId={spanId}
  → Logs 页面识别 URL 参数 → 触发 Command 路径
  → Agent: SELECT * FROM otel_logs WHERE TraceId = ?
  → 注意: 无时间窗口限制（可以查到任意时间的日志）
```

#### 关键设计决策

| 场景 | 时间窗口 | 原因 |
|------|---------|------|
| 默认浏览 | 15min | 内存缓存覆盖 |
| 搜索 | 用户选择 | 灵活查询 |
| Trace 关联 | **无限制** | Trace 可能跨越较长时间，必须能找到关联日志 |

---

## 四、当前存储现状

### OTelSnapshot 中的 Logs 数据

```go
OTelSnapshot {
    // 最近日志条目（5min 窗口，最多 500 条）
    RecentLogs  []log.Entry    // 完整日志条目

    // 日志统计摘要
    LogsSummary *log.Summary   // TotalEntries, SeverityCounts, TopServices
}
```

### log.Entry 结构

```go
type Entry struct {
    Timestamp          time.Time
    TraceID            string
    SpanID             string
    SeverityText       string        // "INFO" / "WARN" / "ERROR"
    SeverityNumber     int32
    ServiceName        string
    Body               string        // 日志内容（可能很长）
    ScopeName          string
    LogAttributes      map[string]string
    ResourceAttributes map[string]string
}
```

### Ring Buffer 存储（90 份）

每 10 秒一份 OTelSnapshot → 90 份中每份都包含:
- `RecentLogs` — **最多 500 条 log.Entry**
- `LogsSummary` — 统计摘要

### 内存开销估算

```
单条 log.Entry（保守估算）:
  固定字段: ~200 bytes
  Body: ~200 bytes（平均）
  Attributes: ~300 bytes（2 个 map）
  ≈ 700 bytes / 条

单份 RecentLogs:
  500 条 × 700 bytes ≈ 350 KB

90 份 Ring Buffer 中的 RecentLogs:
  350 KB × 90 ≈ 31 MB  ← 这是最大的内存浪费
```

---

## 五、问题分析

### 5.1 Ring Buffer 中的冗余

| 字段 | 90 份是否必要 | 分析 |
|------|-------------|------|
| `RecentLogs` | **完全不需要** | 500 条日志是 5min 窗口的缓存，90 份中大量重复 |
| `LogsSummary` | **不需要** | 统计摘要只需最新 1 份 |

**RecentLogs 是 Ring Buffer 中最大的内存浪费源。**

### 5.2 为什么 Ring Buffer 不被 Logs 使用

检查所有消费者:
- `LogsQuery()` 快速路径 → 读 `GetOTelSnapshot().RecentLogs`（最新 1 份）
- `LogsQuery()` Command 路径 → 直接查 ClickHouse
- **没有任何端点从 Ring Buffer 历史中读取日志**

结论: **Logs 从不使用 Ring Buffer 中的历史数据**。

### 5.3 首屏 500 条的局限

用户说"页面打开瞬间就需要展示默认的 15 分钟的全部数据"。但当前实际情况:

- 快速路径只返回 `RecentLogs`（5min 窗口，最多 500 条）
- 如果 15min 内日志量 > 500 条，快速路径无法覆盖全部
- 这种情况下实际上会丢失日志（只看到最近 500 条）

**这是一个已知的权衡**: 500 条足以覆盖大部分场景的首屏展示，更多日志通过分页或搜索获取（走 Command 路径）。

**处理方式**: 接受此权衡，不做修改。理由：

1. 快速路径的目的是**零延迟首屏**，500 条覆盖绝大多数场景
2. 用户翻页或搜索时自动切换到 Command 路径，获取完整数据
3. 增大缓存（如 2000 条）会导致 Ring Buffer 内存开销从 31MB 涨至 125MB，收益不成比例
4. 前端已在 LogToolbar 显示 `显示 {offset+1}-{offset+limit} / 共 {total} 条`，用户可感知数据范围

---

## 六、优化结论

### Logs 数据特征总结

| 特征 | 结论 |
|------|------|
| **首屏需求** | 最近 500 条原始日志（非聚合）— 只需最新 1 份 |
| **搜索/筛选** | 走 ClickHouse Command 路径 |
| **Trace 关联** | 走 ClickHouse Command 路径（无时间限制） |
| **Histogram** | 从 RecentLogs 实时计算（快速路径）或 ClickHouse 返回 |
| **Facet** | 从 RecentLogs 实时计算（快速路径）或 ClickHouse 返回 |
| **Ring Buffer 需求** | **完全不需要** — 没有任何端点使用历史日志 |

### Ring Buffer 中 Logs 的处理建议

```
当前:
  ClusterSnapshot.OTel.RecentLogs (最新 1 份, ~350KB)
  OTelRing[90] → 每份都含 RecentLogs  ← 浪费 ~31MB

优化后:
  ClusterSnapshot.OTel.RecentLogs (最新 1 份, ~350KB) → 不变
  OTelRing[90] → 不存日志数据
```

**节省: ~31MB / 集群**

### 需要保留的存储

```
最新 1 份 OTelSnapshot 中:
├── RecentLogs[]   → 500 条，首屏快速展示
└── LogsSummary    → 统计摘要

其余所有日志查询 → ClickHouse (Command 机制)
```

### 跨信号关联不受影响

Trace ↔ Log 关联走的是 Command 路径（ClickHouse `WHERE TraceId = ?`），与 Ring Buffer 无关。

---

## 七、文件结构分析

### 7.1 当前 Logs 文件分布

```
=== 前端 ===

atlhyper_web/src/
├── app/observe/logs/
│   ├── page.tsx                             # Logs 主页面（搜索/筛选/分页/Brush 状态编排）
│   └── components/
│       ├── LogList.tsx                       # 日志条目列表（50 条/页）
│       ├── LogDetail.tsx                     # 日志详情抽屉（Attributes + Trace 关联链接）
│       ├── LogHistogram.tsx                  # 日志量直方图（支持 Brush 选区）
│       ├── LogFacets.tsx                     # 左侧分面筛选（Services/Severities/Scopes）
│       └── LogToolbar.tsx                    # 搜索框 + 分页信息
├── types/model/log.ts                        # Log TypeScript 类型定义
├── datasource/logs.ts                        # Log 数据源适配（mock/api 切换）
├── api/observe.ts                            # ⚠️ 混合：queryLogs() + getLogsHistogram() 在此文件中
├── mock/logs/
│   ├── index.ts                              # Mock 导出
│   ├── data.ts                               # 1600+ 条模拟日志数据
│   └── queries.ts                            # Mock 查询函数（过滤/分页/分桶）
└── config/data-source.ts                     # 数据源开关

=== Master 后端 ===

atlhyper_master_v2/
├── gateway/handler/
│   ├── observe_logs.go                       # Logs Handler（3 个端点: Query/Histogram/Summary）
│   └── observe.go                            # ⚠️ 共用基础：TTL 缓存 + executeQuery()
├── gateway/routes.go                         # 路由注册
└── service/query/otel.go                     # ⚠️ 混合：GetOTelSnapshot() 返回全部信号数据

=== Agent 后端 ===

atlhyper_agent_v2/
├── repository/
│   ├── interfaces.go                         # LogQueryRepository 接口定义
│   └── ch/
│       ├── query/log.go                      # ClickHouse 日志查询实现（~417 行）
│       └── dashboard.go                      # ⚠️ 混合：OTelDashboardRepository（4 信号域）
├── service/
│   ├── command/ch_query.go                   # ⚠️ 混合：handleQueryLogs() 在通用 Command 分发中
│   └── snapshot/
│       └── snapshot.go                       # ⚠️ 混合：getOTelSnapshot() 采集 4 信号域

=== 共享模型 ===

model_v3/log/
├── log.go                                    # 日志数据模型（Entry, Facets, QueryResult, HistogramResult）
└── summary.go                                # 日志统计摘要（Summary, ServiceCount）
model_v3/cluster/snapshot.go                  # ⚠️ 混合：OTelSnapshot 包含 RecentLogs + LogsSummary
```

### 7.2 耦合问题

#### 问题一：`observe_logs.go` Handler 层职责混合

`LogsQuery()` 方法（~120 行）在 Handler 中直接实现了两条路径的数据加工逻辑：

```go
// observe_logs.go 中的 LogsQuery()
func (h *ObserveHandler) LogsQuery(...) {
    // 快速路径：直接从内存过滤
    if 无搜索条件 && 无 Trace 关联 {
        logs := h.querySvc.GetOTelSnapshot().RecentLogs
        // Handler 内直接做 service/scope/severity 过滤
        // Handler 内直接做 facets 计算
        // Handler 内直接做分页
        return
    }
    // 慢速路径：MQ Command 透传到 Agent
    h.executeQuery(...)
}
```

**影响**:
- 快速路径的过滤/分页/facets 计算逻辑本应在 Service 层
- Handler 直接操作 Store 数据，违反分层架构
- 难以对快速路径逻辑做单元测试

**解决方案**: 将快速路径的过滤逻辑下沉到 Service 层：

```go
// service/query/otel.go 新增方法

// QueryLogsFromSnapshot 从内存快照查询日志（快速路径）
// 封装过滤、facets 计算、分页，Handler 只做参数提取和响应
func (q *QueryService) QueryLogsFromSnapshot(
    ctx context.Context, clusterID string,
    service, level, scope string,
    startTime, endTime string,
    offset, limit int,
) (*log.QueryResult, error) {
    otel, err := q.store.GetOTelSnapshot(ctx, clusterID)
    if err != nil || otel == nil || len(otel.RecentLogs) == 0 {
        return nil, err
    }

    // 1. facets 基于全量数据
    facets := computeLogFacets(otel.RecentLogs)

    // 2. 过滤
    logs := filterLogs(otel.RecentLogs, service, level, scope, startTime, endTime)

    // 3. 分页
    total := len(logs)
    paged := paginateLogs(logs, offset, limit)

    return &log.QueryResult{Logs: paged, Total: total, Facets: facets}, nil
}
```

```go
// observe_logs.go 简化为

func (h *ObserveHandler) LogsQuery(w http.ResponseWriter, r *http.Request) {
    // ... 参数提取（不变）

    if query == "" && traceId == "" && spanId == "" && startTime == "" && endTime == "" {
        result, err := h.querySvc.QueryLogsFromSnapshot(ctx, clusterID, service, level, scope, "", "", offset, limit)
        if err == nil && result != nil {
            writeJSON(w, http.StatusOK, map[string]interface{}{"message": "获取成功", "data": result})
            return
        }
    }
    // Command 路径不变
    h.executeQuery(w, r, clusterID, command.ActionQueryLogs, body, 0)
}
```

**效果**: Handler 从 ~120 行降至 ~30 行，过滤/分页逻辑可独立单元测试。

#### 问题二：缺少独立的 Log Service

与 SLO 有独立的 `slo/` 领域包不同，Logs 没有独立的 Service 层：

```
SLO:  Handler → service/query/slo.go → slo/ 领域包
APM:  Handler → (无独立 Service) → 直接读 OTelSnapshot
Logs: Handler → (无独立 Service) → 直接读 OTelSnapshot + MQ 透传
```

**影响**:
- Handler 承担了本应属于 Service 的业务逻辑
- 快速路径的过滤逻辑无法复用

**解决方案**: 与问题一的方案合并。`QueryLogsFromSnapshot()` 即为 Logs 的 Service 层方法。

当前不需要独立的 `logs/` 领域包（不像 SLO 有复杂的域名匹配 + 路由同步逻辑），在 `service/query/otel.go` 中新增方法即可满足需求。如果未来 Logs 功能复杂化（如日志告警规则管理），再提取为独立包。

#### 问题三：Agent `snapshot.go` 混合采集

同 APM 问题 — `getOTelSnapshot()` 中混合了 Logs 的 `getRecentLogs()` 调用。

**解决方案**: 见 APM 数据策略文档 6.2 问题二的方案 — 提取 `otel_collector.go`，将 12 个并发采集 goroutine（含 `GetLogsSummary` + `ListRecentLogs`）从 `snapshot.go` 移至独立文件。Logs 采集逻辑无需独立文件（仅 2 个调用），跟随 `otel_collector.go` 即可。

#### 问题四：`api/observe.ts` 混合 4 信号域

前端 `api/observe.ts` 中的 `queryLogs()` 和 `getLogsHistogram()` 与 APM/Metrics/SLO 的 API 函数混在同一文件中。

**解决方案**: 将 Logs 相关函数拆分到 `api/logs.ts`：

```
迁出函数:
  queryLogs()           → api/logs.ts
  getLogsHistogram()    → api/logs.ts

更新导入:
  datasource/logs.ts 的 import 从 "@/api/observe" → "@/api/logs"
```

### 7.3 理想文件结构（整理后）

```
=== 前端（已达标，无需修改） ===

atlhyper_web/src/
├── app/observe/logs/
│   ├── page.tsx                              # 页面
│   └── components/*.tsx                      # 组件（5 个）
├── types/model/log.ts                        # 类型
├── datasource/logs.ts                        # 数据源
└── mock/logs/*.ts                            # Mock（3 个）

=== Master 后端（需整理 Handler 职责） ===

atlhyper_master_v2/
├── gateway/handler/
│   └── observe_logs.go                       # Logs Handler（保持，但下沉过滤逻辑到 Service）
└── service/query/
    └── otel.go                               # 可新增 LogsQuery 方法（封装快速路径过滤逻辑）

=== Agent 后端（需拆分 snapshot.go） ===

atlhyper_agent_v2/
├── repository/
│   ├── interfaces.go                         # 保持 LogQueryRepository 独立
│   └── ch/query/log.go                       # ClickHouse 实现（独立 ✅）
└── service/snapshot/
    ├── snapshot.go                           # 通用快照编排
    └── otel_collector.go                     # ← OTel 采集逻辑（含 Logs 采集）

=== 共享模型（已达标） ===

model_v3/log/
├── log.go                                    # 数据模型
└── summary.go                                # 统计摘要
```

### 7.4 整理检查清单

| 检查项 | 当前 | 目标 |
|--------|------|------|
| 前端 Logs 页面/组件是否独立 | ✅ 已隔离 | 无需修改 |
| 前端 Logs 数据源/类型是否独立 | ✅ `datasource/logs.ts` + `types/model/log.ts` | 无需修改 |
| 前端 API 调用是否独立 | ❌ 混在 `api/observe.ts` 中 | 可拆分为 `api/logs.ts`（优先级低） |
| Master Handler 是否独立 | ⚠️ 文件独立但职责混合（Handler 做过滤） | 下沉过滤逻辑到 Service |
| Agent 查询层是否独立 | ✅ `ch/query/log.go` 独立 | 无需修改 |
| Agent 快照采集是否独立 | ❌ 混在 `snapshot.go` 中 | 拆分 OTel 采集逻辑 |
| 共享模型是否独立 | ✅ `model_v3/log/` 独立 | 无需修改 |
