# Master V2 架构边界修复设计文档

## 背景与目标

### 背景

基于 CLAUDE.md 规范符合度审计（2026-03），atlhyper_master_v2 存在 3 类架构边界违规：

| 编号 | 违规 | 风险 | 影响文件 |
|------|------|------|----------|
| V-02 | Gateway 直接持有 `mq.Producer`，调用 `WaitCommandResult()` | P0 | 3 个 Handler |
| V-01 | Gateway 直接持有 `database.SLORepository` | P0 | 1 个 Handler |
| V-03 | Service 层使用 Setter 注入隐藏依赖 | P1 | 2 个文件 |

### 目标

1. Gateway 层只依赖 `service.Query` / `service.Ops` 接口，移除对 MQ 和 Database 的直接依赖
2. Service 层所有依赖通过构造函数注入，消除 Setter 注入
3. 保持所有现有功能不变，零行为变更

### 非目标

- 不拆分超大文件（另案处理）
- 不重构 QueryService 的 God Object 问题
- 不增加新功能

---

## 核心架构

### 当前（违规状态）

```
Gateway Handler
  ├── service.Query ──> Store / MQ / DB   ✅ 正确
  ├── service.Ops   ──> MQ               ✅ 正确
  ├── mq.Producer   ──> MQ               ❌ 跳层
  └── database.SLORepository ──> DB      ❌ 跳层
```

### 目标（合规状态）

```
Gateway Handler
  ├── service.Query ──> Store / MQ / DB   ✅
  └── service.Ops   ──> MQ / DB          ✅
```

所有 MQ 等待和 DB 访问由 Service 层包装，Gateway 只调用 Service 接口方法。

---

## 功能一：MQ 调用下沉到 Service 层（V-02 修复）

### 用户故事

Gateway 的 3 个 Handler 直接调用 `bus.WaitCommandResult()` 获取 Agent 执行结果。这违反了 Gateway 只依赖 Service 接口的规范。需要将「创建指令 + 等待结果」封装为 Service 层的同步方法。

### 当前数据流

```
Handler.PodLogs()
  1. h.svc.CreateCommand(req)     → 创建指令（通过 service.Ops）
  2. h.bus.WaitCommandResult(id)  → 等待结果（直接调用 MQ）❌
  3. 处理 result，返回 HTTP 响应
```

### 目标数据流

```
Handler.PodLogs()
  1. h.svc.ExecuteCommandSync(ctx, req, timeout) → 创建 + 等待（通过 service.Ops）
  2. 处理 result，返回 HTTP 响应
```

### 接口变更

**service/interfaces.go — Ops 接口新增方法：**

```go
// Ops 写入操作接口
type Ops interface {
    CreateCommand(req *model.CreateCommandRequest) (*model.CreateCommandResponse, error)
    // 新增：同步执行指令（创建 + 等待结果）
    ExecuteCommandSync(ctx context.Context, req *model.CreateCommandRequest, timeout time.Duration) (*command.Result, error)
    OpsAdmin
}
```

**service/operations/command.go — 新增实现：**

```go
// ExecuteCommandSync 创建指令并同步等待 Agent 执行结果
func (s *CommandService) ExecuteCommandSync(ctx context.Context, req *model.CreateCommandRequest, timeout time.Duration) (*command.Result, error) {
    resp, err := s.CreateCommand(req)
    if err != nil {
        return nil, fmt.Errorf("create command: %w", err)
    }
    result, err := s.bus.WaitCommandResult(ctx, resp.CommandID, timeout)
    if err != nil {
        return nil, fmt.Errorf("wait command %s: %w", resp.CommandID, err)
    }
    return result, nil
}
```

### 受影响的 Handler 和调用点

