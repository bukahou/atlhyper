# service/query/k8s.go 测试补强设计文档

## 1. 背景与目标

### 背景

`atlhyper_master_v2/service/query/k8s.go`（363 行）是 Master V2 中最大的 K8s 资源查询文件，承担 19 个方法，按模式分为 3 类：

1. **透传方法**（4 个）：直接返回快照字段，无任何逻辑
2. **Namespace 过滤方法**（14 个）：`namespace == ""` 返回全部，否则按 namespace 过滤
3. **复杂查询方法**（1 个）：GetPods — 3 种过滤 + 分页 + Metrics 格式化

当前 **0 个测试**。19 个方法虽然模式高度重复，但作为 query 包核心查询入口，需要建立完整的回归安全网。

### 目标

为 `k8s.go` 中的 19 个方法建立最小测试安全网，覆盖正常路径和关键边界条件。

同时为 `metrics_format.go` 中的 `FormatCPU` 和 `FormatMemory` 2 个纯函数补充单元测试（GetPods 内部调用，属于同包辅助函数）。

**本轮只做测试补强，不做架构重构、不修改任何业务代码。**

### 不做什么

| 不做 | 原因 |
|------|------|
| 修改 `k8s.go` | 本轮只补测试 |
| 修改 `metrics_format.go` | 本轮只补测试 |
| 拆分 k8s.go | 属于重构，不在范围内 |
| 测试 handler / gateway | 超出范围 |
| 测试 model_v3 结构体方法 | 属于模型层 |

---

## 2. 当前职责边界

### 函数清单

**A. 透传方法（4 个）— 直接返回快照字段**

| 函数 | 行数 | 依赖 | 复杂度 |
|------|------|------|--------|
| `GetSnapshot` | 15-17 | store.GetSnapshot | 透传 |
| `GetNodes` | 58-64 | store.GetSnapshot | 低 |
| `GetNamespaces` | 167-173 | store.GetSnapshot | 低 |
| `GetPersistentVolumes` | 256-262 | store.GetSnapshot | 低 |

**B. Namespace 过滤方法（14 个）— 相同模式：无 ns 返回全部，有 ns 过滤**

| 函数 | Namespace 访问方式 |
|------|-------------------|
| `GetDeployments` | `d.GetNamespace()` |
| `GetServices` | `s.GetNamespace()` |
| `GetIngresses` | `i.GetNamespace()` |
| `GetDaemonSets` | `d.GetNamespace()` |
| `GetStatefulSets` | `s.GetNamespace()` |
| `GetConfigMaps` | `c.Namespace`（CommonMeta 字段） |
| `GetSecrets` | `s.Namespace` |
| `GetJobs` | `j.Namespace` |
| `GetCronJobs` | `c.Namespace` |
| `GetPersistentVolumeClaims` | `p.Namespace`（CommonMeta 字段） |
| `GetNetworkPolicies` | `np.Namespace`（直接字段） |
| `GetResourceQuotas` | `rq.Namespace` |
| `GetLimitRanges` | `lr.Namespace` |
| `GetServiceAccounts` | `sa.Namespace` |

> 注：部分类型通过 `GetNamespace()` 方法（有 Summary 子结构），部分直接用 `.Namespace` 字段（嵌入 CommonMeta 或直接定义）。行为一致，但测试时构造数据方式不同。

**C. 复杂查询（1 个）**

| 函数 | 行数 | 逻辑 |
|------|------|------|
| `GetPods` | 20-55 | Namespace + NodeName + Phase 三重过滤 → FormatCPU/FormatMemory 格式化 → Offset/Limit 分页 |

**D. 辅助纯函数（2 个，在 metrics_format.go 中）**

| 函数 | 行数 | 逻辑 |
|------|------|------|
| `FormatCPU` | 15-55 | 纳核(n)/毫核(m)/核 三种输入格式转换 |
| `FormatMemory` | 61-120 | Ki/Mi/Gi/纯字节 四种输入格式转换 |

### 依赖关系

```
k8s.go 依赖:
├── q.store (datahub.Store)
│   └── GetSnapshot() — 全部 19 个方法使用
├── FormatCPU()        — GetPods 内部调用
├── FormatMemory()     — GetPods 内部调用
└── model.PodQueryOpts — GetPods 参数
```

---

## 3. 本轮测试补强范围

### 测试什么

| 功能块 | 方法数 | 优先级 | 理由 |
|--------|--------|--------|------|
| FormatCPU + FormatMemory | 2 | P0 | 纯函数，输入输出明确，最易测试，且 GetPods 依赖 |
| 透传方法 | 4 | P0 | 极简，零成本覆盖 |
| GetPods | 1 | P0 | 最复杂方法，过滤+分页+格式化逻辑 |
| Namespace 过滤（代表性） | 2 | P0 | 深度测试 GetDeployments + GetConfigMaps（两种 namespace 访问方式的代表） |
| Namespace 过滤（烟雾测试） | 12 | P1 | 剩余 12 个同模式方法的最小覆盖 |

