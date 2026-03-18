# QueryService 拆分设计

## 背景与问题定义

### 现状

`service/query/impl.go` 定义了一个 `QueryService` 结构体，承载 6 个功能域、66 个方法、14 个依赖字段：

```go
type QueryService struct {
    store          datahub.Store                    // k8s, otel, overview, slo
    bus            mq.Producer                      // overview (1 方法)
    eventRepo      database.ClusterEventRepository  // overview, admin
    sloRepo        database.SLORepository           // slo
    aiopsEngine    aiops.Engine                     // aiops
    aiopsAI        *enricher.Enricher               // aiops (1 方法)
    auditRepo      database.AuditRepository         // admin
    commandRepo    database.CommandHistoryRepository // admin
    notifyRepo     database.NotifyChannelRepository  // admin
    settingsRepo   database.SettingsRepository       // admin
    aiProviderRepo database.AIProviderRepository     // admin
    aiSettingsRepo database.AISettingsRepository     // admin
    aiModelRepo    database.AIProviderModelRepository // admin
    aiBudgetRepo   database.AIRoleBudgetRepository   // admin
    aiReportRepo   database.AIReportRepository       // admin, aiops
}
```

### 问题

| 问题 | 影响 |
|------|------|
| **上帝对象** | 一个 struct 持有 14 个依赖、66 个方法，违反单一职责 |
| **依赖膨胀** | 每个功能域只用 1-3 个依赖，但被迫注入全部 14 个 |
| **测试负担** | 测试 k8s.go 只需 store，但 mock 必须满足整个 Store 接口才能构造 QueryService |
| **耦合风险** | 新增功能域时 QueryService 继续膨胀，所有测试文件潜在受影响 |

### 有利条件

| 条件 | 说明 |
|------|------|
| **子接口已定义** | `interfaces.go` 已有 6 个子接口（QueryK8s/OTel/SLO/AIOps/Overview/Admin） |
| **方法无名冲突** | 6 个子接口共 69 个方法签名，无一重名 |
| **文件已按域分离** | k8s.go/otel.go/slo.go/aiops.go/overview.go/admin.go 已按职责分文件 |
| **测试地基完备** | k8s(144), overview(36), otel(5), slo(23) 共 208 个测试守护 |
| **Go 嵌入组合** | Go struct embedding 天然支持"多小 struct 组合满足大接口" |

## 职责盘点

### 按功能域分类

| 域 | 文件 | 行数 | 方法数 | 依赖 | 子接口 |
|----|------|------|--------|------|--------|
| **K8s** | k8s.go | 362 | 19 | store | QueryK8s |
| **OTel** | otel.go | 24 | 2 | store | QueryOTel |
| **Overview** | overview.go | 491 | 11 | store, bus, eventRepo | QueryOverview |
| **SLO** | slo.go | 304 | 6 | store, sloRepo | QuerySLO |
| **AIOps** | aiops.go | 113 | 13 | aiopsEngine, aiopsAI, aiReportRepo | QueryAIOps |
| **Admin** | admin.go | 86 | 15 | 10 个 repo | QueryAdmin |
| (工具) | metrics_format.go | 120 | 0 | (无) | (包级函数) |
| (定义) | impl.go | 86 | 0 | (无) | (struct+构造) |
| **合计** | 8 文件 | 1,586 | 66 | 14 字段 | 6 子接口 |

### 依赖使用矩阵

```
依赖 \ 使用文件     k8s  otel  overview  slo  aiops  admin
─────────────────────────────────────────────────────────
store               ✓    ✓     ✓         ✓
bus                              ✓
eventRepo                        ✓                   ✓
sloRepo                                   ✓
aiopsEngine                                    ✓
aiopsAI                                        ✓
auditRepo                                            ✓
commandRepo                                          ✓
notifyRepo                                           ✓
settingsRepo                                         ✓
aiProviderRepo                                       ✓
aiSettingsRepo                                       ✓
aiModelRepo                                          ✓
aiBudgetRepo                                         ✓
aiReportRepo                                   ✓     ✓
```

**关键发现**: 依赖几乎完全不重叠。唯一例外是 `aiReportRepo`（admin + aiops）和 `eventRepo`（overview + admin），但这只是两个 struct 各自持有同一实例的引用——不构成耦合问题。

## 目标拆分结构

### 拆分为 6 个独立 Query Service

