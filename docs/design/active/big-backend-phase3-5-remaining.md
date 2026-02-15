# 大后端小前端重构 — Phase 3~5 剩余工作

> 原设计文档: [big-backend-small-frontend.md](../archive/big-backend-small-frontend.md)
> 本文档仅描述**尚未完成**的剩余工作。

---

## 现状总结

| Phase | 目标 | 完成度 | 备注 |
|-------|------|--------|------|
| Phase 3 | 9 种 K8s 资源 List 扁平化 | **7/9** | DaemonSet/StatefulSet 的 `List()` 仍返回原始 model_v2 |
| Phase 4 | SLO/Command camelCase | **0/6** | 未开始 |
| Phase 5 | 废弃文件清理 | **3/4** | 仅剩 `api/metrics.ts` |

---

## Phase 3: DaemonSet + StatefulSet List 扁平化

### 问题

DaemonSet 和 StatefulSet 的 `List()` Handler 直接返回 `model_v2` 嵌套结构（含 `summary`/`spec`/`status` 嵌套），前端被迫用 `parseDaemonSetList()` / `parseStatefulSetList()` 做数据转换。其余 7 种资源（Pod/Node/Deployment/Service/Namespace/Ingress/Event）的 `List()` 都已通过 `convert.XxxItems()` 完成扁平化。

### 目录结构与文件变更

```
atlhyper_master_v2/
├── model/
│   ├── daemonset.go                          [修改] 新增 DaemonSetItem 类型
│   ├── statefulset.go                        [修改] 新增 StatefulSetItem 类型
│   └── convert/
│       ├── daemonset.go                      [修改] 新增 DaemonSetItem() + DaemonSetItems()
│       ├── daemonset_test.go                 [新增] DaemonSetItem/Items 测试
│       ├── statefulset.go                    [修改] 新增 StatefulSetItem() + StatefulSetItems()
│       └── statefulset_test.go               [新增] StatefulSetItem/Items 测试
├── gateway/
│   └── handler/
│       ├── daemonset.go                      [修改] List() 调用 convert.DaemonSetItems()
│       └── statefulset.go                    [修改] List() 调用 convert.StatefulSetItems()

atlhyper_web/src/
├── api/
│   └── workload.ts                           [修改] 删除 parseDaemonSetList/parseStatefulSetList，
│                                                    更新响应类型 data: DaemonSetListItem[]
├── app/cluster/
│   ├── daemonset/page.tsx                    [修改] 移除 parseDaemonSetList 调用，直接使用 res.data.data
│   └── statefulset/page.tsx                  [修改] 移除 parseStatefulSetList 调用，直接使用 res.data.data
```

### 实现细节

#### Step 1: Master model — 新增 Item 类型

**`atlhyper_master_v2/model/daemonset.go`** — 新增 `DaemonSetItem`（参考 `DeploymentItem` 模式）：

```go
// DaemonSetItem DaemonSet 列表项（扁平，camelCase）
type DaemonSetItem struct {
    Name         string `json:"name"`
    Namespace    string `json:"namespace"`
    Desired      int32  `json:"desired"`
    Current      int32  `json:"current"`
    Ready        int32  `json:"ready"`
    Available    int32  `json:"available"`
    Misscheduled int32  `json:"misscheduled"`
    CreatedAt    string `json:"createdAt"`
    Age          string `json:"age"`
}
```

**`atlhyper_master_v2/model/statefulset.go`** — 新增 `StatefulSetItem`：

```go
// StatefulSetItem StatefulSet 列表项（扁平，camelCase）
type StatefulSetItem struct {
    Name        string `json:"name"`
    Namespace   string `json:"namespace"`
    Replicas    int32  `json:"replicas"`
    Ready       int32  `json:"ready"`
    Current     int32  `json:"current"`
    Updated     int32  `json:"updated"`
    Available   int32  `json:"available"`
    CreatedAt   string `json:"createdAt"`
    Age         string `json:"age"`
    ServiceName string `json:"serviceName"`
}
```

#### Step 2: convert 函数

**`atlhyper_master_v2/model/convert/daemonset.go`** — 新增（参考 `DeploymentItem/Items` 模式）：

```go
// DaemonSetItem 转换为列表项（扁平）
func DaemonSetItem(src *model_v2.DaemonSet) model.DaemonSetItem {
    return model.DaemonSetItem{
        Name:         src.Summary.Name,
        Namespace:    src.Summary.Namespace,
        Desired:      src.Summary.DesiredNumberScheduled,
        Current:      src.Summary.CurrentNumberScheduled,
        Ready:        src.Summary.NumberReady,
        Available:    src.Summary.NumberAvailable,
        Misscheduled: src.Summary.NumberMisscheduled,
        CreatedAt:    src.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
        Age:          src.Summary.Age,
    }
}

// DaemonSetItems 转换多个 DaemonSet 为列表项
func DaemonSetItems(src []model_v2.DaemonSet) []model.DaemonSetItem { ... }
```