| Handler 文件 | 方法 | 当前代码（删除） | 替换代码 |
|-------------|------|------------------|----------|
| `gateway/handler/ops.go:142` | `PodLogs()` | `h.bus.WaitCommandResult(...)` | `h.svc.ExecuteCommandSync(...)` |
| `gateway/handler/ops.go:450` | `ConfigMapData()` | `h.bus.WaitCommandResult(...)` | `h.svc.ExecuteCommandSync(...)` |
| `gateway/handler/ops.go:507` | `SecretData()` | `h.bus.WaitCommandResult(...)` | `h.svc.ExecuteCommandSync(...)` |
| `gateway/handler/observe/observe.go:166` | `executeQuery()` | `h.bus.WaitCommandResult(...)` | `h.svc.ExecuteCommandSync(...)` |
| `gateway/handler/observe/node_metrics.go:245` | `getHistoryFromCH()` | `h.bus.WaitCommandResult(...)` | `h.svc.ExecuteCommandSync(...)` |

### 构造函数变更

| 构造函数 | 当前签名 | 新签名 |
|---------|---------|--------|
| `NewOpsHandler` | `(svc service.Ops, bus mq.Producer)` | `(svc service.Ops)` |
| `NewObserveHandler` | `(svc service.Ops, querySvc service.Query, bus mq.Producer)` | `(svc service.Ops, querySvc service.Query)` |
| `NewNodeMetricsHandler` | `(querySvc service.Query, ops service.Ops, bus mq.Producer)` | `(querySvc service.Query, ops service.Ops)` |

### 后端实现

| 文件 | 变更 |
|------|------|
| `service/interfaces.go` | Ops 接口新增 `ExecuteCommandSync` 方法签名 |
| `service/operations/command.go` | 实现 `ExecuteCommandSync` 方法 |
| `gateway/handler/ops.go` | 移除 `bus` 字段，5 处调用替换为 `h.svc.ExecuteCommandSync()` |
| `gateway/handler/observe/observe.go` | 移除 `bus` 字段，1 处调用替换 |
| `gateway/handler/observe/node_metrics.go` | 移除 `bus` 字段，1 处调用替换 |
| `gateway/routes.go` | 构造函数调用移除 `bus` 参数 |

---

## 功能二：SLO Database 访问下沉到 Service 层（V-01 修复）

### 用户故事

SLO Handler 直接持有 `database.SLORepository`，在 Handler 层执行数据库读写。需要将所有 SLO 数据库操作封装到 Service 层，Handler 只通过 `service.Query` / `service.Ops` 调用。

### 类型泄漏修复

当前 `database.SLOTarget` 和 `database.SLORouteMapping` 是数据库层类型，不应暴露到 Service 接口签名中。Service 接口使用 `model` 层的已有类型或新增 DTO：

**数据对照表：**

| 用途 | database 类型（内部） | Service 接口类型（对外） | 转换位置 |
|------|---------------------|------------------------|----------|
| SLO 目标查询返回 | `database.SLOTarget` | `model.SLOTargetResponse`（已有） | `service/query/slo.go` |
| SLO 目标写入入参 | `database.SLOTarget` | `model.UpdateSLOTargetRequest`（已有） | `service/operations/slo.go` |
| 路由映射查询返回 | `database.SLORouteMapping` | `model.SLORouteMapping`（新增） | `service/query/slo.go` |

**新增 model 类型（model/slo.go）：**

```go
// SLORouteMapping SLO 路由映射（Service 层返回类型，不暴露 database 层）
type SLORouteMapping struct {
    Domain      string `json:"domain"`
    PathPrefix  string `json:"pathPrefix"`
    IngressName string `json:"ingressName"`
    Namespace   string `json:"namespace"`
    TLS         bool   `json:"tls"`
    ServiceKey  string `json:"serviceKey"`
    ServiceName string `json:"serviceName"`
    ServicePort int    `json:"servicePort"`
}
```

**转换函数（model/convert/slo.go 或 service/query/slo.go 内部 helper）：**

