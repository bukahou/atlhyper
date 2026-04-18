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

## AIOps 风险分数单位统一 — 🔄 进行中

> 原设计文档: [aiops-risk-score-unify-design.md](../../design/active/aiops-risk-score-unify-design.md)
>
> 拓扑图 badge 显示 4500（实际 45%）、节点全红的 bug。根治方案：建立 `lib/risk.ts` 单一信任源，迁移 5 个消费者，补 API 类型注释和后端 scale_risk.go 约定注释。

- Phase 1: 新建 `lib/risk.ts` — 待办
  - 导出 `RISK_THRESHOLDS` / `riskLevel` / `riskColor` / `formatRiskScore` / `isRisky`
- Phase 2: 迁移 5 个消费者 — 待办
  - topology-graph-utils.ts（删本地 riskColor，badge/isAnomaly 改用共享函数 + 阈值常量）
  - TopologyGraph.tsx tooltip
  - NodeDetail.tsx 显示
  - RootCauseCard.tsx 显示 + RiskBadge level 判断
  - IncidentDetailModal.tsx 显示
- Phase 3: 补 API + 后端注释 — 待办
  - api/aiops.ts 四个 rXxx 字段 JSDoc `@unit 百分制 [0,100]`
  - scale_risk.go 文件头补单位约定注释
- Phase 4: 编译 + 验证 + commit — 待办
  - `npm run build` PASS
  - 浏览器刷新 /aiops/topology，badge 显示 45 而非 4500，染色合理
  - commit: "fix(aiops): 前端风险分数单位彻底统一（lib/risk 单一信任源）"

---

## Agent 泛用型 Mesh 监控（Kustomize Overlay + OTel Transform + 代码去 Linkerd 化）— 🔄 进行中

> 原设计文档: [agent-mesh-generic-monitoring-design.md](../../design/active/agent-mesh-generic-monitoring-design.md)
>
> 实施顺序：**Config 先行，代码后改**。先重构 config 仓库为 base/mesh overlay 结构，切 mesh/none 验证重构→部署 Istio→切 mesh/istio→ClickHouse 确认有 `mesh_request_total`→最后改 Agent 代码查统一名。
>
> 跨仓库任务：config 仓库在 `~/AtlHyper/GitHub/config/`，本项目 atlhyper 在 `~/work/github/atlhyper/`。

- Phase 0: 本机工具 — 待办
  - 下载 istioctl 1.25.1 到 `~/istio-1.25.1/`
  - 改 `config/local/ubuntu/env/infra.env` 加 ISTIO PATH 导出
  - 新 shell 验证 `istioctl version` 可用
- Phase 1: Config 仓库重构（本地完成，**不推送**）— 待办
  - 建 `config/clusters/zgmf-x10a/apps/atlhyper/base/` + `mesh/{none,linkerd,istio}/`
  - 拆 OTel：`atlhyper-otel-collector.yaml` 只留 Deployment/Service/RBAC；ConfigMap data 移到各 mesh overlay
  - 写 3 份 `atlhyper-otel-config.yaml`（none/linkerd/istio，共享基础段 + 各自 mesh 段）
  - 写 3 份 `mesh/*/kustomization.yaml` 和 `base/kustomization.yaml`
  - 改顶层 `kustomization.yaml`：`resources: [base, mesh/none]`
  - 确认 OTel Collector 镜像是 `contrib` 版（transform processor 必需）
  - 本地 `kustomize build .` 和旧版 diff，确认只有 mesh 相关差异
- Phase 2: 推送 config + 切 mesh/none 验证 — 待办
  - 推送 config 仓库 main 分支
  - 等 Deployer 30-60s 应用
  - 验证：`kubectl -n atlhyper get cm otel-collector-config -o yaml | grep -E 'linkerd|istio'` 无输出
  - 业务 Pod 全 Running
- Phase 3: 部署 Istio — 待办
  - `istioctl install --set profile=default --skip-confirmation`
  - `kubectl label ns geass-v2-api istio-injection=enabled`
  - `kubectl label ns geass-v2-web istio-injection=enabled`
  - `kubectl -n geass-v2-{api,web} rollout restart deployment`
  - 验证：Pod 2/2
- Phase 4: 切 mesh/istio overlay — 待办
  - 顶层 kustomization 改 `- mesh/istio`
  - 推送，OTel Collector 自动重启加载新配置
- Phase 5: ClickHouse 实测 — 待办
  - 访问 `https://geass-api.bukahou.com` 产生流量
  - `SELECT count() FROM otel_metrics_sum WHERE MetricName='mesh_request_total'` 应返回 >0
  - 采集若干行 label，确认 workload/namespace/direction/mtls 等字段正确
  - 把 Istio 真实 label 反馈到设计文档（万一有字段和预期不同需要微调 transform）
- Phase 6: Agent 代码改造 — 待办
  - slo.go：4 处 SQL rename（`mv_linkerd_*` → `mv_mesh_*`、`response_total` → `mesh_request_total`、`deployment` → `workload`、`tls` → `mtls`，删 Linkerd 专属 filter）
  - summary.go：2 处同类改动
  - ClickHouse MV DDL：建 `mv_mesh_request_total` + `mv_mesh_latency_bucket`（找到现有 Linkerd MV DDL 存放位置再加）
  - `go build ./...` + `go test ./atlhyper_agent_v2/repository/ch/query/` PASS
  - commit + 刷新 Web `/observe/slo` 服务网格 tab 验证

### 验证清单（完整列表见设计文档 §5）

| # | 验证点 | 通过标准 |
|---|-------|---------|
| V1 | `which istioctl` | 路径正确 |
| V2 | `kustomize build .` | 输出合理，diff 只含 mesh 差异 |
| V3 | mesh/none 切换后 Pod 状态 | 全 Running |
| V4 | Istio sidecar 注入 | geass-v2-{api,web} Pod 2/2 |
| V5 | OTel ConfigMap 含 transform | `istio-sidecars` + `transform/istio_to_mesh` 可见 |
| V6 | ClickHouse 有 mesh_request_total | `count() > 0` |
| V7 | Agent 编译 + 测试 | PASS |
| V8 | Web mesh 页面 | 有数据 |

### 关键约束

- OTel Collector 镜像必须是 **contrib 版**（`transform` processor 在 contrib 中），Phase 1 确认
- ClickHouse MV DDL 当前**不在 atlhyper 项目代码中**，Phase 5-6 前需定位存放位置
- Phase 3-6 之间 mesh 页面可能暂时空白（OTel 已 rename 但 Agent 还查老名字），用户可接受（集群本来也无 mesh 数据）

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
