# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

---

## Agent SLO Counter Reset 修复 — 🔄 待端到端验证

> 原设计文档: [agent-slo-counter-reset-fix-design.md](../../design/active/agent-slo-counter-reset-fix-design.md)
>
> 代码改动已完成并 commit（`8cb469f` + `712f902`）。剩最后端到端验证一步。

- Phase 1: Collect/Push 拆独立 ctx — ✅ 完成（commit 8cb469f）
- Phase 2: SLO counter reset SQL 替换 — ✅ 完成（commit 712f902）
  - queryIngressSLO countQuery → lagInFrame + sum(if reset) window function
  - GetIngressSLOHistory countQuery → 同算法 + 加桶 ts 分区
  - ClickHouse 实测新 SQL：24h 内 atlhyper-web GET 200 = 1564 次（合理）
- Phase 3: 清理临时 debug 日志 — ✅ 完成（包含在 commit 712f902）
- Phase 4: 编译 + 测试 + commit — ✅ 完成
  - `go build ./...` PASS
  - `go test ./atlhyper_agent_v2/repository/ch/query/ ./scheduler/ ./service/snapshot/` PASS
  - 2 个 commit 已提交（未 push）
- Phase 5: 端到端验证 — 待办
  - 前置：port-forward `svc/clickhouse 9000:9000`
  - 启 Master: `go run ./cmd/atlhyper_master_v2/main.go`
  - 启 Agent: `go run ./cmd/atlhyper_agent_v2/main.go`（等 Agent 首次推送成功）
  - 实测 API: `curl "http://localhost:8080/api/v2/slo/domains/v2?cluster_id=ZGFX-X10A&time_range=1d"` — 期望 totalRequests > 0
  - 刷新前端 `/observe/slo` — 期望顶部卡片显示真实请求数/P95/P99/错误预算
  - 验证通过后：本任务归档到 `docs/tasks/archive/`，设计文档从 active 移到 archive

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