```
service/query/
├── impl.go              ← 删除 QueryService，只保留包级工具函数
├── k8s.go               ← K8sQueryService{store}           → 满足 QueryK8s
├── otel.go              ← OTelQueryService{store}          → 满足 QueryOTel
├── overview.go          ← OverviewQueryService{store,bus,eventRepo} → 满足 QueryOverview
├── slo.go               ← SLOQueryService{store,sloRepo}   → 满足 QuerySLO
├── aiops.go             ← AIOpsQueryService{engine,ai,reportRepo} → 满足 QueryAIOps
├── admin.go             ← AdminQueryService{10 repos}      → 满足 QueryAdmin
├── metrics_format.go    ← 不变（包级纯函数）
├── k8s_test.go          ← 更新 receiver
├── overview_test.go     ← 更新 receiver
├── slo_test.go          ← 更新 receiver
├── metrics_format_test.go ← 不变
└── impl_test.go         ← 更新 receiver
```

### 每个子服务的职责边界

#### K8sQueryService

```go
type K8sQueryService struct {
    store datahub.Store
}
func NewK8sQueryService(store datahub.Store) *K8sQueryService
```

- 承接: k8s.go 全部 19 个方法
- 满足: `service.QueryK8s`
- 依赖: `datahub.Store`（只读 GetSnapshot）

#### OTelQueryService

```go
type OTelQueryService struct {
    store datahub.Store
}
func NewOTelQueryService(store datahub.Store) *OTelQueryService
```

- 承接: otel.go 全部 2 个方法
- 满足: `service.QueryOTel`
- 依赖: `datahub.Store`（只读 GetSnapshot, GetOTelTimeline）

#### OverviewQueryService

```go
type OverviewQueryService struct {
    store     datahub.Store
    bus       mq.Producer
    eventRepo database.ClusterEventRepository
}
func NewOverviewQueryService(store datahub.Store, bus mq.Producer, eventRepo database.ClusterEventRepository) *OverviewQueryService
```

- 承接: overview.go 全部 11 个方法
- 满足: `service.QueryOverview`
- 依赖: `datahub.Store` + `mq.Producer` + `database.ClusterEventRepository`
- 外部消费者: `notifier/enrich.ResourceQuery`（4 个方法子集：GetPod/GetNode/GetDeployment/GetDeploymentByReplicaSet）

#### SLOQueryService

```go
type SLOQueryService struct {
    store   datahub.Store
    sloRepo database.SLORepository
}
func NewSLOQueryService(store datahub.Store, sloRepo database.SLORepository) *SLOQueryService
```

- 承接: slo.go 全部 6 个方法
- 满足: `service.QuerySLO`
- 依赖: `datahub.Store`（GetMeshTopology/GetServiceDetail 读 OTel 快照）+ `database.SLORepository`

#### AIOpsQueryService

```go
type AIOpsQueryService struct {
    engine    aiops.Engine
    ai        *enricher.Enricher
    reportRepo database.AIReportRepository
}
func NewAIOpsQueryService(engine aiops.Engine, ai *enricher.Enricher, reportRepo database.AIReportRepository) *AIOpsQueryService
```

- 承接: aiops.go 全部 13 个方法
- 满足: `service.QueryAIOps`
- 依赖: `aiops.Engine` + `*enricher.Enricher` + `database.AIReportRepository`

#### AdminQueryService

```go
type AdminQueryService struct {
    auditRepo      database.AuditRepository
    commandRepo    database.CommandHistoryRepository
    eventRepo      database.ClusterEventRepository
    notifyRepo     database.NotifyChannelRepository
    settingsRepo   database.SettingsRepository
    aiProviderRepo database.AIProviderRepository
    aiSettingsRepo database.AISettingsRepository
    aiModelRepo    database.AIProviderModelRepository
    aiBudgetRepo   database.AIRoleBudgetRepository
    aiReportRepo   database.AIReportRepository
}
func NewAdminQueryService(repos AdminRepos) *AdminQueryService
```

- 承接: admin.go 全部 15 个方法
- 满足: `service.QueryAdmin`
- 依赖: 10 个 database.Repository
- `AdminRepos` 结构体保留（聚合注入参数）

## 接口层演化

### `service/interfaces.go` — 不变

6 个子接口 + Query 聚合接口 + Service 组合接口 **完全不变**。这是本次重构的核心约束：接口不动，只动实现。