```go
// toModelRouteMapping 将 database.SLORouteMapping 转换为 model.SLORouteMapping
func toModelRouteMapping(src *database.SLORouteMapping) *model.SLORouteMapping {
    if src == nil { return nil }
    return &model.SLORouteMapping{
        Domain:      src.Domain,
        PathPrefix:  src.PathPrefix,
        IngressName: src.IngressName,
        Namespace:   src.Namespace,
        TLS:         src.TLS,
        ServiceKey:  src.ServiceKey,
        ServiceName: src.ServiceName,
        ServicePort: src.ServicePort,
    }
}

// toModelRouteMappings 批量转换
func toModelRouteMappings(src []*database.SLORouteMapping) []*model.SLORouteMapping {
    if src == nil { return []*model.SLORouteMapping{} }
    result := make([]*model.SLORouteMapping, len(src))
    for i := range src { result[i] = toModelRouteMapping(src[i]) }
    return result
}

// toModelTargetResponse 将 database.SLOTarget 转换为 model.SLOTargetResponse
func toModelTargetResponse(src *database.SLOTarget) model.SLOTargetResponse {
    return model.SLOTargetResponse{
        ID:                 src.ID,
        ClusterID:          src.ClusterID,
        Host:               src.Host,
        TimeRange:          src.TimeRange,
        AvailabilityTarget: src.AvailabilityTarget,
        P95LatencyTarget:   src.P95LatencyTarget,
        CreatedAt:          src.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
        UpdatedAt:          src.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
    }
}
```

> **转换位置**：在 `service/query/slo.go` 的方法实现中完成 `database.*` → `model.*` 转换。
> Handler 只接触 `model.*` 类型，彻底消除对 `database` 包的依赖。

### 当前 SLO Handler 对 Database 的 11 处调用

| 文件 | 行号 | 调用 | 类型 |
|------|------|------|------|
| `slo_targets.go:35` | `GetTargets(ctx, clusterID)` | 读 |
| `slo_targets.go:85` | `UpsertTarget(ctx, target)` | 写 |
| `slo_domains.go:47` | `GetTargets(ctx, clusterID)` | 读 |
| `slo_domains.go:122` | `GetRouteMappingByServiceKey(ctx, clusterID, key)` | 读 |
| `slo_domains.go:217` | `GetAllDomains(ctx, clusterID)` | 读 |
| `slo_domains.go:224` | `GetTargets(ctx, clusterID)` | 读 |
| `slo_domains.go:302` | `GetRouteMappingsByDomain(ctx, clusterID, domain)` | 读 |
| `slo_domains.go:577` | `GetTargets(ctx, clusterID)` | 读 |
| `slo_domains.go:624` | `GetTargets(ctx, clusterID)` | 读 |
| `slo_domains.go:642` | `GetRouteMappingsByDomain(ctx, clusterID, host)` | 读 |
| `slo_latency.go:49` | `GetRouteMappingsByDomain(ctx, clusterID, domain)` | 读 |

### 接口变更

**service/interfaces.go — QuerySLO 扩展（全部使用 model 类型）：**

```go
// QuerySLO SLO 服务网格查询 + SLO 目标/路由映射查询
type QuerySLO interface {
    GetMeshTopology(ctx context.Context, clusterID, timeRange string) (*model.ServiceMeshTopologyResponse, error)
    GetServiceDetail(ctx context.Context, clusterID, namespace, name, timeRange string) (*model.ServiceDetailResponse, error)
    // 新增：SLO 目标查询（返回 model 类型，非 database 类型）
    GetSLOTargets(ctx context.Context, clusterID string) ([]model.SLOTargetResponse, error)
    // 新增：SLO 路由映射查询（返回 model 类型，非 database 类型）
    GetSLORouteMappingByServiceKey(ctx context.Context, clusterID, serviceKey string) (*model.SLORouteMapping, error)
    GetSLORouteMappingsByDomain(ctx context.Context, clusterID, domain string) ([]*model.SLORouteMapping, error)
    GetSLOAllDomains(ctx context.Context, clusterID string) ([]string, error)
}
```

**service/interfaces.go — Ops 新增 OpsSLO 子接口（入参使用 model 类型）：**

```go
// OpsSLO SLO 写入操作
type OpsSLO interface {
    UpsertSLOTarget(ctx context.Context, req *model.UpdateSLOTargetRequest) error
}

// Ops 写入操作接口
type Ops interface {
    CreateCommand(req *model.CreateCommandRequest) (*model.CreateCommandResponse, error)
    ExecuteCommandSync(ctx context.Context, req *model.CreateCommandRequest, timeout time.Duration) (*command.Result, error)
    OpsAdmin
    OpsSLO
}
```

### 后端实现

