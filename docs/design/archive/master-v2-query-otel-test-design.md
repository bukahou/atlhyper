# service/query/otel.go 测试补强设计文档

## 1. 背景与目标

### 背景

`atlhyper_master_v2/service/query/otel.go`（24 行）是 Master V2 中最简单的查询文件，只有 2 个方法：

1. **GetOTelSnapshot**：从内存快照中读取 OTel 数据（`snapshot.OTel`）
2. **GetOTelTimeline**：透传到 `store.GetOTelTimeline`

当前 **0 个测试**。两个方法都是简单的读取/透传，但作为 query 包完整覆盖的一环，仍需建立最小安全网。

### 目标

为 `otel.go` 中的 2 个方法建立回归测试，覆盖正常路径和关键边界条件。

**本轮只做测试补强，不做架构重构、不修改任何业务代码。**

### 不做什么

| 不做 | 原因 |
|------|------|
| 修改 `otel.go` | 本轮只补测试 |
| 测试 OTelSnapshot 内部字段语义 | 属于 model_v3 层 |
| 测试 handler / gateway | 超出范围 |
| 测试 DataHub 的 GetOTelTimeline 实现 | 属于 datahub 层，已有独立测试 |

---

## 2. 当前职责边界

### 函数清单

| 函数 | 行数 | 依赖 | 复杂度 | 已有测试 |
|------|------|------|--------|---------|
| `GetOTelSnapshot` | 13-19 | store.GetSnapshot | 低 | ❌ |
| `GetOTelTimeline` | 22-24 | store.GetOTelTimeline | 透传 | ❌ |

### 依赖关系

```
otel.go 依赖:
├── q.store (datahub.Store)
│   ├── GetSnapshot()      — GetOTelSnapshot
│   └── GetOTelTimeline()  — GetOTelTimeline
└── 无其他依赖
```

---

## 3. 本轮测试补强范围

### 测试什么

| 功能块 | 优先级 | 理由 |
|--------|--------|------|
| GetOTelSnapshot（3 场景） | P0 | 需验证快照存在/OTel 为 nil/快照不存在 |
| GetOTelTimeline（2 场景） | P0 | 透传验证 + 无数据边界 |

### 不测什么

| 排除项 | 原因 |
|--------|------|
| OTelSnapshot 内部字段（TotalServices 等） | 属于 model_v3 层 |
| OTelEntry 时间排序逻辑 | 属于 datahub/memory 层 |
| ClickHouse 查询逻辑 | 属于 Agent 侧 |

---

## 4. 测试设计

### 功能块 A：GetOTelSnapshot

**测试对象**: GetOTelSnapshot — 从快照提取 OTel 子结构

Mock: 复用已有 `mockStoreForOverview`（或新建更轻量的 mock，视实现决定）

| 测试用例 | 场景 | 验证重点 |
|---------|------|---------|
| `TestGetOTelSnapshot_Found` | 快照存在且含 OTel | 返回非 nil 的 OTelSnapshot |
| `TestGetOTelSnapshot_NoOTel` | 快照存在但 OTel=nil | 返回 nil, nil |
| `TestGetOTelSnapshot_NoSnapshot` | clusterID 不存在 | 返回 nil, nil |

### 功能块 B：GetOTelTimeline

**测试对象**: GetOTelTimeline — 透传到 store

Mock: 需要 `mockStoreForOverview` 扩展 `GetOTelTimeline` 支持注入返回值（当前固定返回 nil）

| 测试用例 | 场景 | 验证重点 |
|---------|------|---------|
| `TestGetOTelTimeline_Delegate` | store 返回注入的时间线数据 | 透传结果一致 |
| `TestGetOTelTimeline_Empty` | store 返回空切片 | 返回空切片 |

---

## 5. Mock 边界控制

```
本轮 mock 边界:

复用 mockStoreForOverview (datahub.Store)
├── GetSnapshot()       → 返回注入的快照（已有）
├── GetOTelTimeline()   → 需扩展：返回注入的 OTelEntry 列表
└── 其余方法已有空实现

不 mock:
- OTelSnapshot / OTelEntry 等数据结构（直接构造）
```

同包 `package query`，可直接构造 `&QueryService{store: mock}`。

---

## 6. 文件变更清单

```
atlhyper_master_v2/service/query/
└── overview_test.go    [修改] 追加 5 个 otel 测试 + 扩展 mockStoreForOverview
```

> 注：由于 otel.go 方法极少（2 个，5 个测试），不单独新建 `otel_test.go`，
> 直接追加到 `overview_test.go` 中（该文件已包含 mockStoreForOverview，可复用）。

**总计：0 个新增文件，1 个修改测试文件，0 个业务代码修改。**

---

## 7. 执行顺序

### Phase 1：GetOTelSnapshot + GetOTelTimeline（5 个测试）

由于 otel.go 只有 2 个方法、24 行代码，无需分 Phase，一次性覆盖。

```
1. overview_test.go 扩展 mockStoreForOverview（otelTimeline 字段）+ 追加 5 个测试
2. 验证: go test ./atlhyper_master_v2/service/query/ -run "TestGetOTelSnapshot|TestGetOTelTimeline" -v
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
| 1 | 5 个测试全部 PASS | `go test -run "TestGetOTelSnapshot\|TestGetOTelTimeline" -v` |
| 2 | 未修改任何业务代码 | `git diff --name-only -- service/query/ \| grep -v _test.go` = 0 |
| 3 | 2 个方法全覆盖 | 检查覆盖矩阵 |

### 覆盖矩阵

| 方法 | 正常路径 | 边界条件 | Phase |
|------|---------|---------|-------|
| `GetOTelSnapshot` | ✓ (found) | ✓ (no otel, no snapshot) | 1 |
| `GetOTelTimeline` | ✓ (delegate) | ✓ (empty) | 1 |