### `service/factory.go` — 更新嵌入

**Before:**

```go
type serviceImpl struct {
    *query.QueryService          // 单一大 struct 满足 Query
    *operations.CommandService
    *operations.AdminService
    *operations.SLOService
}

func NewService(q *query.QueryService, cmd *operations.CommandService, admin *operations.AdminService, slo *operations.SLOService) Service
```

**After:**

```go
type serviceImpl struct {
    *query.K8sQueryService       // 满足 QueryK8s
    *query.OTelQueryService      // 满足 QueryOTel
    *query.OverviewQueryService  // 满足 QueryOverview
    *query.SLOQueryService       // 满足 QuerySLO
    *query.AIOpsQueryService     // 满足 QueryAIOps
    *query.AdminQueryService     // 满足 QueryAdmin
    *operations.CommandService
    *operations.AdminService
    *operations.SLOService
}

func NewService(
    k8s *query.K8sQueryService,
    otel *query.OTelQueryService,
    overview *query.OverviewQueryService,
    sloQ *query.SLOQueryService,
    aiopsQ *query.AIOpsQueryService,
    adminQ *query.AdminQueryService,
    cmd *operations.CommandService,
    admin *operations.AdminService,
    slo *operations.SLOService,
) Service
```

Go struct embedding 保证 serviceImpl 自动满足 `Query`（= QueryK8s + QueryOTel + ... + QueryAdmin）。

### `master.go` — 更新构造

**Before:**

```go
q := query.NewQueryService(query.QueryServiceDeps{
    Store: store, Bus: bus, EventRepo: db.Event, SLORepo: db.SLO,
    AIOpsEngine: aiopsEngine, AIOpsAI: aiopsEnricher,
    AdminRepos: query.AdminRepos{...},
})
svc := service.NewService(q, cmdOps, adminOps, sloOps)
```

**After:**

```go
k8sQ := query.NewK8sQueryService(store)
otelQ := query.NewOTelQueryService(store)
overviewQ := query.NewOverviewQueryService(store, bus, db.Event)
sloQ := query.NewSLOQueryService(store, db.SLO)
aiopsQ := query.NewAIOpsQueryService(aiopsEngine, aiopsEnricher, db.AIReport)
adminQ := query.NewAdminQueryService(query.AdminRepos{...})

svc := service.NewService(k8sQ, otelQ, overviewQ, sloQ, aiopsQ, adminQ, cmdOps, adminOps, sloOps)
```

**master.go 中 EventTrigger 的引用**（L522）：

```go
// Before: trigger.NewEventTrigger(db.Event, q, alertMgr, ...)
// After:  trigger.NewEventTrigger(db.Event, overviewQ, alertMgr, ...)
```

`overviewQ` 满足 `enrich.ResourceQuery`（4 方法子集），无需额外适配。

## 分阶段重构方案

### 核心原则

- **Strangler Pattern**: 每个 Phase 从 QueryService 剥离一个域，剩余方法仍在 QueryService 上
- **零行为变更**: 每步只做 struct 拆分 + receiver 变更，不改业务逻辑
- **编译即验证**: Go 类型系统保证接口满足性，编译通过 = 接口正确
- **可回退**: 每步独立 commit，`git revert` 即可恢复

### Phase 0: 准备（factory 签名前置扩展）

**目标**: 让 factory.go 同时接受旧 QueryService 和未来的新 struct，为后续 Phase 提供落地空间。

**操作**:
1. `factory.go`: NewService 签名暂不变，但在 serviceImpl 注释中标注即将拆分
2. 确认 `go build` + `go test ./atlhyper_master_v2/service/query/` 全绿

> 实际上 Phase 0 可以跳过——每个 Phase 自带 factory 更新。保留此步骤仅作为检查点。

### Phase 1: 拆分 AdminQueryService

**为什么先拆 Admin**:
- 依赖最多（10 个 repo）但完全自包含，不访问 store/bus
- 与其他 5 个域零依赖重叠（aiReportRepo 虽共享，但各持各的引用即可）
- 目前无 admin 测试（不存在 admin_test.go），拆分时无需迁移测试
- 拆出后 QueryService 的字段从 14 降到 6，立竿见影