### 不测什么

| 排除项 | 原因 |
|--------|------|
| model_v3 结构体的 GetName/GetNamespace/IsHealthy | 属于模型层 |
| datahub.Store 的 GetSnapshot 实现 | 已有独立测试 |
| Handler 层的 HTTP 参数解析 | 属于 gateway 层 |

---

## 4. 功能块拆分与测试设计

### 功能块 A：FormatCPU + FormatMemory 纯函数

**测试对象**: `FormatCPU`、`FormatMemory` — 无需 mock

| 测试用例 | 函数 | 场景 |
|---------|------|------|
| `TestFormatCPU` (表驱动) | FormatCPU | 空字符串→"" / "123456789n"→毫核 / "100m"→"100m" / "1500m"→"1.50" / "2"→"2" / 非法输入原样返回 |
| `TestFormatMemory` (表驱动) | FormatMemory | 空字符串→"" / "131072Ki"→"128Mi" / "2097152Ki"→Gi / "128Mi"→"128Mi" / "2048Mi"→Gi / "1Gi"→"1Gi" / 纯字节→Mi/Ki / 非法输入原样返回 |

### 功能块 B：透传方法

**测试对象**: GetSnapshot、GetNodes、GetNamespaces、GetPersistentVolumes

Mock: 复用 `mockStoreForOverview`

| 测试用例 | 函数 | 场景 |
|---------|------|------|
| `TestGetSnapshot_Delegate` | GetSnapshot | 返回注入快照 |
| `TestGetSnapshot_NoData` | GetSnapshot | clusterID 不存在返回 nil |
| `TestGetNodes_Found` | GetNodes | 返回节点列表 |
| `TestGetNodes_NoSnapshot` | GetNodes | 无快照返回 nil |
| `TestGetNamespaces_Found` | GetNamespaces | 返回命名空间列表 |
| `TestGetNamespaces_NoSnapshot` | GetNamespaces | 无快照返回 nil |
| `TestGetPersistentVolumes_Found` | GetPersistentVolumes | 返回 PV 列表 |
| `TestGetPersistentVolumes_NoSnapshot` | GetPersistentVolumes | 无快照返回 nil |

### 功能块 C：GetPods（复杂查询）

**测试对象**: GetPods — 三重过滤 + 分页 + 格式化

Mock: 复用 `mockStoreForOverview`

| 测试用例 | 场景 |
|---------|------|
| `TestGetPods_NoFilter` | 无过滤返回全部 Pod |
| `TestGetPods_NamespaceFilter` | 按 Namespace 过滤 |
| `TestGetPods_NodeNameFilter` | 按 NodeName 过滤 |
| `TestGetPods_PhaseFilter` | 按 Phase 过滤 |
| `TestGetPods_CombinedFilter` | Namespace + Phase 组合过滤 |
| `TestGetPods_Pagination` | Offset + Limit 分页 |
| `TestGetPods_MetricsFormat` | 验证 FormatCPU/FormatMemory 在 Pod 上的应用 |
| `TestGetPods_NoSnapshot` | 无快照返回 nil |

### 功能块 D：Namespace 过滤（代表性深度测试）

**测试对象**: GetDeployments（代表 `GetNamespace()` 方法类）、GetConfigMaps（代表 `.Namespace` 字段类）

| 测试用例 | 函数 | 场景 |
|---------|------|------|
| `TestGetDeployments_All` | GetDeployments | namespace="" 返回全部 |
| `TestGetDeployments_Filtered` | GetDeployments | 按 namespace 过滤 |
| `TestGetDeployments_NoSnapshot` | GetDeployments | 无快照返回 nil |
| `TestGetConfigMaps_All` | GetConfigMaps | namespace="" 返回全部 |
| `TestGetConfigMaps_Filtered` | GetConfigMaps | 按 namespace 过滤 |
| `TestGetConfigMaps_NoSnapshot` | GetConfigMaps | 无快照返回 nil |

### 功能块 E：Namespace 过滤（烟雾测试）

**测试对象**: 剩余 12 个 namespace 过滤方法

由于模式完全一致，使用两个表驱动测试覆盖全部方法的「有数据返回」和「namespace="" 返回全部」两个基本路径：

| 测试用例 | 场景 |
|---------|------|
| `TestNamespaceFilterMethods_All` (表驱动) | 每个方法 namespace="" 返回全部数据 |
| `TestNamespaceFilterMethods_Filtered` (表驱动) | 每个方法按 namespace 过滤后只返回匹配项 |

> 注：由于 Go 不支持泛型方法引用（每个方法签名中返回类型不同），表驱动测试需要为每个方法编写一个子测试。结构为 `t.Run("GetServices", func(t *testing.T) { ... })`。

---