| 文件 | 变更 |
|------|------|
| `service/interfaces.go` | QuerySLO 新增 4 个方法（model 类型），Ops 新增 OpsSLO 子接口（model 类型） |
| `model/slo.go` | 新增 `SLORouteMapping` 结构体 |
| `service/query/slo.go` | 新增 4 个查询方法 + `toModelRouteMapping` / `toModelTargetResponse` 转换函数 |
| `service/query/impl.go` | QueryService 新增 `sloRepo database.SLORepository` 字段 |
| `service/operations/slo.go` | 新增 `UpsertSLOTarget(req *model.UpdateSLOTargetRequest)` 实现（内部构建 `database.SLOTarget`） |
| `gateway/handler/slo/slo.go` | 移除 `sloRepo` 字段，新增 `opsSvc service.Ops` 字段 |
| `gateway/handler/slo/slo_domains.go` | 10 处 `h.sloRepo.Xxx()` → `h.querySvc.GetSLOXxx()` |
| `gateway/handler/slo/slo_targets.go` | 移除 `database` import，读改为 `h.querySvc.GetSLOTargets()`（直接返回 model 类型，不再在 Handler 做转换），写改为 `h.opsSvc.UpsertSLOTarget()` |
| `gateway/handler/slo/slo_latency.go` | 1 处 `h.sloRepo.GetRouteMappingsByDomain()` → `h.querySvc.GetSLORouteMappingsByDomain()` |
| `gateway/routes.go` | SLOHandler 构造函数调用移除 `sloRepo` 参数，新增 `opsSvc` 参数 |

### SLO Handler 结构体变更

```go
// 当前
type SLOHandler struct {
    querySvc service.Query
    sloRepo  database.SLORepository  // ❌ 直接持有 DB
}

// 目标
type SLOHandler struct {
    querySvc service.Query
    opsSvc   service.Ops  // 用于 UpsertSLOTarget
}
```

### Handler 层 buildTargetMap 辅助函数变更

当前 `buildTargetMap` 接受 `[]*database.SLOTarget`，需改为接受 `[]model.SLOTargetResponse`：

```go
// 当前
func buildTargetMap(targets []*database.SLOTarget) map[string]map[string]*database.SLOTarget

// 目标
func buildTargetMap(targets []model.SLOTargetResponse) map[string]map[string]model.SLOTargetResponse
```

Handler 中使用 `t.AvailabilityTarget` / `t.P95LatencyTarget` / `t.Host` / `t.TimeRange` 的地方字段名不变（model.SLOTargetResponse 已有这些字段），改动最小。

---

## 功能三：消除 Setter 注入，改为构造函数注入（V-03 修复）

### 用户故事

`QueryService` 使用 3 个 Setter 方法延迟注入依赖，导致依赖关系不透明、允许不完整初始化。需要改为构造函数一次性注入所有依赖。

### 范围约束

**本次 `QueryServiceDeps` 的职责严格限定为：**

1. **仅替代现有 Setter 注入**：将 `SetAIOpsEngine()` / `SetAIOpsAI()` / `SetAdminRepos()` 三个 Setter 方法注入的依赖，以及 Phase 2 新增的 `sloRepo`，统一收纳到构造函数参数中
2. **不新增额外领域依赖**：`QueryServiceDeps` 中的字段必须与当前 `QueryService` struct 中已有的字段一一对应，禁止趁机新增未被使用的 Repository 或 Engine
3. **不扩大 QueryService 职责**：如果未来有新的领域查询需求（如 Deploy、GitHub），应新增独立的 Service 实现（如 `DeployQueryService`），而非继续向 `QueryServiceDeps` 塞入更多 Repository

> **检查标准**：`QueryServiceDeps` 中每个字段都必须在当前 `QueryService` 的方法中被实际使用。如果一个字段没有调用方，就不该出现在 Deps 中。

### 当前代码（service/query/impl.go）

```go
// 构造函数只注入部分依赖
q := query.NewQueryServiceWithEventRepo(store, bus, db.Event)
// Setter 延迟注入其余依赖
q.SetAIOpsEngine(aiopsEngine)     // master.go:203
q.SetAIOpsAI(aiopsEnricher)       // master.go:204
q.SetAdminRepos(db)               // master.go:205
```

### 目标代码

