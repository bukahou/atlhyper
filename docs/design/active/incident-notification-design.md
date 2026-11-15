# 事件通知：从检测到通知

> 状态：执行中。「裁决记录」一节的各项已于 2026-11-15 由用户裁定。

## 背景

AIOps 引擎已经能把异常收敛成事件（incident）并跟踪它的状态，但事件只出现在页面上，不会通知到人。
通知模块（notifier）现在只有两个来源：Agent 心跳，和 K8s Warning 事件（每条新事件发一条消息）。
两边互不相通：

| 现状 | 后果 |
|------|------|
| 事件的状态转换只触发 AI 后台分析，不发通知 | 检测到了，没人知道 |
| K8s Warning 事件逐条发送，不分组、不按严重度筛选 | 同一个问题既报事件告警、又会进入 AIOps |
| 没有统一的告警对象（没有标识、状态、开始与结束时间） | 做不了去重、恢复通知、重复提醒 |
| 发送失败只记日志，不重试，没有送达记录 | 渠道配错或对端故障时，通知无声无息地丢了 |
| 所有已启用的渠道收到同样的内容；每种渠道类型只能配一条 | 不能按集群或严重度分流 |
| 没有静默 | 维护窗口里只能关掉整个渠道 |

事件本身也有四处会直接影响通知质量：

| 现状 | 后果 |
|------|------|
| 每个实体各开各的事件，Pod 重建后换了名字就是新对象 | 同一个工作负载的几个 Pod 同时出问题 = 几个事件；历史随 Pod 名字断掉 |
| 只到「警告」就恢复的事件直接关闭，下次再触发是全新事件 | 同一个对象反复抖动 = 一长串互不相干的短事件，「反复」本身看不出来 |
| 事件关闭时严重度被清空 | 历史事件没有严重度，按严重度的统计与分流无从谈起 |
| 「警告」「事件中」两档都不等于「需要人处理」 | 选哪一档作为通知时机都不干净 |

## 目标

- 事件以「主体」为单位：同一工作负载的 Pod 归到一个事件；反复触发归到同一个事件并计数
- 事件的关键变化通知到人：进入「事件中」、严重度升高、复发、恢复；只到「警告」的，反复到一定次数才通知
- 通知管道具备主流告警系统（Alertmanager 模型）的基本能力：分组、去重、抑制、重复提醒、恢复通知、失败重试、送达记录
- 多个接收人、按集群与严重度分流的路由规则、静默

**不在范围内**

- SLO 违规告警（burn rate 接线）、OOM 走 Pod 状态：已另案冻结，本设计不触碰
- 按依赖图把一次故障的多个对象（服务、Pod、节点）归并成一个事件并选根因：另行立项
- 用户自定义阈值规则；新的渠道类型（钉钉、通用 Webhook 的发送器）

## 核心架构

```
AIOps                                  通知模块（独立协程，不阻塞快照处理）
─────                                  ──────────────────────────────────
实体风险 ─► 按主体聚合 ─► 状态机        告警来源                 统一告警对象        管道
                            │           ────────                 ────────────        ────
                            │ 事件变化   事件变化 ──────────┐                        1. 路由规则 → 选出接收人
                            └──────────► Agent 心跳 ────────┼─► Alert ─► 落库 ─►     2. 静默：命中的不发
                                         K8s Warning（旧）──┘                        3. 抑制：集群离线时压住该集群的其他告警
                                                                                     4. 分组：同一集群在等待窗口内合并成一条
                                                                                     5. 重复：仍在持续的，按间隔再提醒
                                                                                     6. 发送 → 失败重试 → 送达记录
```

三期一次发布，所以数据模型直接按最终形态实现（接收人表取代原来「每种类型一条」的渠道表），不做过渡形态。

## 功能零：事件按主体收敛（AIOps）

**用户故事**：一个 Deployment 的三个 Pod 同时探针失败，事件页只出现一个事件，写明是哪个工作负载、涉及哪几个 Pod；
这个工作负载当天第三次抖动时，还是同一个事件，复发次数是 2；事件关闭后仍然看得到它的严重度。

### 主体

状态机与事件以「主体」（subject）为单位，而不是单个实体。

| 实体 | 主体 |
|------|------|
| Pod，上级是 ReplicaSet 且该 ReplicaSet 属于某个 Deployment | `<ns>/deployment/<名字>` |
| Pod，上级是 StatefulSet / DaemonSet | `<ns>/statefulset/<名字>`、`<ns>/daemonset/<名字>` |
| Pod，上级是 Job 且该 Job 属于某个 CronJob | `<ns>/cronjob/<名字>`；Job 没有上级时 `<ns>/job/<名字>` |
| 其他 Pod（没有上级，或上级在快照里找不到）、service、node、ingress、logs | 实体自身的键 |

- 主体的风险 = 成员实体中最高的 `RFinal`；风险最高的成员称为「领头实体」
- 风险计算、基线、依赖图、风险页面都不变，仍以实体为单位；只有状态机与事件改用主体

