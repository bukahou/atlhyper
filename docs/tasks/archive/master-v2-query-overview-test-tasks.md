# service/query/overview.go 测试补强 — 完成记录

> 设计文档: [master-v2-query-overview-test-design.md](../../design/archive/master-v2-query-overview-test-design.md)
> 完成日期: 2026-03-15

## 背景

`atlhyper_master_v2/service/query/overview.go`（492 行）是 Master V2 中最大的查询文件，承担 4 类职责：

1. **集群查询**（2 个方法）：ListClusters、GetCluster
2. **概览查询**（1 个方法）：GetOverview — 最复杂的单方法（284 行）
3. **Event 查询**（2 个方法）：GetEvents（含过滤+分页）、GetEventsByResource
4. **单资源查询**（4 个方法）：GetPod、GetNode、GetDeployment、GetDeploymentByReplicaSet
5. **透传方法**（2 个方法）：GetAgentStatus、GetCommandStatus

当前 **0 个测试**。按复杂度分 3 Phase 逐步覆盖全部 11 个方法。

## Phase 1: 单资源查询 + Event 查询 ✅

**新增测试数**: 18

| 测试 | 覆盖函数 |
|------|---------|
| `TestGetPod_Found/NotFound/NoSnapshot` (3) | GetPod — 命中/未命中/无快照 |
| `TestGetNode_Found/NotFound` (2) | GetNode — 命中/未命中 |
| `TestGetDeployment_Found/NotFound` (2) | GetDeployment — 命中/未命中 |
| `TestGetDeploymentByReplicaSet_Found/NotFound/WrongNamespace` (3) | GetDeploymentByReplicaSet — 前缀匹配/未命中/namespace 不匹配 |
| `TestGetEvents_NoFilter/TypeFilter/ReasonFilter/SinceFilter/Pagination/NoSnapshot` (6) | GetEvents — 6 种过滤+分页场景 |
| `TestGetEventsByResource_Found/NotFound` (2) | GetEventsByResource — 按资源命中/未命中 |

Mock: `mockStoreForOverview`（GetSnapshot + GetEvents）

结果: 18/18 PASS

## Phase 2: 透传 + 集群查询 ✅

**新增测试数**: 6

| 测试 | 覆盖行为 |
|------|---------|
| `TestGetAgentStatus_Delegate` | 验证透传到 store，返回注入对象 |
| `TestGetCommandStatus_Delegate` | 验证透传到 bus，返回注入对象 |
| `TestListClusters_WithSnapshots` | 多 Agent + 快照丰富（NodeCount/PodCount/OTelAvailable） |
| `TestListClusters_Empty` | 无 Agent 返回空切片 |
| `TestGetCluster_Found` | 返回 ClusterDetail 含 Status + Snapshot |
| `TestGetCluster_NoSnapshot` | 快照为 nil 时仍返回 ClusterDetail |

Mock: 扩展 `mockStoreForOverview`（agentStatuses）+ `mockBusForOverview`

结果: 6/6 PASS

## Phase 3: GetOverview ✅

**新增测试数**: 7

| 测试 | 覆盖行为 |
|------|---------|
| `TestGetOverview_NoSnapshot` | clusterID 不存在返回 nil |
| `TestGetOverview_BasicCards` | 最小快照下 Cards 健康状态、NodeReady、PodReady 百分比 |
| `TestGetOverview_WorkloadStats` | Deployment/StatefulSet/DaemonSet/Job/Pod 各项统计 |
| `TestGetOverview_NodeUsageAndPeak` | 节点 Metrics → CPU/Mem 使用率、Peak 节点识别 |
| `TestGetOverview_NoMetrics` | 无 Metrics → HasData=false、Usage 为空 |
| `TestGetOverview_AlertsFromDB` | eventRepo 返回趋势+告警 → 组装验证 |
| `TestGetOverview_NilEventRepo` | eventRepo=nil 不 panic，告警字段零值 |

Mock: `mockEventRepoForOverview`（CountByHourAndKind + ListByCluster）

### 初次红灯说明

`TestGetOverview_BasicCards` 初次运行红灯：health status 断言值写为 `"Critical"`，实际 `CalculateHealthStatus` 返回 `"Unhealthy"`（三级状态为 Healthy/Degraded/Unhealthy）。查看源码确认后修正断言。**这是测试断言错误，非业务缺陷。**

结果: 7/7 PASS

## 最终验收

| 指标 | 结果 |
|------|------|
| 新增测试数 | 31 |
| 现有测试数 | 25（slo 23 + impl 2，保持不变） |
| 全量测试 | 56/56 PASS |
| 新增测试文件 | 1（`overview_test.go`） |
| 业务代码变更 | 0 |
| 11 个方法覆盖 | 全部 |
| GetOverview 7 子场景覆盖 | 全部 |
| 发现缺陷 | 0 |
