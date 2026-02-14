# 大后端小前端：Master 专用模型 + Gateway 转换层

> 状态：已批准，待实施
> 最终审计日期：2026-02-14

---

## Context

### 问题

当前 Master 的 Gateway 层直接返回 `model_v2` 结构体给前端。`model_v2` 是 Agent - Master 的通信模型，不是为 Web API 设计的。导致：

1. **NodeMetrics / Overview**: 后端 snake_case，前端需要 camelCase -> 16+1 个 transform 函数（~280 行）
2. **K8s 资源（9 种）**: 后端嵌套结构（summary/spec/status），前端需要扁平化 -> 每种 2~3 个 transform
3. **K8s 资源 Overview 聚合**: 前端对列表做统计计算（count running/ready/failed 等）-> 每种 1 个聚合函数
4. **K8s 单位解析**: 前端自行解析 K8s 内存/CPU 单位（Ki/Mi/Gi/m/n）-> `parseMemoryToGiB`, `parseCPUCores`
5. **SLO 业务计算**: error budget（公式错误）、趋势预测、拓扑过滤散落在前端组件中
6. **SLO/Mesh**: `model/slo.go` 已有 Master 专用响应类型但用 snake_case -> 前端类型也被迫 snake_case
7. **架构耦合**: Web API 响应格式被 Agent 通信模型绑定，改一个影响两头

### 目标

- `model_v2/` — **只用于 Agent - Master 通信**，不暴露给前端
- `atlhyper_master_v2/model/` — **Master 专用模型**，定义 Web API 响应类型（统一 camelCase JSON tag）
- Gateway Handler 负责 `model_v2` -> `master/model` 转换
- 前端删除所有 transform 函数，API 返回即直接使用

### 现状