```go
// aiops/types.go
// SubjectRisk 一个主体在本轮评估中的风险
type SubjectRisk struct {
    SubjectKey string
    RFinal     float64        // = Lead.RFinal
    Lead       *EntityRisk    // 风险最高的成员
    Members    []*EntityRisk  // RFinal > 0 的成员，按风险从高到低（含 Lead）
}

// aiops/subject/resolver.go
// ResolveSubjects 从快照得出「实体键 → 主体键」；不在返回值里的实体，主体就是它自己
func ResolveSubjects(snap *cluster.ClusterSnapshot) map[string]string
// AggregateBySubject 把实体风险按主体聚合
func AggregateBySubject(entityRisks map[string]*aiops.EntityRisk, subjects map[string]string) map[string]*aiops.SubjectRisk
```

### 状态机

| 转换 | 条件 | 说明 |
|------|------|------|
| 健康 → 警告 | 风险 > 0.2 持续 2 分钟 | 建事件（不变） |
| 警告 → 事件中 | 风险 > 0.5 持续 5 分钟，或集群风险 > 80 | 不变；首次发生时记下 `escalated_at` |
| 警告 → 恢复观察 | 风险 < 0.15 持续 5 分钟 | **改**：原来是直接关闭 |
| 事件中 → 恢复观察 | 风险 < 0.15 持续 10 分钟 | 不变 |
| 恢复观察 → 警告（复发） | 风险 > 0.2；升级过的事件立即生效，没升级过的需持续 2 分钟 | 复发次数 +1 |
| 恢复观察 → 关闭 | 观察期满且期间没有复发：升级过的事件 48 小时，没升级过的 6 小时 | **改**：观察期按是否升级过区分 |
| 任意 → 关闭（失联） | 主体 30 分钟没有被评估（对象已删除或 Agent 断连） | 不变；标记为「失联关闭」 |

状态机条目新增（都能在重启后从库里取回）：

| 字段 | 含义 | 重启后的来源 |
|------|------|-------------|
| `ReachedIncident bool` | 是否升级到过「事件中」 | `escalated_at` 非空 |
| `PeakRisk float64` | 事件至今的最高风险 | `peak_risk` |
| `OccurrenceTimes []int64` | 近 24 小时内每次「发生」的时刻（建事件 + 每次复发） | 时间线里的建事件与复发条目 |

### 严重度

- `severity` 的含义改为「峰值严重度」：只升不降，关闭时保留
- 处于警告 / 事件中时，本轮风险落入比已记录更高的一档，就回调 `severity_raised`：更新 `peak_risk`、`severity`，时间线加一条

### 事件变化的发布

五个回调改为一个：引擎把每次变化作为 `IncidentTransition` 发给订阅者。AI 后台分析与通知各自订阅。
回调在持有状态机锁、且位于快照处理主路径上被调用，订阅者只允许做「放进队列」这类不阻塞的事。

```go
// aiops/types.go
type TransitionKind string
const (
    TransitionCreated         TransitionKind = "created"          // 健康 → 警告
    TransitionEscalated       TransitionKind = "escalated"        // 警告 → 事件中
    TransitionSeverityRaised  TransitionKind = "severity_raised"  // 严重度升高一档以上
    TransitionRecoveryStarted TransitionKind = "recovery_started" // → 恢复观察
    TransitionRecurred        TransitionKind = "recurred"         // 恢复观察 → 警告
    TransitionClosed          TransitionKind = "closed"           // 观察期满
    TransitionClosedStale     TransitionKind = "closed_stale"     // 失联关闭
)

type IncidentTransition struct {
    Kind            TransitionKind
    IncidentID      string
    ClusterID       string
    SubjectKey      string
    RootCause       string      // 领头实体
    Members         []string    // 本轮有风险的成员实体键
    FromState       EntityState
    ToState         EntityState
    Severity        string      // 峰值严重度
    RFinal          float64
    PeakRisk        float64
    Recurrence      int
    Occurrences24h  int         // 近 24 小时内的发生次数（含本次）
    ReachedIncident bool
    At              time.Time
}

// aiops/interfaces.go（Engine）
// SubscribeTransitions 订阅事件变化。订阅者在快照处理主路径上被同步调用，不得阻塞
SubscribeTransitions(fn func(IncidentTransition))
```

### 数据对照

| API JSON（事件） | Go（`aiops.Incident`） | 表 `aiops_incidents` |
|---|---|---|
| `subject` | `Subject` | `subject TEXT NOT NULL` **[新增]** |
| `subjectType` / `subjectNamespace` / `subjectName` | 由 `Subject` 解析（集群级对象的命名空间为空串） | — |
| `rootCause` | `RootCause` | `root_cause TEXT`（领头实体，不变） |
| `severity` | `Severity` | `severity TEXT NOT NULL DEFAULT 'low'`（峰值，关闭时保留） |
| `escalatedAt` | `EscalatedAt *time.Time` | `escalated_at TEXT` **[新增]** |
| `closeReason` | `CloseReason` | `close_reason TEXT NOT NULL DEFAULT ''` **[新增]**：`''` / `recovered` / `stale` |
| `recurrence` | `Recurrence` | `recurrence INTEGER`（不变；现在警告级的反复也计入） |

事件实体表不变：领头实体 `role = root_cause`，其余成员 `role = affected`；状态转换时把本轮新出现的成员补进去。
按实体查历史（`ListByEntity`、事件模式）同时匹配 `subject` 与成员实体。

索引：`idx_aiops_incidents_subject (cluster_id, subject, started_at DESC)`。

### 实现