```go
// 所有依赖通过构造函数一次性注入
q := query.NewQueryService(query.QueryServiceDeps{
    Store:       store,
    Bus:         bus,
    EventRepo:   db.Event,
    SLORepo:     db.SLO,           // Phase 2 已新增的字段
    AIOpsEngine: aiopsEngine,      // 可选，nil = 禁用
    AIOpsAI:     aiopsEnricher,    // 可选，nil = 禁用
    AdminRepos:  query.AdminRepos{
        Audit:      db.Audit,
        Command:    db.Command,
        Notify:     db.Notify,
        Settings:   db.Settings,
        AIProvider: db.AIProvider,
        AISettings: db.AISettings,
        AIModel:    db.AIModel,
        AIBudget:   db.AIRoleBudget,
        AIReport:   db.AIReport,
    },
})
```

### 接口变更

**service/query/impl.go — 新增 Deps 结构体，重写构造函数：**

```go
// AdminRepos 管理查询所需的 Repository 集合
// 对应 QueryAdmin 接口的所有方法所需依赖
type AdminRepos struct {
    Audit      database.AuditRepository
    Command    database.CommandHistoryRepository
    Notify     database.NotifyChannelRepository
    Settings   database.SettingsRepository
    AIProvider database.AIProviderRepository
    AISettings database.AISettingsRepository
    AIModel    database.AIProviderModelRepository
    AIBudget   database.AIRoleBudgetRepository
    AIReport   database.AIReportRepository
}

// QueryServiceDeps QueryService 全部依赖
// 严格限定：每个字段对应 QueryService 已有的 struct 字段，禁止新增未使用的依赖
type QueryServiceDeps struct {
    Store       datahub.Store                    // 必需
    Bus         mq.Producer                      // 必需
    EventRepo   database.ClusterEventRepository  // 必需（Alert Trends）
    SLORepo     database.SLORepository           // 必需（Phase 2 新增）
    AIOpsEngine aiops.Engine                     // 可选，nil = AIOps 查询返回空
    AIOpsAI     *enricher.Enricher               // 可选，nil = AI 增强禁用
    AdminRepos  AdminRepos                       // 必需（管理查询）
}

// NewQueryService 创建 QueryService（全部依赖通过构造函数注入）
func NewQueryService(deps QueryServiceDeps) *QueryService {
    return &QueryService{
        store:          deps.Store,
        bus:            deps.Bus,
        eventRepo:      deps.EventRepo,
        sloRepo:        deps.SLORepo,
        aiopsEngine:    deps.AIOpsEngine,
        aiopsAI:        deps.AIOpsAI,
        auditRepo:      deps.AdminRepos.Audit,
        commandRepo:    deps.AdminRepos.Command,
        notifyRepo:     deps.AdminRepos.Notify,
        settingsRepo:   deps.AdminRepos.Settings,
        aiProviderRepo: deps.AdminRepos.AIProvider,
        aiSettingsRepo: deps.AdminRepos.AISettings,
        aiModelRepo:    deps.AdminRepos.AIModel,
        aiBudgetRepo:   deps.AdminRepos.AIBudget,
        aiReportRepo:   deps.AdminRepos.AIReport,
    }
}
```

### 后端实现

| 文件 | 变更 |
|------|------|
| `service/query/impl.go` | 新增 `QueryServiceDeps` + `AdminRepos` 结构体，重写 `NewQueryService()`，删除 `NewQueryServiceWithEventRepo()`、`SetAIOpsEngine()`、`SetAIOpsAI()`、`SetAdminRepos()` 共 4 个函数 |
| `master.go` | 用 `query.NewQueryService(query.QueryServiceDeps{...})` 替换旧的构造 + 3 行 Set* 调用 |

---

## 实施阶段

### Phase 1：MQ 调用下沉（V-02）

**目标**：Gateway Handler 不再直接持有 `mq.Producer`。

**TDD 流程**：
1. 编写 `command_test.go`（4 个测试用例）
2. 运行 `go test`，**确认全部 FAIL**（红灯确认：`ExecuteCommandSync` 方法尚不存在，编译应失败）
3. 在 `service/interfaces.go` 添加签名 + `service/operations/command.go` 实现方法
4. 运行 `go test`，**确认全部 PASS**（绿灯）
5. 替换 3 个 Handler 中的 `bus` 字段和调用
6. 编译验证 + grep 合规检查

