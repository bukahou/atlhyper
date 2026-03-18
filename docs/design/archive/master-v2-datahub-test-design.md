# DataHub 测试补强设计文档

## 1. 背景与目标

### 背景

`atlhyper_master_v2/datahub/` 是 Master V2 的核心数据存储层，承担快照存取、Agent 状态管理、OTel 时间线等职责。当前模块共 861 行代码、5 个文件、**0 个测试文件**。

作为 Processor（写入）和 Service/Query（读取）之间的关键中间层，datahub 的正确性直接影响上下游全链路。缺乏测试意味着任何重构或 bug 修复都没有安全网。

### 目标

为 datahub 模块建立**最小测试骨架**，覆盖核心行为的正常路径和关键边界条件。

**本轮只做测试补强，不做架构重构、不修改任何业务代码。**

### 不做什么

| 不做 | 原因 |
|------|------|
| 重构 datahub 接口或实现 | 本轮目标是补测试安全网，为未来重构铺路 |
| 测试 RedisStore | 需要 Redis 实例，属于集成测试范畴 |
| 测试 `service/query/`、`gateway/` | 超出范围 |
| 修改任何 `*.go`（非 `*_test.go`） | 严格只新增测试文件 |
| 实现并发压力测试 | 只规划并发验证点，不在本轮实现 |

---

## 2. DataHub 当前职责边界

### 模块结构

```
datahub/
├── interfaces.go         # Store 接口（9 方法）+ Config
├── factory.go            # NewStore 工厂函数
└── memory/
    ├── store.go          # MemoryStore 实现（332 行）
    └── otel_ring.go      # OTelRing 环形缓冲区（97 行）
```

> `redis/store.go`（343 行）不在本轮测试范围。

### Store 接口（9 个方法）

| 分类 | 方法 | 说明 |
|------|------|------|
| **快照** | `SetSnapshot(clusterID, snapshot)` | 存储集群快照 + 追加 OTel 到 Ring |
| | `GetSnapshot(clusterID)` | 读取集群快照 |
| **Agent 状态** | `UpdateHeartbeat(clusterID)` | 更新心跳（标记在线） |
| | `GetAgentStatus(clusterID)` | 读取 Agent 状态 |
| | `ListAgents()` | 列出所有 Agent |
| **事件** | `GetEvents(clusterID)` | 读取集群事件 |
| **OTel 时间线** | `GetOTelTimeline(clusterID, since)` | 读取 OTel 历史快照 |
| **生命周期** | `Start()` / `Stop()` | 启停清理协程 |

### OTelRing（4 个方法）

| 方法 | 说明 |
|------|------|
| `Add(snapshot, ts)` | O(1) 写入，循环覆盖 |
| `Latest()` | 读取最新条目 |
| `Since(since)` | 过滤时间范围内的条目 |
| `Count()` | 当前条目数 |

### 关键内部行为

| 行为 | 位置 | 说明 |
|------|------|------|
| `lightweightOTelCopy()` | `store.go` | 轻量复制 OTel 防 OOM（23 字段，排除时序数据） |
| `cleanupLoop()` | `store.go` | 后台协程：标记离线 + 清理数据 |
| `updateAgentStatus()` | `store.go` | 心跳超时 → 标记离线 |
| `cleanupOfflineClusterData()` | `store.go` | 离线超 2×retention → 删除全部数据 |

---

## 3. 本轮测试补强范围

### 测试范围（测什么）

| 功能块 | 优先级 | 理由 |
|--------|--------|------|
| OTelRing 基本行为 | P0 | 独立数据结构，无外部依赖，最易测试 |
| MemoryStore 快照存取 | P0 | 核心读写路径，上下游全依赖 |
| MemoryStore Agent 状态 | P0 | 心跳/离线检测的正确性关键 |
| MemoryStore OTel 时间线 | P1 | SetSnapshot → Ring → GetOTelTimeline 端到端 |
| MemoryStore 事件查询 | P1 | 简单委托，但需验证快照不存在时的边界 |
| MemoryStore 生命周期 | P1 | Start/Stop 重复调用的安全性 |
| Factory 路由 | P2 | 简单分支，但覆盖零成本 |
| 并发安全验证点 | P2 | 本轮只规划不实现 |