- `aiops/subject/` [新增]：主体解析与聚合
- `aiops/statemachine/`：以主体为键；新增观察期区分、峰值跟踪、发生次数；回调接口换成单个 `OnTransition`
- `aiops/incident/store.go`：按 `SubjectRisk` 建事件与更新；关闭不再清空严重度；写 `escalated_at`、`close_reason`
- `aiops/core/engine.go`：评估前按主体聚合；发布 `IncidentTransition`；重启恢复用 `subject`
- `aiops/enricher/`：改为订阅 `IncidentTransition`（`created` 与 `escalated` 对应原来的两种触发）
- `database/`：事件表三列；仓库方法 `UpdateState(id, state)`、`RaiseSeverity(id, severity, peakRisk)`、`MarkEscalated(id, at)`、`Resolve(id, at, reason)`
- 前端：事件列表与详情显示主体（类型 + 名称）、成员实体、复发次数、升级时间；状态筛选与文案覆盖「恢复观察」

## 统一告警对象

```go
// notifier/alert
type Alert struct {
    Fingerprint string            // 稳定标识：同一个问题始终是同一个值
    Source      Source            // incident / heartbeat / k8s_event
    ClusterID   string
    Severity    Severity          // low / medium / high / critical（与 AIOps 的四档一致）
    Status      Status            // firing / resolved
    Kind        Kind              // 最近一次变化：firing / escalated / recurred / flapping / resolved
    Title       string
    Summary     string
    Labels      map[string]string // namespace、subject、subjectType，供规则与静默匹配
    Details     map[string]string // 根因实体、成员、峰值风险、复发次数、事件号等，供模板使用
    StartsAt    time.Time
    EndsAt      *time.Time
    UpdatedAt   time.Time
}
```

| 来源 | 指纹 | 严重度 | 何时结束 |
|------|------|--------|---------|
| 事件 | `incident:<事件号>` | 事件的峰值严重度 | 见下表 |
| Agent 心跳 | `heartbeat:<集群>` | critical | Agent 恢复上报 |
| K8s Warning 事件（旧，默认关） | `k8s_event:<集群>:<Kind>/<ns>/<名字>:<Reason>` | low | 10 分钟内没有再出现则静默结束（不发恢复通知） |

### 事件 → 告警

| 事件的变化 | 告警动作 |
|-----------|---------|
| `escalated`（进入事件中） | 触发，发通知（`firing`） |
| `severity_raised`，且事件处于事件中 | 更新严重度，再发一次（`escalated`） |
| `recovery_started`，且告警在持续 | 结束告警，发恢复通知（`resolved`） |
| `recurred`，事件升级过 | 重新触发，发通知（`recurred`，带复发次数） |
| `recurred`，事件没升级过，近 24 小时发生次数 ≥ 阈值（默认 3） | 触发，发通知（`flapping`，带次数） |
| `created`、没升级过且次数不到阈值的 `recurred` | 不产生告警 |
| `closed` | 告警若仍在持续（反复类）则结束并发恢复通知 |
| `closed_stale` | 结束告警，**不发**恢复通知（并不是真的恢复了） |

### 分组、抑制、重复

- **路由**：告警按顺序匹配路由规则，第一条命中的规则决定接收人与时间参数；都不命中用默认策略
- **分组**：同一条规则、同一个集群的告警在「分组等待」（默认 30 秒）内合并成一条消息，列出每个受影响的主体；
  之后同组有新内容时，最短间隔「分组间隔」（默认 5 分钟）再发
- **抑制**：某集群的 Agent 离线告警在持续时，该集群的其他告警不发，标记 `suppressedBy = "cluster-offline"`
- **静默**：命中静默的告警照常落库与显示，标记 `silencedBy`，不发；静默到期时仍在持续的，按新告警发出
- **重复提醒**：告警持续未结束，每隔「重复间隔」（默认 4 小时）再提醒一次
- **去重**：同一个指纹、同一种变化，在重复间隔内只发一次

### 发送与送达记录

- 一条消息发给一个接收人 = 一个批次；批次里每条告警一条送达记录（表 `notify_deliveries`）
- 失败按 30 秒、2 分钟、10 分钟重试三次，仍失败记为「已放弃」；重试时按告警当前内容重新渲染
- 发送带超时（Slack 10 秒，SMTP 连接与整体各有超时）
- 没有任何可用接收人时，告警照常落库并记一条 `no-receiver` 的送达记录；配好接收人后，重复提醒会把仍在持续的告警补发出去
- 送达记录保留 30 天；已结束的告警保留 30 天

### 通知内容

- 标题：`[集群] 严重度 · 主体 · 一句话摘要`；分组时：`[集群] N 条告警（最高严重度）`
- 正文：变化（触发 / 升级 / 复发 / 反复 / 恢复）、主体、根因实体、成员数、峰值风险、持续时间、复发次数
- 页面链接：配置了 `MASTER_PUBLIC_URL` 时带上事件详情页的链接
- 邮件主题按 RFC 2047 编码

## 功能一：事件通知闭环（第 1 期）

**用户故事**：某个服务出问题并升级为事件，30 秒内 Slack / 邮件收到一条通知，写明是哪个集群的哪个主体、有多严重；
同一集群同时出问题的几个主体合并在一条里。问题持续 4 小时会再提醒。恢复后收到一条恢复通知。
在设置页能看到当前有哪些告警、每条通知发给了谁、成没成功。

**API**（权限均为 `notify.manage`，2 级；写操作审计）