**验收标准**：
1. `go build ./atlhyper_master_v2/...` 通过
2. `grep -r "mq.Producer" atlhyper_master_v2/gateway/` 返回 0 结果
3. 所有 Handler 构造函数中无 `bus` 参数
4. `go test ./atlhyper_master_v2/service/operations/...` 全部 PASS

**依赖**：无前置依赖

### Phase 2：SLO Database 访问下沉（V-01）

**目标**：SLO Handler 不再直接持有 `database.SLORepository`，Service 接口不暴露 database 类型。

**TDD 流程**：
1. 在 `model/slo.go` 新增 `SLORouteMapping` 结构体
2. 编写 `service/query/slo_test.go`（5 个测试）+ `service/operations/slo_test.go`（2 个测试）
3. 运行 `go test`，**确认全部 FAIL**（红灯确认：新方法尚不存在，编译应失败）
4. 在 `service/interfaces.go` 添加签名 + `service/query/slo.go` 和 `service/operations/slo.go` 实现
5. 运行 `go test`，**确认全部 PASS**（绿灯）
6. 替换 SLO Handler 中的 `sloRepo` 字段和 11 处调用
7. 编译验证 + grep 合规检查

**验收标准**：
1. `go build ./atlhyper_master_v2/...` 通过
2. `grep -r "database\." atlhyper_master_v2/gateway/handler/slo/` 返回 0 结果（import 和类型引用均消除）
3. Service 接口签名中无 `database.*` 类型引用
4. `go test ./atlhyper_master_v2/service/...` 全部 PASS

**依赖**：无前置依赖（可与 Phase 1 并行，但建议串行降低冲突）

### Phase 3：消除 Setter 注入（V-03）

**目标**：`QueryService` 所有依赖通过构造函数注入，删除所有 Set* 方法。

**TDD 流程**：
1. 编写 `service/query/impl_test.go`（2 个测试）
2. 运行 `go test`，**确认全部 FAIL**（红灯确认：`NewQueryService(deps)` 签名尚不存在）
3. 在 `service/query/impl.go` 新增 Deps 结构体、重写构造函数、删除 Set* 方法
4. 运行 `go test`，**确认全部 PASS**（绿灯）
5. 更新 `master.go` 调用点
6. 编译验证 + grep 合规检查

**验收标准**：
1. `go build ./atlhyper_master_v2/...` 通过
2. `grep -r "\.Set" atlhyper_master_v2/service/query/impl.go` 返回 0 结果
3. `grep -r "SetAIOps\|SetAdmin" atlhyper_master_v2/master.go` 返回 0 结果
4. `go test ./atlhyper_master_v2/service/query/...` 全部 PASS

**依赖**：Phase 2（Phase 2 新增 `sloRepo` 字段到 QueryService，应纳入 Deps 结构体）

---

## 文件变更清单

```
atlhyper_master_v2/
├── model/
│   └── slo.go                           [修改] 新增 SLORouteMapping 结构体
├── service/
│   ├── interfaces.go                    [修改] Ops 新增 ExecuteCommandSync + OpsSLO；QuerySLO 新增 4 方法（均使用 model 类型）
│   ├── query/
│   │   ├── impl.go                      [修改] 新增 Deps/AdminRepos 结构体，重写构造函数，删除 Set* 方法
│   │   ├── impl_test.go                 [新增] 构造函数注入测试
│   │   ├── slo.go                       [修改] 新增 4 个 SLO 查询方法 + toModel 转换函数
│   │   └── slo_test.go                  [新增] SLO 查询单元测试
│   └── operations/
│       ├── command.go                   [修改] 新增 ExecuteCommandSync 实现
│       ├── command_test.go              [新增] ExecuteCommandSync 单元测试
│       ├── slo.go                       [新增] UpsertSLOTarget 实现（model → database 转换）
│       └── slo_test.go                  [新增] SLO 写入单元测试
├── gateway/
│   ├── server.go                        [修改] Server 结构体移除 bus 字段（如 Handler 不再需要）
│   ├── routes.go                        [修改] Handler 构造函数调用移除 bus/sloRepo 参数
│   └── handler/
│       ├── ops.go                       [修改] 移除 bus 字段，替换 WaitCommandResult 调用
│       ├── observe/
│       │   ├── observe.go               [修改] 移除 bus 字段，替换 WaitCommandResult 调用
│       │   └── node_metrics.go          [修改] 移除 bus 字段，替换 WaitCommandResult 调用
│       └── slo/
│           ├── slo.go                   [修改] 移除 sloRepo，新增 opsSvc，移除 database import
│           ├── slo_domains.go           [修改] 10 处 h.sloRepo → h.querySvc，buildTargetMap 改用 model 类型
│           ├── slo_targets.go           [修改] 移除 database import，读直接用 model 返回，写改 opsSvc
│           └── slo_latency.go           [修改] 1 处 h.sloRepo → h.querySvc
└── master.go                            [修改] QueryService 构造函数改为 NewQueryService(deps)
```

