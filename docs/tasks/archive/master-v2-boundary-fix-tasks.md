# Master V2 架构边界修复 — 完成记录

> 设计文档: [master-v2-boundary-fix-design.md](../../design/archive/master-v2-boundary-fix-design.md)
> 完成日期: 2026-03-13

## 概述

修复 Master V2 的 3 类架构边界违规，严格遵循 TDD（红-绿-重构）流程。

## Phase 1: MQ 调用下沉到 Service 层 ✅

Gateway Handler 不再直接持有 `mq.Producer`，所有 MQ 操作通过 `service.Ops` 接口调用。

- `service/operations/command.go` 新增 `ExecuteCommandSync` 方法
- `gateway/handler/` 中 `ops.go`、`observe.go`、`node_metrics.go` 移除 `bus` 字段
- 验证: `grep -r "mq.Producer" gateway/` = 0 匹配

## Phase 2: SLO Database 访问下沉到 Service 层 ✅

SLO Handler 不再直接持有 `database.SLORepository`，所有 DB 访问通过 Service 层。

- `model/slo.go` 新增 `SLORouteMapping` 结构体（Service 层返回类型，不暴露 database 类型）
- `service/interfaces.go` 新增 `QuerySLO`（4 方法）+ `OpsSLO`（1 方法）子接口
- `service/query/slo.go` 实现查询 + `toModel*` 转换函数
- `service/operations/slo.go` 实现 `UpsertSLOTarget`
- `service/factory.go` 嵌入 `*operations.SLOService`
- SLO Handler 11 处调用替换
- 验证: `grep -r "database\." gateway/handler/slo/` = 0 匹配

## Phase 3: 消除 Setter 注入 ✅

所有 QueryService 依赖通过构造函数一次性注入，消除 `Set*` 方法。

- `service/query/impl.go` 新增 `QueryServiceDeps` + `AdminRepos` 结构体
- 重写 `NewQueryService(deps QueryServiceDeps)` 构造函数
- 删除 `NewQueryServiceWithEventRepo`、`SetAIOpsEngine`、`SetAIOpsAI`、`SetAdminRepos`
- `master.go` 迁移为统一构造调用
- 验证: `grep -r "SetAIOps\|SetAdmin\|NewQueryServiceWithEventRepo" atlhyper_master_v2/` = 0 匹配

## 测试覆盖

| 文件 | 测试数 |
|------|--------|
| `service/operations/command_test.go` | 4 |
| `service/query/slo_test.go` | 5 |
| `service/operations/slo_test.go` | 2 |
| `service/query/impl_test.go` | 2 |