```
GET /api/v2/notify/policy
→ { "message": "获取成功", "data": {
      "incidentAlertsEnabled": true,
      "minSeverity": "low",              // 低于此严重度的告警不通知
      "notifyResolved": true,
      "warningFlapThreshold": 3,         // 警告级事件 24 小时内发生几次才通知；0 = 不通知
      "groupWaitSeconds": 30,
      "groupIntervalSeconds": 300,
      "repeatIntervalSeconds": 14400,
      "k8sEventAlertsEnabled": false,    // 旧的逐条 K8s 事件告警
      "defaultReceiverIds": []           // 没有规则命中时发给谁；空 = 全部已启用的接收人
  } }

PUT /api/v2/notify/policy   （同上结构）→ 200 { "message": "已保存", "data": { ... } }
  校验：minSeverity ∈ low/medium/high/critical；warningFlapThreshold 0–20；
        groupWaitSeconds 0–600；groupIntervalSeconds 30–86400；repeatIntervalSeconds 300–604800；
        defaultReceiverIds 必须都存在

GET /api/v2/notify/alerts?status=firing|resolved&cluster_id=&limit=&offset=
→ { "message": "获取成功", "data": [
      { "fingerprint": "incident:inc-123", "source": "incident", "clusterId": "c1",
        "severity": "high", "status": "firing", "kind": "firing",
        "title": "...", "summary": "...",
        "subject": "shop/deployment/api", "incidentId": "inc-123",
        "startsAt": "...", "endsAt": "", "lastNotifiedAt": "...", "notifyCount": 2,
        "suppressedBy": "", "silencedBy": 0 }
  ], "total": 1 }

GET /api/v2/notify/deliveries?fingerprint=&status=&limit=&offset=
→ { "message": "获取成功", "data": [
      { "id": 1, "batchId": "b-...", "fingerprint": "incident:inc-123", "alertTitle": "...",
        "receiverId": 2, "receiverName": "值班频道", "receiverType": "slack",
        "kind": "firing",                // firing / escalated / recurred / flapping / resolved / repeat
        "status": "success",             // pending / success / failed / abandoned / no-receiver
        "attempts": 1, "lastError": "", "createdAt": "...", "updatedAt": "..." }
  ], "total": 1 }
```

**数据对照**

| API JSON（policy） | Go（`model.NotifyPolicy`） | 表 `notify_policy`（单行，`id = 1`） |
|---|---|---|
| `incidentAlertsEnabled` | `IncidentAlertsEnabled` | `incident_alerts_enabled INTEGER NOT NULL DEFAULT 1` |
| `minSeverity` | `MinSeverity` | `min_severity TEXT NOT NULL DEFAULT 'low'` |
| `notifyResolved` | `NotifyResolved` | `notify_resolved INTEGER NOT NULL DEFAULT 1` |
| `warningFlapThreshold` | `WarningFlapThreshold` | `warning_flap_threshold INTEGER NOT NULL DEFAULT 3` |
| `groupWaitSeconds` | `GroupWaitSeconds` | `group_wait_seconds INTEGER NOT NULL DEFAULT 30` |
| `groupIntervalSeconds` | `GroupIntervalSeconds` | `group_interval_seconds INTEGER NOT NULL DEFAULT 300` |
| `repeatIntervalSeconds` | `RepeatIntervalSeconds` | `repeat_interval_seconds INTEGER NOT NULL DEFAULT 14400` |
| `k8sEventAlertsEnabled` | `K8sEventAlertsEnabled` | `k8s_event_alerts_enabled INTEGER NOT NULL DEFAULT 0` |
| `defaultReceiverIds` | `DefaultReceiverIDs []int64` | `default_receiver_ids TEXT NOT NULL DEFAULT '[]'` |

| API JSON（alert） | Go（`model.NotifyAlert`） | 表 `alerts` |
|---|---|---|
| `fingerprint` | `Fingerprint` | `fingerprint TEXT PRIMARY KEY` |
| `source` / `clusterId` / `severity` / `status` / `kind` | 同名 | `source` / `cluster_id` / `severity` / `status` / `kind`（TEXT NOT NULL） |
| `title` / `summary` | 同名 | `title` / `summary` |
| `subject` / `incidentId` | 由 `Labels` / `Details` 取出 | `labels TEXT`（JSON）、`details TEXT`（JSON） |
| `startsAt` / `endsAt` | `StartsAt` / `EndsAt` | `starts_at TEXT NOT NULL` / `ends_at TEXT` |
| `lastNotifiedAt` / `notifyCount` | 同名 | `last_notified_at TEXT` / `notify_count INTEGER NOT NULL DEFAULT 0` |
| `suppressedBy` / `silencedBy` | 同名 | `suppressed_by TEXT NOT NULL DEFAULT ''` / `silenced_by INTEGER NOT NULL DEFAULT 0` |
| —（内部） | `LastNotifiedKind`、`UpdatedAt` | `last_notified_kind TEXT`、`updated_at TEXT NOT NULL` |

