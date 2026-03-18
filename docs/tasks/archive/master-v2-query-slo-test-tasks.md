# service/query/slo.go 测试补强 — 完成记录

> 设计文档: [master-v2-query-slo-test-design.md](../../design/archive/master-v2-query-slo-test-design.md)
> 完成日期: 2026-03-15

## 背景

`service/query/slo.go`（305 行）包含 15 个函数，此前仅有 5 个测试覆盖 SLO DB 查询方法。OTel 直读方法（GetMeshTopology、GetServiceDetail）和 6 个辅助函数完全没有测试。

## Phase 1: 纯辅助函数测试 ✅

**新增测试数**: 9

| 测试 | 覆盖函数 |
|------|---------|
| `TestDetermineMeshStatus_Healthy/Warning/Critical` (3) | `determineMeshStatus` — 三级状态判定 |
| `TestServiceToNode` | `serviceToNode` — 字段映射、ErrorRate=100-SuccessRate |
| `TestConvertEdge` | `convertEdge` — Source/Target 拼接 |
| `TestConvertEdges_Empty` | `convertEdges` — nil 输入返回空切片 |
| `TestTotalFromStatusCodes` / `_Empty` (2) | `totalFromStatusCodes` — 累加 + nil 边界 |
| `TestGetTimeStart` (8 子用例) | `getTimeStart` — 1h/6h/24h/1d/7d/30d/unknown/empty |

结果: 9/9 PASS

## Phase 2: GetMeshTopology + GetServiceDetail 测试 ✅

**新增测试数**: 9

**Mock 边界**: `mockStoreForSLO` 实现 `datahub.Store` 全部 9 个方法，仅 `GetSnapshot` 返回注入的快照数据，其余空实现。测试通过构造 `&QueryService{store: mock}` 直接验证 OTel 直读逻辑。

| 测试 | 覆盖行为 |
|------|---------|
| `TestGetMeshTopology_FromSLOWindows` | SLOWindows 匹配时间范围 |
| `TestGetMeshTopology_FallbackToSnapshot` | SLOWindows 无匹配，回退到 otel.SLOServices |
| `TestGetMeshTopology_NoOTel` | 快照无 OTel，返回空 response |
| `TestGetMeshTopology_NoSnapshot` | clusterID 不存在，返回空 response |
| `TestGetServiceDetail_FullData` | 全 5 段数据组装（节点+时序+状态码+延迟桶+上下游） |
| `TestGetServiceDetail_ServiceNotFound` | 服务不在 mesh 列表中，返回 nil |
| `TestGetServiceDetail_NoOTel` | 快照无 OTel，返回 nil |
| `TestGetServiceDetail_UpstreamDownstream` | 中间节点上下游边方向正确 |
| `TestGetServiceDetail_FallbackToSnapshot` | SLOWindows 无匹配，回退到快照 |

结果: 9/9 PASS

## 最终验收

| 指标 | 结果 |
|------|------|
| 新增测试数 | 18 |
| 现有测试数 | 5（保持不变） |
| 全量测试 | 23/23 PASS |
| 修改文件 | 1（`slo_test.go` 追加测试） |
| 业务代码变更 | 0 |
| 辅助函数覆盖 | 6/6 |
| OTel 直读方法覆盖 | 2/2（GetMeshTopology + GetServiceDetail） |
| 发现缺陷 | 0 |
