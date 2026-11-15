# Master V2 API 参考文档

> 本文档基于 `atlhyper_master_v2/gateway/routes.go` 梳理，记录 Master 对 Web 前端提供的 HTTP API。
> **权限以代码为准**：权限表在 `gateway/middleware/permission.go`（唯一权限源），每条路由在 `routes.go` 声明所需权限。
> 运行时可用 `GET /api/v2/auth/permissions` 取完整权限表。设计见 `docs/design/active/permission-model-design.md`。

---

## 目录

1. [认证与权限模型](#1-认证与权限模型)
2. [中间件栈](#2-中间件栈)
3. [API 端点清单（按权限等级）](#3-api-端点清单按权限等级)
4. [审计覆盖](#4-审计覆盖)
5. [通用约定](#5-通用约定)
6. [Handler 包映射](#6-handler-包映射)

---

## 1. 认证与权限模型

### 权限等级

| level | 主体 | 新增的权限 |
|-------|------|-----------|
| **0** | 访客（未登录）= 新注册成员 | `cluster.read` `events.read` `metrics.read` `slo.read` `apm.summary.read` `logs.stats.read` `aiops.read` |
| **1** | Viewer | `workload.env.read` `logs.read` `traces.read` `commands.read` `deploy.history.read` `github.status.read` `ai.chat` |
| **2** | Operator | `secret.read` `pod.logs.read` `cluster.operate` `audit.read` `notify.manage` `slo.manage` `ai.analyze` `ai.settings.read` |
| **3** | Admin | `user.manage` `ai.settings.write` `github.manage` `deploy.manage` `clusters.manage` |

高等级自动继承低等级的全部权限。新注册账号（含 OIDC 自动建立的账号）默认 0 级，由 Admin 在用户管理中提升。

### 认证机制

- Token 通过 `Authorization: Bearer {token}` 传递；不带 token = 访客（0 级）
- `Authenticate` 中间件**每次请求从库中读取**当前角色与账号状态，⛔ 不信任 token 里的角色；账号被禁用 / 删除后立即降为访客
- 前端唯一的权限来源：`GET /api/v2/auth/me`（返回当前主体、level 与权限列表）

### 同一接口内的字段裁剪

- 资源列表 / 详情（0 级可见）统一套 `RedactEnvValues`：无 `workload.env.read` 时，任意层级的 `envs[].value` 置空并标 `valueHidden`
- 读类指令（Secret / ConfigMap 内容、Pod 日志）的输出只交给同步等待方，不落库；指令历史与状态接口不返回输出

---

## 2. 中间件栈

```
请求 → Logging → CORS → Authenticate（解析主体，查库取角色）→ mux
                         → [Audit →] RequirePermission(perm) → Handler
```

| 中间件 | 职责 |
|--------|------|
| `Logging` | 请求日志 |
| `CORS` | 跨域支持 |
| `Authenticate` | 解析 token → 主体；无 token / 无效 token = 访客 |
| `RequirePermission(perm)` | 权限检查：访客不足 → 401，已登录不足 → 403 |
| `Audit(action, resource)` | 审计记录（标记的路由） |

**审计包装在权限检查之外**：无论成功还是权限不足，都会记录。

路由注册只有四种写法（`routes.go`），⛔ 不存在「公开路由组」式旁路：

| 写法 | 含义 |
|------|------|
| `open(method, path, h)` | 无需主体（健康检查、登录、权限查询、OIDC） |
| `openAudited(method, path, action, resource, h)` | 无需主体 + 审计（登录、第三方回跳） |
| `route(method, perm, path, h)` | 需要权限 |
| `routeAudited(method, perm, path, action, resource, h)` | 需要权限 + 审计 |

**按（方法，路径）注册**：同一路径的读与写分别声明权限，写方法不继承读方法的权限；没声明的方法由 ServeMux 直接返回 405。
`routes_test.go` 遍历全部路由声明，守住三条规则：未声明的方法一律 405；访客能用的写方法只有登录类入口；2 级及以上的非 GET 路由一律审计。

---

## 3. API 端点清单（按权限等级）

> 路径以 `/` 结尾表示前缀匹配（如 `/api/v2/pods/` 捕获 `/api/v2/pods/{name}`）。审计列为 `action / resource`。

### 3.1 无需主体

| 方法 | 路径 | 审计 | Handler | 说明 |
|------|------|------|---------|------|
| GET | `/health` | — | `healthCheck` | 健康检查 |
| GET | `/api/v2/auth/me` | — | `AuthMeHandler.Get` | 当前主体与权限列表 |
| GET | `/api/v2/auth/permissions` | — | `GetPermissionTable` | 完整权限表（角色说明页用） |
| POST | `/api/v2/user/login` | login / user | `UserHandler.Login` | 本地账号登录（外部身份账号不可用密码登录） |
| GET | `/api/v2/auth/providers` | — | `oidc.Providers` | 外部登录方式（未启用时为空列表） |
| GET | `/api/v2/auth/oidc/start` | — | `oidc.Start` | 发起 OIDC 登录（仅启用时注册） |
| GET | `/api/v2/auth/oidc/callback` | login / user | `oidc.Callback` | OIDC 回调（仅启用时注册） |

### 3.2 0 级：集群状态与形状（全部 GET）

| 权限 | 路径 | Handler |
|------|------|---------|
| `cluster.read` | `/api/v2/overview` | `OverviewHandler.Get` |
| `cluster.read` | `/api/v2/clusters` | `ClusterHandler.List` |
| `cluster.read` | `/api/v2/{pods,nodes,deployments,daemonsets,statefulsets,services,ingresses,jobs,cronjobs,pvs,pvcs,network-policies,resource-quotas,limit-ranges,service-accounts,namespaces}` 及各自 `/` 前缀 | 各资源 `List` / `Get`（env 字面值裁剪） |
| `cluster.read` | `/api/v2/configmaps` | `ConfigMapHandler.List`（仅名字与键数；详情见 2 级） |
| `events.read` | `/api/v2/events` · `/api/v2/events/by-resource` | `EventHandler.List` / `ListByResource` |
| `metrics.read` | `/api/v2/node-metrics` · `/api/v2/node-metrics/` | `NodeMetricsHandler.Route` |
| `metrics.read` | `/api/v2/observe/freshness` | `ObserveHandler.Freshness` |
| `metrics.read` | `/api/v2/observe/metrics/{summary,nodes,hardware,compare}` · `/api/v2/observe/metrics/nodes/` | `ObserveHandler.Metrics*` |
| `slo.read` | `/api/v2/slo/domains` · `/domains/v2` · `/domains/detail` · `/domains/history` · `/domains/latency` · `/targets` · `/status-history` | `SLOHandler.*`（目标写入见 2 级 `slo.manage`） |
| `slo.read` | `/api/v2/observe/slo/{summary,ingress,timeseries}` | `ObserveHandler.SLO*` |
| `apm.summary.read` | `/api/v2/observe/traces/{services,stats,topology,operations}` · `/api/v2/observe/traces/services/` | `ObserveHandler.Traces*` / `APMServiceSeries` |
| `logs.stats.read` | `/api/v2/observe/logs/{summary,histogram}` | `ObserveHandler.LogsSummary` / `LogsHistogram` |
| `aiops.read` | `/api/v2/aiops/graph` · `/graph/trace` · `/baseline` | `AIOpsGraphHandler` / `AIOpsBaselineHandler` |
| `aiops.read` | `/api/v2/aiops/risk/{cluster,entities,entity}` | `AIOpsRiskHandler.*` |
| `aiops.read` | `/api/v2/aiops/incidents` · `/incidents/stats` · `/incidents/patterns` · `/incidents/` | `AIOpsIncidentHandler.*` |

### 3.3 1 级：内容级数据（Viewer）

| 权限 | 方法 | 路径 | 审计 | Handler |
|------|------|------|------|---------|
| `workload.env.read` | — | （不对应独立路由）资源详情中的 env 字面值 | — | `RedactEnvValues` 裁剪 |
| `logs.read` | POST | `/api/v2/observe/logs/query` | — | `ObserveHandler.LogsQuery`（查询条件放请求体） |
| `traces.read` | GET | `/api/v2/observe/traces` · `/api/v2/observe/traces/` · `/api/v2/observe/traces/correlations` | — | `ObserveHandler.TracesList` / `TracesDetail` / `TracesCorrelations` |
| `commands.read` | GET | `/api/v2/commands/history` · `/api/v2/commands/` | — | `CommandHandler.ListHistory` / `GetStatus`（不含输出） |
| `ai.chat` | GET · POST | `/api/v2/ai/conversations` | — | `AIHandler.Conversations`（只列自己的会话） |
| `ai.chat` | GET | `/api/v2/ai/conversations/` | — | `AIHandler.ConversationByID`（`{id}/messages`；别人的会话与不存在同样 404） |
| `ai.chat` | DELETE | `/api/v2/ai/conversations/` | delete / ai_conversation | `AIHandler.ConversationByID`（只能删自己的；handler 内另需 Operator） |
| `ai.chat` | POST | `/api/v2/ai/chat` | — | `AIHandler.Chat`（SSE；只能续聊自己的会话；工具按调用者本人的权限过滤，见下） |
| `github.status.read` | GET | `/api/github/connection` | — | `GitHubHandler.Connection` |
| `deploy.history.read` | GET | `/api/deploy/history` | — | `DeployHandler.History` |

> **AI 工具的权限**（`ai/tool_policy.go`）：每个工具读取的数据所需权限与网关上读同一份数据的接口一致，按「谁会看到结果」授权 —— 对话 = 调用者本人；后台深度分析 = 能看 AI 报告的人（`ai.analyze` 所在等级）。例：读容器日志需 `pod.logs.read`，读 ConfigMap 内容需 `secret.read`，触发事件总结需 `ai.analyze`。调用者没有的工具不发给模型；未登记权限的工具一律拒绝。
> GitHub 工具（`ai/github_scope.go`）只读公开仓库，以及 `MASTER_AI_GITHUB_PRIVATE_REPOS` 显式列出的私有仓库；读到的 Kubernetes Secret 清单，值一律替换为 `<redacted>`（`ai/github_redact.go`）。

### 3.4 2 级：凭据与写操作（Operator）

| 权限 | 方法 | 路径 | 审计 | Handler |
|------|------|------|------|---------|
| `secret.read` | GET | `/api/v2/configmaps/` | — | `ConfigMapHandler.Get` |
| `secret.read` | GET | `/api/v2/secrets` | — | `SecretHandler.List` |
| `secret.read` | POST | `/api/v2/ops/configmaps/data` | read / configmap | `OpsHandler.ConfigMapData` |
| `secret.read` | POST | `/api/v2/ops/secrets/data` | read / secret | `OpsHandler.SecretData` |
| `pod.logs.read` | POST | `/api/v2/ops/pods/logs` | read / pod | `OpsHandler.PodLogs` |
| `cluster.operate` | POST | `/api/v2/commands` | execute / command | `CommandHandler.Create` |
| `cluster.operate` | POST | `/api/v2/ops/pods/restart` | execute / pod | `OpsHandler.PodRestart` |
| `cluster.operate` | POST | `/api/v2/ops/deployments/{scale,restart,image}` | execute / deployment | `OpsHandler.Deployment*` |
| `cluster.operate` | POST | `/api/v2/ops/nodes/{cordon,uncordon}` | execute / node | `OpsHandler.NodeCordon` / `NodeUncordon` |
| `audit.read` | GET | `/api/v2/audit/logs` | — | `AuditHandler.List` |
| `notify.manage` | GET | `/api/v2/notify/policy` · `/smtp` · `/alerts` · `/deliveries` · `/receivers` · `/rules` · `/silences` | — | `NotifyHandler.GetPolicy` / `GetSMTP` / `ListAlerts` / `ListDeliveries` / `ListReceivers` / `ListRules` / `ListSilences` |
| `notify.manage` | PUT | `/api/v2/notify/policy` | update / notify_policy | `NotifyHandler.UpdatePolicy` |
| `notify.manage` | PUT | `/api/v2/notify/smtp` | update / notify_smtp | `NotifyHandler.UpdateSMTP` |
| `notify.manage` | POST | `/api/v2/notify/receivers` | create / notify_receiver | `NotifyHandler.CreateReceiver` |
| `notify.manage` | PUT · DELETE | `/api/v2/notify/receivers/{id}` | update · delete / notify_receiver | `NotifyHandler.UpdateReceiver` / `DeleteReceiver` |
| `notify.manage` | POST | `/api/v2/notify/receivers/{id}/test` | execute / notify_test | `NotifyHandler.TestReceiver` |
| `notify.manage` | POST | `/api/v2/notify/rules` | create / notify_rule | `NotifyHandler.CreateRule` |
| `notify.manage` | PUT · DELETE | `/api/v2/notify/rules/{id}` | update · delete / notify_rule | `NotifyHandler.UpdateRule` / `DeleteRule` |
| `notify.manage` | PUT | `/api/v2/notify/rules/order` | update / notify_rule_order | `NotifyHandler.ReorderRules` |
| `notify.manage` | POST | `/api/v2/notify/rules/match` | read / notify_rule_match | `NotifyHandler.MatchRule`（试匹配，只读） |
| `notify.manage` | POST | `/api/v2/notify/silences` | create / notify_silence | `NotifyHandler.CreateSilence` |
| `notify.manage` | POST | `/api/v2/notify/silences/{id}/expire` | update / notify_silence | `NotifyHandler.ExpireSilence` |
| `slo.manage` | PUT | `/api/v2/slo/targets` | update / slo_target | `SLOHandler.Targets`（host / 窗口 / 目标值校验） |
| `ai.analyze` | POST | `/api/v2/aiops/ai/summarize` | execute / ai_summary | `AIOpsAIHandler.Summarize` |
| `ai.analyze` | POST | `/api/v2/aiops/ai/recommend` | execute / ai_recommendation | `AIOpsAIHandler.Recommend` |
| `ai.analyze` | GET | `/api/v2/aiops/ai/reports` | — | `AIOpsAIHandler.ReportsHandler` |
| `ai.analyze` | GET | `/api/v2/aiops/ai/reports/` | read / ai_report | `AIOpsAIHandler.ReportDetailHandler` |
| `ai.analyze` | POST | `/api/v2/aiops/ai/analyze` | execute / ai_analysis | `AIOpsAIHandler.AnalyzeHandler` |
| `ai.analyze` | GET | `/api/v2/aiops/ai/analyze/status` | — | `AIOpsAIHandler.AnalyzeStatus` |
| `ai.settings.read` | GET | `/api/v2/ai/{providers,settings,roles,budgets,reports}` | — | `AIProviderHandler.*` |

### 3.5 3 级：系统配置（Admin）

| 权限 | 方法 | 路径 | 审计 | Handler |
|------|------|------|------|---------|
| `user.manage` | GET | `/api/v2/user/list` | — | `UserHandler.List` |
| `user.manage` | POST | `/api/v2/user/register` | create / user | `UserHandler.Register` |
| `user.manage` | POST | `/api/v2/user/{update-role,update-status}` | update / user | `UserHandler.UpdateRole` / `UpdateStatus` |
| `user.manage` | POST | `/api/v2/user/delete` | delete / user | `UserHandler.Delete` |
| `ai.settings.write` | POST | `/api/v2/ai/providers` | create / ai_provider | `AIProviderHandler.ProvidersHandler` |
| `ai.settings.write` | GET | `/api/v2/ai/providers/` | — | `AIProviderHandler.ProviderHandler`（单个详情） |
| `ai.settings.write` | PUT · PATCH · DELETE | `/api/v2/ai/providers/` | update · delete / ai_provider | `AIProviderHandler.ProviderHandler` |
| `ai.settings.write` | PUT · PATCH | `/api/v2/ai/settings` | update / ai_settings | `AIProviderHandler.SettingsHandler` |
| `ai.settings.write` | PUT | `/api/v2/ai/budgets/` | update / ai_budget | `AIProviderHandler.BudgetHandler` |
| `github.manage` | POST | `/api/github/connect` | create / github | `GitHubHandler.Connect` |
| `github.manage` | POST | `/api/github/callback` | create / github | `GitHubHandler.Callback`（GitHub App 安装回跳；只接受 App 所有者账号名下的安装） |
| `github.manage` | DELETE | `/api/github/disconnect` | delete / github | `GitHubHandler.Disconnect` |
| `github.manage` | GET | `/api/github/repos` · `/api/github/repos/` | — | `GitHubHandler.Repos` / `RepoSubRoute` |
| `deploy.manage` | GET | `/api/deploy/{config,kustomize-paths,status}` | — | `DeployHandler.*` |
| `deploy.manage` | PUT | `/api/deploy/config` | update / deploy_config | `DeployHandler.Config` |
| `deploy.manage` | POST | `/api/deploy/{test-connection,sync,rollback}` | execute / deploy | `DeployHandler.TestConnection` / `SyncNow` / `Rollback` |
| `clusters.manage` | GET | `/api/v2/admin/clusters` | — | `ClusterRegistryHandler.List` |
| `clusters.manage` | POST | `/api/v2/admin/clusters` | create / cluster | `ClusterRegistryHandler.Register` |
| `clusters.manage` | PUT | `/api/v2/admin/clusters/{clusterId}` | update / cluster | `ClusterRegistryHandler.UpdateProfile` |
| `clusters.manage` | DELETE | `/api/v2/admin/clusters/{clusterId}` | delete / cluster | `ClusterRegistryHandler.Unregister` |
| `clusters.manage` | POST | `/api/v2/admin/clusters/{clusterId}/token` | create / cluster_token | `ClusterRegistryHandler.IssueToken` |

> GitHub / 部署类路由仅在配置了 GitHub 客户端时注册；AI 对话路由仅在 AI 服务启用时注册。

### 3.6 集群注册表与 Agent 令牌（`clusters.manage`）

集群是管理员登记的对象；每个集群一枚令牌，Agent 用它向 Master 证明自己属于哪个集群。设计见 `docs/design/active/agent-auth-design.md`。

**列表** `GET /api/v2/admin/clusters`

返回已登记的集群，以及正在上报但尚未登记的集群（`registered: false`，可直接登记收编），按集群 ID 排序。

```json
{ "message": "获取成功", "data": [
    { "clusterId": "c1", "name": "生产", "description": "",
      "registered": true, "tokenIssued": true, "tokenPrefix": "atlk_Ab3", "tokenIssuedAt": "2026-11-14T10:00:00Z",
      "status": "online", "lastSeen": "2026-11-14T10:05:00Z", "authenticated": true }
  ], "total": 1 }
```

| 字段 | 说明 |
|------|------|
| `registered` | 是否已登记。未登记的行 `name` / `description` 为空 |
| `tokenIssued` / `tokenPrefix` / `tokenIssuedAt` | 是否已签发令牌、令牌开头的几个字符（供辨认）、最近一次签发时间。未签发时后两项为空串 |
| `status` | `online` / `offline` / `never`（登记了但从未上报） |
| `lastSeen` | 最近一次心跳或快照的时间；从未上报时为空串 |
| `authenticated` | 该集群最近一次上报是否带了有效令牌 |

时间一律是 UTC 的 RFC 3339 字符串。

**登记** `POST /api/v2/admin/clusters`

```json
{ "clusterId": "c1", "name": "生产", "description": "" }
```

→ `201 { "message": "已登记", "data": { ...同列表的一行... } }`

- `clusterId`：1–63 个字符，只允许小写字母、数字、`-`、`.`，首尾必须是字母或数字；须与 Agent 上报的集群 ID 逐字一致
- `name`：必填，不超过 64 个字；`description`：可空，不超过 256 个字；两者首尾空白会被去掉，不能含控制字符
- 输入不合规 → 400；集群 ID 已登记 → 409

**修改显示名与描述** `PUT /api/v2/admin/clusters/{clusterId}`

```json
{ "name": "生产", "description": "东京" }
```

→ `200 { "message": "已更新", "data": { ...修改后的一行... } }`；输入不合规 → 400；集群未登记 → 404

**签发令牌** `POST /api/v2/admin/clusters/{clusterId}/token`（无请求体）

```json
{ "message": "已签发", "data": { "token": "atlk_...", "tokenPrefix": "atlk_Ab3", "tokenIssuedAt": "2026-11-14T10:00:00Z" } }
```

- 令牌明文**只在这一次响应里出现**：库里只保存哈希与前缀，之后任何接口都取不回明文。响应带 `Cache-Control: no-store`
- 对已有令牌的集群再次签发即为轮换，旧令牌立即失效
- 集群未登记 → 404

**删除登记** `DELETE /api/v2/admin/clusters/{clusterId}`

→ `200 { "message": "已删除登记" }`；集群未登记 → 404。令牌随之失效；该集群的历史数据（事件、指令历史、SLO 等）不受影响。

### 3.7 AIOps 事件的响应字段（`aiops.read`）

事件以「主体」为单位：有上级工作负载的 Pod 归到工作负载，同一个工作负载的几个 Pod 同时出问题只有一个事件；
反复触发并入同一个事件并计数。设计见 `docs/design/active/incident-notification-design.md`「功能零」。

**列表** `GET /api/v2/aiops/incidents?cluster=&state=&severity=&from=&to=&limit=&offset=`

```json
{ "message": "获取成功", "data": [
    { "id": "inc-17672544000000001", "clusterId": "c1", "state": "recovery", "severity": "high",
      "subject": "shop/deployment/api", "subjectType": "deployment", "rootCause": "shop/pod/api-7d9f-abc",
      "peakRisk": 75, "startedAt": "2026-01-01T08:00:00Z", "escalatedAt": "2026-01-01T08:07:00Z",
      "resolvedAt": null, "closeReason": "", "durationS": 3600, "recurrence": 2, "summary": "",
      "createdAt": "2026-01-01T08:00:00Z" }
  ], "total": 1 }
```

**详情** `GET /api/v2/aiops/incidents/{id}`：上面的事件字段直接位于响应顶层（不包 `data`），另有 `entities`（成员实体）与 `timeline`。

| 字段 | 说明 |
|------|------|
| `subject` | 主体键 `<命名空间>/<类型>/<名字>`。Pod 归到所属的 Deployment / StatefulSet / DaemonSet / CronJob（Job 没有 CronJob 上级时归到 Job）；没有上级或上级不在快照里的 Pod，以及 service、node、ingress、logs，主体就是实体自身 |
| `subjectType` | 由 `subject` 解析出的类型：`deployment` / `statefulset` / `daemonset` / `cronjob` / `job` / `pod` / `service` / `node` / `ingress` / `logs` |
| `rootCause` | 领头实体：主体的成员里风险最高的那个。建事件、升级、严重度升高、复发时跟到当前的领头 |
| `state` | `warning` 警告 / `incident` 事件中 / `recovery` 恢复观察 / `stable` 已关闭。只到警告的事件恢复后同样先进入恢复观察 |
| `severity` | **峰值**严重度 `low` / `medium` / `high` / `critical`：只升不降，事件关闭后保留，可用于筛选与统计已关闭的事件 |
| `escalatedAt` | 首次升级到「事件中」的时刻；从未升级为 `null`。升级过的事件恢复观察 48 小时、观察期内复发立即生效；没升级过的观察 6 小时、需风险持续 2 分钟才算复发 |
| `closeReason` | 未关闭为空串；`recovered` 恢复观察期满；`stale` 失联（主体 30 分钟没有数据：对象已删除或 Agent 断连，并不代表真的恢复了） |
| `recurrence` | 复发次数。只到警告级的反复也计入 |
| `peakRisk` | 事件至今的最高风险，[0, 100]（列表、详情、事件模式同一刻度） |
| `durationS` | 异常持续了多久（秒），不含恢复观察期：仍在异常中的按当前时刻算；进入恢复观察期时定下；还在异常中就失联关闭的算到关闭时刻 |

详情的 `entities[]`：`entityKey`、`entityType`、`rLocal`、`rFinal`（均为 [0, 100]）、`role`。领头实体 `role = "root_cause"`（始终只有一个），其余成员 `"affected"`；
成员在建事件及之后每次风险抬升类的变化时写入：已有的更新为当时的风险值，新出现的补进去，已不在当轮的保留最后一次的值。

详情的 `timeline[].eventType`：`state_change`（建事件、升级、关闭）、`anomaly_detected`（领头实体的触发信号）、`severity_raised`（严重度升高）、`recovery_started`、`recurrence`。条目的 `entityKey` 是当时的领头实体。

`GET /api/v2/aiops/incidents/patterns?cluster_id=&entity=&period=` 的 `entity` 既可以传主体键，也可以传成员实体键：主体是它、或成员里有它的事件都会返回。

---

### 3.8 通知设置（`notify.manage`）

事件、Agent 心跳、（可选的）K8s Warning 事件先变成统一的告警，再经同一条管道通知到人：
路由规则选出接收人 → 静默 → 抑制（集群离线时压住该集群的其他告警）→ 分组 → 重复提醒 → 发送与重试 → 送达记录。
设计见 `docs/design/active/incident-notification-design.md`。

通用约定：时间一律是 UTC 的 RFC 3339 字符串，「没有」时为空串；列表的 `data` 恒为数组、`total` 恒为数字；
错误响应为 `{ "error": "..." }`。**凭据（接收人的 Webhook 地址、发件设置的 SMTP 密码）加密存储，任何接口都不返回，只返回是否已配置。**

**策略** `GET /api/v2/notify/policy` · `PUT /api/v2/notify/policy`

```json
{ "message": "获取成功", "data": {
    "incidentAlertsEnabled": true, "minSeverity": "low", "notifyResolved": true,
    "warningFlapThreshold": 3, "groupWaitSeconds": 30, "groupIntervalSeconds": 300,
    "repeatIntervalSeconds": 14400, "k8sEventAlertsEnabled": false, "defaultReceiverIds": [] } }
```

| 字段 | 说明 | 取值 |
|------|------|------|
| `incidentAlertsEnabled` | 事件是否产生告警（心跳告警不受影响） | — |
| `minSeverity` | 没有规则命中时，低于此严重度的告警不通知 | `low` / `medium` / `high` / `critical` |
| `notifyResolved` | 是否发恢复通知 | — |
| `warningFlapThreshold` | 只到「警告」的事件 24 小时内发生几次才通知；0 = 不通知 | 0–20 |
| `groupWaitSeconds` | 分组等待：同一集群的告警在这段时间内合并成一条 | 0–600 |
| `groupIntervalSeconds` | 分组间隔：同一组两条消息的最短间隔 | 30–86400 |
| `repeatIntervalSeconds` | 持续未结束的告警多久再提醒一次 | 300–604800 |
| `k8sEventAlertsEnabled` | 旧的逐条 K8s 事件告警 | 默认 `false` |
| `defaultReceiverIds` | 没有规则命中时发给谁；空 = 全部已启用的接收人 | 必须都存在 |

`PUT` 是整体替换：九个字段都必须给出，缺任何一个 → 400（少传一个开关不会被当成把它关掉）。成功 → `200 { "message": "已保存", "data": { ... } }`。

**当前告警** `GET /api/v2/notify/alerts?status=firing|resolved&cluster_id=&limit=&offset=`

```json
{ "message": "获取成功", "data": [
    { "fingerprint": "incident:inc-123", "source": "incident", "clusterId": "prod",
      "severity": "high", "status": "firing", "kind": "firing",
      "title": "shop/deployment/api 风险异常", "summary": "风险持续偏高，已升级为事件（峰值风险 72）",
      "subject": "shop/deployment/api", "incidentId": "inc-123",
      "startsAt": "2026-11-15T10:00:00Z", "endsAt": "", "lastNotifiedAt": "2026-11-15T10:00:30Z", "notifyCount": 2,
      "suppressedBy": "", "silencedBy": 0 }
  ], "total": 1 }
```

- `source`：`incident` / `heartbeat` / `k8s_event`；`kind`（最近一次变化）：`firing` / `escalated` / `recurred` / `flapping` / `resolved`
- `suppressedBy`：`cluster-offline` 表示该集群的 Agent 离线告警在持续，这条被压住没发；`silencedBy`：命中的静默 id，0 = 没有
- 不带 `status` = 全部；`limit` 默认 50、上限 200；开始时间新的在前；`total` 是符合条件的总数

**送达记录** `GET /api/v2/notify/deliveries?fingerprint=&status=&limit=&offset=`

```json
{ "message": "获取成功", "data": [
    { "id": 1, "batchId": "b-3f9a0c1d2e4b5a67", "fingerprint": "incident:inc-123", "alertTitle": "shop/deployment/api 风险异常",
      "receiverId": 2, "receiverName": "值班频道", "receiverType": "slack",
      "kind": "firing", "status": "success", "attempts": 1, "lastError": "",
      "createdAt": "2026-11-15T10:00:30Z", "updatedAt": "2026-11-15T10:00:30Z" }
  ], "total": 1 }
```

- `kind`：告警的五种变化，另加 `repeat`（重复提醒）
- `status`：`pending`（待发）/ `success` / `failed`（等待重试）/ `abandoned`（重试用尽，或接收人已删除 / 停用）/ `no-receiver`（没有任何可用接收人；此时 `receiverId` 为 0、名称与类型为空串）
- 发送失败按 30 秒、2 分钟、10 分钟重试三次。`lastError` 是失败原因，**不含 Webhook 地址与密码**
- 送达记录与已结束的告警保留 30 天

**发件设置** `GET /api/v2/notify/smtp` · `PUT /api/v2/notify/smtp`

邮件从哪台服务器、用哪个账号发，全局只配一次，所有邮件接收人共用。

```json
{ "message": "获取成功", "data": {
    "configured": true, "host": "smtp.example.com", "port": 587, "user": "ops@example.com",
    "tls": true, "fromAddress": "", "passwordConfigured": true, "updatedAt": "2026-11-15T10:00:00Z" } }
```

- `configured`：主机、端口、账号、密码都已配置；从没保存过时为 `false`，`updatedAt` 为 `null`
- 保存的请求体：`{ "host", "port", "user", "password", "tls", "fromAddress" }`（整体替换）→ `200 { "message": "已保存", "data": { ...同上... } }`
- 校验：主机、端口（1–65535）、账号必填；`fromAddress` 可空（用 `user`），填了必须是纯邮箱地址；
  `password` **留空 = 保持原值**，第一次保存必须填写 → 400
- 改完立即生效：每次发邮件时读取，不用重启

**接收人** `GET /api/v2/notify/receivers` · `POST /api/v2/notify/receivers` · `PUT|DELETE /api/v2/notify/receivers/{id}` · `POST /api/v2/notify/receivers/{id}/test`

```json
{ "message": "获取成功", "data": [
    { "id": 1, "name": "值班频道", "type": "slack", "enabled": true,
      "slack": { "webhookConfigured": true }, "createdAt": "...", "updatedAt": "..." },
    { "id": 2, "name": "运维邮件", "type": "email", "enabled": true,
      "email": { "toAddresses": ["a@example.com"] },
      "createdAt": "...", "updatedAt": "..." }
  ], "total": 2 }
```

新建 / 修改的请求体：

```json
{ "name": "值班频道", "type": "slack", "enabled": true, "slack": { "webhookUrl": "https://..." } }
{ "name": "运维邮件", "type": "email", "enabled": true, "email": { "toAddresses": ["a@example.com"] } }
```

- 新建 → `201 { "message": "已创建", "data": { ...同列表项... } }`；名称重复 → 409；没传 `enabled` 时为启用
- 校验：名称必填、不超过 64 字；Slack 的 `webhookUrl` 必须是 `https://` 地址；邮件至少一个收件人，必须是纯邮箱地址。
  邮件接收人只有收件人，发件服务器用上面的发件设置
- 修改时 `webhookUrl` **留空 = 保持原值**；不带 `slack` / `email` 段 = 整段配置保持原值；没传 `enabled` = 保持原值。
  启停没有单独接口：原样带上其余字段、凭据传空串即可。换类型时必须给出新类型的完整配置
- 删除 → `200`；仍被路由规则或默认策略引用 → 409，错误文字写明被谁引用
- 测试发送：按已保存的配置发一条测试消息（不管是否启用；不产生告警、不留送达记录）→ `200 { "message": "测试消息已发送" }`；
  接收人不存在 → 404；配置不完整 → 400，错误文字写明缺什么（Webhook 地址 / 收件人 / 发件设置）；
  发送失败 → 502，错误文字带原因（不含 Webhook 地址与密码）

**路由规则** `GET /api/v2/notify/rules` · `POST /api/v2/notify/rules` · `PUT|DELETE /api/v2/notify/rules/{id}` · `PUT /api/v2/notify/rules/order` · `POST /api/v2/notify/rules/match`

```json
{ "message": "获取成功", "data": [
    { "id": 1, "name": "生产高严重度", "position": 1, "enabled": true,
      "clusterIds": ["prod"], "sources": ["incident"], "minSeverity": "high", "receiverIds": [1, 2],
      "groupWaitSeconds": null, "groupIntervalSeconds": null, "repeatIntervalSeconds": 3600,
      "createdAt": "...", "updatedAt": "..." }
  ], "total": 1 }
```

- 按 `position` 从小到大匹配（列表也按此顺序返回）：第一条已启用且条件全部满足的规则生效，不再往下；都不命中用默认策略
- 条件：集群在 `clusterIds` 里（空 = 全部）、来源在 `sources` 里（空 = 全部）、严重度不低于 `minSeverity`（没填 = `low`）
- `receiverIds` 必须都存在；可以为空 —— 命中这条规则的告警不发给任何人（送达记录记 `no-receiver`）
- 三个时间参数的取值范围与策略里的同名项相同；`null` = 用策略里的全局值
- 新建不带 `id` / `position`，新规则排在最后 → 201；修改不改位置
- 重排 `PUT /rules/order`，请求体 `{ "ids": [3, 1, 2] }`，必须恰好是全部规则的一个排列，否则 400 → `200 { "message": "已保存", "data": [ ...重排后的列表... ] }`
- 试匹配 `POST /rules/match`，请求体 `{ "clusterId": "prod", "source": "incident", "severity": "high" }`：

```json
{ "message": "匹配完成", "data": {
    "matchedRule": { "id": 1, "name": "生产高严重度" },
    "receivers": [ { "id": 1, "name": "值班频道", "type": "slack", "enabled": true } ],
    "belowMinSeverity": false } }
```

  `matchedRule` 为 `null` = 走默认策略；`receivers` 含已停用的接收人（`enabled: false`，实际不会收到）；
  `belowMinSeverity` 为真 = 默认策略下严重度不到 `minSeverity`，不会发送。试匹配与真实发送用的是同一个路由函数

**静默** `GET /api/v2/notify/silences?state=active|pending|expired` · `POST /api/v2/notify/silences` · `POST /api/v2/notify/silences/{id}/expire`

```json
{ "message": "获取成功", "data": [
    { "id": 1, "clusterId": "prod", "namespace": "shop", "subject": "",
      "startsAt": "2026-11-15T10:00:00Z", "endsAt": "2026-11-15T12:00:00Z",
      "comment": "数据库升级", "createdBy": "admin", "createdAt": "2026-11-15T09:58:00Z", "state": "active" }
  ], "total": 1 }
```

- 三个条件为空 = 不限；`subject` 填了则精确匹配主体键。心跳告警没有命名空间与主体，只有不限定这两项的静默才会压住它
- 命中静默的告警照常落库与显示（`silencedBy` 记下静默 id），不发；静默结束时仍在持续的按新告警发出；静默期间结束的告警不补发触发、也不发恢复
- 新建请求体同上（不带 `id` / `createdBy` / `createdAt` / `state`；`startsAt` 留空 = 现在）→ 201。
  校验：`endsAt` 晚于 `startsAt` 且晚于现在；时长不超过 30 天；`comment` 必填、不超过 256 字。创建人取自当前登录的用户
- 提前结束 `POST /silences/{id}/expire`：结束时间置为现在 → 200，带回该静默。还没开始的静默也可以结束（建错了撤销），之后它的状态是 `expired`、不再生效
- 静默不删除，只会过期；过期的保留 30 天

---

## 4. 审计覆盖

审计记录无论成功还是权限不足都会写入。覆盖范围 = 上表「审计」列非空的路由：

- **登录**：本地登录、OIDC 回调
- **敏感读**：ConfigMap / Secret 内容、Pod 日志、AI 报告详情
- **写操作**：指令下发、Pod / Deployment / Node 操作、通知设置（策略、接收人含测试发送、路由规则含重排与试匹配、静默）、SLO 目标、AI 总结 / 建议 / 分析触发、删除 AI 会话
- **系统配置**：用户管理、AI Provider / 设置 / 预算、GitHub 连接（含安装回跳）、部署配置 / 同步 / 回滚 / 连接测试、集群注册表（登记 / 修改 / 删除登记 / 签发令牌）

规则（`routes_test.go` 守住）：2 级及以上的非 GET 路由一律审计。

审计记录的内容：操作人、动作与资源类型、HTTP 方法、脱敏后的请求体（凭据类键的值替换为 `***`，最多 500 字符）、状态码、来源 IP 与 User-Agent、耗时。**不记录响应体** —— 签发令牌接口返回的令牌明文不会进入审计日志。

---

## 5. 通用约定

### 查询参数

| 参数 | 类型 | 说明 | 适用范围 |
|------|------|------|----------|
| `cluster_id` | string | 集群 ID（必需） | 几乎所有资源查询 |
| `namespace` | string | 命名空间过滤 | 有 namespace 的资源 |
| `limit` | int | 返回数量限制 | 列表 API |
| `offset` | int | 分页偏移量 | 列表 API |

### 响应格式

**成功（列表）：**
```json
{ "message": "获取成功", "data": [...], "total": 10 }
```

**成功（详情）：**
```json
{ "message": "获取成功", "data": { ... } }
```

**错误：**
```json
{ "error": "错误信息" }
```

HTTP 状态码：200 成功，201 已创建（登记集群、新建通知接收人 / 规则 / 静默），400 参数错误，401 未登录且所需权限高于 0 级，403 已登录但权限不足，404 未找到，409 配置冲突（如 AI 角色未分配 Provider、集群 ID 已登记、接收人重名或仍被引用），500 内部错误，502 上游失败，504 超时。

AI 类接口失败时另带稳定错误码 `code`（`ai_role_unassigned` / `ai_budget_exhausted` / `ai_timeout` / `ai_upstream`），前端按码显示文案。

### 路由匹配规则

Go `http.ServeMux` 的匹配规则（Go 1.22 起的「方法 路径」模式）：
- `GET /api/v2/pods` — 精确匹配列表路由；GET 模式同时匹配 HEAD
- `GET /api/v2/pods/` — 前缀匹配，捕获 `/api/v2/pods/{name}`
- `PUT /api/v2/admin/clusters/{clusterId}` — 通配段匹配单个路径段，handler 用 `r.PathValue("clusterId")` 取值（集群注册表、通知设置接口使用）
- 固定路径与同级的通配路径并存时（`/api/v2/notify/rules/order` 与 `/api/v2/notify/rules/{id}`），固定路径优先；
  固定路径没声明的方法由通配路由接走（如 `DELETE /rules/order` 落到 `DELETE /rules/{id}`，id 不是数字 → 400）
- 路径命中但方法没声明 → ServeMux 返回 405（带 `Allow` 头），请求到不了 handler
- `/x` 没声明某方法、而 `/x/` 声明了 → ServeMux 307 到 `/x/`，仍落在已声明的路由上

Handler 内部从 URL 路径提取资源名称：`strings.TrimPrefix(r.URL.Path, "/api/v2/pods/")`

---

## 6. Handler 包映射

| 包（`gateway/handler/`） | 职责 |
|------|------|
| 根包 | 概览、集群、事件、运维操作（`ops.go`）、当前主体（`auth_me.go`）、权限表（`auth_permissions.go`） |
| `k8s/` | 各 K8s 资源列表 / 详情、Secret 列表、ConfigMap 详情 |
| `observe/` | ClickHouse 按需查询：Metrics / Logs / Traces / SLO、节点硬件指标、数据新鲜度 |
| `slo/` | SLO 域名、延迟分布、目标、状态历史 |
| `aiops/` | 依赖图、基线、风险、事件、AI 增强与深度分析、AI 对话、AI 错误码 |
| `admin/` | 用户、指令、通知、AI Provider / 设置 / 预算 / 报告、审计、部署、集群注册表与 Agent 令牌 |
| `settings/` | GitHub 连接与仓库 |

OIDC 端点的实现在 `atlhyper_master_v2/auth/oidc/`，由 Router 注入。