| API JSON（delivery） | Go（`model.NotifyDelivery`） | 表 `notify_deliveries` |
|---|---|---|
| `id` | `ID` | `id INTEGER PRIMARY KEY AUTOINCREMENT` |
| `batchId` | `BatchID` | `batch_id TEXT NOT NULL` |
| `fingerprint` / `alertTitle` | 同名 | `fingerprint TEXT NOT NULL` / `alert_title TEXT NOT NULL` |
| `receiverId` / `receiverName` / `receiverType` | 同名 | `receiver_id INTEGER` / `receiver_name TEXT` / `receiver_type TEXT`（接收人删除后记录仍可读） |
| `kind` / `status` / `attempts` / `lastError` | 同名 | `kind` / `status` / `attempts INTEGER` / `last_error TEXT` |
| —（内部） | `NextAttemptAt` | `next_attempt_at TEXT` |
| `createdAt` / `updatedAt` | 同名 | `created_at` / `updated_at` |

索引：`alerts(status, cluster_id)`；`notify_deliveries(created_at DESC)`、`notify_deliveries(fingerprint)`、`notify_deliveries(status, next_attempt_at)`。

**后端实现**

- `notifier/alert/`：`Alert`、严重度与状态、指纹
- `notifier/pipeline/`：入队（有界）、路由、静默、抑制、分组、重复、发送与重试；时钟可替换以便测试
- `notifier/source/`：`incident.go`（`IncidentTransition` → 告警）、`heartbeat.go`、`k8s_event.go`（两个旧触发器改为产出告警，经同一条管道）
- `notifier/template/`：告警消息的模板（单条 / 分组，Slack / 邮件）；旧的三组模板删除
- `notifier/channel/`：发送器按接收人创建；SMTP 加超时
- `database/`：`notify_policy`、`alerts`、`notify_deliveries` 三张表与仓库
- `config/`：`MASTER_PUBLIC_URL`（可空）
- `service/`、`gateway/handler/admin/`：上述接口

**前端实现**

- `app/settings/notif/`：页签「当前告警」「送达记录」「通知策略」
- `queries/notify.ts`、`api/notify.ts`：查询与写操作的工厂
- 无 `notify.manage` 时整页演示数据 + 横幅（沿用现状）

**边界**

- 快照处理不因通知阻塞：订阅者只入队；队列满时丢弃并计数、记日志
- Master 重启：持续中的告警从库里恢复，不重复发「触发」，重复提醒的计时接着算；`pending` / `failed` 的批次接着重试
- 渠道全部失败：三次重试后记「已放弃」，不影响下一条
- Agent 离线：只发一条离线告警，该集群其他告警被抑制；恢复上线后各自按状态继续
- `incidentAlertsEnabled = false`：事件不产生告警；心跳告警不受影响

## 功能二：接收人与路由规则（第 2 期）

**用户故事**：生产集群的高严重度告警发到值班频道和邮件，测试集群的只发到开发频道。
可以配多个 Slack 频道、多组邮件收件人；改完规则可以先「试匹配」看看会发给谁。

**API**

```
GET /api/v2/notify/receivers
→ { "message": "获取成功", "data": [
      { "id": 1, "name": "值班频道", "type": "slack", "enabled": true,
        "slack": { "webhookConfigured": true },
        "createdAt": "...", "updatedAt": "..." },
      { "id": 2, "name": "运维邮件", "type": "email", "enabled": true,
        "email": { "toAddresses": ["a@example.com"] },      // 发件服务器见「全局发件设置」
        "createdAt": "...", "updatedAt": "..." }
  ], "total": 2 }

POST /api/v2/notify/receivers
  { "name": "值班频道", "type": "slack", "enabled": true, "slack": { "webhookUrl": "https://..." } }
  { "name": "运维邮件", "type": "email", "enabled": true, "email": { "toAddresses": ["..."] } }
→ 201 { "message": "已创建", "data": { ...同列表项... } }        # 名称重复 → 409

PUT /api/v2/notify/receivers/{id}      （同创建；webhookUrl 留空 = 保持原值）→ 200
DELETE /api/v2/notify/receivers/{id}   → 200；仍被规则或默认策略引用 → 409，message 写明被谁引用
POST /api/v2/notify/receivers/{id}/test → 200 { "message": "测试消息已发送" }；发送失败 → 502，message 带原因

GET /api/v2/notify/rules
→ { "message": "获取成功", "data": [
      { "id": 1, "name": "生产高严重度", "position": 1, "enabled": true,
        "clusterIds": ["prod"],          // 空 = 全部集群
        "sources": ["incident"],         // 空 = 全部来源
        "minSeverity": "high",
        "receiverIds": [1, 2],
        "groupWaitSeconds": null, "groupIntervalSeconds": null, "repeatIntervalSeconds": 3600,
        "createdAt": "...", "updatedAt": "..." }
  ], "total": 1 }

POST /api/v2/notify/rules              （不带 id / position；新规则排在最后）→ 201
PUT /api/v2/notify/rules/{id}          → 200
DELETE /api/v2/notify/rules/{id}       → 200
PUT /api/v2/notify/rules/order         { "ids": [3, 1, 2] } → 200      # 必须恰好是全部规则的一个排列

POST /api/v2/notify/rules/match        { "clusterId": "prod", "source": "incident", "severity": "high" }
→ 200 { "message": "匹配完成", "data": {
      "matchedRule": { "id": 1, "name": "生产高严重度" },   // null = 走默认策略
      "receivers": [ { "id": 1, "name": "值班频道", "type": "slack", "enabled": true } ],
      "belowMinSeverity": false      // 默认策略下严重度不到 minSeverity，不会发送
  } }
```

