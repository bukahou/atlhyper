# Agent SLO Counter Reset 修复 — 已完成

> 原设计文档: [agent-slo-counter-reset-fix-design.md](../../design/archive/agent-slo-counter-reset-fix-design.md)
> 完成日期: 2026-04-18

## 背景

前端 SLO 页面 Ingress 数据异常：`totalRequests=0`、`P95/P99=0`、`错误预算=100%`，但同一页面的延迟分布图却显示 24h 内有大量请求，数据互相矛盾。

## 根因

Traefik Pod 在 24h 窗口内重启过，导致 counter（`traefik_service_requests_total`）被重置。Agent 原 SQL 用 `argMax(Value) - argMin(Value)` 算 delta，遇 reset 得负数，被 Go 侧 `if cnt <= 0 { continue }` 全部丢弃，`totalRequests` 归零。`totalRequests=0` 级联导致 Master 端 Summary 聚合中 P95/P99 权重分母为零跳过计算，`CalculateAvailability(0,0)` 和 `CalculateErrorBudgetRemaining(100,95)` 都返回 100。

## Phase 1: Collect/Push 拆独立 ctx ✅ (commit 8cb469f)

顺手修掉的设计缺陷：`collectAndPushSnapshot` 原来 Collect 和 Push 共享同一个 30s ctx，Collect 慢耗尽预算后 Push 立即 deadline exceeded。

- scheduler.go: Config 新增 SnapshotPushTimeout 字段
- scheduler.go: collectAndPushSnapshot 拆两个独立 WithTimeout
- agent.go: 构造时 SnapshotPushTimeout 复用 cfg.Timeout.HTTPClient

## Phase 2: SLO counter reset SQL 替换 ✅ (commit 712f902)

用 ClickHouse `lagInFrame` window function 实现 Prometheus `rate()/increase()` 的 `counterCorrection` 等价算法：
- 相邻样本 `v[i] >= v[i-1]`: 正常递增，delta = v[i] - v[i-1]
- 相邻样本 `v[i] < v[i-1]`: counter reset，delta = v[i]（reset 后从 0 开始）
- `sum()` 累加所有相邻 delta，正确处理任意次数 reset

改动位置：
- slo.go `queryIngressSLO` (L119-132 一带): countQuery 替换为 window function 版本
- slo.go `GetIngressSLOHistory` (L654-666 一带): countQuery 同算法，按 (svc, code, method, bucket) 四维分区

## Phase 3: 清理临时 debug 日志 ✅ (包含在 commit 712f902)

调试过程中加的时间测量日志全部删除，净改动为零：
- scheduler.go collectAndPushSnapshot
- gateway/master_gateway.go PushSnapshot
- service/snapshot/snapshot.go Collect

## Phase 4: 编译 + 测试 + commit ✅

- `go build ./...` PASS
- `go test ./atlhyper_agent_v2/repository/ch/query/ ./scheduler/ ./service/snapshot/` PASS

## Phase 5: 端到端验证 ✅

本地启 Master + Agent + Web，刷新 `/observe/slo` 页面，对比修复前后：

| 指标 | 修复前 | 修复后 |
|------|--------|--------|
| 状态 | unknown（灰色）| 健康（绿色）|
| 可用性 | 100% | 97.98% |
| P95 / P99 | 0ms / 0ms | 284ms / 5000ms |
| 错误率 | 0.000% | 2.020% |
| 总请求数 | 0 | 99 |
| 错误预算 | 100% + 消耗图 -100% | 59.6% + 消耗图正常递减 |

数据自洽性交叉验证通过：
- 99 = GET 25 + POST 74 ✅
- 99 = 200码 96 + 401码 1 + 500码 2 ✅
- 错误率 2.02% = 2/99 ✅

## 后续独立 issue

**Node metrics / Summary 5min 窗口 7 处升级为 counter-reset-safe**
- 位置: `atlhyper_agent_v2/repository/ch/query/metrics.go:316/495/580/677/809` + `atlhyper_agent_v2/repository/ch/summary.go:61/94`
- 方案: 裸 `argMax - argMin` 替换为已有的 `gaugeCounterDelta` 片段
- 优先级: 低（5min 窗口 reset 概率极低，单次 reset 处理足够）

## Commits

```
ba69647 docs(tasks): SLO counter reset 修复进度更新为待端到端验证
712f902 fix(agent): SLO Ingress counter reset 改用 Prometheus 算法
8cb469f fix(agent): Collect 与 Push 拆独立 ctx 避免互相拖累
```
