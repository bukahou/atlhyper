# AIOps 事件证据补全 — 设计文档

> 状态：设计已定（2026-10-14 用户裁定「1、2、3 一起做」），待实施
> 任务追踪：`docs/tasks/active/tracker.md`「AIOps 事件证据补全」

## 背景

一次真实事件暴露了 AI 分析的证据断层。某 Pod 在 18:21 进入 warning，18:27 升级为 incident（风险 56），18:39 进入 recovery。AI 后台分析的结论是「K8s 事件仅保留约 1 小时，数据不足」，处置建议是去 `kubectl describe` 里找 event。

实际上，master 的 `cluster_events` 表里**有直接证据**：该 Pod 在 18:24 有一条 `Unhealthy: Readiness probe failed ... context deadline exceeded`。同一时刻，另一个应用的两个 Pod 也出现了同样的事件。两个应用唯一明确的共同依赖是同一个外部数据库。

逐项核对后，问题出在三处：

| # | 缺口 | 现状 |
|---|------|------|
| 1 | 持久化的 K8s event 没进入 AI 上下文 | 后台分析（`Summarize`）只拼一次 prompt，上下文里没有 event。深度分析和对话里的 `query_cluster get_events` 走 Agent 实时查 API Server，只能看到约 1 小时 |
| 2 | 事件时间线不记录「是什么信号触发的」 | 时间线只有「状态变更 → incident（风险 56）」。触发它的异常（`critical_event` / 基线偏离 / 容器异常……）在 `engine.anomalyCache` 里，用完就丢 |
| 3 | 上下文缺三样关键事实 | ① **真正的异常时长**：recovery 要观察 48h 才关闭，`DurationS` 一直累计到当前时刻，AI 把 18 分钟说成「约 2.4 小时」② **同时段其他事件**：只给本事件实体，AI 看不到其他应用同时出事 ③ **Pod 所属 Deployment**：AI 把 ReplicaSet 名当成 Deployment 名 |

## 目标

- 后台分析与深度分析能直接引用 atlhyper 自己持久化的 K8s event，不再以「event 已过期」为由给出「数据不足」
- 时间线上每次升级都能看出是哪个信号触发的，人和 AI 都能读到
- 给 AI 的事实里：异常时长与观察期分开；同时段其他事件可见；Pod 的归属写准确

**非目标**（不在本设计内，另行决策）：
- 检测灵敏度（单次探针超时即升级为 incident）
- 前端「持续时间」显示口径
- `/api/v2/events?source=history` 忽略 `namespace` 参数的 bug

## 核心架构

```
                      ┌────────────── 现有 ──────────────┐
Agent 快照 → engine ─┤ anomalies → scorer → statemachine → incident.Store → 时间线
                      └──────────────────────────────────┘
                              │ 功能二：异常 → RiskSignal 挂到 EntityRisk 上
                              ▼
                     incident.Store 在状态变更时写 anomaly_detected 时间线

Enricher（后台 / 深度两条路径共用一个组装函数）
  assembleIncidentContext(incident)
    ├─ 现有：summary / entities / timeline / historical / OTel / PodStates
    ├─ 功能一：EventHistoryReader → 事件窗口内的持久化 event
    ├─ 功能三①：ActiveWindow(timeline) → 异常时长 + 观察期
    ├─ 功能三②：incidentRepo.List(同集群、开始时间 ±15min) → 同时段事件
    └─ 功能三③：Pod → ReplicaSet → Deployment 归属（快照内解析）
```

**依赖方向**：
- enricher 只依赖一个最小接口 `EventHistoryReader`，由 master.go 注入 `db.Event`，沿用现有 `SetStore` 的注入方式。
- prompts 包仍只接收纯文本，不依赖 aiops / database。

**两条路径统一**：
- 现状：后台分析走 `buildPromptWithTruncation`，深度分析（`runAnalysis`）只调 `BuildIncidentContext(…, nil)`，后者没有历史、OTel、Pod 状态。
- 本设计抽出 `assembleIncidentContext`，两条路径共用，深度分析也一并获得新证据。

---

## 功能一：持久化 K8s event 进入 AI 上下文

### 用户故事

事件发生 1 小时后才看分析报告时，AI 仍然能说出「18:24 该 Pod readiness 探针超时（累计 N 次）」，而不是「数据不足」。

### 数据来源

复用现有 `EventRepository.ListByInvolvedResource(ctx, clusterID, kind, namespace, name)`，不新增 SQL。

`cluster_events` 的语义（影响如何呈现给 AI）：
- 同一资源的同一原因（DedupKey）只存一行；
- `FirstTimestamp` 是首次发生，`LastTimestamp` 是最近一次，`Count` 是**自首次以来的累计次数**，不是窗口内次数；
- 只存 Warning。

