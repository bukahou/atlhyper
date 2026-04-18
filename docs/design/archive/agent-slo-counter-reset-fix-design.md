# Agent SLO Counter Reset 修复设计

> 修复 Ingress SLO `totalRequests=0` 连带 Summary 全归零的 bug，用 Prometheus `rate()/increase()` 等价算法替换 `argMax - argMin`。

---

## 1. 背景

前端 SLO 概览页（`/observe/slo`）出现：
- 顶部统计卡片 `总请求数=0`、`P95/P99=0ms`、`错误预算=100%`
- 延迟分布图却显示 24h 内明显有请求（2-3ms 处 ~83 个）
- SLO 趋势图也有历史数据（13:00 峰值 3355ms）

同一页面的数据源互相矛盾。

## 2. 根因（已实测确认）

### 2.1 API 实测
`GET /api/v2/slo/domains/v2?cluster_id=ZGFX-X10A&time_range=1d` 返回：
```json
{
  "services": [{
    "current": {
      "p95Latency": 284,       // Histogram 路径：有值
      "p99Latency": 5000,
      "totalRequests": 0       // Counter 路径：归零
    }
  }],
  "summary": {
    "availability": 100,       // totalRequests=0 时除零保护
    "p95Latency": 0,           // 权重=0，计算跳过
    "totalRequests": 0
  }
}
```

### 2.2 ClickHouse 实测
`atlhyper.otel_metrics_sum` 24h 内 `traefik_service_requests_total` 有 **6846 行**数据。

但跑 Agent 的原 SQL：
```sql
SELECT Attributes['service'] AS svc, ...,
       (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) AS delta
FROM atlhyper.otel_metrics_sum
WHERE MetricName = 'traefik_service_requests_total'
  AND TimeUnix >= now() - INTERVAL 86400 SECOND
GROUP BY svc, code, method
HAVING count() >= 2
```

**所有 delta 为负**（-1 到 -22），因 Traefik 在 24h 窗口内重启过，counter 被重置。

### 2.3 Agent 丢弃逻辑
`atlhyper_agent_v2/repository/ch/query/slo.go:159-162`：
```go
cnt := int64(delta)
if cnt <= 0 { continue }   // 所有负数 delta 被丢弃 → totalReqs=0
```

### 2.4 级联效应
Master `gateway/handler/slo/slo_domains.go:418-442` 构建 Summary 时，`totalRequests=0` 导致：
- P95/P99 不计算（权重分母为 0），保持 0
- `CalculateAvailability(0, 0) = 100%`
- `CalculateErrorBudgetRemaining(100, 95) = 100%`

## 3. 项目现状：Counter Delta 处理现况

| 位置 | 指标 | SQL 方式 | Reset 处理 | 窗口 | 状态 |
|------|------|---------|-----------|------|------|
| **slo.go:126** (Ingress) | `traefik_service_requests_total` | 裸 `argMax-argMin` | ❌ 无 | 1d/7d/30d | 🔴 当前 bug |
| slo.go:252/311/414/496 (Linkerd) | `linkerd_*` | `gaugeCounterDelta` | ✅ 单次 reset | 1d/7d/30d | 已保护 |
| slo.go:addHistogramDelta | histogram bucket | Go 侧判 reset | ✅ 单次 reset | 全部 | 已保护 |
| metrics.go:316 | `node_cpu_seconds_total` | 裸 rate | ❌ 无 | 5min | 🟡 短窗口隐患 |
| metrics.go:495/580/677/809 | node disk/net/PSI/softnet | 裸 rate | ❌ 无 | 5min | 🟡 短窗口隐患 |
| summary.go:61 | `traefik_service_requests_total` | 裸 rate | ❌ 无 | 5min | 🟡 短窗口隐患 |
| summary.go:94 | `node_cpu_seconds_total` | 裸 rate | ❌ 无 | 5min | 🟡 短窗口隐患 |

## 4. 算法对比

### Prometheus `rate()/increase()` 源算法
```
counterCorrection = 0
for i := 1; i < len(samples); i++ {
    if samples[i].v < samples[i-1].v {    // 相邻样本递减 = reset
        counterCorrection += samples[i-1].v
    }
}
delta = samples[last].v - samples[0].v + counterCorrection
```
精确处理任意次 reset。

### 现有 `gaugeCounterDelta`
```sql
if(max(Value) > argMax(Value, TimeUnix),
    (max(Value) - argMin(Value, TimeUnix)) + argMax(Value, TimeUnix),
    argMax(Value, TimeUnix) - argMin(Value, TimeUnix))
```
- 单次 reset：与 Prometheus 等价
- 多次 reset：**只补偿最高峰值那一次**，低估

### 长窗口 vs 短窗口的选型

- **1d/7d/30d SLO 窗口**：多次 reset 是常态（Traefik 升级/扩缩） → **必须用 Prometheus 完整算法**
- **5min 节点指标窗口**：多次 reset 概率 ≈ 0（node_exporter DaemonSet 稳定） → `gaugeCounterDelta` 足够

## 5. 方案

### 本次任务：仅修 slo.go 的 Ingress counter（1d/7d/30d 长窗口）

用 ClickHouse window function 实现 Prometheus 算法：

```sql
SELECT svc, code, method,
       sum(if(Value >= prevValue, Value - prevValue, Value)) AS delta
FROM (
    SELECT Attributes['service'] AS svc,
           Attributes['code']    AS code,
           Attributes['method']  AS method,
           Value, TimeUnix,
           lagInFrame(Value, 1, Value) OVER
               (PARTITION BY Attributes['service'], Attributes['code'], Attributes['method']
                ORDER BY TimeUnix) AS prevValue
    FROM otel_metrics_sum
    WHERE MetricName = 'traefik_service_requests_total'
      <timeFilter>
)
GROUP BY svc, code, method
HAVING delta > 0
```