**`atlhyper_master_v2/model/convert/statefulset.go`** — 同理。

#### Step 3: Handler 修改

**`atlhyper_master_v2/gateway/handler/daemonset.go`** 第 45-49 行：

```go
// 修改前
"data": daemonsets,

// 修改后
"data": convert.DaemonSetItems(daemonsets),
```

**`atlhyper_master_v2/gateway/handler/statefulset.go`** 第 45-49 行：同理。

#### Step 4: 前端清理

**`atlhyper_web/src/api/workload.ts`**：
- 删除 `parseDaemonSetList()` 函数（第 280-295 行）
- 删除 `parseStatefulSetList()` 函数（第 297-313 行）
- 删除 `RawItem` 接口（第 275-278 行）
- 修改 `DaemonSetListResponse` / `StatefulSetListResponse` 的 `data` 类型：`unknown[]` → `DaemonSetListItem[]` / `StatefulSetListItem[]`

**`atlhyper_web/src/app/cluster/daemonset/page.tsx`** 第 6、171 行：
```ts
// 修改前
import { getDaemonSetList, parseDaemonSetList, type DaemonSetListItem } from "@/api/workload";
setItems(parseDaemonSetList(res.data.data || []));

// 修改后
import { getDaemonSetList, type DaemonSetListItem } from "@/api/workload";
setItems(res.data.data || []);
```

**`atlhyper_web/src/app/cluster/statefulset/page.tsx`**：同理。

### 测试

```bash
go test ./atlhyper_master_v2/model/convert/... -run "DaemonSet|StatefulSet" -v
go build ./atlhyper_master_v2/...
cd atlhyper_web && npx next build
```

---

## Phase 4: SLO/Command camelCase + 业务逻辑后端化

### 问题

`model_v2/command.go` 是 Agent-Master **共享传输模型**，JSON tag 使用 snake_case（如 `cluster_id`、`created_at`、`command_id`）。当 Master 将指令状态返回给前端时，前端收到的也是 snake_case。按照「大后端小前端」原则，Web API 应返回 camelCase。

**注意**：`model_v2/command.go` 的 JSON tag **不能修改**——它是 Agent ↔ Master 的通信协议，修改会破坏兼容性。正确做法是在 Master 侧新增 `model/command.go`（camelCase）+ `convert/command.go`。

### 目录结构与文件变更

```
atlhyper_master_v2/
├── model/
│   ├── command.go                            [新增] CommandStatus/CommandResult camelCase 类型
│   └── convert/
│       ├── command.go                        [新增] model_v2.CommandStatus → model.CommandStatus
│       └── command_test.go                   [新增] 转换测试
├── gateway/
│   └── handler/
│       └── command.go                        [修改] 返回 convert.CommandStatus() 而非原始 model_v2

atlhyper_web/src/
├── types/
│   └── slo.ts                                [修改] SLOTarget 改为 camelCase
├── api/
│   └── slo.ts                                [修改] SLOTarget API 请求/响应 camelCase
├── components/slo/
│   └── (使用 SLOTarget 的组件)               [修改] 属性名同步改为 camelCase
```

### 实现细节

#### Step 1: Master model — Command camelCase 类型

**`atlhyper_master_v2/model/command.go`** [新增]：

```go
package model

// CommandStatusResponse 指令状态 (Web API 响应, camelCase)
type CommandStatusResponse struct {
    CommandID  string  `json:"commandId"`
    Status     string  `json:"status"`
    Result     *CommandResultResponse `json:"result,omitempty"`
    CreatedAt  string  `json:"createdAt"`
    StartedAt  string  `json:"startedAt,omitempty"`
    FinishedAt string  `json:"finishedAt,omitempty"`
}

// CommandResultResponse 指令执行结果 (Web API 响应)
type CommandResultResponse struct {
    Success    bool   `json:"success"`
    Output     string `json:"output,omitempty"`
    Error      string `json:"error,omitempty"`
    ExecTimeMs int64  `json:"execTimeMs,omitempty"`
    ExecutedAt string `json:"executedAt"`
}
```

#### Step 2: Convert 函数

**`atlhyper_master_v2/model/convert/command.go`** [新增]：
- `CommandStatusResponse(src *model_v2.CommandStatus) model.CommandStatusResponse`
- 将 `time.Time` → ISO 8601 字符串，`time.Duration` → 毫秒整数

#### Step 3: Handler 修改

**`atlhyper_master_v2/gateway/handler/command.go`**：
- 现有返回 `model_v2.CommandStatus` 的地方改为 `convert.CommandStatusResponse()`

#### Step 4: 前端 SLOTarget camelCase

**`atlhyper_web/src/types/slo.ts`** 第 82-91 行：