### 不测试（明确排除）

| 排除项 | 原因 |
|--------|------|
| `redis/store.go` | 需要 Redis 实例，属于集成测试 |
| `lightweightOTelCopy` 字段完整性 | 内部函数，通过 OTel 时间线端到端测试间接覆盖 |
| `cleanupLoop` 定时触发 | 涉及 time.Sleep 和后台协程，测试成本高收益低 |
| 并发压力测试实现 | 只规划验证点，留给后续迭代 |

---

## 4. 功能块拆分与测试设计

### 功能块 A：OTelRing 环形缓冲区

**测试文件**: `datahub/memory/otel_ring_test.go`

**测试对象**: `OTelRing` 的 `Add`、`Latest`、`Since`、`Count`

| 测试用例 | 类型 | 场景描述 |
|---------|------|----------|
| `TestOTelRing_AddAndCount` | 正常 | 添加 N 个条目，Count 正确递增 |
| `TestOTelRing_Latest_Empty` | 边界 | 空 Ring 调用 Latest 返回 nil |
| `TestOTelRing_Latest_NonEmpty` | 正常 | 添加多个条目，Latest 返回最后一个 |
| `TestOTelRing_CircularOverwrite` | 边界 | 添加超过 capacity 的条目，验证旧数据被覆盖、Count 不超 capacity |
| `TestOTelRing_Since_FilterByTime` | 正常 | 添加不同时间戳的条目，Since 只返回指定时间之后的 |
| `TestOTelRing_Since_Empty` | 边界 | 空 Ring 调用 Since 返回空切片 |
| `TestOTelRing_Since_AllFiltered` | 边界 | since 时间晚于所有条目，返回空 |
| `TestOTelRing_DefaultCapacity` | 边界 | capacity ≤ 0 时使用默认值 90 |

### 功能块 B：MemoryStore 快照存取

**测试文件**: `datahub/memory/store_test.go`

**测试对象**: `SetSnapshot`、`GetSnapshot`

| 测试用例 | 类型 | 场景描述 |
|---------|------|----------|
| `TestSetGetSnapshot_Basic` | 正常 | Set 后 Get 返回相同快照 |
| `TestGetSnapshot_NotFound` | 边界 | 未 Set 的 clusterID，Get 返回 nil |
| `TestSetSnapshot_Overwrite` | 正常 | 同一 clusterID 连续 Set 两次，Get 返回最新快照 |
| `TestSetSnapshot_MultiCluster` | 正常 | 不同 clusterID 的快照互不干扰 |
| `TestSetSnapshot_WithOTel` | 正常 | 含 OTel 的快照，验证 Ring 被追加（通过 GetOTelTimeline 间接验证） |
| `TestSetSnapshot_NilOTel` | 边界 | OTel 为 nil 的快照，Ring 不应被追加 |
| `TestSetSnapshot_UpdatesAgentInfo` | 正常 | SetSnapshot 后 Agent 状态被自动更新（LastSnapshot 字段） |

### 功能块 C：MemoryStore Agent 状态

**测试文件**: `datahub/memory/store_test.go`（同上，按功能分 Test 函数）

**测试对象**: `UpdateHeartbeat`、`GetAgentStatus`、`ListAgents`

| 测试用例 | 类型 | 场景描述 |
|---------|------|----------|
| `TestUpdateHeartbeat_NewAgent` | 正常 | 首次心跳创建 Agent 记录，状态为 online |
| `TestUpdateHeartbeat_ExistingAgent` | 正常 | 已有 Agent 更新心跳，LastHeartbeat 刷新 |
| `TestGetAgentStatus_NotFound` | 边界 | 未注册的 clusterID，返回 nil 和 nil error |
| `TestGetAgentStatus_ReturnsCorrectFields` | 正常 | 验证返回的 AgentStatus 包含正确的 4 个字段 |
| `TestListAgents_Empty` | 边界 | 无 Agent 时返回空切片 |
| `TestListAgents_MultipleAgents` | 正常 | 多个 Agent 全部返回 |