**规则**

- 按 `position` 从小到大匹配；第一条 `enabled` 且条件全部满足的规则生效，不再往下匹配
- 条件：集群在 `clusterIds` 里（空 = 全部）、来源在 `sources` 里（空 = 全部）、严重度 ≥ `minSeverity`
- 规则的三个时间参数为 `null` 时用策略里的全局值
- 都不命中：用默认策略（`minSeverity` + `defaultReceiverIds`）
- 接收人的凭据（Webhook 地址）与发件设置的 SMTP 密码加密存储，接口永不返回原值

**数据对照**

| API JSON（receiver） | Go（`model.NotifyReceiver`） | 表 `notify_receivers` |
|---|---|---|
| `id` / `name` / `type` / `enabled` | 同名 | `id INTEGER PRIMARY KEY AUTOINCREMENT` / `name TEXT UNIQUE NOT NULL` / `type TEXT NOT NULL` / `enabled INTEGER NOT NULL DEFAULT 1` |
| `slack` / `email` | `Slack *…` / `Email *…` | `config TEXT NOT NULL`（JSON，整体加密） |
| `createdAt` / `updatedAt` | 同名 | `created_at` / `updated_at` |

| API JSON（rule） | Go（`model.NotifyRule`） | 表 `notify_rules` |
|---|---|---|
| `id` / `name` / `position` / `enabled` | 同名 | `id INTEGER PRIMARY KEY AUTOINCREMENT` / `name TEXT NOT NULL` / `position INTEGER NOT NULL` / `enabled INTEGER NOT NULL DEFAULT 1` |
| `clusterIds` / `sources` / `receiverIds` | `[]string` / `[]string` / `[]int64` | `cluster_ids` / `sources` / `receiver_ids`（TEXT，JSON 数组） |
| `minSeverity` | `MinSeverity` | `min_severity TEXT NOT NULL DEFAULT 'low'` |
| `groupWaitSeconds` 等三项 | `*int` | `group_wait_seconds` 等三列，可空 |
| `createdAt` / `updatedAt` | 同名 | `created_at` / `updated_at` |

原来的表 `notify_channels` 与接口 `/api/v2/notify/channels` 删除。

**实现**

- `database/`：两张表与仓库；`notify_channels` 及其仓库删除
- `notifier/routing/` [新增]：规则匹配（纯函数）
- `service/`、`gateway/handler/admin/`：接口
- 前端：页签「接收人」（列表 + 新建 / 编辑弹窗 + 测试发送）、「路由规则」（可上下调序 + 试匹配）

### 全局发件设置（2026-11-15 追加）

**用户故事**：邮件从哪台服务器、用哪个账号发，全局只配一次；每个邮件接收人只填收件人。
换密码只改一处。与 Alertmanager / Grafana 的做法一致（全局 SMTP，接收人只写收件地址）。

**API**

```
GET /api/v2/notify/smtp
→ { "message": "获取成功", "data": {
      "configured": true,                 // 主机、端口、账号、密码都已配置
      "host": "smtp.example.com", "port": 587, "user": "ops@example.com",
      "tls": true, "fromAddress": "ops@example.com",   // 空 = 使用 user
      "passwordConfigured": true,
      "updatedAt": "..."                  // 从没保存过 = null
  } }

PUT /api/v2/notify/smtp
  { "host": "smtp.example.com", "port": 587, "user": "ops@example.com", "password": "...",
    "tls": true, "fromAddress": "" }
→ 200 { "message": "已保存", "data": { ...同 GET... } }
  # password 留空 = 保持原值；第一次保存必须填写 → 400
```

权限同其他通知接口（`notify.manage`）；PUT 记审计（`update` / `notify_smtp`，请求体里的 password 由审计脱敏）。

**发送**

- 邮件接收人发送时，每次从库里读发件设置（单行，读取很轻），改完立即生效，不用重启
- 发件设置没配齐 → 送达失败，原因写「未配置发件设置」；测试发送同样返回这条原因
- 发送失败的原因里去掉 SMTP 密码

**数据对照**

| API JSON | Go（`model.NotifySMTP` / `database.NotifySMTP`） | 表 `notify_smtp` |
|---|---|---|
| `host` / `port` / `user` / `tls` / `fromAddress` | `Host` / `Port` / `User` / `TLS` / `FromAddress` | `config TEXT NOT NULL`（JSON，整体加密） |
| `password`（只写） / `passwordConfigured`（只读） | `Password` | 同上 |
| `configured` | 派生：主机、端口、账号、密码都非空 | — |
| `updatedAt` | `UpdatedAt` | `updated_at TEXT NOT NULL` |

表是单行（`id INTEGER PRIMARY KEY CHECK (id = 1)`）。新表用 `CREATE TABLE IF NOT EXISTS`，已有的库启动时自动建出，**不用清库**；
接收人配置本来就整份加密存在 `config` 一列，去掉其中的 SMTP 字段不涉及表结构。

**前端**：页签「接收人」顶部加一块「邮件发件设置」（查看 / 编辑，密码只写不读）；邮件接收人的表单只剩收件人；
发件设置没配时，接收人页签提示「邮件接收人需要先配置发件设置」。

## 功能三：静默（第 3 期）

**用户故事**：今晚维护某个命名空间，两小时内它的告警都不要发；维护提前结束时可以手动结束静默。

**API**