**操作**:
1. `admin.go`: 新增 `AdminQueryService` struct + `NewAdminQueryService` 构造函数
2. `admin.go`: 所有 15 个方法 receiver 从 `*QueryService` 改为 `*AdminQueryService`
3. `impl.go`: 从 `QueryService` 删除 10 个 admin repo 字段；`AdminRepos` struct 保留；`QueryServiceDeps` 删除 `AdminRepos` 字段；`NewQueryService` 删除 admin 注入
4. `factory.go`: serviceImpl 新增嵌入 `*query.AdminQueryService`；NewService 签名新增参数
5. `master.go`: 构造 `adminQ := query.NewAdminQueryService(...)`，传入 NewService

**验收**:
- `go build ./...`
- `go test ./atlhyper_master_v2/service/query/ -v`（144/144 PASS，admin 无测试不受影响）

**风险**: 极低。Admin 方法完全自包含。

**回退**: `git revert` 单次 commit。

### Phase 2: 拆分 AIOpsQueryService

**为什么第二拆 AIOps**:
- 依赖独特（aiopsEngine, aiopsAI），与 store 系列零重叠
- 目前无 aiops 测试，无需迁移

**操作**:
1. `aiops.go`: 新增 `AIOpsQueryService` struct + `NewAIOpsQueryService`
2. `aiops.go`: 13 个方法 receiver 改为 `*AIOpsQueryService`
3. `impl.go`: 从 `QueryService` 删除 aiopsEngine, aiopsAI, aiReportRepo 字段
4. `factory.go` + `master.go`: 同 Phase 1 模式

**验收**: `go build` + `go test`（144/144 PASS）

**风险**: 极低。

### Phase 3: 拆分 SLOQueryService

**为什么第三拆 SLO**:
- 依赖: store + sloRepo（store 是共享依赖，但 SLO 是首个涉及 store 的拆分）
- 已有 23 个测试（slo_test.go），需要迁移 mock + receiver

**操作**:
1. `slo.go`: 新增 `SLOQueryService` struct + `NewSLOQueryService`
2. `slo.go`: 6 个方法 receiver 改为 `*SLOQueryService`
3. `impl.go`: 从 `QueryService` 删除 sloRepo 字段
4. `slo_test.go`: mock 和测试构造从 `&QueryService{store: ..., sloRepo: ...}` 改为 `&SLOQueryService{store: ..., sloRepo: ...}`
5. `factory.go` + `master.go`: 同上

**验收**: `go build` + `go test`（144/144 PASS，重点关注 slo_test.go 的 23 个测试）

**风险**: 低。测试迁移是机械操作。

### Phase 4: 拆分 OTelQueryService

**操作**:
1. `otel.go`: 新增 `OTelQueryService{store}` + `NewOTelQueryService`
2. `otel.go`: 2 个方法 receiver 改为 `*OTelQueryService`
3. `overview_test.go`: OTel 相关的 5 个测试构造对象从 `&QueryService{store: ...}` 改为 `&OTelQueryService{store: ...}`，测试仍留在 overview_test.go 中
4. `factory.go` + `master.go`: 同上

**验收**: `go build` + `go test`（144/144 PASS，重点关注 5 个 OTel 测试）

**风险**: 低。OTel 测试留在 overview_test.go 原位，只改构造对象，不搬迁文件。

> **测试文件归属说明**: Phase 4 不强制将 OTel 测试从 overview_test.go 迁移到 otel_test.go。结构拆分阶段应专注于 struct + receiver 变更，测试文件的物理归属整理可以在全部 6 个 Phase 完成后作为独立的收尾任务执行。混入文件搬迁会增加 diff 噪音和出错概率，与"最小变更、可回退"原则冲突。

### Phase 5: 拆分 K8sQueryService

**为什么 K8s 和 Overview 分开拆**:
- K8s (19 方法) 只依赖 store，与 Overview 的依赖（store + bus + eventRepo）和外部消费者（EventTrigger）完全无关
- 分开后每个 Phase 的联动范围更小：Phase 5 只动 k8s.go + k8s_test.go，Phase 6 只动 overview.go + overview_test.go + EventTrigger
- 如果 Phase 6 的 EventTrigger 引用变更出问题，不会连带 K8s 的 26 个测试回滚