```ts
// 修改前
export interface SLOTarget {
  id?: number;
  cluster_id: string;
  host: string;
  time_range: string;
  availability_target: number;
  p95_latency_target: number;
  created_at?: string;
  updated_at?: string;
}

// 修改后
export interface SLOTarget {
  id?: number;
  clusterId: string;
  host: string;
  timeRange: string;
  availabilityTarget: number;
  p95LatencyTarget: number;
  createdAt?: string;
  updatedAt?: string;
}
```

**前提条件**：Master SLO Target API 需要返回 camelCase JSON。当前 SLOTarget 直接从 SQLite 查询返回，如果 DB 层返回的就是 snake_case，需要在 service/query 层或 handler 层做转换。

#### (可选) Error Budget / 拓扑 BFS 后端化

这两项标记为可选，暂不纳入本次迭代。

### 测试

```bash
go test ./atlhyper_master_v2/model/convert/... -run "Command" -v
go build ./atlhyper_master_v2/...
cd atlhyper_web && npx next build
```

---

## Phase 5: 废弃文件清理

### 现状

| 文件 | 状态 | 原因 |
|------|------|------|
| `api/metrics.ts` | **待删除** | 旧指标 API（`/uiapi/metrics/*`），无任何组件引用 |
| `api/config.ts` | ✅ 已删除 | — |
| `api/test.ts` | ✅ 已删除 | — |
| `utils/safeData.ts` | ✅ 已删除 | — |

### 文件变更

```
atlhyper_web/src/
├── api/
│   └── metrics.ts                            [删除] 旧指标 API，已被 api/node-metrics.ts 替代
├── types/
│   └── cluster.ts                            [审查] MetricsOverview / NodeMetricsDetail 类型
│                                                    若仅被 metrics.ts 使用则一并清理
```

### 验证

确认 `getMetricsOverview` 和 `getNodeMetricsDetail` 无任何引用后删除。

```bash
# 确认无引用
grep -r "api/metrics" atlhyper_web/src/ --include="*.ts" --include="*.tsx"
# 删除后构建
cd atlhyper_web && npx next build
```

---

## 执行顺序

```
Phase 3 (DaemonSet/StatefulSet 扁平化)
  ├── Step 1: Master model 新增 Item 类型
  ├── Step 2: convert 函数 + 测试
  ├── Step 3: Handler 修改
  ├── Step 4: 前端删除 parse 函数
  └── 验证: go test + go build + next build

Phase 5 (废弃文件清理)  ← 可与 Phase 3 同步进行
  └── 删除 api/metrics.ts + 验证

Phase 4 (SLO/Command camelCase)  ← 依赖 Phase 3 完成
  ├── Step 1: Master model/command.go
  ├── Step 2: convert/command.go + 测试
  ├── Step 3: Handler 修改
  ├── Step 4: 前端 SLOTarget camelCase
  └── 验证
```

---

## 文件变更总结

| 操作 | 文件 | 说明 |
|:----:|------|------|
| **新增** | `atlhyper_master_v2/model/command.go` | CommandStatusResponse camelCase 类型 |
| **新增** | `atlhyper_master_v2/model/convert/command.go` | Command 转换函数 |
| **新增** | `atlhyper_master_v2/model/convert/command_test.go` | Command 转换测试 |
| **新增** | `atlhyper_master_v2/model/convert/daemonset_test.go` | DaemonSet Item 转换测试 |
| **新增** | `atlhyper_master_v2/model/convert/statefulset_test.go` | StatefulSet Item 转换测试 |
| **修改** | `atlhyper_master_v2/model/daemonset.go` | +DaemonSetItem 类型 |
| **修改** | `atlhyper_master_v2/model/statefulset.go` | +StatefulSetItem 类型 |
| **修改** | `atlhyper_master_v2/model/convert/daemonset.go` | +DaemonSetItem/Items 函数 |
| **修改** | `atlhyper_master_v2/model/convert/statefulset.go` | +StatefulSetItem/Items 函数 |
| **修改** | `atlhyper_master_v2/gateway/handler/daemonset.go` | List() 调用 convert |
| **修改** | `atlhyper_master_v2/gateway/handler/statefulset.go` | List() 调用 convert |
| **修改** | `atlhyper_master_v2/gateway/handler/command.go` | 返回 camelCase |
| **修改** | `atlhyper_web/src/api/workload.ts` | 删除 parse 函数，更新类型 |
| **修改** | `atlhyper_web/src/app/cluster/daemonset/page.tsx` | 移除 parse 调用 |
| **修改** | `atlhyper_web/src/app/cluster/statefulset/page.tsx` | 移除 parse 调用 |
| **修改** | `atlhyper_web/src/types/slo.ts` | SLOTarget camelCase |
| **修改** | `atlhyper_web/src/api/slo.ts` | SLOTarget 请求/响应适配 |
| **删除** | `atlhyper_web/src/api/metrics.ts` | 旧指标 API，无引用 |

共 ~18 文件变更（5 新增，12 修改，1 删除）。
