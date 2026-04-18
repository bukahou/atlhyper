# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

---

## Agent SLO Counter Reset 修复 — 🔄 进行中

> 原设计文档: [agent-slo-counter-reset-fix-design.md](../../design/active/agent-slo-counter-reset-fix-design.md)
>
> 背景: Ingress SLO 页面 totalRequests=0、P95/P99=0、错误预算=100% 异常。根因: Traefik 重启导致 counter reset，Agent 原 SQL `argMax-argMin` 产生负值被全部丢弃。修复: 用 ClickHouse window function 实现 Prometheus `rate()/increase()` 等价算法。

- Phase 1: 前置改动（方案 B Collect/Push 拆 ctx）— ✅ 完成
  - scheduler.go: Config 新增 SnapshotPushTimeout 字段，collectAndPushSnapshot 拆独立 ctx
  - agent.go: scheduler.Config 构造时 `SnapshotPushTimeout: cfg.Timeout.HTTPClient`
  - 编译通过 + scheduler 测试 PASS
- Phase 2: SLO counter reset SQL 替换 — 待办
  - slo.go:119-132 `queryIngressSLO` 的 countQuery 替换为 `lagInFrame + sum(if reset)` 的 window function 版本
  - slo.go:159-162 删除（或保留为防御）`if cnt <= 0 { continue }`
  - slo.go:660-690 排查 `queryIngressHistorySLO` 是否同类 SQL（若是同步替换）
  - 验证: ClickHouse 直测 SQL → Agent 启动 → `curl /api/v2/slo/domains/v2` → 前端刷新
- Phase 3: 清理临时 debug 日志 — 待办
  - scheduler.go `collectAndPushSnapshot` 中的 [DEBUG] 采集/推送日志
  - gateway/master_gateway.go `PushSnapshot` 中的 [DEBUG push] fmt.Printf 日志
  - service/snapshot/snapshot.go `Collect` 中的 [DEBUG] K8s/OTel 日志
- Phase 4: 编译 + 测试 + commit — 待办
  - `go build ./...` PASS
  - `go test ./atlhyper_agent_v2/repository/ch/query/ -v` PASS
  - commit: "fix(agent): SLO ingress counter reset 正确处理（Prometheus 算法）"

### 后续 issue（独立任务，不在本次范围）

**Node metrics / Summary 5min 窗口 7 处升级为 counter-reset-safe**
- 位置: `atlhyper_agent_v2/repository/ch/query/metrics.go:316/495/580/677/809` + `atlhyper_agent_v2/repository/ch/summary.go:61/94`
- 方案: 裸 `argMax - argMin` 替换为已有的 `gaugeCounterDelta` 片段（5min 窗口 reset 概率极低，单次 reset 处理足够）
- 优先级: 低

---

## QueryService 拆分重构 — 待办

> 原设计文档: [master-v2-query-service-split-design.md](../../design/active/master-v2-query-service-split-design.md)

- Phase 1: AdminQueryService 拆分 — 待办
  - admin.go: 新增 AdminQueryService struct + 15 个方法 receiver 变更
  - impl.go: 删除 10 个 admin repo 字段
  - factory.go + master.go: 构造注入更新
  - 验证: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v` 全 PASS
- Phase 2: AIOpsQueryService 拆分 — 待办
  - aiops.go: 新增 AIOpsQueryService struct + 13 个方法 receiver 变更
  - impl.go: 删除 aiopsEngine, aiopsAI, aiReportRepo 字段
  - factory.go + master.go: 构造注入更新
  - 验证: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v` 全 PASS
- Phase 3: SLOQueryService 拆分 — 待办
  - slo.go: 新增 SLOQueryService struct + 6 个方法 receiver 变更
  - impl.go: 删除 sloRepo 字段
  - slo_test.go: mock + 构造更新（23 个测试迁移）
  - factory.go + master.go: 构造注入更新
  - 验证: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v` 全 PASS
- Phase 4: OTelQueryService 拆分 — 待办
  - otel.go: 新增 OTelQueryService struct + 2 个方法 receiver 变更
  - overview_test.go: OTel 测试构造对象从 QueryService 改为 OTelQueryService（原位更新，不搬迁文件）
  - factory.go + master.go: 构造注入更新
  - 验证: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v` 全 PASS
- Phase 5: K8sQueryService 拆分 — 待办
  - k8s.go: 新增 K8sQueryService struct + 19 个方法 receiver 变更
  - k8s_test.go: mock + 构造更新（26 个测试迁移）
  - factory.go + master.go: 构造注入更新
  - 验证: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v` 全 PASS
- Phase 6: OverviewQueryService 拆分 + 收尾 — 待办
  - overview.go: 新增 OverviewQueryService struct + 11 个方法 receiver 变更
  - overview_test.go: Overview 测试 mock + 构造更新（31 个测试迁移）
  - impl_test.go: 删除旧构造测试
  - impl.go: 删除 QueryService、QueryServiceDeps、NewQueryService
  - master.go: EventTrigger 引用从 q 改为 overviewQ
  - factory.go: 最终形态
  - 验证: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v` 全 PASS
