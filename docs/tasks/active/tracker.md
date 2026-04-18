# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

---

## Agent Metrics / Summary Counter Reset 修复 — 🔄 进行中

> 原设计文档: [agent-metrics-counter-reset-fix-design.md](../../design/active/agent-metrics-counter-reset-fix-design.md)
>
> 清理 SLO 修复遗留的 7 处裸 `argMax-argMin` rate 计算。5min 短窗口复用 `gaugeCounterDelta`（单次 reset 安全）+ 抽 DRY 常量 `counterRateExpr`。

- Phase 1: 新增 `counterRateExpr` 常量 — 待办
  - slo.go: `gaugeCounterDelta` 下方追加常量定义 + 说明注释
- Phase 2: metrics.go 5 处 SQL 替换 — 待办
  - fillCPU / fillDisks ioQuery / fillNetworks rateQuery / fillPSI / fillVMStat rateQuery 模板
- Phase 3: summary.go 2 处 SQL 替换 — 待办
  - GetSLOSummary ingressQuery / GetMetricsSummary cpuQuery
- Phase 4: 编译 + 测试 + commit — 待办
  - `go build ./...` PASS
  - `go test ./atlhyper_agent_v2/repository/ch/...` PASS
  - commit: "fix(agent): metrics/summary 5min 窗口 rate 计算 counter-reset-safe"
- Phase 5: 本地端到端验证 — 待办
  - 节点监控页 CPU/Disk/Network/PSI/VMStat 数据正常
  - 概览页集群 CPU / Ingress RPS 正常

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
