# service/query/k8s.go 测试补强 — 完成记录

> 设计文档: [master-v2-query-k8s-test-design.md](../../design/archive/master-v2-query-k8s-test-design.md)
> 完成日期: 2026-03-15

## 背景

`atlhyper_master_v2/service/query/k8s.go`（363 行）是 Master V2 中最大的查询文件，包含 19 个方法：

- **透传方法** (4): GetSnapshot, GetNodes, GetNamespaces, GetPersistentVolumes
- **复杂查询** (1): GetPods（三重过滤 + 分页 + Metrics 格式化）
- **Namespace 过滤** (14): GetDeployments, GetServices, GetIngresses, GetConfigMaps, GetSecrets, GetDaemonSets, GetStatefulSets, GetJobs, GetCronJobs, GetPersistentVolumeClaims, GetNetworkPolicies, GetResourceQuotas, GetLimitRanges, GetServiceAccounts

另有 `metrics_format.go`（120 行）包含 2 个纯函数：FormatCPU, FormatMemory。

当前 **0 个测试**。作为 query 包完整覆盖的一环，建立全方位安全网。

## Phase 1: 纯函数 + 透传方法

**目标**: 覆盖 FormatCPU/FormatMemory 纯函数和 4 个透传方法。

### metrics_format_test.go（新增）

| 测试 | 子测试数 | 覆盖行为 |
|------|---------|----------|
| `TestFormatCPU` | 11 | n→m、m→cores、plain、空值、无效输入 |
| `TestFormatMemory` | 14 | Ki→Mi→Gi、bytes→适当单位、空值、无效输入 |

### k8s_test.go（新增）

| 测试 | 覆盖行为 |
|------|----------|
| `TestGetSnapshot_Delegate` | 快照存在 → 透传原始指针 |
| `TestGetSnapshot_NoData` | clusterID 不存在 → nil |
| `TestGetNodes_Found` | 快照含 2 个 Node → 返回 2 个 |
| `TestGetNodes_NoSnapshot` | 无快照 → nil |
| `TestGetNamespaces_Found` | 快照含 3 个 Namespace → 返回 3 个 |
| `TestGetNamespaces_NoSnapshot` | 无快照 → nil |
| `TestGetPersistentVolumes_Found` | 快照含 2 个 PV → 返回 2 个 |
| `TestGetPersistentVolumes_NoSnapshot` | 无快照 → nil |

**Mock**: `mockStoreForK8s`（最小实现，只有 GetSnapshot 返回注入数据）

**验证**: 10 个测试（含 25 个子测试）全 PASS，全量 104/104 PASS

## Phase 2: GetPods 复杂查询

**目标**: 覆盖 GetPods 的三重过滤（namespace/nodeName/phase）、组合过滤、分页、Metrics 格式化。

### 测试数据

5 个 Pod，覆盖不同 namespace/node/phase/metrics 组合：

| Pod | Namespace | NodeName | Phase | CPU | Memory |
|-----|-----------|----------|-------|-----|--------|
| pod-1 | default | node-a | Running | 100000000n | 2097152Ki |
| pod-2 | default | node-b | Running | 2500m | 128Mi |
| pod-3 | kube-system | node-a | Running | 50m | 64Mi |
| pod-4 | kube-system | node-b | Pending | (空) | (空) |
| pod-5 | monitoring | node-a | Failed | 1500000000n | 2048Mi |

### 测试用例

| 测试 | 覆盖行为 |
|------|----------|
| `TestGetPods_NoFilter` | 无条件 → 返回全部 5 个 |
| `TestGetPods_NamespaceFilter` | namespace=kube-system → 2 个 |
| `TestGetPods_NodeNameFilter` | nodeName=node-a → 3 个 |
| `TestGetPods_PhaseFilter` | phase=Running → 3 个 |
| `TestGetPods_CombinedFilter` | namespace+nodeName 交集 (1个)、namespace+phase 交集 (1个) |
| `TestGetPods_Pagination` | limit=2、offset+limit、offset=4 三种分页场景 |
| `TestGetPods_MetricsFormat` | 5 个 Pod 的 CPU/Memory 格式化验证 |
| `TestGetPods_NoSnapshot` | 无快照 → nil |

**验证**: 8/8 PASS，全量 112/112 PASS

## Phase 3: Namespace 过滤方法

**目标**: 覆盖 14 个 namespace 过滤方法的三种访问模式。

### 三种 Namespace 访问模式

| 模式 | 方法 | 覆盖的资源类型 |
|------|------|---------------|
| **Pattern A**: Summary.GetNamespace() | GetDeployments, GetServices, GetIngresses, GetDaemonSets, GetStatefulSets | 有 Summary 结构体 |
| **Pattern B**: CommonMeta.Namespace | GetConfigMaps, GetSecrets, GetJobs, GetCronJobs, GetPersistentVolumeClaims | 嵌入 CommonMeta |
| **Pattern C**: 直接 .Namespace 字段 | GetNetworkPolicies, GetResourceQuotas, GetLimitRanges, GetServiceAccounts | 扁平结构体 |

### 代表性深度测试（6 个）

| 测试 | 覆盖行为 |
|------|----------|
| `TestGetDeployments_All` | namespace="" → 返回全部 3 个（Pattern A 代表） |
| `TestGetDeployments_Filtered` | namespace="default" → 返回匹配的 2 个 |
| `TestGetDeployments_NoSnapshot` | 无快照 → nil |
| `TestGetConfigMaps_All` | namespace="" → 返回全部 3 个（Pattern B 代表） |
| `TestGetConfigMaps_Filtered` | namespace="kube-system" → 返回匹配的 1 个 |
| `TestGetConfigMaps_NoSnapshot` | 无快照 → nil |

### 烟雾测试（2 个表驱动，24 个子测试）

| 测试 | 组织方式 | 覆盖 |
|------|----------|------|
| `TestNamespaceFilter_SmokeAll` | 12 个子测试 | 每个方法 namespace="" → 返回全部 3 个 |
| `TestNamespaceFilter_SmokeFiltered` | 12 个子测试 | 每个方法 namespace="default" → 返回匹配的 2 个 |

共用 `testSnapshotForSmoke()` 构造函数，每类资源 3 个（2 个 default + 1 个 kube-system）。

**验证**: 8 个测试（含 30 个子测试）全 PASS，全量 144/144 PASS

## 最终验收

| 指标 | 结果 |
|------|------|
| 新增测试数 | 26 个（含 79 个子测试） |
| 全量测试 | 144/144 PASS |
| 新增测试文件 | 1（`metrics_format_test.go`） |
| 修改测试文件 | 1（`k8s_test.go`） |
| 业务代码变更 | 0 |
| 19 个方法覆盖 | 全部 |
| 2 个纯函数覆盖 | 全部 |
| 发现缺陷 | 0 |