```
GET /api/v2/notify/silences?state=active|pending|expired
→ { "message": "获取成功", "data": [
      { "id": 1, "clusterId": "prod",          // 空 = 全部集群
        "namespace": "shop",                   // 空 = 全部命名空间
        "subject": "",                         // 空 = 全部主体；填了则精确匹配主体键
        "startsAt": "...", "endsAt": "...",
        "comment": "数据库升级", "createdBy": "admin", "createdAt": "...",
        "state": "active" }                    // pending（未开始）/ active / expired
  ], "total": 1 }

POST /api/v2/notify/silences
  { "clusterId": "prod", "namespace": "shop", "subject": "", "startsAt": "...", "endsAt": "...", "comment": "..." }
→ 201        # 校验：endsAt > startsAt 且 endsAt > 现在；时长不超过 30 天；comment 必填、不超过 256 字

POST /api/v2/notify/silences/{id}/expire → 200      # 提前结束：endsAt 置为现在
```

**规则**

- 告警命中静默 = 静默处于 `active`，且三个条件都满足（空条件视为满足）。
  心跳告警没有命名空间与主体，所以只有不限定命名空间与主体的静默才会压住它（维护某个命名空间不该连集群离线也不报）
- 命中的告警照常落库与显示，`silencedBy` 记下静默的 id；静默结束后仍在持续的，下一轮按新告警发出
- 静默期间结束的告警不补发触发通知，也不发恢复通知
- 静默不删除，只会过期；过期的保留 30 天

**数据对照**

| API JSON | Go（`model.NotifySilence`） | 表 `notify_silences` |
|---|---|---|
| `id` | `ID` | `id INTEGER PRIMARY KEY AUTOINCREMENT` |
| `clusterId` / `namespace` / `subject` | 同名 | `cluster_id` / `namespace` / `subject`（TEXT NOT NULL DEFAULT ''） |
| `startsAt` / `endsAt` | 同名 | `starts_at TEXT NOT NULL` / `ends_at TEXT NOT NULL` |
| `comment` / `createdBy` / `createdAt` | 同名 | `comment TEXT NOT NULL` / `created_by TEXT NOT NULL` / `created_at TEXT NOT NULL` |
| `state` | 由时间算出 | — |

**实现**

- `database/`：表与仓库；`notifier/silence/` [新增]：匹配（纯函数）
- 前端：页签「静默」（列表 + 新建弹窗 + 提前结束）；「当前告警」里标出「已静默」「已抑制」

## 实施阶段

```
功能零 事件按主体收敛 ──► 第 1 期 事件通知闭环 ──► 第 2 期 接收人与路由 ──► 第 3 期 静默
```

每期结束都能在界面上验收。各期都有表结构变更，上线需清库。

## 测试要点（先写测试）

- 主体解析：Deployment / StatefulSet / DaemonSet / CronJob / 裸 Pod / 上级不在快照里
- 状态机：同一工作负载两个 Pod 只建一个事件；警告恢复后进入观察期；6 小时内再触发是复发而不是新事件；
  没升级过的事件 6 小时关闭、升级过的 48 小时；严重度只升不降；`Occurrences24h` 的计数与 24 小时滑出
- 事件存储：关闭后严重度仍在；`escalated_at` 只写一次；成员实体补写
- 事件 → 告警：上表每一行一个用例
- 管道：分组等待内合并；分组间隔；重复提醒；去重；抑制；静默及其到期；失败重试三次后放弃；
  重启后不重复发「触发」；没有接收人时记 `no-receiver`
- 路由：顺序、第一条命中即停、空条件、禁用的规则、默认策略、时间参数覆盖
- 接口：校验、权限、凭据不外泄

## 裁决记录（2026-11-15）

| 事项 | 背景 | 裁定 |
|------|------|------|
| 范围 | 事件按单个实体开立、反复触发各算各的，只做通知会原样继承；完整的拓扑归并量大 | 通知第 1 期里一并做四项修改（主体收敛、反复并入同一事件、严重度保留、按严重度与反复次数通知）。完整归并、周期性基线、环境重要性、反馈回路另行立项 |
| 什么时候通知 | 「警告」「事件中」都是风险分数切出来的档位 | 进入「事件中」立即通知；只到「警告」的，同一主体 24 小时内反复到阈值才通知一次 |
| 旧的逐条 K8s 事件告警 | 与事件通知并行会重复 | 默认关闭，保留开关 |
| 做几期 | 三期都要改表 | 三期全做，一次发布 |
| 邮件发件设置放哪里 | 原实现每个邮件接收人各填一套 SMTP；用户预期「发件人配一次，接收人只填收件人」，主流系统也是全局 SMTP | 改成全局只配一次（2026-11-15 追加裁定） |

### 工程选择（由实现方决定，供查阅）

- 恢复通知在进入恢复观察期时发出，而不是观察期满才发；观察期内复发会重新触发告警
- 主体只用于状态机与事件；依赖图不新增工作负载节点（风险传播与拓扑页不受影响）
- `root_cause` 仍是领头实体，AI 分析的上下文构建不变
- 三期一次发布，接收人表直接取代渠道表，不做过渡形态
- 旧的 K8s 事件告警改走同一条管道，开启后也有分组与去重

功能零落地时的选择：

