# 事件通知：从检测到通知

> 状态：设计稿，待用户裁决「待裁决」一节后动工。

## 背景

AIOps 引擎已经能把异常收敛成事件（incident）并跟踪它的状态，但事件只出现在页面上，不会通知到人。
通知模块（notifier）现在只有两个来源：Agent 心跳，和 K8s Warning 事件（每条新事件发一条消息）。
两边互不相通：

| 现状 | 后果 |
|------|------|
| 事件的状态转换只触发 AI 后台分析，不发通知 | 检测到了，没人知道 |
| K8s Warning 事件逐条发送，不分组、不按严重度筛选 | 测试环境 7 分钟 22 条；同一个问题既报事件告警、又会进入 AIOps |
| 没有统一的告警对象（没有标识、状态、开始与结束时间） | 做不了去重、恢复通知、重复提醒 |
| 发送失败只记日志，不重试，没有送达记录 | 渠道配错或对端故障时，通知无声无息地丢了 |
| 所有已启用的渠道收到同样的内容；每种渠道类型只能配一条 | 不能按集群或严重度分流 |
| 没有静默 | 维护窗口里只能关掉整个渠道 |

## 目标

- 事件的关键状态转换通知到人：进入「事件中」、严重度升高、复发、恢复
- 通知管道具备主流告警系统（Alertmanager 模型）的基本能力：分组、去重、抑制、重复提醒、恢复通知、失败重试、送达记录
- 第二、三期补上接收人与路由规则、静默

**不在范围内**

- SLO 违规告警（burn rate 接线）、OOM 走 Pod 状态：已另案冻结，本设计不触碰
- 用户自定义阈值规则（路线图 2.1 的规则引擎）
- 新的渠道类型（钉钉、通用 Webhook 的发送器）

## 核心架构

```
告警来源                      统一告警对象                 通知管道（独立协程，不阻塞快照处理）
────────                      ────────────                 ──────────────────────────────────
AIOps 事件的状态转换  ─┐                                   1. 匹配策略 / 规则 → 选出接收人
Agent 心跳（离线/恢复）─┼─►  Alert ─► 入队（有界，不阻塞）─► 2. 静默：命中的不发（第 3 期）
K8s Warning 事件（旧） ─┘     落库                          3. 抑制：集群离线时压住该集群的其他告警
                                                           4. 分组：同一集群在等待窗口内合并成一条
                                                           5. 重复：仍在持续的，按间隔再提醒
                                                           6. 发送 → 失败重试 → 送达记录
```

### 统一告警对象

```go
// notifier/alert
type Alert struct {
    Fingerprint string            // 稳定标识：同一个问题始终是同一个值（事件告警 = "incident:<事件ID>"）
    Source      string            // incident / heartbeat / k8s_event
    ClusterID   string
    Severity    Severity          // low / medium / high / critical（与 AIOps 的四档一致）
    Status      Status            // firing / resolved
    Title       string
    Summary     string
    Labels      map[string]string // namespace、entityType、entityName 等，供规则与静默匹配
    Details     map[string]string // 根因实体、峰值风险、复发次数、AI 摘要等，供模板使用
    StartsAt    time.Time
    EndsAt      *time.Time
    UpdatedAt   time.Time
}
```

- 告警落库（表 `alerts`）：Master 重启后仍知道哪些在持续、上次什么时候通知过
- 严重度全模块统一为四档；「恢复」是状态，不是严重度

### 事件 → 告警

状态机的五个回调都只做一件事：把一条状态变化放进有界队列后立即返回。
回调是在持有状态机锁、且位于快照处理主路径上被调用的，所以绝不在回调里发送。

| 事件的变化 | 告警动作 |
|-----------|---------|
| 进入「事件中」（Warning → Incident） | 触发（firing），发通知 |
| 持续期间严重度升高一档以上 | 更新，再发一次（标明「升级」） |
| 进入恢复观察期（Incident → Recovery） | 更新，不发 |
| 复发（Recovery → Warning 后再次进入事件中） | 再发一次（标明「复发」与次数） |
| 关闭（→ Stable） | 置为 resolved，发恢复通知 |
| 因 Agent 断连超时被关闭 | 置为 resolved，**不发**恢复通知（并不是真的恢复了） |
| 只到「警告」就自行恢复 | 不产生告警（见待裁决 1） |

### 分组、抑制、重复

