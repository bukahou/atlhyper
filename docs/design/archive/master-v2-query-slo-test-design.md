# service/query/slo.go 测试补强设计文档

## 1. 背景与目标

### 背景

`atlhyper_master_v2/service/query/slo.go` 是 SLO 查询的核心实现（305 行），包含两类职责：

1. **OTel 直读方法**（2 个）：从内存 OTelSnapshot 读取服务网格拓扑和服务详情
2. **SLO DB 查询方法**（4 个）：从 database.SLORepository 查询 Target 和 RouteMapping

当前 `slo_test.go` 已有 **5 个测试**，覆盖了 SLO DB 查询方法（第 2 类）。但 OTel 直读方法（第 1 类）和 6 个辅助函数 **完全没有测试**。

### 目标

为 `slo.go` 中未覆盖的 OTel 直读方法和辅助函数建立测试安全网。

**本轮只做测试补强，不做架构重构、不修改任何业务代码。**

### 不做什么

| 不做 | 原因 |
|------|------|
| 修改 `slo.go` 实现 | 本轮只补测试 |
| 测试 `otel.go`（GetOTelSnapshot/GetOTelTimeline） | 超出范围 |
| 测试 Gateway handler | 属于 handler 层 |
| 测试 `operations/slo.go` | 已有测试，属于写入侧 |
| 重构已有 `slo_test.go` | 已有测试保持不动 |

---

## 2. 当前职责边界

### 函数清单与测试覆盖现状

| 函数 | 类型 | 行数 | 已有测试 |
|------|------|------|---------|
| `GetMeshTopology` | OTel 直读 | 18-50 | ❌ |
| `GetServiceDetail` | OTel 直读 | 53-140 | ❌ |
| `serviceToNode` | 辅助（转换） | 146-163 | ❌ |
| `determineMeshStatus` | 辅助（状态判定） | 166-174 | ❌ |
| `convertEdge` | 辅助（转换） | 177-185 | ❌ |
| `convertEdges` | 辅助（批量转换） | 188-194 | ❌ |
| `totalFromStatusCodes` | 辅助（聚合） | 197-203 | ❌ |
| `getTimeStart` | 辅助（时间计算） | 289-304 | ❌ |
| `GetSLOTargets` | DB 查询 | 208-218 | ✅ (2 tests) |
| `GetSLORouteMappingByServiceKey` | DB 查询 | 221-227 | ✅ (1 test) |
| `GetSLORouteMappingsByDomain` | DB 查询 | 230-236 | ✅ (1 test) |
| `GetSLOAllDomains` | DB 查询 | 239-241 | ✅ (1 test) |
| `toModelTargetResponse` | DB 转换 | 246-257 | ✅ (间接) |
| `toModelRouteMapping` | DB 转换 | 260-274 | ✅ (间接) |
| `toModelRouteMappings` | DB 转换 | 277-286 | ✅ (间接) |

**本轮补强目标：上表中 ❌ 的 8 个函数。**

### 依赖关系

```
GetMeshTopology / GetServiceDetail
    └── q.GetOTelSnapshot(ctx, clusterID)  ← 依赖 q.store (datahub.Store)
        └── q.store.GetSnapshot(clusterID) → snapshot.OTel

测试策略: mock q.store，注入含 OTel 的快照，直接测试 SLO 方法的转换逻辑
```

---

## 3. 本轮测试补强范围

### 测试什么

| 功能块 | 优先级 | 理由 |
|--------|--------|------|
| 辅助函数（6 个） | P0 | 纯函数，无依赖，测试成本最低 |
| GetMeshTopology | P0 | 核心路径，含 SLOWindows 回退逻辑 |
| GetServiceDetail | P0 | 最复杂方法（87 行），含 5 个数据组装步骤 |

### 不测什么

| 排除项 | 原因 |
|--------|------|
| SLO DB 查询方法（4 个） | 已有 5 个测试覆盖 |
| `toModel*` 转换函数（3 个） | 已被现有测试间接覆盖 |
| `GetOTelSnapshot` / `GetOTelTimeline` | 属于 `otel.go`，不在本轮范围 |

---

## 4. 功能块拆分与测试设计

### 功能块 A：纯辅助函数（无依赖）

**测试文件**: `service/query/slo_test.go`（在现有文件追加）

| 测试用例 | 函数 | 场景 |
|---------|------|------|
| `TestDetermineMeshStatus_Healthy` | `determineMeshStatus` | errRate ≤ 1, p99 ≤ 500 → "healthy" |
| `TestDetermineMeshStatus_Warning` | `determineMeshStatus` | errRate > 1 或 p99 > 500 → "warning" |
| `TestDetermineMeshStatus_Critical` | `determineMeshStatus` | errRate > 5 → "critical" |
| `TestServiceToNode` | `serviceToNode` | 验证字段映射、ErrorRate=100-SuccessRate、ID=ns/name |
| `TestConvertEdge` | `convertEdge` | 验证 Source/Target 拼接、ErrorRate=100-SuccessRate |
| `TestConvertEdges_Empty` | `convertEdges` | 空切片输入返回空切片 |
| `TestTotalFromStatusCodes` | `totalFromStatusCodes` | 多个状态码计数累加 |
| `TestTotalFromStatusCodes_Empty` | `totalFromStatusCodes` | nil 输入返回 0 |
| `TestGetTimeStart` | `getTimeStart` | 表驱动：1h/6h/24h/1d/7d/30d/unknown 各返回正确时间差 |

### 功能块 B：GetMeshTopology（需 mock store）

**测试文件**: `service/query/slo_test.go`（追加）