## 5. Mock 边界控制

```
本轮 mock 边界:

复用 mockStoreForOverview (datahub.Store)
├── GetSnapshot() → 返回注入的快照（已有）
└── 其余方法已有空实现（不需要扩展）

不 mock:
- FormatCPU / FormatMemory（直接测试纯函数）
- model_v3 数据结构（直接构造）
- PodQueryOpts（直接构造）
```

同包 `package query`，可直接构造 `&QueryService{store: mock}` 和调用 `FormatCPU/FormatMemory`。

---

## 6. 文件变更清单

```
atlhyper_master_v2/service/query/
├── k8s_test.go            [新增] k8s.go 全量测试（19 个方法）
└── metrics_format_test.go [新增] FormatCPU + FormatMemory 纯函数测试
```

**总计：2 个新增测试文件，0 个修改文件，0 个业务代码修改。**

> 不追加到 overview_test.go，因为 k8s.go 是独立的功能域（19 个方法），体量足以独立成文件。
> FormatCPU/FormatMemory 跟随标准命名约定（metrics_format.go → metrics_format_test.go）。

---

## 7. 执行顺序

### Phase 1：纯函数 + 透传方法（~16 个测试）

最简单的部分，无外部依赖（纯函数）或极简透传。

```
1. 新增 metrics_format_test.go：FormatCPU + FormatMemory 表驱动测试
2. 新增 k8s_test.go：透传方法 8 个测试
3. 验证: go test ./atlhyper_master_v2/service/query/ -run "TestFormatCPU|TestFormatMemory|TestGetSnapshot|TestGetNodes_|TestGetNamespaces_|TestGetPersistentVolumes_" -v
```

### Phase 2：GetPods 复杂查询（8 个测试）

最复杂的单方法，覆盖三重过滤 + 分页 + 格式化。

```
1. k8s_test.go 追加 GetPods 8 个测试
2. 验证: go test ./atlhyper_master_v2/service/query/ -run TestGetPods -v
```

### Phase 3：Namespace 过滤方法（~20 个测试）

14 个同模式方法，代表性深度测试 + 烟雾测试。

```
1. k8s_test.go 追加代表性测试（GetDeployments + GetConfigMaps 各 3 个）
2. k8s_test.go 追加烟雾测试（剩余 12 个方法的表驱动子测试）
3. 验证: go test ./atlhyper_master_v2/service/query/ -run "TestGetDeployments_|TestGetConfigMaps_|TestNamespaceFilter" -v
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
| 1 | ~44 个测试全部 PASS | `go test ./atlhyper_master_v2/service/query/ -v` |
| 2 | 未修改任何业务代码 | `git diff --name-only -- service/query/ \| grep -v _test.go` = 0 |
| 3 | 19 个 k8s.go 方法全覆盖 | 检查覆盖矩阵 |
| 4 | 2 个纯函数全覆盖 | FormatCPU + FormatMemory 关键输入格式全覆盖 |

### 覆盖矩阵

| 方法 | 正常路径 | 边界条件 | Phase |
|------|---------|---------|-------|
| `FormatCPU` | ✓ (n/m/核) | ✓ (空/非法) | 1 |
| `FormatMemory` | ✓ (Ki/Mi/Gi/字节) | ✓ (空/非法) | 1 |
| `GetSnapshot` | ✓ (delegate) | ✓ (no data) | 1 |
| `GetNodes` | ✓ (found) | ✓ (no snapshot) | 1 |
| `GetNamespaces` | ✓ (found) | ✓ (no snapshot) | 1 |
| `GetPersistentVolumes` | ✓ (found) | ✓ (no snapshot) | 1 |
| `GetPods` | ✓ (no filter/each filter/combined) | ✓ (pagination/format/no snapshot) | 2 |
| `GetDeployments` | ✓ (all/filtered) | ✓ (no snapshot) | 3 |
| `GetConfigMaps` | ✓ (all/filtered) | ✓ (no snapshot) | 3 |
| `GetServices` | ✓ (all/filtered) | — | 3 |
| `GetIngresses` | ✓ (all/filtered) | — | 3 |
| `GetDaemonSets` | ✓ (all/filtered) | — | 3 |
| `GetStatefulSets` | ✓ (all/filtered) | — | 3 |
| `GetSecrets` | ✓ (all/filtered) | — | 3 |
| `GetJobs` | ✓ (all/filtered) | — | 3 |
| `GetCronJobs` | ✓ (all/filtered) | — | 3 |
| `GetPersistentVolumeClaims` | ✓ (all/filtered) | — | 3 |
| `GetNetworkPolicies` | ✓ (all/filtered) | — | 3 |
| `GetResourceQuotas` | ✓ (all/filtered) | — | 3 |
| `GetLimitRanges` | ✓ (all/filtered) | — | 3 |
| `GetServiceAccounts` | ✓ (all/filtered) | — | 3 |
