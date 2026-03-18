# DataHub 测试补强 — 完成记录

> 设计文档: [master-v2-datahub-test-design.md](../../design/archive/master-v2-datahub-test-design.md)
> 完成日期: 2026-03-15

## 背景

`atlhyper_master_v2/datahub/` 是 Master V2 的核心数据存储层（861 行代码、5 个文件），承担快照存取、Agent 状态管理、OTel 时间线等职责。此前 **0 个测试文件**，任何重构或 bug 修复都没有安全网。

本轮目标：建立最小测试骨架，覆盖核心行为的正常路径和关键边界条件。只补测试，不做架构重构。

## Phase 1: OTelRing 单元测试 ✅

**文件**: `datahub/memory/otel_ring_test.go`（新增）
**测试数**: 8

覆盖 `OTelRing` 的 Add/Latest/Since/Count 四个方法：
- 添加与计数递增
- 空 Ring 的 Latest/Since 边界
- 循环覆盖（超过 capacity 后旧数据被替换）
- Since 时间过滤（只返回指定时间之后的条目）
- 默认容量（capacity ≤ 0 时回退到 90）

结果: 8/8 PASS

## Phase 2: MemoryStore 快照 + Agent + 事件 + OTel 时间线 + 生命周期 ✅

**文件**: `datahub/memory/store_test.go`（新增）
**测试数**: 22

| 分类 | 测试数 | 覆盖方法 |
|------|--------|---------|
| 快照存取 | 7 | SetSnapshot / GetSnapshot |
| Agent 状态 | 6 | UpdateHeartbeat / GetAgentStatus / ListAgents |
| 事件查询 | 2 | GetEvents |
| OTel 时间线 | 4 | GetOTelTimeline（端到端：SetSnapshot → Ring → Timeline） |
| 生命周期 | 2 | Start / Stop |
| 轻量复制回归 | 1 | lightweightOTelCopy 剥离大字段验证 |

结果: 22/22 PASS

### 发现并修复的缺陷: Stop() 非幂等

**问题**: `store.go:70` 的 `Stop()` 直接 `close(s.stopCh)`，第二次调用触发 `panic: close of closed channel`。

**修复**: 新增 `stopOnce sync.Once` 字段，`Stop()` 用 `stopOnce.Do()` 包裹关闭逻辑。

**TDD 流程**:
1. 测试从 SKIP 改为真正执行 → 红灯确认（panic）
2. 最小修复（2 处改动）→ 绿灯确认

## Phase 3: Factory 路由测试 ✅

**文件**: `datahub/factory_test.go`（新增）
**测试数**: 2

- Type 为空时返回 MemoryStore
- 未知 Type 回退到 MemoryStore

结果: 2/2 PASS

## 最终验收

| 指标 | 结果 |
|------|------|
| 总测试数 | 32 |
| PASS | 32 |
| FAIL | 0 |
| 新增测试文件 | 3（`otel_ring_test.go`、`store_test.go`、`factory_test.go`） |
| 修改业务文件 | 1（`store.go` — Stop 幂等修复，2 处改动） |
| Store 接口 9 方法覆盖 | 全部 |
| OTelRing 4 方法覆盖 | 全部 |
| 发现缺陷 | 1（Stop 非幂等，已修复） |