- 状态机只认一个处理方 `TransitionHandler.OnTransition(ctx, transition, risk)`（引擎实现：先落库、再发给订阅者）；
  事件号在建事件时由状态机通过注入的 `incident.NewIncidentID` 分配，不等写库
- 状态机条目除表中三项外，还带 `Severity`、`Recurrence`、`RootCause`（分别来自 `severity`、`recurrence`、`root_cause`），
  用来把每条变化的字段填齐；键由实体改为主体（`SubjectKey`）
- `root_cause` 跟随当前领头：建事件、升级、严重度升高、复发时，如果领头换了（Pod 重建换了名字等），
  事件的 `root_cause` 与实体表的角色一起换，原领头降为 `affected`；恢复与关闭类的变化沿用最近一次记下的领头
- `peak_risk` 在每次变化时都写库（含恢复与关闭），两次变化之间涨上去的部分只在内存里
- 时间线新增类型 `severity_raised`；关闭的条目仍是 `state_change`，文案区分观察期满与失联
- 同一状态的两个出口条件各计各的时（条目新增 `ConditionTarget`）：原来共用一个计时起点，
  「风险先低后高」时会把低风险攒下的时间算给升级
- 属于 ReplicaSet、而该 ReplicaSet 不属于任何 Deployment 的 Pod，按「其他 Pod」处理，主体是自身
- 事件的 API 响应直接是 `aiops.Incident`（`subjectType` 在事件存储的类型转换里由 `subject` 解析），没有另设 `model/` 下的响应模型

通知管道落地时的选择：

- 反复类告警（没升级过的事件）在事件进入恢复观察时不结束，等事件关闭才结束；否则每抖一次都是「恢复 + 反复」两条
- 去重只在同一次持续之内生效；恢复后再次出现总会通知
- 静默到期或集群恢复上线时，仍在持续的告警按「触发」再发一次
- 恢复通知还没发出去告警又触发：撤掉那条恢复，算同一次持续，两条都不发
- 没有可用接收人时不算通知过；满一个重复间隔后补发的是原来的变化，不是「重复提醒」
- 重试共 4 次尝试（首次 + 30 秒 / 2 分钟 / 10 分钟）；等重试期间接收人被删或停用，直接放弃
- 订阅回调只入队，「是否产生告警」在管道协程里按当时的策略判断；队列容量 1024，满了丢弃并计数
- 加密器不可用时拒绝保存接收人，不落明文；Slack 不跟随重定向；写入送达记录与日志的错误信息先去掉 Webhook 地址
- 环境变量 `MASTER_EVENT_ALERT_ENABLED` 删除，改由策略里的 `k8sEventAlertsEnabled` 控制；旧的事件告警消息里的资源状态补充信息不再提供
- 事件时长是「异常持续了多久」，不含恢复观察期；活跃事件只数警告与事件中；出事件最多的对象按主体归并

全局发件设置落地时的选择：

- 单独一张单行表 `notify_smtp`，不放进 `notify_policy`：凭据要整体加密，策略表是明文列；新表也免去清库
- 接收人不能覆盖全局发件设置（暂不需要；接收人配置是 JSON，以后要加不涉及表结构）
- 不另设「测试发件设置」接口：建一个邮件接收人后用它的测试发送即可验证
- 界面放在「接收人」页签顶部，不新开页签：发件设置只服务于邮件接收人
- 前端：卡片分查看 / 编辑两态；「需要先配置发件设置」在卡片、列表的邮件接收人行、新建 / 编辑弹窗三处提示，
  只在接口明确返回 `configured: false` 时提示；首次保存不填密码在前端先拦下；`updatedAt` 用 `null` 表示从没保存过

## 文件变更清单

```
atlhyper_master_v2/
├── aiops/
│   ├── subject/resolver.go                      [新增]
│   ├── types.go, interfaces.go                  [修改] SubjectRisk、IncidentTransition、SubscribeTransitions
│   ├── statemachine/{machine,trigger}.go        [修改]；suppressor.go [删除]（无引用）
│   ├── incident/store.go                        [修改]
│   ├── core/engine.go                           [修改]
│   └── enricher/                                [修改] 订阅方式
├── notifier/
│   ├── alert/                                   [新增]
│   ├── pipeline/                                [新增]
│   ├── routing/, silence/                       [新增]
│   ├── source/{incident,heartbeat,k8s_event}.go [新增]；trigger/ [删除]
│   ├── template/                                [修改]
│   ├── channel/                                 [修改]
│   └── manager.go, interfaces.go, alert.go      [重写 / 删除]
├── database/                                    [修改] 事件表三列；notify_policy / alerts / notify_deliveries /
│                                                        notify_receivers / notify_rules / notify_silences / notify_smtp；删除 notify_channels
├── model/, model/convert/                       [新增] notify_*.go；[修改] aiops 事件模型
├── service/{interfaces.go,query/,operations/}   [修改]
├── gateway/{routes.go,handler/admin/,handler/aiops/} [修改]
├── config/                                      [修改] MASTER_PUBLIC_URL
└── master.go                                    [修改] 装配
atlhyper_web/
├── src/app/settings/notif/                      [重写] 六个页签
├── src/app/aiops/（事件列表与详情）              [修改] 主体、成员、复发、升级时间
├── src/api/notify.ts, src/queries/notify.ts     [重写]
├── src/mock/notify/                             [重写]
└── src/types/, src/i18n/                        [修改]
```
