# service/query/overview.go 测试补强设计文档

## 1. 背景与目标

### 背景

`atlhyper_master_v2/service/query/overview.go`（492 行）是 Master V2 中最大的查询文件，承担 4 类职责：

1. **集群查询**（2 个方法）：ListClusters、GetCluster
2. **概览查询**（1 个方法）：GetOverview — 最复杂的单方法（284 行），包含健康状态、资源使用率、工作负载统计、告警数据、Peak 统计
3. **Event 查询**（2 个方法）：GetEvents（含过滤+分页）、GetEventsByResource
4. **单资源查询**（4 个方法）：GetPod、GetNode、GetDeployment、GetDeploymentByReplicaSet
5. **透传方法**（2 个方法）：GetAgentStatus、GetCommandStatus

当前 **0 个测试**。文件职责杂且 `GetOverview` 复杂度高，需要分阶段覆盖。

### 目标

为 `overview.go` 中的 11 个方法建立最小测试安全网，按复杂度分 Phase 逐步覆盖。

**本轮只做测试补强，不做架构重构、不修改任何业务代码。**

### 不做什么

| 不做 | 原因 |
|------|------|
| 修改 `overview.go` | 本轮只补测试 |
| 拆分 GetOverview | 属于重构，不在范围内 |
| 测试 handler / gateway | 超出范围 |
| 测试其他 query 文件 | 已独立完成或待单独规划 |
| 拆分 QueryService | 不在本轮范围 |

---

## 2. 当前职责边界

### 函数清单

| 函数 | 行数 | 依赖 | 复杂度 | 已有测试 |
|------|------|------|--------|---------|
| `ListClusters` | 20-45 | store.ListAgents + store.GetSnapshot | 低 | ❌ |
| `GetCluster` | 48-61 | store.GetSnapshot + store.GetAgentStatus | 低 | ❌ |
| `GetAgentStatus` | 66-68 | store.GetAgentStatus | 透传 | ❌ |
| `GetCommandStatus` | 73-75 | bus.GetCommandStatus | 透传 | ❌ |
| `GetEvents` | 80-110 | store.GetEvents | 中（过滤+分页） | ❌ |
| `GetEventsByResource` | 113-127 | store.GetEvents | 低 | ❌ |
| `GetOverview` | 132-416 | store.GetSnapshot + eventRepo | **高**（284 行） | ❌ |
| `GetPod` | 421-434 | store.GetSnapshot | 低 | ❌ |
| `GetNode` | 437-450 | store.GetSnapshot | 低 | ❌ |
| `GetDeployment` | 453-466 | store.GetSnapshot | 低 | ❌ |
| `GetDeploymentByReplicaSet` | 470-491 | store.GetSnapshot | 中（前缀匹配） | ❌ |

### 依赖关系

```
overview.go 依赖:
├── q.store (datahub.Store)
│   ├── GetSnapshot()      — 8 个方法使用
│   ├── ListAgents()       — ListClusters
│   ├── GetAgentStatus()   — GetAgentStatus, GetCluster
│   └── GetEvents()        — GetEvents, GetEventsByResource
├── q.bus (mq.Producer)
│   └── GetCommandStatus() — GetCommandStatus
└── q.eventRepo (database.ClusterEventRepository)  [可选, nil-safe]
    ├── CountByHourAndKind() — GetOverview 告警趋势
    └── ListByCluster()      — GetOverview 最近告警
```

---

## 3. 本轮测试补强范围

### 测试什么

| 功能块 | 优先级 | 理由 |
|--------|--------|------|
| 单资源查询（4 个） | P0 | 模式统一、无外部依赖、最易测试 |
| Event 查询（2 个） | P0 | 含过滤和分页逻辑，需验证边界 |
| 透传方法（2 个） | P1 | 简单委托，但覆盖零成本 |
| 集群查询（2 个） | P1 | ListClusters 含快照丰富逻辑 |
| GetOverview | P1 | 最复杂方法，拆为子场景逐步验证 |

### 不测什么

| 排除项 | 原因 |
|--------|------|
| `CalculateHealthStatus/Reason` | 属于 `model_v3/cluster/` 的纯函数，不在 query 层 |
| `ParseCPU/ParseMemory` | 属于 `model_v3/` 工具函数 |
| `model_v3` 结构体的 `IsHealthy/GetName` 方法 | 属于模型层 |

---

## 4. 功能块拆分与测试设计

### 功能块 A：单资源查询