**操作**:
1. `k8s.go`: 新增 `K8sQueryService{store}` + `NewK8sQueryService`
2. `k8s.go`: 19 个方法 receiver 改为 `*K8sQueryService`
3. `k8s_test.go`: 构造改为 `&K8sQueryService{store: ...}`
4. `impl.go`: 从 QueryService 删除 store 字段（如果此时 store 仍被 overview 使用，则保留，Phase 6 再删）
5. `factory.go` + `master.go`: 更新嵌入和构造

**验收**: `go build` + `go test`（144/144 PASS，重点关注 k8s_test.go 的 26 个测试）

**风险**: 低。K8s 只依赖 store，mock 最简单，无外部消费者。

**注意**: Phase 5 完成后 QueryService 仍存在（持有 overview 所需的 store + bus + eventRepo），Phase 6 才彻底删除。

### Phase 6: 拆分 OverviewQueryService + 收尾

**为什么最后拆 Overview**:
- Overview 依赖最复杂（store + bus + eventRepo），且有外部消费者 `notifier/enrich.ResourceQuery`
- 此步同时完成 QueryService 彻底清除（删除 struct + QueryServiceDeps + NewQueryService）
- EventTrigger 参数引用从 `q` 改为 `overviewQ`，需要确认 `enrich.ResourceQuery` 接口兼容

**操作**:
1. `overview.go`: 新增 `OverviewQueryService{store, bus, eventRepo}` + `NewOverviewQueryService`
2. `overview.go`: 11 个方法 receiver 改为 `*OverviewQueryService`
3. `overview_test.go`: Overview 测试构造改为 `&OverviewQueryService{store: ..., bus: ..., eventRepo: ...}`
4. `impl_test.go`: 删除旧 QueryService 构造测试，替换为对应子服务的构造验证
5. `impl.go`: 删除 `QueryService` struct、`QueryServiceDeps`、`NewQueryService`（彻底清除）
6. `factory.go`: 最终形态（serviceImpl 嵌入 6 个子服务）
7. `master.go`: 最终形态 + EventTrigger 参数从 `q` 改为 `overviewQ`