---

## 测试计划

### Phase 1 测试：ExecuteCommandSync 单元测试

**文件**：`service/operations/command_test.go` [新增]

**测试对象**：`CommandService.ExecuteCommandSync(ctx, req, timeout)`

**Mock 边界**：
- `mq.Producer` 接口：mock `EnqueueCommand()` 和 `WaitCommandResult()`。`CommandService.CreateCommand()` 内部调用 `EnqueueCommand` 入队指令，`ExecuteCommandSync` 调用 `WaitCommandResult` 等待结果。测试通过控制这两个方法的返回值覆盖所有路径
- `database.CommandHistoryRepository` 接口：mock `Create()`。`CreateCommand()` 内部会持久化命令历史，mock 为直接返回 nil
- **不依赖真实 MQ / DB**：所有外部依赖均为接口 mock

**构造方式**：
```go
// 测试中直接构造 CommandService，注入 mock
svc := &CommandService{
    bus:     mockProducer,
    cmdRepo: mockCmdRepo,
}
```

**断言重点**：

| 测试函数 | 输入条件 | 断言 |
|---------|---------|------|
| `TestExecuteCommandSync_Success` | `EnqueueCommand` 返回 nil，`WaitCommandResult` 返回有效 `*command.Result` | 返回值非 nil，err 为 nil，Result 内容与 mock 一致 |
| `TestExecuteCommandSync_CreateFail` | `EnqueueCommand` 返回 error | err 非 nil，包含 `"create command"` 前缀，Result 为 nil |
| `TestExecuteCommandSync_WaitTimeout` | `WaitCommandResult` 返回 `context.DeadlineExceeded` | err 非 nil，包含 `"wait command"` 前缀 |
| `TestExecuteCommandSync_WaitError` | `WaitCommandResult` 返回其他 error | err 非 nil，包含 `"wait command"` 前缀，原始错误被包装 |

**TDD 执行顺序**：
1. **写测试**：编写 `command_test.go`，包含上述 4 个测试函数 + mock 结构体
2. **红灯确认**：运行 `go test ./atlhyper_master_v2/service/operations/...`，确认编译失败（`ExecuteCommandSync` 方法不存在）
3. **写实现**：在 `command.go` 和 `interfaces.go` 中添加方法
4. **绿灯确认**：运行 `go test`，确认 4/4 PASS

**编译验证**：
- Phase 1 实现完成后：`grep -r "mq.Producer" atlhyper_master_v2/gateway/` = 0 匹配

### Phase 2 测试：SLO Service 层单元测试

**文件**：`service/query/slo_test.go` [新增]

**测试对象**：`QueryService` 的 4 个新增 SLO 查询方法

**Mock 边界**：
- `database.SLORepository` 接口：mock `GetTargets()` / `GetRouteMappingByServiceKey()` / `GetRouteMappingsByDomain()` / `GetAllDomains()`
- **不依赖真实 DB**：SLO Repository 为接口 mock
- 测试同时验证 `database.*` → `model.*` 类型转换的正确性

**构造方式**：
```go
// 测试中构造 QueryService，只注入 sloRepo（其他字段可为 nil）
svc := &QueryService{
    sloRepo: mockSLORepo,
}
```