⇒ 呈现时必须写清「累计 N 次，首次 X，最近 Y」，⛔ 不得写成「窗口内 N 次」。

### 规则

| 项 | 规则 |
|----|------|
| 查询对象 | `pod` 实体 → `kind=Pod`；`node` 实体 → `kind=Node`；`service` / `ingress` 实体不查（event 不挂在这些对象上） |
| 时间窗 | `[StartedAt − 15min, 异常结束 + 5min]`。异常结束 = 功能三①的 recovery 时刻，仍在异常中则取 now |
| 命中条件 | 该行的 `[FirstTimestamp, LastTimestamp]` 与时间窗有交集 |
| 排序 / 上限 | 按 `LastTimestamp` 倒序，最多 20 行（常量 `MaxEventEntries`）；message 截断 200 字符 |
| 读失败 | 记 Warn 日志，该节写「（读取失败）」，不中断分析 |

### 呈现给 AI 的格式

```
## K8s 事件（atlhyper 持久化，同资源同原因合并为一行）
- [Pod ns-a/api-7d9f-x2k] Unhealthy（累计 6 次，首次 10-09 13:45，最近 10-14 18:24）
  Readiness probe failed: Get "http://…/healthz": context deadline exceeded
```

窗口内无事件时写「事件窗口内无 Warning 事件」。这是明确的事实，和「没查」区分开。

### 对话工具：`get_event_history`

- 在 `query_cluster` 的 action 枚举中新增 `get_event_history`。
- 由 **master 侧直接读库**，不下发 Agent。
- 参数：
  - 复用现有 `namespace` / `involved_kind` / `involved_name`；
  - 新增 `since_minutes`（默认 180，上限 10080）。
- 工具说明里写明「`get_events` 只能看到约 1 小时；更早的事件用 `get_event_history`」。
- 权限：AI 对话本身要求 Viewer 及以上，event 历史是 0 级可见数据，不放宽任何权限。

### 后端实现

| 文件 | 职责 |
|------|------|
| `aiops/enricher/event_context.go` [新增] | `EventHistoryReader` 接口；`buildEventContext(ctx, reader, clusterID, entities, window) string` |
| `aiops/enricher/enricher.go` [修改] | `SetEventSource(EventHistoryReader)`；`assembleIncidentContext` 调用 |
| `master.go` [修改] | `aiopsEnricher.SetEventSource(db.Event)` |
| `ai/prompts/background.go` [修改] | `IncidentPromptContext.K8sEvents`，渲染「K8s 事件」节 |
| `ai/prompts/analysis.go` [修改] | 深度分析 user prompt 同样渲染该节 |
| `ai/prompts/tools.go` / `ai/tool.go` [修改] | `get_event_history` action：master 侧读库并格式化 |
| `ai/factory.go` / `ai/service.go` [修改] | AI Service 注入 `EventHistoryReader`（同一最小接口） |

---

## 功能二：时间线记录触发信号

### 用户故事

页面时间线上，「状态变更 → incident（风险 56）」的同一时刻多出一行「异常检测：K8s 事件 Unhealthy — Readiness probe failed…」。不用翻日志就知道是什么把它推上去的。

### 设计

**① `AnomalyResult` 增加 `Detail string`（`json:"detail,omitempty"`）**，由各个确定性提取器在生成时填写人可读的证据：

| MetricName | Detail 示例 |
|---|---|
| `critical_event` | `Unhealthy: Readiness probe failed: …`（取该 Pod 窗口内最近一条 critical event，截断 160 字符） |
| `container_anomaly` | `容器 app: CrashLoopBackOff` |
| `deployment_impact` | `Deployment 不可用 2/3` |
| `memory_pressure` 等节点压力 | `Node condition MemoryPressure=True` |
| OTel 确定性（`apm_high_error_rate` 等） | `错误率 12.5%` / `P99 2300ms` / `ERROR 日志 48 条` |

统计基线异常不填 Detail，渲染时由字段生成：`cpu_usage 当前 92.1，基线 41.3（偏离 4.2σ）`。

**② `EntityRisk` 增加 `Signals []RiskSignal`（`json:"signals,omitempty"`）**

```go
// RiskSignal 推高该实体本地风险的异常信号（按 Score 倒序，最多 3 条）
type RiskSignal struct {
    Metric string  `json:"metric"`
    Score  float64 `json:"score"`
    Detail string  `json:"detail"` // 已渲染好的人可读文本
}
```

- `Scorer.Calculate` 已持有本轮全部 anomalies，按实体分组后，在 `buildEntityRisks` 里挂上。
- 风险接口（`/aiops/risk/entities`、`/aiops/risk/entity`）会随之多出 `signals` 字段，属于纯增量，不影响现有前端。