**验收**: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v`（144/144 PASS）

**风险**: 中低。Overview 测试迁移是机械操作。EventTrigger 的 `enrich.ResourceQuery` 是 QueryOverview 的子集（4 方法），`OverviewQueryService` 天然满足，编译器会强制验证。

**回退**: `git revert` 单次 commit。

## 每个 Phase 的风险点与回退策略

| Phase | 改动文件 | 测试迁移 | 外部消费者 | 风险 | 回退 |
|-------|---------|----------|-----------|------|------|
| 1 Admin | 4 | 无 | 无 | 极低 | git revert |
| 2 AIOps | 4 | 无 | 无 | 极低 | git revert |
| 3 SLO | 5 | slo_test.go (23) | 无 | 低 | git revert |
| 4 OTel | 4 | overview_test.go 中 5 个（原位更新构造） | 无 | 低 | git revert |
| 5 K8s | 5 | k8s_test.go (26) | 无 | 低 | git revert |
| 6 Overview+收尾 | 6 | overview_test.go (31) + impl_test.go (2) | EventTrigger | 中低 | git revert |

## 测试策略

### 现有安全网

| 测试文件 | 测试数 | 覆盖域 | 受影响 Phase |
|---------|--------|--------|-------------|
| k8s_test.go | 26 (含 55 子测试) | K8s 19 方法 | Phase 5 |
| metrics_format_test.go | 2 (含 25 子测试) | 纯函数 | 不受影响 |
| overview_test.go | 36 (含 OTel 5) | Overview 11 + OTel 2 | Phase 4 (OTel 构造), Phase 6 (Overview 构造) |
| slo_test.go | 23 | SLO 6 方法 | Phase 3 |
| impl_test.go | 2 | 构造函数 | Phase 6 |
| **合计** | **89 (含子测试 ~200)** | | |

### 迁移策略

每个 Phase 的测试迁移是**机械操作**：
1. 将 `&QueryService{store: mock}` 改为 `&XxxQueryService{store: mock}`
2. 删除不再需要的 mock 字段（如 k8s_test.go 的 mock 本来就只用了 store）
3. 编译 + 运行 → 全绿

### 是否需要补新测试

**不需要**。本次重构是纯结构拆分（receiver 变更），不改业务逻辑。现有 144 个测试已覆盖所有被拆分的方法。如果拆分后所有测试仍然 PASS，即证明重构正确。

唯一例外：如果 impl_test.go 中有测试 `NewQueryService` 构造函数的用例，Phase 6 删除 QueryService 后需要替换为各子服务的构造函数测试。但这是删旧补新，不是额外工作。

### 测试文件物理归属

结构拆分阶段（Phase 1-6）不要求测试文件的物理搬迁。例如 OTel 测试可以继续留在 overview_test.go 中，只需更新构造对象。测试文件整理（如将 OTel 测试移到独立的 otel_test.go）可在全部拆分完成后作为独立收尾任务执行。原因：文件搬迁增加 diff 噪音，与"最小变更、可回退"原则冲突。

## 文件变更清单

### Phase 1: Admin

```
atlhyper_master_v2/
├── service/
│   ├── factory.go                [修改] serviceImpl 嵌入 + NewService 签名
│   └── query/
│       ├── admin.go              [修改] 新增 AdminQueryService, receiver 变更
│       └── impl.go              [修改] 删除 admin 相关字段
├── master.go                     [修改] 构造 AdminQueryService
```

### Phase 2: AIOps

```
atlhyper_master_v2/
├── service/
│   ├── factory.go                [修改]
│   └── query/
│       ├── aiops.go              [修改] 新增 AIOpsQueryService, receiver 变更
│       └── impl.go              [修改] 删除 aiops 相关字段
├── master.go                     [修改]
```

### Phase 3: SLO

```
atlhyper_master_v2/
├── service/
│   ├── factory.go                [修改]
│   └── query/
│       ├── slo.go                [修改] 新增 SLOQueryService, receiver 变更
│       ├── slo_test.go           [修改] mock + 构造更新
│       └── impl.go              [修改] 删除 sloRepo 字段
├── master.go                     [修改]
```

### Phase 4: OTel

```
atlhyper_master_v2/
├── service/
│   ├── factory.go                [修改]
│   └── query/
│       ├── otel.go               [修改] 新增 OTelQueryService, receiver 变更
│       ├── overview_test.go      [修改] OTel 测试构造从 QueryService 改为 OTelQueryService（原位更新，不搬迁文件）
│       └── impl.go              [修改] (如有残留字段)
├── master.go                     [修改]
```

### Phase 5: K8s

```
atlhyper_master_v2/
├── service/
│   ├── factory.go                [修改]
│   └── query/
│       ├── k8s.go                [修改] 新增 K8sQueryService, receiver 变更
│       ├── k8s_test.go           [修改] mock + 构造更新
│       └── impl.go              [修改] (store 字段如仍被 overview 使用则保留)
├── master.go                     [修改]
```

### Phase 6: Overview + 收尾

```
atlhyper_master_v2/
├── service/
│   ├── factory.go                [修改] 最终形态
│   └── query/
│       ├── overview.go           [修改] 新增 OverviewQueryService, receiver 变更
│       ├── overview_test.go      [修改] Overview 测试 mock + 构造更新
│       ├── impl_test.go          [修改] 删除旧构造测试
│       └── impl.go              [修改] 删除 QueryService、QueryServiceDeps、NewQueryService
├── master.go                     [修改] 最终形态 + EventTrigger 引用变更
```

## 不变的文件

| 文件 | 原因 |
|------|------|
| `service/interfaces.go` | 子接口 + 聚合接口完全不变 |
| `metrics_format.go` | 包级纯函数，不属于任何 struct |
| `metrics_format_test.go` | 不涉及 QueryService |
| `gateway/handler/**` | Handler 持有 `service.Query` 或 `service.Service`，不直接引用 query 包内部类型 |
| `gateway/routes.go` | 通过 `service.Service` 传递，与内部拆分解耦 |
| `gateway/server.go` | 同上 |

## Gateway Handler 不被联动的原因

所有 Handler 的构造函数接收的是 `service.Service`（或其子接口 `service.Query`/`service.Ops`），而非 `*query.QueryService`：

```go
// routes.go 中的调用
podH := k8sHandler.NewPodHandler(r.service)   // r.service 是 service.Service
aiopsGraphH := aiopsHandler.NewAIOpsGraphHandler(r.service)
```

`service.Service` → `Query` → 各子接口的满足关系由 `factory.go` 中的 `serviceImpl` 嵌入保证。只要 serviceImpl 嵌入的 struct 组合满足 Query 接口（编译器强制），Handler 层无需任何修改。

这正是**面向接口编程 + 依赖倒置**的设计收益。