**测试对象**: GetPod、GetNode、GetDeployment、GetDeploymentByReplicaSet

Mock: `mockStoreForOverview`（同 SLO 测试的 mock 模式，只实现 `GetSnapshot`）

| 测试用例 | 函数 | 场景 |
|---------|------|------|
| `TestGetPod_Found` | GetPod | 命中 namespace+name |
| `TestGetPod_NotFound` | GetPod | 无匹配返回 nil |
| `TestGetPod_NoSnapshot` | GetPod | clusterID 不存在 |
| `TestGetNode_Found` | GetNode | 命中 name |
| `TestGetNode_NotFound` | GetNode | 无匹配返回 nil |
| `TestGetDeployment_Found` | GetDeployment | 命中 namespace+name |
| `TestGetDeployment_NotFound` | GetDeployment | 无匹配返回 nil |
| `TestGetDeploymentByReplicaSet_Found` | GetDeploymentByReplicaSet | rs 名 = dep名-hash 前缀匹配 |
| `TestGetDeploymentByReplicaSet_NotFound` | GetDeploymentByReplicaSet | 无匹配返回 nil |
| `TestGetDeploymentByReplicaSet_WrongNamespace` | GetDeploymentByReplicaSet | namespace 不匹配返回 nil |

### 功能块 B：Event 查询

**测试对象**: GetEvents、GetEventsByResource

Mock: `mockStoreForOverview`

| 测试用例 | 函数 | 场景 |
|---------|------|------|
| `TestGetEvents_NoFilter` | GetEvents | 无过滤返回全部 |
| `TestGetEvents_TypeFilter` | GetEvents | 按 Type 过滤 |
| `TestGetEvents_ReasonFilter` | GetEvents | 按 Reason 过滤 |
| `TestGetEvents_SinceFilter` | GetEvents | 按 Since 时间过滤 |
| `TestGetEvents_Pagination` | GetEvents | Offset+Limit 分页 |
| `TestGetEvents_NoSnapshot` | GetEvents | clusterID 不存在返回 nil |
| `TestGetEventsByResource_Found` | GetEventsByResource | 按 kind/ns/name 命中 |
| `TestGetEventsByResource_NotFound` | GetEventsByResource | 无匹配返回空切片 |

### 功能块 C：透传 + 集群查询

**测试对象**: GetAgentStatus、GetCommandStatus、ListClusters、GetCluster

Mock: `mockStoreForOverview`（需扩展 ListAgents/GetAgentStatus）+ `mockBusForOverview`（GetCommandStatus）

| 测试用例 | 函数 | 场景 |
|---------|------|------|
| `TestGetAgentStatus_Delegate` | GetAgentStatus | 验证透传到 store |
| `TestGetCommandStatus_Delegate` | GetCommandStatus | 验证透传到 bus |
| `TestListClusters_WithSnapshots` | ListClusters | 多 Agent + 快照统计（NodeCount/PodCount/OTelAvailable） |
| `TestListClusters_Empty` | ListClusters | 无 Agent 返回空切片 |
| `TestGetCluster_Found` | GetCluster | 返回 ClusterDetail 含 Status + Snapshot |
| `TestGetCluster_NoSnapshot` | GetCluster | 快照为 nil 时仍返回 ClusterDetail |

### 功能块 D：GetOverview

**测试对象**: GetOverview — 最复杂方法，拆为子场景

Mock: `mockStoreForOverview` + `mockEventRepoForOverview`（CountByHourAndKind + ListByCluster）

| 测试用例 | 场景 | 验证重点 |
|---------|------|---------|
| `TestGetOverview_NoSnapshot` | clusterID 不存在 | 返回 nil |
| `TestGetOverview_BasicCards` | 最小快照（Nodes+Pods+Summary） | Cards 健康状态、NodeReady、Pod 百分比 |
| `TestGetOverview_WorkloadStats` | 含 Deployments/StatefulSets/DaemonSets/Jobs/Pods | Workloads 各项统计正确 |
| `TestGetOverview_NodeUsageAndPeak` | 节点含 Metrics 数据 | CPU/Mem 使用率、Peak 统计 |
| `TestGetOverview_NoMetrics` | 节点无 Metrics | PeakStats.HasData=false, NodeUsage 为空 |
| `TestGetOverview_AlertsFromDB` | eventRepo 返回告警数据 | AlertTrend 填充、RecentAlerts 转换 |
| `TestGetOverview_NilEventRepo` | eventRepo 为 nil | 告警部分为零值，不 panic |