**③ `incident.Store` 在 Create / UpdateState（升级）/ IncrementRecurrence 时追加一条时间线**
- `eventType = aiops.TimelineAnomalyDetected`（常量与前端图标、i18n 已存在，前端零改动）；
- `detail` = 信号逐条以「；」连接；
- `RLocal == 0`（本地无异常、风险全部来自传播）时写「无本地异常信号（风险由依赖传播）」。

**④ recovery 改用 `aiops.TimelineRecoveryStarted` 写入**（现为 `state_change`）。前端已有该类型，功能三①据此结构化识别，不解析文案。

### 数据对照

| 前端 TS（`IncidentTimeline`） | 后端 Go（`AIOpsIncidentTimeline`） | DB `aiops_incident_timeline` |
|---|---|---|
| `eventType: "anomaly_detected"` | `EventType` | `event_type` |
| `detail` | `Detail` | `detail` |

表结构不变。

### 后端实现

| 文件 | 职责 |
|------|------|
| `aiops/types.go` [修改] | `AnomalyResult.Detail`、`RiskSignal`、`EntityRisk.Signals` |
| `aiops/baseline/extractor.go` / `extractor_enhanced.go` [修改] | 各确定性提取器填 Detail |
| `aiops/risk/scorer.go` [修改] | 按实体挑 Top 3 信号；统计异常的 Detail 渲染 |
| `aiops/incident/store.go` [修改] | 追加 `anomaly_detected` 时间线；recovery 用 `recovery_started` |

---

## 功能三：上下文补三样事实

### ① 异常时长与观察期分开

`incident.ActiveWindow(timeline) (start, end time.Time, recovered bool)`：
- `start` = 事件 `StartedAt`；
- `end` = **最后一次**复发之后的首个 `recovery_started`；
- 兼容旧数据：同时识别 `state_change` 且 detail 以「状态变更 → recovery」开头的条目。旧条目只存在于本设计上线前的库里。

`buildSummary` 改为：

```
异常持续: 18m（18:21:34 → 18:39:14）
当前: 恢复观察期（距开始 2h27m；48h 内无复发才关闭）
```

仍在异常中则写「异常持续: 12m（进行中）」。⛔ 不再只输出一个累计到当前时刻的「持续」。

### ② 同时段其他事件

- 查询：`incidentRepo.List({ClusterID, From: StartedAt−15min, To: StartedAt+15min, Limit: 20})`，排除自身；
- 呈现：根因实体、开始时间、峰值风险、状态，每行一条；
- 同一时段若出现 ≥2 个**不同命名空间**的事件，在该节末尾附一行**事实**提示：「同时段 N 个不同命名空间出现事件」。

system prompt 新增一条规则：

> 同时段多个不同工作负载出现同类信号时，须考虑共同依赖（数据库、网络、节点），结论必须引用上下文中的具体证据；无证据时只能列为待验证假设。

### ③ Pod 所属 Deployment

- 在 `buildPodStateContext` 里解析：Pod 的 `OwnerKind/OwnerName` → ReplicaSet → 该 RS 的 `OwnerKind/OwnerName`（`model_v3.CommonMeta` 已有）；
- 输出：`所属: Deployment ns/name（经 ReplicaSet rs-name）`；
- RS 不在快照里：`所属: ReplicaSet rs-name（上级未知）`。⛔ 不靠去掉 hash 后缀猜 Deployment 名。

system prompt 新增一条规则：

> 资源名称只能使用上下文中出现的完整名称，不得自行推断或改写。

### 后端实现

| 文件 | 职责 |
|------|------|
| `aiops/incident/timeline.go` [修改] | `ActiveWindow` |
| `aiops/enricher/context_builder.go` [修改] | `buildSummary` 用 ActiveWindow；`buildConcurrent` |
| `aiops/enricher/pod_state.go` [修改] | 归属解析 |
| `aiops/enricher/enricher.go` [修改] | 抽出 `assembleIncidentContext`，后台与深度两条路径共用；截断顺序加入 events / concurrent |
| `ai/prompts/background.go` / `analysis.go` [修改] | `ConcurrentIncidents` 字段与渲染；两条新规则 |

### Prompt 长度

- 新增两节都有条数上限：event 20 行、同时段 20 行。
- 截断顺序改为：同时段 → historical → K8s 事件 → timeline → entities。理由：K8s 事件是直接证据，比历史模式更有价值，截断时应保留得更久。

---

## 实施阶段

单个 PR，按功能分 commit，每个 commit 都能独立通过测试：

