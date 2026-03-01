# Agent V3 测试方案

## 背景

Agent V3 Phase 1 迁移完成后，`atlhyper_agent_v2/` 有 43 个文件 ~7,700 行代码，但 **0 个测试**。
Master V2 已有 27 个测试文件（表驱动、手工 Mock、原生断言），需要为 Agent 补充对应的测试覆盖。

## 工具选择

| 维度 | 选择 | 理由 |
|------|------|------|
| 断言 | 原生 `if/t.Errorf` | 与 Master V2 现有 27 个测试保持风格一致 |
| Mock | 手工编写 | 接口方法数少（多数 2-3 个），Master V2 已验证此模式 |
| 模式 | 表驱动测试 | CLAUDE.md 明确要求 |

## Mock 策略

**共享 mock 包** `testutil/mock/`：被多个测试复用的 Mock（PodRepo、GenericRepo、Gateway）
**文件内联**：仅单个测试用的简单 Mock

```
atlhyper_agent_v2/testutil/mock/
├── repository.go    # PodRepository + GenericRepository
├── k8s_repos.go     # 20 个 K8s 仓库（Snapshot 测试用）
├── otel_repo.go     # OTelSummaryRepository
├── gateway.go       # MasterGateway
└── service.go       # SnapshotService + CommandService（Scheduler 测试用）
```

---

## Phase 1: 纯函数测试（converter + summary + command 辅助）

**价值**: 极高。~2,200 行无外部依赖的纯逻辑，Agent 最核心的数据准确性保障。

### 文件

| 测试文件 | 测试目标 | 用例数 |
|----------|----------|--------|
| `repository/k8s/converter_test.go` | 16 个 Convert 函数 (Pod/Node/Deployment 等) | ~30 |
| `service/command/summary_test.go` | summarizeList + formatAge + 辅助函数 | ~15 |
| `service/command/command_helpers_test.go` | buildAPIPath + stripManagedFields + buildEventFieldSelector | ~10 |

### 关键用例

**converter_test.go**:
- `TestConvertPod_BasicFields` — 完整 Pod 转换字段映射
- `TestConvertPod_ContainerMerge` — spec + status 按 name 合并
- `TestConvertPod_SidecarReasonFallback` — sidecar 异常原因
- `TestConvertNode_RoleLabels` — 多角色标签解析
- `TestConvertNode_IPv4Priority` — 多地址 IPv4 优先
- `TestConvertDeployment_RolloutPhases` — 表驱动各状态
- 每个 Convert 函数至少: 正常路径 + 空/nil 输入
- converter_test.go 如超 500 行可拆为 `converter_pod_test.go` 等

**辅助构造函数模式**:
```go
func makeK8sPod(name, ns, phase string, opts ...func(*corev1.Pod)) *corev1.Pod { ... }
func withOwner(kind, name string) func(*corev1.Pod) { ... }
func withContainers(names ...string) func(*corev1.Pod) { ... }
```

**summary_test.go**:
- `TestSummarizeList_PodList` — 表格包含 NAME/NAMESPACE/STATUS 列
- `TestSummarizeList_EmptyItems` — 空列表返回 "0 items"
- `TestFormatAge` — 表驱动时间格式化

**command_helpers_test.go**:
- `TestBuildAPIPath_AllKindMappings` — 表驱动遍历所有 Kind
- `TestStripManagedFields_RemoveFromList` — items 中递归移除

---

## Phase 2: Command Service 测试

**价值**: 高。10 种 Action 路由 + 参数验证。

### 文件

| 测试文件 | Mock 依赖 | 用例数 |
|----------|-----------|--------|
| `service/command/command_test.go` | `mock/repository.go` | ~30 |

### 关键用例

- `TestExecute_Scale_Success` — params 解析 + ScaleDeployment 调用
- `TestExecute_Scale_InvalidParams` — 非法 params → error
- `TestExecute_GetLogs_AutoSelectContainer` — 自动选非 sidecar 容器
- `TestExecute_GetLogs_TailLinesClamp` — 0→100, 500→200
- `TestExecute_Dynamic_AISource_SummarizeList` — AI 来源走表格摘要
- `TestExecute_Dynamic_4xxError` — HTTP 错误码处理
- `TestExecute_Delete_Pod` vs `TestExecute_Delete_Generic` — 路由分支
- `TestExecute_UnknownAction` — 未知 action
- `TestExecute_ResultFormat` — ExecTime > 0, CommandID 正确