### 功能块 D：MemoryStore OTel 时间线（端到端）

**测试文件**: `datahub/memory/store_test.go`

**测试对象**: `SetSnapshot`（含 OTel）→ `GetOTelTimeline`

| 测试用例 | 类型 | 场景描述 |
|---------|------|----------|
| `TestGetOTelTimeline_Basic` | 正常 | 多次 SetSnapshot（含 OTel），GetOTelTimeline 返回时间线 |
| `TestGetOTelTimeline_NotFound` | 边界 | 未知 clusterID 返回 nil |
| `TestGetOTelTimeline_SinceFilter` | 正常 | since 参数正确过滤旧条目 |
| `TestGetOTelTimeline_LightweightCopy` | 回归 | 验证时间线条目不含时序数据（MetricsSummary 等为 nil） |

### 功能块 E：MemoryStore 事件查询

**测试文件**: `datahub/memory/store_test.go`

**测试对象**: `GetEvents`

| 测试用例 | 类型 | 场景描述 |
|---------|------|----------|
| `TestGetEvents_Basic` | 正常 | SetSnapshot 含 Events，GetEvents 返回 |
| `TestGetEvents_NotFound` | 边界 | 未知 clusterID 返回 nil |

### 功能块 F：MemoryStore 生命周期

**测试文件**: `datahub/memory/store_test.go`

**测试对象**: `Start`、`Stop`

| 测试用例 | 类型 | 场景描述 |
|---------|------|----------|
| `TestStartStop_Basic` | 正常 | Start 后 Stop 不 panic |
| `TestStop_Idempotent` | 边界 | 连续调用两次 Stop 不 panic |

### 功能块 G：Factory 路由

**测试文件**: `datahub/factory_test.go`

**测试对象**: `NewStore`

| 测试用例 | 类型 | 场景描述 |
|---------|------|----------|
| `TestNewStore_DefaultMemory` | 正常 | Type 为空或 "memory" 返回 MemoryStore |
| `TestNewStore_UnknownType` | 边界 | 未知 Type 回退到 MemoryStore |

> 注意：不测试 `Type="redis"`，因为需要 Redis 实例。

### 功能块 H：并发安全验证点（仅规划）

以下测试点**本轮只规划，不实现**，留给后续迭代：

| 验证点 | 方法 | 场景 |
|--------|------|------|
| 并发读写快照 | `SetSnapshot` + `GetSnapshot` | 多 goroutine 同时读写不 panic、不数据竞争 |
| 并发心跳更新 | `UpdateHeartbeat` × N | 多 goroutine 并发心跳不丢失 |
| 并发 Ring 读写 | `Add` + `Since` | 写入同时查询不 panic |
| `-race` 检测 | 全部方法 | `go test -race` 无 data race 报告 |

> 实现时建议使用 `go test -race -count=1` 配合 `sync.WaitGroup` 编排并发。

---

## 5. 文件变更清单

```
atlhyper_master_v2/datahub/
├── factory_test.go          [新增] Factory 路由测试（功能块 G）
└── memory/
    ├── otel_ring_test.go    [新增] OTelRing 单元测试（功能块 A）
    └── store_test.go        [新增] MemoryStore 测试（功能块 B/C/D/E/F）
```

**总计：3 个新增测试文件，0 个修改文件。**

---

## 6. TDD 执行顺序

### Phase 1：OTelRing 单元测试

**文件**: `datahub/memory/otel_ring_test.go`
**测试数**: 8 个
**理由**: OTelRing 是独立数据结构，无任何外部依赖，最适合作为第一个 TDD 目标。

```
1. 编写 8 个测试 → 运行确认全部 PASS（OTelRing 已有实现，测试应直接绿灯）
2. 验证: go test ./atlhyper_master_v2/datahub/memory/ -run TestOTelRing -v
```