**mock 策略**: 复用现有 `impl_test.go` 的 `mockStore` 模式，或在 `slo_test.go` 内新增一个最小 `mockStoreForSLO`，只实现 `GetSnapshot`（其余方法空实现）。因为 `GetMeshTopology` 只调用 `q.GetOTelSnapshot` → `q.store.GetSnapshot`。

| 测试用例 | 场景 |
|---------|------|
| `TestGetMeshTopology_FromSLOWindows` | OTel 含 SLOWindows["1d"]，返回对应 MeshServices/MeshEdges |
| `TestGetMeshTopology_FallbackToSnapshot` | SLOWindows 无匹配时间范围，回退到 otel.SLOServices/SLOEdges |
| `TestGetMeshTopology_NoOTel` | 快照无 OTel，返回空 response |
| `TestGetMeshTopology_NoSnapshot` | clusterID 不存在，返回空 response |

### 功能块 C：GetServiceDetail（需 mock store）

**测试文件**: `service/query/slo_test.go`（追加）

| 测试用例 | 场景 |
|---------|------|
| `TestGetServiceDetail_FullData` | 含 SLOWindows + TimeSeries + StatusCodes + LatencyBuckets + Edges，验证所有 5 个数据段组装 |
| `TestGetServiceDetail_ServiceNotFound` | 服务不在 mesh 列表中，返回 nil |
| `TestGetServiceDetail_NoOTel` | 快照无 OTel，返回 nil |
| `TestGetServiceDetail_UpstreamDownstream` | 验证 edge 的 Source/Target 方向正确分到 Upstreams/Downstreams |
| `TestGetServiceDetail_FallbackToSnapshot` | SLOWindows 无匹配，回退到 otel.SLOServices |

---

## 5. Mock 边界控制

```
本轮 mock 边界:
┌─────────────────────────────────────┐
│  slo_test.go                        │
│                                     │
│  mockStoreForSLO ──────────────┐    │
│    implements datahub.Store     │    │
│    只需实现 GetSnapshot()       │    │
│    其余方法空实现               │    │
│                                │    │
│  QueryService{store: mock}     │    │
│    ↓                           │    │
│  GetMeshTopology / Detail      │    │
│    └── GetOTelSnapshot()       │    │
│        └── store.GetSnapshot() │    │
└────────────────────────────────┘    │
                                      │
不 mock:                              │
  - OTelSnapshot 数据结构（直接构造）  │
  - 辅助函数（纯函数直接调用）         │
  - database.SLORepository（已有测试） │
└─────────────────────────────────────┘
```

测试同包（`package query`），可直接构造 `&QueryService{store: mock}` 和调用未导出函数。

---

## 6. 文件变更清单

```
atlhyper_master_v2/service/query/
└── slo_test.go    [修改] 追加 ~18 个测试（现有 5 个保持不变）
```

**总计：1 个修改文件（仅追加测试），0 个新增文件，0 个业务代码修改。**

---

## 7. 执行顺序

### Phase 1：纯辅助函数测试（9 个）

无外部依赖，直接调用函数验证返回值。

```
1. slo_test.go 追加 9 个辅助函数测试
2. 验证: go test ./atlhyper_master_v2/service/query/ -run "TestDetermine|TestServiceToNode|TestConvert|TestTotal|TestGetTimeStart" -v
```

### Phase 2：GetMeshTopology + GetServiceDetail 测试（9 个）

需要 mock store，验证 OTel 直读路径。

```
1. slo_test.go 追加 mockStoreForSLO + 9 个测试
2. 验证: go test ./atlhyper_master_v2/service/query/ -run "TestGetMesh|TestGetServiceDetail" -v
```

### 最终验证

```bash
# 全量测试（含现有 5 个 + 新增 ~18 个）
go test ./atlhyper_master_v2/service/query/ -v

# 确认无业务代码变更
git diff --name-only -- atlhyper_master_v2/service/query/ | grep -v "_test.go"
# 预期: 0 行
```

---

## 8. 验收标准

| # | 标准 | 验证方式 |
|---|------|---------|
| 1 | 新增 ~18 个测试全部 PASS | `go test ./atlhyper_master_v2/service/query/ -v` |
| 2 | 现有 5 个测试仍然 PASS | 同上 |
| 3 | 未修改任何业务代码 | `git diff --name-only -- service/query/ \| grep -v _test.go` = 0 |
| 4 | 辅助函数 6 个全覆盖 | 检查覆盖矩阵 |
| 5 | GetMeshTopology 4 个场景覆盖 | 正常/回退/无 OTel/无快照 |
| 6 | GetServiceDetail 5 个场景覆盖 | 全数据/未找到/无 OTel/上下游/回退 |

### 覆盖矩阵

| 函数 | 正常路径 | 边界/回退 | 测试位置 |
|------|---------|----------|---------|
| `determineMeshStatus` | ✓ (healthy) | ✓ (warning, critical) | Phase 1 |
| `serviceToNode` | ✓ | — | Phase 1 |
| `convertEdge` | ✓ | — | Phase 1 |
| `convertEdges` | — | ✓ (empty) | Phase 1 |
| `totalFromStatusCodes` | ✓ | ✓ (nil/empty) | Phase 1 |
| `getTimeStart` | ✓ (6 值) | ✓ (unknown 回退) | Phase 1 |
| `GetMeshTopology` | ✓ (SLOWindows) | ✓ (回退/无 OTel/无快照) | Phase 2 |
| `GetServiceDetail` | ✓ (全数据) | ✓ (未找到/无 OTel/回退/上下游) | Phase 2 |