---

## Phase 3: Snapshot Service 测试

**价值**: 中高。验证 20 goroutine 并发采集、错误处理、OTel 缓存。

### 文件

| 测试文件 | Mock 依赖 | 用例数 |
|----------|-----------|--------|
| `service/snapshot/snapshot_test.go` | `mock/k8s_repos.go` + `mock/otel_repo.go` | ~12 |

### 关键用例

- `TestCollect_AllResourcesPopulated` — 全部 Mock 返回数据，20 字段非 nil
- `TestCollect_PartialFailure` — PodRepo 报错，其他正常
- `TestCalculateNamespaceResources_BasicCounting` — 按 namespace 统计
- `TestCalculateNamespaceResources_QuotaAssociation` — Quota/LimitRange 关联
- `TestGetOTelSummary_CacheBehavior` — TTL 内缓存命中
- `TestGetOTelSummary_NilRepo` — repo=nil 时 OTel 为 nil

---

## Phase 4: Scheduler 测试

**价值**: 中。生命周期管理 + 循环编排。

### 文件

| 测试文件 | Mock 依赖 | 用例数 |
|----------|-----------|--------|
| `scheduler/scheduler_test.go` | `mock/gateway.go` + `mock/service.go` | ~8 |

### 关键用例

- `TestScheduler_StartStop` — 启动后立即停止不 panic
- `TestCollectAndPushSnapshot_Success` — 采集+推送正常流程
- `TestCollectAndPushSnapshot_CollectError` — 采集失败不推送
- `TestPollAndExecuteCommands_WithCommands` — 多指令并发执行+上报
- `TestPollAndExecuteCommands_PollError` — 拉取失败不 panic

注意：直接测试 `collectAndPushSnapshot()` / `pollAndExecuteCommands()` 方法（同包测试），避免 `time.Sleep` 导致 flaky。

---

## Phase 5: Gateway 测试 (httptest)

**价值**: 中。HTTP 请求构建、Gzip 压缩、响应解析。

### 文件

| 测试文件 | Mock 依赖 | 用例数 |
|----------|-----------|--------|
| `gateway/master_gateway_test.go` | 无（httptest） | ~10 |

### 关键用例

- `TestPushSnapshot_RequestFormat` — Content-Type/Content-Encoding/X-Cluster-ID
- `TestPushSnapshot_CompressedPayload` — 解压后是有效 JSON
- `TestPollCommands_HasCommand` — 解析返回 Command
- `TestPollCommands_NoContent` — 204 → nil, nil
- `TestPollCommands_ContextCanceled` — context 取消不报错
- `TestReportResult_RequestFormat`

---

## 汇总

| Phase | 测试文件 | Mock 文件 | 用例数 | 行数 |
|-------|---------|-----------|--------|------|
| P1 纯函数 | 3 | 0 | ~55 | ~1,600 |
| P2 Command | 1 | 1 | ~30 | ~800 |
| P3 Snapshot | 1 | 2 | ~12 | ~600 |
| P4 Scheduler | 1 | 2 | ~8 | ~400 |
| P5 Gateway | 1 | 0 | ~10 | ~350 |
| **合计** | **7** | **5** | **~115** | **~3,750** |

## 验证

每个 Phase 完成后：
```bash
go test ./atlhyper_agent_v2/...
go test -v -count=1 ./atlhyper_agent_v2/repository/k8s/...   # Phase 1
go test -v -count=1 ./atlhyper_agent_v2/service/command/...   # Phase 1+2
go test -v -count=1 ./atlhyper_agent_v2/service/snapshot/...  # Phase 3
go test -v -count=1 ./atlhyper_agent_v2/scheduler/...         # Phase 4
go test -v -count=1 ./atlhyper_agent_v2/gateway/...           # Phase 5
```

## 注意事项

- **时间相关测试**: converter 的 `age` 计算依赖 `time.Now()`，使用足够远的过去时间确保稳定（如 10 天前 → "10d"）
- **Scheduler 不测循环**: 只测被循环调用的方法，避免 `time.Sleep` flaky
- **converter 拆分**: 如超 500 行按资源类型拆分 (`converter_pod_test.go` 等)