> 注意：本轮是补测试而非 TDD 新功能。OTelRing 实现已存在，测试写完应直接绿灯。
> 如果测试红灯，说明发现了 bug——记录但不在本轮修复（除非是测试本身的问题）。

### Phase 2：MemoryStore 快照 + Agent + 事件

**文件**: `datahub/memory/store_test.go`
**测试数**: 15 个（功能块 B: 7 + C: 6 + E: 2）
**理由**: 快照和 Agent 是 MemoryStore 的核心路径，事件查询是快照的简单委托。

```
1. 编写 15 个测试 → 运行确认全部 PASS
2. 验证: go test ./atlhyper_master_v2/datahub/memory/ -run "TestSet|TestGet|TestUpdate|TestList" -v
```

### Phase 3：MemoryStore OTel 时间线 + 生命周期 + Factory

**文件**: `datahub/memory/store_test.go`（追加）+ `datahub/factory_test.go`（新增）
**测试数**: 8 个（功能块 D: 4 + F: 2 + G: 2）
**理由**: OTel 时间线是 SetSnapshot → Ring → GetOTelTimeline 的端到端路径；生命周期和 Factory 是收尾。

```
1. store_test.go 追加 6 个测试，factory_test.go 新增 2 个测试
2. 验证: go test ./atlhyper_master_v2/datahub/... -v
```

### 最终验证

```bash
# 全量测试
go test ./atlhyper_master_v2/datahub/... -v

# 确认测试数量
go test ./atlhyper_master_v2/datahub/... -v 2>&1 | grep -c "^--- PASS"
# 预期: 31

# 编译检查（确保未修改业务代码）
go build ./atlhyper_master_v2/...

# 确认无业务代码变更
git diff --name-only | grep -v "_test.go"
# 预期: 0 行输出（只有 test 文件和文档）
```

---

## 7. 验收标准

| # | 标准 | 验证方式 |
|---|------|---------|
| 1 | 31 个测试全部 PASS | `go test ./atlhyper_master_v2/datahub/... -v` |
| 2 | 未修改任何非测试文件 | `git diff --name-only \| grep -v "_test.go"` 只有文档 |
| 3 | 编译通过 | `go build ./atlhyper_master_v2/...` |
| 4 | 3 个测试文件位于正确路径 | `find datahub/ -name "*_test.go"` |
| 5 | 覆盖 Store 接口全部 9 个方法 | 检查测试用例覆盖矩阵 |
| 6 | 覆盖 OTelRing 全部 4 个方法 | 检查测试用例覆盖矩阵 |

### 覆盖矩阵

| 方法 | 正常路径 | 边界条件 | 测试文件 |
|------|---------|---------|---------|
| `SetSnapshot` | ✓ (B1,B3,B4,B5,B7) | ✓ (B6) | `store_test.go` |
| `GetSnapshot` | ✓ (B1) | ✓ (B2) | `store_test.go` |
| `UpdateHeartbeat` | ✓ (C1,C2) | — | `store_test.go` |
| `GetAgentStatus` | ✓ (C4) | ✓ (C3) | `store_test.go` |
| `ListAgents` | ✓ (C6) | ✓ (C5) | `store_test.go` |
| `GetEvents` | ✓ (E1) | ✓ (E2) | `store_test.go` |
| `GetOTelTimeline` | ✓ (D1,D3) | ✓ (D2) | `store_test.go` |
| `Start` | ✓ (F1) | — | `store_test.go` |
| `Stop` | ✓ (F1) | ✓ (F2) | `store_test.go` |
| `OTelRing.Add` | ✓ (A1) | ✓ (A4) | `otel_ring_test.go` |
| `OTelRing.Latest` | ✓ (A3) | ✓ (A2) | `otel_ring_test.go` |
| `OTelRing.Since` | ✓ (A5) | ✓ (A6,A7) | `otel_ring_test.go` |
| `OTelRing.Count` | ✓ (A1) | — | `otel_ring_test.go` |
| `NewOTelRing` | — | ✓ (A8) | `otel_ring_test.go` |
| `NewStore` | ✓ (G1) | ✓ (G2) | `factory_test.go` |