| 层 | 当前状态 |
|---|---|
| `model_v2/` | Agent - Master 共享，17 个文件。K8s 资源用 camelCase + 嵌套结构，NodeMetrics/Overview 用 snake_case |
| `atlhyper_master_v2/model/` | 已有 SLO (slo.go)、Command (command.go)、Query (query.go) 三个文件，SLO 类型 snake_case JSON tag |
| `service/interfaces.go` | 返回 `model_v2.*` 类型（K8s/Overview/Cluster），少量返回 `model.*`（SLO/Command） |
| Gateway handler | 直接把 service 层返回值 `writeJSON` 给前端，无转换 |
| 前端 api/*.ts | 每个 API 文件都有 transform 函数，共 37 个 |
| 前端 types/ | K8s 资源全在 `cluster.ts`（978 行），命名风格混乱（camelCase/snake_case/PascalCase 混用） |

---

## 前端 transform 函数完整审计清单

### 命名转换（snake_case -> camelCase）

| 文件 | 函数 | 行数 | 类型 |
|------|------|------|------|
| `api/node-metrics.ts` | transformSummary, transformCPU, transformMemory, transformDisk, transformNetwork, transformSensor, transformTemperature, transformProcess, transformPSI, transformTCP, transformSystem, transformVMStat, transformNTP, transformSoftnet, transformSnapshot, transformDataPoint | ~200 | 命名转换 |
| `api/overview.ts` | transformResponse | 81 | 命名转换 |
| `app/overview/utils.ts` | transformOverview | 84 | 安全转换+空值处理 |

### 结构扁平化（嵌套 -> 扁平）

| 文件 | 函数 | 行数 | 类型 |
|------|------|------|------|
| `api/pod.ts` | transformPodToDetail | 81 | summary/spec/status -> 扁平 |
| `api/pod.ts` | transformPod | 43 | 扁平化 + labels 推断 deployment |
| `api/node.ts` | transformNodeItem | 15 | 扁平化 + 单位转换 |
| `api/node.ts` | transformToNodeDetail | 95 | 扁平化 + 使用率百分比计算 |
| `api/deployment.ts` | transformDeploymentItem | 14 | 扁平化 + 镜像提取 |
| `api/deployment.ts` | transformToDeploymentDetail | 104 | 扁平化 |
| `api/service.ts` | transformServiceItem | 28 | 端口格式化字符串 |
| `api/service.ts` | transformToServiceDetail | 64 | 扁平化 |
| `api/namespace.ts` | transformNamespaceItem | 13 | label/annotation 计数 |
| `api/namespace.ts` | transformNamespaceToDetail | 57 | 扁平化 |
| `api/ingress.ts` | expandIngressToRows | 58 | 1 Ingress -> N 行（host*path 展开） |
| `api/ingress.ts` | transformIngressDetail | 33 | 扁平化 |
| `api/event.ts` | transformEventItem | 23 | type->severity 枚举映射 |
| `api/workload.ts` | transformStatefulSetDetail | 46 | summary/spec/status -> 扁平 |
| `api/workload.ts` | transformDaemonSetDetail | 42 | summary/spec/status -> 扁平 |

### 数据聚合统计（列表 -> 统计摘要）

| 文件 | 函数 | 行数 | 计算内容 |
|------|------|------|----------|
| `api/pod.ts` | transformPodListToOverview | 36 | running/pending/failed/succeeded 计数 |
| `api/node.ts` | transformToNodeOverview | 23 | ready 节点数、总 CPU/内存容量 |
| `api/deployment.ts` | transformToDeploymentOverview | 22 | namespace 数、总 replicas |
| `api/service.ts` | transformToServiceOverview | 29 | external/internal/headless 分类计数 |
| `api/namespace.ts` | transformToNamespaceOverview | 25 | active/terminating 计数、总 pods |
| `api/ingress.ts` | transformToIngressOverview | 27 | hosts/TLS/paths 计数 |
| `api/event.ts` | transformToEventOverview | 37 | kinds/categories/severity 计数 |

### 单位转换

| 文件 | 函数 | 行数 | 内容 |
|------|------|------|------|
| `api/node.ts` | parseMemoryToGiB | 21 | K8s 内存字符串 -> GiB（Ki/Mi/Gi/Ti/K/M/G/T） |
| `api/node.ts` | parseCPUCores | 18 | K8s CPU 字符串 -> 核心数（m/n 后缀） |
| `api/ingress.ts` | hasTLS | 9 | 检查 host 是否有 TLS |

### SLO 业务计算（组件层）

| 文件 | 逻辑 | 行数 | 内容 |
|------|------|------|------|
| `components/slo/OverviewTab.tsx` | budgetHistory useMemo | 7 | error_rate -> error_budget（公式错误：100-error_rate*20） |
| `components/slo/DomainCard.tsx` | prevXxx 计算 | 15 | services 数组聚合 previous 指标 |
| `components/slo/DomainCard.tsx` | loadMeshData 过滤 | 10 | namespace 过滤拓扑节点（应为 BFS） |

### 无 transform 的 API 文件

cluster.ts, slo.ts, mesh.ts, auth.ts, commands.ts, notify.ts, settings.ts, ai-provider.ts, ai.ts — 直接透传后端响应。

### 统计

**总计消除前端代码**: ~1360 行 transform/parse/聚合逻辑

---

## 设计

### 转换边界

```
                    转换边界（Gateway 层）
                          │
Agent ──model_v2──> 内部各层 ──│── model/convert ──> master/model ──> Web 前端
                              │
   不动                不动     │    新增               新增/改造      删 transform
```

**关键决策：只改 Gateway 层 + model 层，不动 Service/DataHub/Processor/Database 层。**

Service 层继续返回 `model_v2.*` 类型，Gateway Handler 调用 `model/convert.*()` 转换后再返回前端。

### 不需要修改的层

| 层 | 路径 | 理由 |
|---|---|---|
| model_v2/ | `model_v2/*.go` | Agent 通信模型，改动影响 Agent，不动 |
| Service 接口 | `service/interfaces.go` | 返回类型保持 model_v2，不变 |
| Service Query 实现 | `service/query/*.go` | 内部逻辑不变，返回类型不变 |
| Service Operations | `service/operations/*.go` | 写入逻辑不变 |
| Service Sync | `service/sync/*.go` | 持久化逻辑不变 |
| DataHub | `datahub/` | 内存存储，类型不变 |
| Processor | `processor/` | 数据处理，类型不变 |
| Database | `database/` | 持久化层，类型不变 |
| AgentSDK | `agentsdk/` | Agent 通信，类型不变 |
| SLO 处理器 | `slo/` | 领域逻辑，类型不变 |
| MQ | `mq/` | 消息队列，类型不变 |

### `atlhyper_master_v2/model/` 扩展结构

```
atlhyper_master_v2/model/
├── query.go              # 已有，内部查询选项，不改
├── command.go            # 已有，JSON tags -> camelCase
├── slo.go                # 已有，JSON tags -> camelCase
├── cluster.go            # 新建 — ClusterInfoResponse / ClusterDetailResponse
├── overview.go           # 新建 — OverviewResponse
├── node_metrics.go       # 新建 — NodeMetricsSnapshotResponse
├── pod.go                # 新建 — PodItemResponse / PodDetailResponse（扁平）
├── node.go               # 新建 — NodeItemResponse / NodeDetailResponse（扁平）
├── deployment.go         # 新建 — DeploymentItemResponse / DeploymentDetailResponse
├── statefulset.go        # 新建 — StatefulSetDetailResponse
├── daemonset.go          # 新建 — DaemonSetDetailResponse
├── service.go            # 新建 — ServiceItemResponse / ServiceDetailResponse
├── namespace.go          # 新建 — NamespaceItemResponse / NamespaceDetailResponse
├── ingress.go            # 新建 — IngressItemResponse / IngressDetailResponse
├── event.go              # 新建 — EventItemResponse
└── convert/              # 新建子目录 — 转换函数
    ├── cluster.go
    ├── overview.go
    ├── node_metrics.go
    ├── pod.go
    ├── node.go
    ├── deployment.go
    ├── statefulset.go
    ├── daemonset.go
    ├── service.go
    ├── namespace.go
    ├── ingress.go
    └── event.go
```

---

## 全量目录树（标注变更状态）

图例：`[新建]` 新文件 | `[修改]` 需修改 | `[删除]` 删除 | 无标注 = 不动

### model_v2/（Agent - Master 共享模型 — 全部不动）

```
model_v2/
├── agent.go
├── command.go
├── common.go
├── deployment.go
├── event.go
├── job.go
├── namespace.go
├── node.go
├── node_metrics.go
├── overview.go
├── pod.go
├── policy.go
├── service.go
├── slo.go
├── snapshot.go
├── storage.go
└── workload.go
```

### atlhyper_master_v2/（Master 后端）

```
atlhyper_master_v2/
├── master.go
│
├── model/                                  # ★ 变更集中区
│   ├── query.go                            #   已有，不改（内部查询选项）
│   ├── command.go                          #   [修改] P4: JSON tags -> camelCase
│   ├── slo.go                              #   [修改] P4: JSON tags -> camelCase
│   ├── cluster.go                          #   [新建] P3: ClusterInfoResponse / ClusterDetailResponse
│   ├── overview.go                         #   [新建] P2: OverviewResponse
│   ├── node_metrics.go                     #   [新建] P1: NodeMetricsSnapshotResponse / CPUResponse ...
│   ├── pod.go                              #   [新建] P3: PodItemResponse / PodDetailResponse
│   ├── node.go                             #   [新建] P3: NodeItemResponse / NodeDetailResponse
│   ├── deployment.go                       #   [新建] P3: DeploymentItemResponse / DeploymentDetailResponse
│   ├── statefulset.go                      #   [新建] P3: StatefulSetDetailResponse
│   ├── daemonset.go                        #   [新建] P3: DaemonSetDetailResponse
│   ├── service.go                          #   [新建] P3: ServiceItemResponse / ServiceDetailResponse
│   ├── namespace.go                        #   [新建] P3: NamespaceItemResponse / NamespaceDetailResponse
│   ├── ingress.go                          #   [新建] P3: IngressItemResponse / IngressDetailResponse
│   ├── event.go                            #   [新建] P3: EventItemResponse
│   └── convert/                            #   [新建] 转换函数子目录
│       ├── node_metrics.go                 #     [新建] P1
│       ├── overview.go                     #     [新建] P2
│       ├── cluster.go                      #     [新建] P3
│       ├── pod.go                          #     [新建] P3
│       ├── node.go                         #     [新建] P3
│       ├── deployment.go                   #     [新建] P3
│       ├── statefulset.go                  #     [新建] P3
│       ├── daemonset.go                    #     [新建] P3
│       ├── service.go                      #     [新建] P3
│       ├── namespace.go                    #     [新建] P3
│       ├── ingress.go                      #     [新建] P3
│       └── event.go                        #     [新建] P3
│
├── gateway/                                # ★ 变更集中区
│   ├── server.go
│   ├── routes.go
│   ├── middleware/
│   │   ├── audit.go
│   │   ├── auth.go
│   │   ├── cors.go
│   │   └── logging.go
│   └── handler/
│       ├── helper.go
│       ├── node_metrics.go                 #   [修改] P1: 3 处返回点加 convert 调用
│       ├── overview.go                     #   [修改] P2: 加 convert.Overview() 调用
│       ├── cluster.go                      #   [修改] P3: List/Get 加 convert 调用
│       ├── pod.go                          #   [修改] P3: List/Get 加 convert 调用
│       ├── node.go                         #   [修改] P3: List/Get 加 convert 调用
│       ├── deployment.go                   #   [修改] P3: List/Get 加 convert 调用
│       ├── statefulset.go                  #   [修改] P3: List/Get 加 convert 调用
│       ├── daemonset.go                    #   [修改] P3: List/Get 加 convert 调用
│       ├── service.go                      #   [修改] P3: List/Get 加 convert 调用
│       ├── namespace.go                    #   [修改] P3: List/Get 加 convert 调用
│       ├── ingress.go                      #   [修改] P3: List/Get 加 convert 调用
│       ├── event.go                        #   [修改] P3: 3 处查询结果加 convert 调用
│       ├── command.go                      #   [修改] P4: map key "command_id" -> "commandId"
│       ├── slo.go                          #   P4 JSON tag 自动生效，Go 代码不改
│       ├── slo_latency.go                  #   P4 JSON tag 自动生效，Go 代码不改
│       ├── slo_mesh.go                     #   P4 JSON tag 自动生效，Go 代码不改
│       ├── user.go
│       ├── ops.go
│       ├── ai.go
│       ├── ai_provider.go
│       ├── audit.go
│       ├── configmap.go
│       ├── secret.go
│       ├── notify.go
│       └── settings.go
│
├── agentsdk/                               # 不动
│   ├── server.go
│   ├── snapshot.go
│   ├── heartbeat.go
│   ├── command.go
│   ├── result.go
│   └── types.go
│
├── processor/                              # 不动
│   └── processor.go
│
├── datahub/                                # 不动
│   ├── interfaces.go
│   ├── factory.go
│   └── memory/
│       └── store.go
│
├── service/                                # 不动
│   ├── interfaces.go
│   ├── factory.go
│   ├── query/
│   │   ├── impl.go
│   │   ├── slo.go
│   │   └── metrics_format.go
│   ├── operations/
│   │   └── command.go
│   └── sync/
│       ├── event_persist.go
│       ├── metrics_persist.go
│       └── slo_persist.go
│
├── database/                               # 不动
│   ├── interfaces.go
│   ├── factory.go
│   ├── sync.go
│   ├── repo/
│   │   ├── init.go
│   │   ├── audit.go
│   │   ├── cluster.go
│   │   ├── command.go
│   │   ├── event.go
│   │   ├── node_metrics.go
│   │   ├── notify.go
│   │   ├── settings.go
│   │   ├── slo.go
│   │   ├── slo_edge.go
│   │   ├── slo_service.go
│   │   ├── user.go
│   │   ├── ai_active.go
│   │   ├── ai_conversation.go
│   │   ├── ai_message.go
│   │   ├── ai_model.go
│   │   └── ai_provider.go
│   └── sqlite/
│       ├── dialect.go
│       ├── helpers.go
│       ├── migrations.go
│       ├── audit.go
│       ├── cluster.go
│       ├── command.go
│       ├── event.go
│       ├── node_metrics.go
│       ├── notify.go
│       ├── settings.go
│       ├── slo.go
│       ├── slo_edge.go
│       ├── slo_service.go
│       ├── user.go
│       └── ai.go
│
├── mq/                                     # 不动
│   ├── interfaces.go
│   ├── factory.go
│   └── memory/
│       └── bus.go
│
├── slo/                                    # 不动
│   ├── interfaces.go
│   ├── processor.go
│   ├── aggregator.go
│   ├── calculator.go
│   ├── cleaner.go
│   └── status_checker.go
│
├── ai/                                     # 不动
│   ├── interfaces.go
│   ├── service.go
│   ├── chat.go
│   ├── prompt.go
│   ├── prompts.go
│   ├── tool.go
│   ├── blacklist.go
│   └── llm/
│       ├── interfaces.go
│       ├── factory.go
│       ├── gemini/
│       │   └── client.go
│       ├── openai/
│       │   └── client.go
│       └── anthropic/
│           └── client.go
│
├── notifier/                               # 不动
│   ├── interface.go
│   ├── manager.go
│   ├── alert.go
│   ├── channel/
│   │   ├── interface.go
│   │   ├── slack.go
│   │   └── email.go
│   ├── enrich/
│   │   ├── interface.go
│   │   └── enricher.go
│   ├── template/
│   │   └── renderer.go
│   └── trigger/
│       ├── event.go
│       └── heartbeat.go
│
├── config/                                 # 不动
│   ├── types.go
│   ├── loader.go
│   └── defaults.go
│
└── tester/                                 # 不动
    ├── interfaces.go
    ├── server.go
    ├── handler.go
    ├── registry.go
    └── notifier.go
```

### atlhyper_web/src/（前端）

```
atlhyper_web/src/
├── api/
│   ├── request.ts                          #   不动（Axios 封装）
│   ├── node-metrics.ts                     #   [修改] P1: 删除 16 个 transform（~200 行）
│   ├── overview.ts                         #   [修改] P2: 删除 transformResponse（~81 行）
│   ├── pod.ts                              #   [修改] P3: 删除 3 个 transform（~190 行）
│   ├── node.ts                             #   [修改] P3: 删除 5 个 transform（~170 行）
│   ├── deployment.ts                       #   [修改] P3: 删除 3 个 transform（~130 行）
│   ├── service.ts                          #   [修改] P3: 删除 3 个 transform（~70 行）
│   ├── namespace.ts                        #   [修改] P3: 删除 3 个 transform（~80 行）
│   ├── ingress.ts                          #   [修改] P3: 删除 4 个 transform（~120 行）
│   ├── event.ts                            #   [修改] P3: 删除 2 个 transform（~30 行）
│   ├── workload.ts                         #   [修改] P3: 删除 2 个 transform（~90 行）
│   ├── cluster.ts                          #   不动（无 transform）
│   ├── slo.ts                              #   不动（无 transform）
│   ├── mesh.ts                             #   不动（无 transform）
│   ├── commands.ts                         #   不动
│   ├── auth.ts                             #   不动
│   ├── ai.ts                               #   不动
│   ├── ai-provider.ts                      #   不动
│   ├── notify.ts                           #   不动
│   ├── settings.ts                         #   不动
│   ├── config.ts                           #   [删除] P5: 已整合到 settings
│   ├── metrics.ts                          #   [删除] P5: 已被 node-metrics.ts 替代
│   └── test.ts                             #   [删除] P5: 测试用文件
│
├── types/
│   ├── index.ts                            #   不动
│   ├── common.ts                           #   不动
│   ├── auth.ts                             #   不动
│   ├── i18n.ts                             #   不动
│   ├── cluster.ts                          #   [修改] P3: ClusterInfo snake->camel，确认扁平类型对齐
│   ├── overview.ts                         #   [修改] P2: 确认与后端 OverviewResponse 字段对齐
│   ├── node-metrics.ts                     #   不动（已经是 camelCase）
│   ├── slo.ts                              #   [修改] P4: snake_case -> camelCase
│   └── mesh.ts                             #   [修改] P4: snake_case -> camelCase
│
├── utils/
│   ├── index.ts                            #   不动
│   └── safeData.ts                         #   [审查] P5: 删 transform 后可能不再需要
│
└── app/
    └── overview/
        ├── page.tsx                        #   不动
        ├── utils.ts                        #   [修改] P1/P2: 删除 NodeMetrics/Overview 转换代码
        └── components/
            ├── index.ts
            ├── HealthCard.tsx
            ├── StatCard.tsx
            ├── WorkloadSummaryCard.tsx
            ├── NodeResourceCard.tsx
            ├── RecentAlertsCard.tsx
            ├── AlertDetailModal.tsx
            └── SloOverviewCard.tsx
```

---

### JSON 命名规范

| 层 | JSON tag | 理由 |
|---|---|---|
| `model_v2/` (Agent - Master) | 保持现状不动 | 改动影响 Agent |
| `atlhyper_master_v2/model/` (Web API) | **统一 camelCase** | 前端 JS/TS 原生命名 |
| 请求参数 (query string) | snake_case | URL 惯例 |

### Node 单位转换后端化

当前 `node.ts` 的 `parseMemoryToGiB` 和 `parseCPUCores` 解析 K8s 单位字符串。后端 `convert/node.go` 在转换时直接计算好：

```go
type NodeItemResponse struct {
    CPUCapacity        float64 `json:"cpuCapacity"`        // 核心数（已从 "4000m" 转为 4.0）
    MemoryCapacity     float64 `json:"memoryCapacity"`     // GiB（已从 "16384Ki" 转为 16.0）
    CPUUsagePercent    float64 `json:"cpuUsagePercent"`    // 百分比
    MemoryUsagePercent float64 `json:"memoryUsagePercent"` // 百分比
}
```

### 转换模式示例

```go
// gateway/handler/node_metrics.go — 修改前
nodes := snapshot.NodeMetrics
writeJSON(w, 200, map[string]interface{}{"nodes": nodes})

// gateway/handler/node_metrics.go — 修改后
import "AtlHyper/atlhyper_master_v2/model/convert"
nodes := convert.NodeMetricsSnapshots(snapshot.NodeMetrics)
writeJSON(w, 200, map[string]interface{}{"nodes": nodes})
```

---

## 全量修改文件清单

### 统计总览

| 分类 | 新建 | 修改 | 删除 | 合计 |
|------|------|------|------|------|
| 后端 model/ 响应类型 | 12 | 2 | 0 | 14 |
| 后端 model/convert/ | 12 | 0 | 0 | 12 |
| 后端 gateway/handler/ | 0 | 14 | 0 | 14 |
| 前端 api/ | 0 | 10 | 3 | 13 |
| 前端 types/ | 0 | 4 | 0 | 4 |
| 前端 其他 | 0 | 1 | 0 | 1 |
| **合计** | **24** | **31** | **3** | **58** |

---

## Phase 1: NodeMetrics（收益最大，~280 行前端代码消除）

### 后端新建文件

| # | 文件 | 操作 | 内容 |
|---|------|------|------|
| 1 | `atlhyper_master_v2/model/node_metrics.go` | **新建** | `NodeMetricsSnapshotResponse`, `CPUResponse`, `MemoryResponse`, `DiskResponse`, `NetworkResponse` 等，全部 camelCase JSON tag，对齐前端 `types/node-metrics.ts` |
| 2 | `atlhyper_master_v2/model/convert/node_metrics.go` | **新建** | `NodeMetricsSnapshot()`, `NodeMetricsSnapshots()`, `MetricsDataPoints()` 转换函数 + 单元测试 |

### 后端修改文件

| # | 文件 | 当前行为 | 修改内容 |
|---|------|---------|---------|
| 3 | `atlhyper_master_v2/gateway/handler/node_metrics.go` (259行) | `List()` 直接返回 `model_v2.NodeMetricsSnapshot`；`getDetail()` 直接返回 `model_v2.NodeMetricsSnapshot`；`getHistory()` 直接返回 `[]model_v2.MetricsDataPoint` | 3 处返回点加 `convert.*()` 调用：`List()` 中 summary + nodes 转换，`getDetail()` 单个转换，`getHistory()` 历史数据点转换 |

### 前端修改文件

| # | 文件 | 当前行为 | 修改内容 |
|---|------|---------|---------|
| 4 | `atlhyper_web/src/api/node-metrics.ts` | 16 个 transform 函数（transformSummary/CPU/Memory/Disk/Network/Sensor/Temperature/Process/PSI/TCP/System/VMStat/NTP/Softnet/Snapshot/DataPoint），~200 行 | 删除全部 16 个 transform 函数，API 直接返回类型化数据 |
| 5 | `atlhyper_web/src/app/overview/utils.ts` | transformOverview 中有 NodeMetrics 相关转换（~84 行） | 删除 NodeMetrics 相关转换代码 |

---

## Phase 2: Overview（~81 行前端代码消除）

### 后端新建文件

| # | 文件 | 操作 | 内容 |
|---|------|------|------|
| 6 | `atlhyper_master_v2/model/overview.go` | **新建** | `OverviewResponse`（camelCase），含 cards/workloads/alerts/nodes 子结构 |
| 7 | `atlhyper_master_v2/model/convert/overview.go` | **新建** | `Overview()` 转换函数：`model_v2.ClusterOverview` -> `model.OverviewResponse` |

### 后端修改文件

| # | 文件 | 当前行为 | 修改内容 |
|---|------|---------|---------|
| 8 | `atlhyper_master_v2/gateway/handler/overview.go` (54行) | `Get()` 直接返回 `service.GetOverview()` 结果（`*model_v2.ClusterOverview`） | 加 `convert.Overview()` 调用后返回 |

### 前端修改文件

| # | 文件 | 当前行为 | 修改内容 |
|---|------|---------|---------|
| 9 | `atlhyper_web/src/api/overview.ts` | `transformResponse()`（~81 行 snake_case -> camelCase） | 删除 `transformResponse()`，API 直接返回 |
| 10 | `atlhyper_web/src/types/overview.ts` (203行) | 类型定义已经是 camelCase | 确认与新 `model.OverviewResponse` 字段一一对应，必要时微调 |

---

## Phase 3: K8s 资源扁平化（9 个资源类型）

将嵌套的 `model_v2.Pod` (summary/spec/status) 转为扁平的 `model.PodItemResponse`。

### 后端新建文件（每资源 2 文件 x 9 = 18 文件）

| # | 文件 | 操作 | 内容 |
|---|------|------|------|
| 11 | `model/pod.go` | **新建** | `PodItemResponse`（列表）+ `PodDetailResponse`（详情，含容器/卷/网络） |
| 12 | `model/convert/pod.go` | **新建** | `PodItem()`, `PodItems()`, `PodDetail()` |
| 13 | `model/node.go` | **新建** | `NodeItemResponse` + `NodeDetailResponse`（含容量/条件/污点） |
| 14 | `model/convert/node.go` | **新建** | `NodeItem()`, `NodeItems()`, `NodeDetail()` |
| 15 | `model/deployment.go` | **新建** | `DeploymentItemResponse` + `DeploymentDetailResponse`（含 Spec/条件/ReplicaSet） |
| 16 | `model/convert/deployment.go` | **新建** | `DeploymentItem()`, `DeploymentItems()`, `DeploymentDetail()` |
| 17 | `model/statefulset.go` | **新建** | `StatefulSetDetailResponse` |
| 18 | `model/convert/statefulset.go` | **新建** | `StatefulSetDetail()` |
| 19 | `model/daemonset.go` | **新建** | `DaemonSetDetailResponse` |
| 20 | `model/convert/daemonset.go` | **新建** | `DaemonSetDetail()` |
| 21 | `model/service.go` | **新建** | `ServiceItemResponse` + `ServiceDetailResponse`（含端口/端点/网络配置） |
| 22 | `model/convert/service.go` | **新建** | `ServiceItem()`, `ServiceItems()`, `ServiceDetail()` |
| 23 | `model/namespace.go` | **新建** | `NamespaceItemResponse` + `NamespaceDetailResponse`（含配额/限制） |
| 24 | `model/convert/namespace.go` | **新建** | `NamespaceItem()`, `NamespaceItems()`, `NamespaceDetail()` |
| 25 | `model/ingress.go` | **新建** | `IngressItemResponse` + `IngressDetailResponse` |
| 26 | `model/convert/ingress.go` | **新建** | `IngressItem()`, `IngressItems()`, `IngressDetail()` |
| 27 | `model/event.go` | **新建** | `EventItemResponse` |
| 28 | `model/convert/event.go` | **新建** | `EventItem()`, `EventItems()` |

### 后端新建文件 — Cluster

| # | 文件 | 操作 | 内容 |
|---|------|------|------|
| 29 | `model/cluster.go` | **新建** | `ClusterInfoResponse`（camelCase）+ `ClusterDetailResponse` |
| 30 | `model/convert/cluster.go` | **新建** | `ClusterInfo()`, `ClusterInfos()`, `ClusterDetail()` |

### 后端修改文件 — Gateway Handler（10 个文件）

| # | 文件 | 行数 | 当前返回类型 | 修改内容 |
|---|------|------|-------------|---------|
| 31 | `gateway/handler/pod.go` | 114 | `map{data: []model_v2.Pod}` | `List()`: data 改为 `convert.PodItems(pods)`；`Get()`: data 改为 `convert.PodDetail(pod)` |
| 32 | `gateway/handler/node.go` | 89 | `map{data: []model_v2.Node}` | `List()`: `convert.NodeItems()`；`Get()`: `convert.NodeDetail()` |
| 33 | `gateway/handler/deployment.go` | 93 | `map{data: []model_v2.Deployment}` | `List()`: `convert.DeploymentItems()`；`Get()`: `convert.DeploymentDetail()` |
| 34 | `gateway/handler/statefulset.go` | ~80 | `map{data: []model_v2.StatefulSet}` | `List()`: `convert.StatefulSetItems()`；`Get()`: `convert.StatefulSetDetail()` |
| 35 | `gateway/handler/daemonset.go` | ~80 | `map{data: []model_v2.DaemonSet}` | `List()`: `convert.DaemonSetItems()`；`Get()`: `convert.DaemonSetDetail()` |
| 36 | `gateway/handler/service.go` | 93 | `map{data: []model_v2.Service}` | `List()`: `convert.ServiceItems()`；`Get()`: `convert.ServiceDetail()` |
| 37 | `gateway/handler/namespace.go` | 88 | `map{data: []model_v2.Namespace}` | `List()`: `convert.NamespaceItems()`；`Get()`: `convert.NamespaceDetail()` |
| 38 | `gateway/handler/ingress.go` | 92 | `map{data: []model_v2.Ingress}` | `List()`: `convert.IngressItems()`；`Get()`: `convert.IngressDetail()` |
| 39 | `gateway/handler/event.go` | 213 | `map{events: []model_v2.Event}` + DB 事件 | `listFromQuery()`: `convert.EventItems()`；`listFromDatabase()`: DB 事件也包装为 `EventItemResponse`；`ListByResource()`: 同 |
| 40 | `gateway/handler/cluster.go` | 69 | `map{clusters: []model_v2.ClusterInfo}` + `*model_v2.ClusterDetail` | `List()`: `convert.ClusterInfos()`；`Get()`: `convert.ClusterDetail()` |

### 前端修改文件 — API（8 个文件删 transform）

| # | 文件 | 删除的函数 | 约行数 |
|---|------|-----------|--------|
| 41 | `api/pod.ts` | `transformPod()`, `transformPodToDetail()`, `transformPodListToOverview()` | ~160 |
| 42 | `api/node.ts` | `parseMemoryToGiB()`, `parseCPUCores()`, `transformNodeItem()`, `transformToNodeDetail()`, `transformToNodeOverview()` | ~172 |
| 43 | `api/deployment.ts` | `transformDeploymentItem()`, `transformToDeploymentDetail()`, `transformToDeploymentOverview()` | ~140 |
| 44 | `api/service.ts` | `transformServiceItem()`, `transformToServiceDetail()`, `transformToServiceOverview()` | ~121 |
| 45 | `api/namespace.ts` | `transformNamespaceItem()`, `transformNamespaceToDetail()`, `transformToNamespaceOverview()` | ~95 |
| 46 | `api/ingress.ts` | `hasTLS()`, `expandIngressToRows()`, `transformIngressDetail()`, `transformToIngressOverview()` | ~127 |
| 47 | `api/event.ts` | `transformEventItem()`, `transformToEventOverview()` | ~60 |
| 48 | `api/workload.ts` | `transformStatefulSetDetail()`, `transformDaemonSetDetail()` | ~88 |

**Phase 3 总计消除前端代码**: ~963 行

### 前端修改文件 — Types

| # | 文件 | 修改内容 |
|---|------|---------|
| 49 | `types/cluster.ts` (978行) | 全面审查：(1) `ClusterInfo` 从 snake_case (`cluster_id`, `last_seen`) 改为 camelCase (`clusterId`, `lastSeen`)；(2) 确认 Pod/Node/Deployment 等类型与后端新的扁平 Response 类型字段一一对应；(3) 删除不再需要的嵌套类型（如 `DeploymentSpec`, `DeploymentTemplate` 等中间结构） |

---

## Phase 4: SLO/Mesh camelCase 统一

### 4a: 现有 model JSON tags 改 camelCase

| # | 文件 | 修改内容 |
|---|------|---------|
| 50 | `atlhyper_master_v2/model/slo.go` | 所有 snake_case JSON tag 改 camelCase：`error_budget_remaining` -> `errorBudgetRemaining`, `p95_latency` -> `p95Latency`, `requests_per_sec` -> `requestsPerSec`, `total_requests` -> `totalRequests`, `p50_latency_ms` -> `p50LatencyMs` 等 |
| 51 | `atlhyper_master_v2/model/command.go` | 检查并统一 JSON tag 为 camelCase（如 `command_id` -> `commandId`） |

**注意**：`model/slo.go` 的类型被以下位置构造和使用，但由于 **Go struct 字段名不变**（只改 JSON tag），这些文件的 **Go 代码无需修改**：
- `service/query/slo.go` — 构造 `model.ServiceMeshTopologyResponse`, `model.ServiceDetailResponse`
- `gateway/handler/slo.go` — 构造 `model.DomainSLO`, `model.SLODomainsResponse` 等
- `gateway/handler/slo_latency.go` — 构造 `model.LatencyDistributionResponse`
- `gateway/handler/slo_mesh.go` — 直接返回 service 层结果

### 后端修改文件 — Handler map key 修正

| # | 文件 | 修改内容 |
|---|------|---------|
| 52 | `gateway/handler/command.go` (221行) | `Create()` 返回的 `"command_id"` -> `"commandId"`；`ListHistory()` 内部 `CommandHistoryResponse` 检查 JSON tag |

### 前端修改文件

| # | 文件 | 修改内容 |
|---|------|---------|
| 53 | `types/slo.ts` (213行) | 所有 snake_case 属性改 camelCase：`p95_latency` -> `p95Latency`, `error_rate` -> `errorRate`, `requests_per_sec` -> `requestsPerSec`, `error_budget_remaining` -> `errorBudgetRemaining`, `total_requests` -> `totalRequests` 等 |
| 54 | `types/mesh.ts` (81行) | 所有 snake_case 属性改 camelCase：`avg_latency` -> `avgLatency`, `p50_latency` -> `p50Latency`, `error_rate` -> `errorRate`, `mtls_percent` -> `mtlsPercent`, `status_codes` -> `statusCodes` 等 |

**前端组件联动修改**：`types/slo.ts` 和 `types/mesh.ts` 的属性名变更后，所有引用这些属性的组件文件需要同步改名。需要全局搜索受影响的组件：
- SLO 相关组件（使用 `p95_latency`、`error_rate` 等属性的地方）
- Mesh 拓扑相关组件（使用 `avg_latency`、`mtls_percent` 等属性的地方）

### 4b: Error budget / 趋势预测后端化（可选，独立任务）

当前前端 `OverviewTab.tsx` 用错误公式 `100 - error_rate * 20` 算 error budget。
后端 `slo.go` handler 中用正确公式计算并在响应中新增 `errorBudget`、`trend`、`exhaustDate` 字段。

### 4c: 拓扑 BFS 过滤后端化（可选，独立任务）

当前前端 `DomainCard.tsx` 按 namespace 过滤拓扑。改为后端 `GetMeshTopology` 接收 domain 参数，BFS 过滤后返回连通子图。

---

## Phase 5: 废弃文件清理

| # | 文件 | 操作 |
|---|------|------|
| 55 | `atlhyper_web/src/api/metrics.ts` | **删除**（已被 node-metrics.ts 替代） |
| 56 | `atlhyper_web/src/api/config.ts` | **删除**（已整合到 settings） |
| 57 | `atlhyper_web/src/api/test.ts` | **删除**（测试用，不应存在） |
| 58 | `atlhyper_web/src/utils/safeData.ts` | **审查** — `safeTransform()` 等工具函数在删除所有 transform 后可能不再需要，确认无其他调用后删除 |

---

## 实施顺序

```
Phase 1 (NodeMetrics) -> Phase 2 (Overview) -> Phase 3 (K8s 资源) -> Phase 4 (SLO) -> Phase 5 (清理)
```

- 每 Phase 独立可验证，完成后 commit
- Phase 1~2 收益最大（snake_case 命名转换消除），优先执行
- Phase 3 工作量最大（9 个资源类型），可分批（每 2~3 个资源一个 commit）
- Phase 4 涉及 SLO/Mesh 类型的属性名全局变更，需全局搜索受影响的前端组件
- Phase 4b/4c 为可选独立任务，可单独排期

---

## 验证

每 Phase 完成后：
1. `go build ./...` — 后端编译通过
2. `go test ./atlhyper_master_v2/model/convert/...` — 转换函数单元测试通过
3. `cd atlhyper_web && npx next build` — 前端编译通过
4. 部署后验证：对应页面数据正常显示

---

## 关联问题（不在本次范围，记录备查）

### Handler 分层违规

审计发现 10 个 handler 跳过 Service 层直接访问 Database/Repository：

| Handler | 影响 API |
|---------|---------|
| node_metrics.go | `/api/v2/node-metrics` |
| slo.go | `/api/v2/slo/domains/*` |
| slo_latency.go | `/api/v2/slo/domains/latency` |
| event.go (history) | `/api/v2/events?source=history` |
| user.go | `/api/v2/user/*` |
| audit.go | `/api/v2/audit/logs` |
| settings.go | `/api/v2/settings/ai` |
| ai_provider.go | `/api/v2/ai/providers/*` |
| notify.go | `/api/v2/notify/channels/*` |
| command.go (history) | `/api/v2/commands/history` |

这些违规需要单独修复（Service 层补齐），但不阻塞本次"大后端小前端"改造。本次改造在 convert 层做转换即可，后续再整理分层。