| 步骤 | 内容 | 可验证点 |
|------|------|----------|
| 1 | 功能二（信号 + 时间线） | dev 上新事件的时间线出现「异常检测」行；`/aiops/risk/entities` 带 `signals` |
| 2 | 功能三①③（时长、归属）+ `assembleIncidentContext` 抽取 | 单测；深度分析 prompt 含 Pod 状态 |
| 3 | 功能一（event 上下文 + 对话工具）+ 功能三② | dev 上对一个 1 小时前的事件点「重新分析」，报告引用持久化 event |

**TDD**：每步先写测试（红）再实现（绿）。

**验收**：在 dev 上找一个已超过 1 小时、库中有对应 event 的事件，重新分析，核对报告：
- ① 引用了 event 原文与时间；
- ② 异常时长正确，不是累计到当前时刻；
- ③ 若有同时段事件，有列出；
- ④ Deployment 名正确。

## 测试清单

| 测试 | 断言 |
|------|------|
| `TestBuildEventContext_WindowOverlap` | 与窗口有交集的行入选；完全在窗口外的排除；按 Last 倒序；上限 20 |
| `TestBuildEventContext_CountSemantics` | 输出含「累计 N 次，首次 …，最近 …」 |
| `TestBuildEventContext_EmptyAndError` | 无事件 →「窗口内无 Warning 事件」；读失败 →「（读取失败）」且不 panic |
| `TestBuildEventContext_EntityKinds` | pod→Pod、node→Node；service 不查 |
| `TestExtractEventAnomalies_Detail` | critical_event 的 Detail 取最近一条，含 reason 与 message |
| `TestScorer_SignalsTop3` | 按 Score 倒序，最多 3 条；统计异常 Detail 含当前值、基线、σ |
| `TestStore_AppendsSignalTimeline` | Create / 升级 / 复发各追加一条 `anomaly_detected`；RLocal=0 写传播说明 |
| `TestStore_RecoveryUsesRecoveryStarted` | recovery 的 eventType 为 `recovery_started` |
| `TestActiveWindow` | 正常、复发后再恢复（取最后一段）、仍在异常、旧 `state_change` 数据兼容 |
| `TestBuildSummary_SeparatesActiveAndObservation` | 输出异常时长与观察期两行；不含累计时长当作持续的写法 |
| `TestBuildConcurrent` | ±15min、排除自身、不同命名空间计数提示 |
| `TestPodOwner_Resolution` | Pod→RS→Deployment；RS 缺失时写「上级未知」，不猜名 |
| `TestAssembleIncidentContext_BothPathsShareEvidence` | 后台与深度两条路径的上下文都含新增各节 |
| `TestToolGetEventHistory` | 参数映射、since 上限、master 侧读库不下发 Agent |

## 文件变更清单

```
atlhyper_master_v2/
├── master.go                                   [修改] 注入 EventHistoryReader（enricher + ai）
├── aiops/
│   ├── types.go                                [修改] AnomalyResult.Detail / RiskSignal / EntityRisk.Signals
│   ├── baseline/extractor.go                   [修改] 确定性提取器填 Detail
│   ├── baseline/extractor_enhanced.go          [修改] OTel 确定性提取器填 Detail
│   ├── risk/scorer.go                          [修改] 信号 Top 3
│   ├── incident/store.go                       [修改] anomaly_detected / recovery_started 时间线
│   ├── incident/timeline.go                    [修改] ActiveWindow
│   └── enricher/
│       ├── event_context.go                    [新增] EventHistoryReader + buildEventContext
│       ├── context_builder.go                  [修改] 时长 / 同时段
│       ├── pod_state.go                        [修改] 归属解析
│       └── enricher.go                         [修改] assembleIncidentContext + SetEventSource
└── ai/
    ├── prompts/background.go                   [修改] 新字段、新节、两条规则
    ├── prompts/analysis.go                     [修改] 同上（深度分析）
    ├── prompts/tools.go                        [修改] get_event_history
    ├── tool.go                                 [修改] master 侧执行 get_event_history
    ├── factory.go / service.go                 [修改] 注入 EventHistoryReader
atlhyper_web/                                   无改动（anomaly_detected / recovery_started 已有图标与文案）
```

## 附录：约束与风险

- **event 的 Count 是累计值**：一行可能跨越多天（首次在 10-09，最近在 10-14）。只有用「首次 / 最近 / 累计」的写法才不会误导 AI。
- **event 保留期**：由现有清理任务（`DeleteBefore` / `DeleteOldest`）决定，本设计不改。事件比 event 保留期还老时，该节写「窗口内无 Warning 事件」，可能是真的没有，也可能是已清理。这个歧义暂时接受，需要时再在文案里区分。
- **dev 库已持久化**（2026-10-13 起使用 NFS）：本设计不改表结构，不涉及删库。
- **信号字段进入风险 API**：属于纯增量字段，前端不消费也不影响。
