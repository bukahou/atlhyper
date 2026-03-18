# service/query/otel.go 测试补强 — 完成记录

> 设计文档: [master-v2-query-otel-test-design.md](../../design/archive/master-v2-query-otel-test-design.md)
> 完成日期: 2026-03-15

## 背景

`atlhyper_master_v2/service/query/otel.go`（24 行）是 Master V2 中最简单的查询文件，只有 2 个方法：

1. **GetOTelSnapshot**：从内存快照中读取 OTel 子结构（`snapshot.OTel`）
2. **GetOTelTimeline**：透传到 `store.GetOTelTimeline`

当前 **0 个测试**。作为 query 包完整覆盖的一环，建立最小安全网。

## 测试范围

由于文件极其轻量（2 个方法、24 行），不分 Phase，一次性覆盖。

测试追加到已有的 `overview_test.go` 中，复用 `mockStoreForOverview`。

## 测试用例

**新增测试数**: 5

| 测试 | 覆盖行为 |
|------|---------|
| `TestGetOTelSnapshot_Found` | 快照存在且含 OTel → 返回原始指针 |
| `TestGetOTelSnapshot_NoOTel` | 快照存在但 OTel=nil → 返回 nil |
| `TestGetOTelSnapshot_NoSnapshot` | clusterID 不存在 → 返回 nil |
| `TestGetOTelTimeline_Delegate` | store 返回注入的 2 条时间线 → 透传结果一致 |
| `TestGetOTelTimeline_Empty` | store 返回 nil → 透传 nil |

## Mock 扩展

`mockStoreForOverview` 最小扩展：

- 新增 `otelTimeline []cluster.OTelEntry` 字段
- `GetOTelTimeline` 方法从固定返回 nil 改为返回 `m.otelTimeline`
- 未引入新的 mock 类型

## 最终验收

| 指标 | 结果 |
|------|------|
| 新增测试数 | 5 |
| 全量测试 | 61/61 PASS（含 overview 31 + slo 23 + impl 2 + otel 5） |
| 修改测试文件 | 1（`overview_test.go` 追加测试 + 扩展 mock） |
| 业务代码变更 | 0 |
| 2 个方法覆盖 | 全部 |
| 发现缺陷 | 0 |