**断言重点**：

| 测试函数 | 输入条件 | 断言 |
|---------|---------|------|
| `TestGetSLOTargets_Success` | mock 返回 2 个 `database.SLOTarget` | 返回 2 个 `model.SLOTargetResponse`，字段映射正确（含时间格式化） |
| `TestGetSLOTargets_Empty` | mock 返回空切片 | 返回空切片（非 nil），err 为 nil |
| `TestGetSLORouteMappingsByDomain_Success` | mock 返回 3 个 `database.SLORouteMapping` | 返回 3 个 `model.SLORouteMapping`，字段逐一匹配 |
| `TestGetSLOAllDomains_Success` | mock 返回 `["a.com", "b.com"]` | 返回相同切片 |
| `TestGetSLORouteMappingByServiceKey_NotFound` | mock 返回 nil, nil | 返回 nil, nil |

**文件**：`service/operations/slo_test.go` [新增]

**测试对象**：SLO 写入 Service 的 `UpsertSLOTarget(ctx, req)`

**Mock 边界**：
- `database.SLORepository` 接口：mock `UpsertTarget()`
- 测试验证 `model.UpdateSLOTargetRequest` → `database.SLOTarget` 转换正确性

**断言重点**：

| 测试函数 | 输入条件 | 断言 |
|---------|---------|------|
| `TestUpsertSLOTarget_Success` | mock `UpsertTarget` 返回 nil | err 为 nil，mock 收到的 `database.SLOTarget` 字段与 req 一致 |
| `TestUpsertSLOTarget_Error` | mock `UpsertTarget` 返回 error | err 非 nil，错误被传播 |

**TDD 执行顺序**：
1. **写 model 类型**：在 `model/slo.go` 新增 `SLORouteMapping` 结构体
2. **写测试**：编写 `slo_test.go` 和 `slo_test.go`（operations），包含 7 个测试 + mock
3. **红灯确认**：运行 `go test`，确认编译失败（`GetSLOTargets` 等方法不存在）
4. **写实现**：`interfaces.go` 签名 + `query/slo.go` 查询实现 + `operations/slo.go` 写入实现
5. **绿灯确认**：运行 `go test`，确认 7/7 PASS

**编译验证**：
- Phase 2 实现完成后：`grep -r "database\." atlhyper_master_v2/gateway/handler/slo/` = 0 匹配

### Phase 3 测试：构造函数注入

**文件**：`service/query/impl_test.go` [新增]

**测试对象**：`NewQueryService(deps QueryServiceDeps)` 构造函数

**Mock 边界**：
- 所有 Repository 接口 + `datahub.Store` + `mq.Producer` + `aiops.Engine`：使用空 mock 实现
- 测试仅验证构造函数正确将 Deps 字段映射到 QueryService 内部字段

**断言重点**：

| 测试函数 | 输入条件 | 断言 |
|---------|---------|------|
| `TestNewQueryService_AllDeps` | 全部 Deps 字段非 nil | 返回的 QueryService 内部所有字段均非 nil |
| `TestNewQueryService_OptionalNil` | `AIOpsEngine` 和 `AIOpsAI` 为 nil，其余非 nil | 构造不 panic，可选字段为 nil，必需字段非 nil |

**TDD 执行顺序**：
1. **写测试**：编写 `impl_test.go`，使用新的 `NewQueryService(deps)` 签名
2. **红灯确认**：运行 `go test`，确认编译失败（`QueryServiceDeps` 类型和 `NewQueryService(deps)` 不存在）
3. **写实现**：新增 Deps 结构体，重写构造函数，删除 Set* 方法和旧构造函数
4. **绿灯确认**：运行 `go test`，确认 2/2 PASS

**编译验证**：
- `grep -r "\.Set" atlhyper_master_v2/service/query/impl.go` = 0 匹配
- `grep -r "SetAIOps\|SetAdmin" atlhyper_master_v2/master.go` = 0 匹配

### 回归测试

每个 Phase 完成后：
1. `go build ./atlhyper_master_v2/...` 编译通过
2. `go vet ./atlhyper_master_v2/...` 无警告
3. 现有测试（如有）全部通过