- **分组**：同一集群的告警在「分组等待」（默认 30 秒）内合并成一条消息，列出每个受影响的实体；
  之后同组有新内容时，最短间隔「分组间隔」（默认 5 分钟）再发
- **抑制**：某集群的 Agent 离线告警在持续时，该集群的其他告警不发（Agent 都连不上了，其他告警都是它的后果）
- **重复提醒**：告警持续未恢复，每隔「重复间隔」（默认 4 小时）再提醒一次
- **去重**：同一个指纹、同一种状态，在重复间隔内只发一次

### 发送与送达记录

- 每个接收人一次发送 = 一条送达记录（表 `notify_deliveries`）：告警指纹、接收人、状态（成功 / 失败 / 已放弃）、尝试次数、最后一次错误、时间
- 失败按 30 秒、2 分钟、10 分钟重试三次，仍失败记为「已放弃」
- 发送带超时（SMTP 现在没有超时，一台卡住的邮件服务器会卡住整个循环）
- 没有任何可用接收人时，告警照常落库并记一条「无接收人」的送达记录，配好渠道后重复提醒会把仍在持续的告警补发出去

### 通知内容

- 标题：`[集群] 严重度 · 实体 · 一句话摘要`
- 正文：状态（触发 / 升级 / 复发 / 恢复）、根因实体、峰值风险、持续时间、复发次数；分组时列出全部实体
- AI 摘要：分组等待结束时如果后台分析已经产出摘要，就带上；没有就不等
- 页面链接：配置了 `MASTER_PUBLIC_URL` 时带上事件详情页的链接

## 功能一：事件通知闭环（第 1 期）

**用户故事**：某个服务出问题并升级为事件，30 秒内 Slack / 邮件收到一条通知，写明是哪个集群的哪个实体、有多严重；
同一集群同时出问题的几个实体合并在一条里。问题持续 4 小时会再提醒。恢复后收到一条恢复通知。
在设置页能看到每条通知发给了谁、成没成功。

**API**

```
GET /api/v2/notify/policy
→ { "message": "获取成功", "data": {
      "incidentAlertsEnabled": true,
      "minSeverity": "low",              // 低于此严重度的事件不通知
      "notifyResolved": true,
      "groupWaitSeconds": 30,
      "groupIntervalSeconds": 300,
      "repeatIntervalSeconds": 14400,
      "k8sEventAlertsEnabled": false     // 旧的逐条 K8s 事件告警
  } }

PUT /api/v2/notify/policy   （同上结构）→ 200

GET /api/v2/notify/alerts?status=firing|resolved&cluster_id=&limit=&offset=
→ { "message": "获取成功", "data": [
      { "fingerprint": "incident:inc-123", "source": "incident", "clusterId": "c1",
        "severity": "high", "status": "firing", "title": "...", "summary": "...",
        "startsAt": "...", "endsAt": "", "lastNotifiedAt": "...", "notifyCount": 2,
        "suppressedBy": "" }             // 被抑制时写明原因（如 "cluster-offline"）
  ], "total": 1 }

GET /api/v2/notify/deliveries?fingerprint=&limit=&offset=
→ { "message": "获取成功", "data": [
      { "id": 1, "fingerprint": "incident:inc-123", "alertTitle": "...", "receiver": "slack",
        "kind": "firing",                // firing / escalated / recurred / resolved / repeat
        "status": "success",             // success / failed / abandoned / no-receiver
        "attempts": 1, "lastError": "", "createdAt": "...", "updatedAt": "..." }
  ], "total": 1 }
```

权限：三个接口都用现有的 `notify.manage`（2 级）；写操作审计。

**数据对照**

| API JSON（policy） | Go（`model.NotifyPolicy`） | 表 `notify_policy`（单行） |
|---|---|---|
| `incidentAlertsEnabled` | `IncidentAlertsEnabled` | `incident_alerts_enabled INTEGER` |
| `minSeverity` | `MinSeverity` | `min_severity TEXT` |
| `notifyResolved` | `NotifyResolved` | `notify_resolved INTEGER` |
| `groupWaitSeconds` | `GroupWaitSeconds` | `group_wait_seconds INTEGER` |
| `groupIntervalSeconds` | `GroupIntervalSeconds` | `group_interval_seconds INTEGER` |
| `repeatIntervalSeconds` | `RepeatIntervalSeconds` | `repeat_interval_seconds INTEGER` |
| `k8sEventAlertsEnabled` | `K8sEventAlertsEnabled` | `k8s_event_alerts_enabled INTEGER` |