---

## 5. Mock 边界控制

```
本轮 mock 边界:

mockStoreForOverview (datahub.Store)
├── GetSnapshot()   → 返回注入的快照
├── ListAgents()    → 返回注入的 Agent 列表
├── GetAgentStatus()→ 返回注入的 AgentStatus
├── GetEvents()     → 返回注入的 Event 列表
└── 其余方法空实现

mockBusForOverview (mq.Producer)
├── GetCommandStatus() → 返回注入的 command.Status
└── 其余方法空实现

mockEventRepoForOverview (database.ClusterEventRepository)
├── CountByHourAndKind() → 返回注入的 HourlyKindCount
├── ListByCluster()      → 返回注入的 ClusterEvent
└── 其余方法空实现

不 mock:
- ClusterSnapshot / Node / Pod 等数据结构（直接构造）
- model_v3 工具函数（ParseCPU/ParseMemory、CalculateHealthStatus）
```

同包 `package query`，可直接构造 `&QueryService{store: mock, bus: mock, eventRepo: mock}`。

---

## 6. 文件变更清单

```
atlhyper_master_v2/service/query/
└── overview_test.go    [新增] overview.go 全量测试
```

**总计：1 个新增测试文件，0 个修改文件，0 个业务代码修改。**

---

## 7. 执行顺序

### Phase 1：单资源查询 + Event 查询（18 个测试）

纯 store mock，无 bus/eventRepo 依赖。覆盖简单方法和过滤/分页逻辑。

```
1. 新增 overview_test.go，包含 mockStoreForOverview + 18 个测试
2. 验证: go test ./atlhyper_master_v2/service/query/ -run "TestGetPod|TestGetNode|TestGetDeployment|TestGetEvents" -v
```

### Phase 2：透传 + 集群查询（6 个测试）

需要 mockBusForOverview，覆盖委托和 ListClusters 的丰富逻辑。

```
1. overview_test.go 追加 mockBusForOverview + 6 个测试
2. 验证: go test ./atlhyper_master_v2/service/query/ -run "TestGetAgent|TestGetCommand|TestListClusters|TestGetCluster" -v
```

### Phase 3：GetOverview（7 个测试）

需要 mockEventRepoForOverview，覆盖最复杂方法的各子场景。

```
1. overview_test.go 追加 mockEventRepoForOverview + 7 个测试
2. 验证: go test ./atlhyper_master_v2/service/query/ -run TestGetOverview -v
```

### 最终验证

```bash
go test ./atlhyper_master_v2/service/query/ -v
# 确认无业务代码变更
git diff --name-only -- atlhyper_master_v2/service/query/ | grep -v "_test.go"
```

---

## 8. 验收标准

| # | 标准 | 验证方式 |
|---|------|---------|
| 1 | ~31 个测试全部 PASS | `go test ./atlhyper_master_v2/service/query/ -v` |
| 2 | 未修改任何业务代码 | `git diff --name-only -- service/query/ \| grep -v _test.go` = 0 |
| 3 | 11 个方法全覆盖 | 检查覆盖矩阵 |
| 4 | GetOverview 7 个子场景覆盖 | 包含无快照/Cards/工作负载/Metrics/无Metrics/告警/nil eventRepo |

### 覆盖矩阵

| 方法 | 正常路径 | 边界条件 | Phase |
|------|---------|---------|-------|
| `GetPod` | ✓ (found) | ✓ (not found, no snapshot) | 1 |
| `GetNode` | ✓ (found) | ✓ (not found) | 1 |
| `GetDeployment` | ✓ (found) | ✓ (not found) | 1 |
| `GetDeploymentByReplicaSet` | ✓ (found) | ✓ (not found, wrong ns) | 1 |
| `GetEvents` | ✓ (no filter) | ✓ (type/reason/since/pagination/no snapshot) | 1 |
| `GetEventsByResource` | ✓ (found) | ✓ (not found) | 1 |
| `GetAgentStatus` | ✓ (delegate) | — | 2 |
| `GetCommandStatus` | ✓ (delegate) | — | 2 |
| `ListClusters` | ✓ (with snapshots) | ✓ (empty) | 2 |
| `GetCluster` | ✓ (found) | ✓ (no snapshot) | 2 |
| `GetOverview` | ✓ (cards/workloads/metrics/alerts) | ✓ (no snapshot/no metrics/nil eventRepo) | 3 |
