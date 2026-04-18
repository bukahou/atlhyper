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
>
> **实施三大步，每步独立可验证闭环**：
> - **Step 1 修改配置**（Phase 0-2）：config 仓库重构 + 切 mesh/none，集群与现状等价
> - **Step 2 部署 Istio 并验证**（Phase 3-5）：Istio 装机 + 切 mesh/istio + ClickHouse 实测有 `mesh_request_total`
> - **Step 3 修改代码并测试**（Phase 6）：Agent 代码去 Linkerd 化 + Web 端到端验证

### Step 1：修改配置（Config 仓库重构 + 切 mesh/none 验证）

> 闭环目标：重构后集群所有业务功能和现状完全等价（零回退），mesh/none 下验证重构本身不破坏链路。

- Phase 0: 本机工具 — ✅ 完成
  - istioctl 1.25.1 已下载到 `~/istio-1.25.1/`
  - `config/local/ubuntu/env/infra.env` 已加 ISTIO_HOME + PATH（commit `5e8f90a`）
  - 验证：`istioctl version` = 1.25.1，`istioctl x precheck` = No issues found
- Phase 1: Config 仓库重构（本地完成，**先不推送**）— ✅ 完成（config 仓库 commit `2bd01ce`）
  - `base/` 含 9 个 yaml（git rename 100% 相似度）+ kustomization.yaml
  - `mesh/{none,linkerd,istio}/` 各自 atlhyper-otel-config.yaml + kustomization.yaml
  - 原 atlhyper-otel-authz.yaml 搬到 mesh/linkerd/（仍保持独立手动 apply）
  - 顶层 kustomization.yaml 当前 `resources: [base, mesh/none]`
  - OTel 镜像确认 contrib 版（0.96.0）
  - `kubectl kustomize .` 输出 23 个资源；mesh/none 与旧版 diff 仅含 linkerd scrape 删除（预期）
  - **附带发现**：Linkerd MV DDL 在 `base/atlhyper-clickhouse.yaml` 的 init-db.sh，Phase 6 在此处添加 mesh MV
- Phase 2: 推送 config + 切 mesh/none 验证 — ✅ 完成
  - config 仓库已推送到 origin（3 个 commit：5e8f90a / d220dff / 2bd01ce）
  - `kubectl apply -k .` 手动应用（atlhyper 不在 Deployer 监听范围内）
  - `rollout restart deploy/otel-collector` 加载新 ConfigMap
  - 验证：ConfigMap 扫描 linkerd/istio/metricstransform/transform 关键字均无输出
  - atlhyper ns 所有 Pod Running；OTel 日志「Begin running and processing data」且 scrape 只有 traefik + node-exporter
  - **Step 1 闭环通过**

### Step 2：部署 Istio 以及验证（Istio 装机 + 切 mesh/istio + ClickHouse 实测）

> 闭环目标：Istio 部署到 geass-v2-{api,web} 两个 ns，OTel Collector 通过新 transform 将 `istio_requests_total` 重命名为 `mesh_request_total` 写入 ClickHouse，SQL 实测能查到数据。

- Phase 3: 部署 Istio — ✅ 完成
  - 首装用默认大 sidecar（2000m limits）撞 quota=16 上限
  - 重装 `--set values.global.proxy.resources.limits.cpu=100m` 小 sidecar
  - 两个 ns 打 label + rollout restart 触发注入
  - 临时扩 quota 16→24 让滚动完成；auth/web 再 rollout restart 替换小 sidecar；quota 缩回 16
  - 最终 geass-v2-api 13 Pod + geass-v2-web 2 Pod 全部 2/2，quota used=12.3 core
- Phase 4: 切 mesh/istio overlay — ✅ 完成（config 3 个 fix commit）
  - 97c4749: 顶层 kustomization mesh/none → mesh/istio
  - 23dca26: 移除 v0.99+ 才支持的 `conditions` 字段，改用每条 statement 自带 where 过滤
  - 353db81: scrape 走 istio-proxy 声明的 15090 端口 + `metrics_path: /stats/prometheus`
- Phase 5: ClickHouse 实测 — ✅ 完成
  - OTel 8889 可见 `otel_mesh_request_total`，7 个统一 label 正确填充（workload / namespace / dst_workload / dst_namespace / direction / status_code / mtls）
  - ClickHouse 5 min 内有 76 行 `mesh_request_total`，**没有 istio_requests_total**（已被完全重命名）
  - 附带小噪音：istio-ingressgateway 声明多端口（15021/8443/8080），对这些端口的 scrape 会失败，仅 warn 日志，不影响数据
  - **Step 2 闭环通过**

### Step 3：修改代码，进行测试（Agent 代码去 Linkerd 化 + 端到端验证）

> 闭环目标：Agent SQL 查 `mesh_request_total`，ClickHouse MV 建好，Web `/observe/slo` 服务网格页面显示真实数据。

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