新增表 `alerts`（主键 `fingerprint`）与 `notify_deliveries`（自增主键，按 `created_at` 索引，保留 30 天）。

**后端实现**

- `notifier/alert/`：`Alert`、严重度与状态、指纹
- `notifier/pipeline/`：入队、分组、抑制、重复、发送与重试；时钟可替换以便测试
- `notifier/source/`：`incident.go`（事件 → 告警）、`heartbeat.go`、`k8s_event.go`（两个旧触发器改为产出 Alert，经同一条管道）
- `aiops/core`：五个回调改为发出带完整上下文的状态变化（集群、实体、前后状态、风险、时间），允许多个订阅者（AI 分析与通知各订各的）
- `notifier/template`：新增事件通知与恢复通知的模板（Slack / 邮件）；邮件主题按 RFC 2047 编码
- `notifier/channel`：SMTP 加超时；渠道凭据改为加密存储（沿用 AI Provider 用的加密器）
- `database/`：三张表与对应仓库；`config/`：`MASTER_PUBLIC_URL`
- `service/`、`gateway/`：三个接口

**前端实现**

- `app/settings/notif/`：新增「通知策略」卡片与「送达记录」表；「当前告警」列表
- `queries/notify.ts`：查询与写操作的工厂

**边界**

- 快照处理不因通知阻塞：回调只入队；队列满时丢弃并计数、记日志
- Master 重启：持续中的告警从库里恢复，不重复发「触发」，重复提醒的计时接着算
- 渠道全部失败：三次重试后记「已放弃」，不影响下一条
- Agent 离线：只发一条离线告警，该集群其他告警被抑制；恢复上线后各自按状态继续

## 功能二：接收人与路由规则（第 2 期）

**用户故事**：生产集群的高严重度告警发到值班频道和邮件，测试集群的只发到开发频道。

- `notify_receivers`：可以有多个同类型的接收人（多个 Slack 频道、多组邮件收件人），各自命名
- `notify_rules`：按顺序匹配——集群（全部或指定）、来源、最低严重度 → 一组接收人；可各自覆盖分组与重复的时间
- 第 1 期的全局策略成为「默认规则」
- 界面：接收人管理、规则列表（可排序）、规则的「试匹配」（输入集群与严重度，看会发给谁）

## 功能三：静默（第 3 期）

**用户故事**：今晚维护某个命名空间，两小时内它的告警都不要发。

- `notify_silences`：匹配条件（集群、命名空间、实体）、起止时间、创建人、备注
- 命中的告警照常落库与显示，只是不发；到期自动失效
- 界面：新建 / 提前结束静默；告警列表上标出「已静默」

## 实施阶段

```
第 1 期 事件通知闭环 ──► 第 2 期 接收人与路由 ──► 第 3 期 静默
```

每期结束都能在界面上验收。三期都有表结构变更，上线需清库。

## 待裁决

1. **什么时候开始通知**
   - A. 进入「事件中」才通知（风险 > 0.5 持续 5 分钟，或集群整体风险很高）；只到「警告」就自行恢复的不通知
   - B. 一进入「警告」就通知（风险 > 0.2 持续 2 分钟），更早，也更吵
2. **旧的逐条 K8s 事件告警**
   - A. 默认关闭，保留开关
   - B. 保持开启，与事件通知并行
   - C. 删除
3. **范围**：只做第 1 期，还是三期全做

## 文件变更清单

```
atlhyper_master_v2/
├── notifier/
│   ├── alert/                     [新增]
│   ├── pipeline/                  [新增]
│   ├── source/                    [新增] incident / heartbeat / k8s_event
│   ├── trigger/                   [删除] 并入 source/
│   ├── template/                  [修改] 新增事件与恢复模板
│   ├── channel/                   [修改] SMTP 超时
│   └── manager.go, interfaces.go  [修改]
├── aiops/core/engine.go           [修改] 状态变化可被多个订阅者接收
├── aiops/types.go                 [修改] 状态变化的结构
├── database/                      [修改] alerts / notify_deliveries / notify_policy（第 2、3 期：receivers / rules / silences）
├── service/, gateway/, model/     [修改] 三个接口
├── config/                        [修改] MASTER_PUBLIC_URL
└── master.go                      [修改] 装配
atlhyper_web/
├── src/app/settings/notif/        [修改] 通知策略、送达记录、当前告警
└── src/queries/notify.ts          [新增]
```