**逻辑说明**：
- `lagInFrame(Value, 1, Value)`：按 (svc, code, method) 分区、按时间排序，取前一个样本的 Value；首行取自身值（diff=0）
- `if(Value >= prevValue, Value - prevValue, Value)`：相邻样本递增则取差；递减视为 reset，delta = 当前值（等同 Prometheus `counterCorrection += prev + (v - 0)` 的化简）
- `sum(...)`：累加所有相邻 delta = Prometheus `increase()` 窗口总增量
- `HAVING delta > 0`：只保留有实际请求的组合（单样本或全零的组合会被过滤）

**完全等价 Prometheus 的 `increase()`**，正确处理任意次 reset。

### 后续 issue（不在本次范围）

**Node metrics / Summary 5min 窗口 7 处升级为 counter-reset-safe**
- 位置：`metrics.go:316/495/580/677/809` + `summary.go:61/94`
- 方案：裸 `argMax - argMin` 替换为 `gaugeCounterDelta` 片段（复用已有常量）
- 优先级：低（5min 窗口 reset 概率 ≈ 0，现实影响可忽略）
- 预估工作量：半天

## 6. 文件变更清单

### 本次任务修改

```
atlhyper_agent_v2/
└── repository/ch/query/
    └── slo.go [修改]
        ├── L119-132  queryIngressSLO 的 countQuery SQL 替换为 window function 版本
        ├── L159-162  删除 `if cnt <= 0 { continue }`（新 SQL 已过滤，保留不必要）
        └── L660-690? queryIngressHistorySLO 若有同类 SQL，同步替换（待确认）
```

### 前置已完成（方案 B context 拆分，commit 前一并验证）

```
atlhyper_agent_v2/
├── scheduler/scheduler.go [已改]
│   ├── Config 新增 SnapshotPushTimeout 字段
│   └── collectAndPushSnapshot 拆 Collect/Push 独立 ctx
└── agent.go [已改]
    └── scheduler.Config 构造时填 SnapshotPushTimeout: cfg.Timeout.HTTPClient
```

### 临时 debug 日志（commit 前需清理）

```
atlhyper_agent_v2/
├── scheduler/scheduler.go
│   └── collectAndPushSnapshot 中的 [DEBUG] 采集完成 / 推送完成 日志
├── gateway/master_gateway.go
│   └── PushSnapshot 中的 [DEBUG push] marshalDur/gzipDur/httpDur 日志（fmt.Printf）
└── service/snapshot/snapshot.go
    └── Collect 中的 [DEBUG] K8s 采集完成 / OTel 聚合完成 日志
```

## 7. 验证计划

### 7.1 SQL 单元验证（ClickHouse 直测）
保持 port-forward 8123 开启，本地运行新 SQL：
```bash
curl -s "http://localhost:8123/" --data-binary @/tmp/new_slo_query.sql
```
**预期**：delta 全部 > 0，每个 (svc, code, method) 返回实际请求增量。对比手工估算值应接近。

### 7.2 Agent 端日志
```bash
go run ./cmd/atlhyper_agent_v2/main.go
```
观察 OTel 聚合日志，无 `query ingress counts` 相关错误。

### 7.3 API 实测
```bash
curl -s "http://localhost:8080/api/v2/slo/domains/v2?cluster_id=ZGFX-X10A&time_range=1d" | jq .
```
**预期**：
- Service.current.totalRequests > 0
- Domain.summary.totalRequests > 0
- P95/P99 在 Service 和 Summary 两级一致（不再 Summary=0）
- errorBudgetRemaining 为合理值（不再固定 100）

### 7.4 前端验证
刷新 `/observe/slo` 页面，顶部卡片显示真实的总请求数、P95/P99、错误预算。

### 7.5 编译 + 测试
```bash
go build ./...
go test ./atlhyper_agent_v2/repository/ch/query/ -v
```

## 8. 风险与回滚

### 风险
- ClickHouse `lagInFrame` 需要 v21.12+ 支持。生产集群 `clickhouse-server:24.8` ✅ 支持。
- window function 性能开销：相比 `argMax - argMin`，需要先排序分区，预估 30d 窗口查询耗时增加 10-30%，但 SLO 查询本就在后台异步执行，可接受。

### 回滚
Git revert 单个 commit 即可，SQL 修改不涉及表结构和数据迁移。

## 9. 附录

### 9.1 关键文件路径
- **Agent SLO 查询**：`atlhyper_agent_v2/repository/ch/query/slo.go`
- **Master SLO Handler**：`atlhyper_master_v2/gateway/handler/slo/slo_domains.go`
- **Master Service 层**：`atlhyper_master_v2/service/query/slo.go`
- **前端 API**：`atlhyper_web/src/api/slo.ts`（端点 `/api/v2/slo/domains/v2`）
- **前端页面**：`atlhyper_web/src/app/observe/slo/page.tsx`
- **类型定义**：`atlhyper_web/src/types/slo.ts`（`DomainSLOListResponseV2`）

### 9.2 本地调试环境
- ClickHouse port-forward：`kubectl -n atlhyper port-forward svc/clickhouse 9000:9000 8123:8123`
- Agent 环境变量：`AGENT_CLICKHOUSE_ENDPOINT=clickhouse://localhost:9000`（native）或 `http://localhost:8123`（HTTP）
- Master 本地运行：`go run ./cmd/atlhyper_master_v2/main.go`（8080 gateway / 8081 agentsdk）
