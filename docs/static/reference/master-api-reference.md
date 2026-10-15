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
| **2** | Operator | `secret.read` `pod.logs.read` `cluster.operate` `audit.read` `notify.manage` `ai.analyze` `ai.settings.read` |
| **3** | Admin | `user.manage` `ai.settings.write` `github.manage` `deploy.manage` |

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
| `open(path, h)` | 无需主体（健康检查、登录、权限查询、OIDC） |
| `openAudited(path, action, resource, h)` | 无需主体 + 审计（登录、第三方回跳） |
| `route(perm, path, h)` | 需要权限 |
| `routeAudited(perm, path, action, resource, h)` | 需要权限 + 审计 |

---

## 3. API 端点清单（按权限等级）

> 路径以 `/` 结尾表示前缀匹配（如 `/api/v2/pods/` 捕获 `/api/v2/pods/{name}`）。审计列为 `action / resource`。

### 3.1 无需主体

| 路径 | 审计 | Handler | 说明 |
|------|------|---------|------|
| `/health` | — | `healthCheck` | 健康检查 |
| `/api/v2/auth/me` | — | `AuthMeHandler.Get` | 当前主体与权限列表 |
| `/api/v2/auth/permissions` | — | `GetPermissionTable` | 完整权限表（角色说明页用） |
| `/api/v2/user/login` | login / user | `UserHandler.Login` | 本地账号登录（外部身份账号不可用密码登录） |
| `/api/v2/auth/providers` | — | `oidc.Providers` | 外部登录方式（未启用时为空列表） |
| `/api/v2/auth/oidc/start` | — | `oidc.Start` | 发起 OIDC 登录（仅启用时注册） |
| `/api/v2/auth/oidc/callback` | login / user | `oidc.Callback` | OIDC 回调（仅启用时注册） |
| `/api/github/callback` | create / github | `GitHubHandler.Callback` | GitHub OAuth 回跳（仅配置 GitHub 时注册） |

### 3.2 0 级：集群状态与形状

| 权限 | 路径 | Handler |
|------|------|---------|
| `cluster.read` | `/api/v2/overview` | `OverviewHandler.Get` |
| `cluster.read` | `/api/v2/clusters` · `/api/v2/clusters/` | `ClusterHandler.List` / `Get` |
| `cluster.read` | `/api/v2/{pods,nodes,deployments,daemonsets,statefulsets,services,ingresses,jobs,cronjobs,pvs,pvcs,network-policies,resource-quotas,limit-ranges,service-accounts,namespaces}` 及各自 `/` 前缀 | 各资源 `List` / `Get`（env 字面值裁剪） |
| `cluster.read` | `/api/v2/configmaps` | `ConfigMapHandler.List`（仅名字与键数；详情见 2 级） |
| `events.read` | `/api/v2/events` · `/api/v2/events/by-resource` | `EventHandler.List` / `ListByResource` |
| `metrics.read` | `/api/v2/node-metrics` · `/api/v2/node-metrics/` | `NodeMetricsHandler.Route` |
| `metrics.read` | `/api/v2/observe/freshness` | `ObserveHandler.Freshness` |
| `metrics.read` | `/api/v2/observe/metrics/{summary,nodes,hardware,compare}` · `/api/v2/observe/metrics/nodes/` | `ObserveHandler.Metrics*` |
| `slo.read` | `/api/v2/slo/domains` · `/domains/v2` · `/domains/detail` · `/domains/history` · `/domains/latency` · `/targets` · `/status-history` | `SLOHandler.*` |
| `slo.read` | `/api/v2/observe/slo/{summary,ingress,timeseries}` | `ObserveHandler.SLO*` |
| `apm.summary.read` | `/api/v2/observe/traces/{services,stats,topology,operations}` · `/api/v2/observe/traces/services/` | `ObserveHandler.Traces*` / `APMServiceSeries` |
| `logs.stats.read` | `/api/v2/observe/logs/{summary,histogram}` | `ObserveHandler.LogsSummary` / `LogsHistogram` |
| `aiops.read` | `/api/v2/aiops/graph` · `/graph/trace` · `/baseline` | `AIOpsGraphHandler` / `AIOpsBaselineHandler` |
| `aiops.read` | `/api/v2/aiops/risk/{cluster,entities,entity}` | `AIOpsRiskHandler.*` |
| `aiops.read` | `/api/v2/aiops/incidents` · `/incidents/stats` · `/incidents/patterns` · `/incidents/` | `AIOpsIncidentHandler.*` |

### 3.3 1 级：内容级数据（Viewer）

| 权限 | 路径 | Handler |
|------|------|---------|
| `workload.env.read` | （不对应独立路由）资源详情中的 env 字面值 | `RedactEnvValues` 裁剪 |
| `logs.read` | `/api/v2/observe/logs/query` | `ObserveHandler.LogsQuery` |
| `traces.read` | `/api/v2/observe/traces` · `/api/v2/observe/traces/` · `/api/v2/observe/traces/correlations` | `ObserveHandler.TracesList` / `TracesDetail` / `TracesCorrelations` |
| `commands.read` | `/api/v2/commands/history` · `/api/v2/commands/` | `CommandHandler.ListHistory` / `GetStatus`（不含输出） |
| `ai.chat` | `/api/v2/ai/conversations` · `/api/v2/ai/conversations/` · `/api/v2/ai/chat` | `AIHandler.*`（SSE；删除会话在 handler 内另需 Operator） |
| `github.status.read` | `/api/github/connection` | `GitHubHandler.Connection` |
| `deploy.history.read` | `/api/deploy/history` | `DeployHandler.History` |

### 3.4 2 级：凭据与写操作（Operator）

| 权限 | 路径 | 审计 | Handler |
|------|------|------|---------|
| `secret.read` | `/api/v2/configmaps/` | — | `ConfigMapHandler.Get` |
| `secret.read` | `/api/v2/secrets` | — | `SecretHandler.List` |
| `secret.read` | `/api/v2/ops/configmaps/data` | read / configmap | `OpsHandler.ConfigMapData` |
| `secret.read` | `/api/v2/ops/secrets/data` | read / secret | `OpsHandler.SecretData` |
| `pod.logs.read` | `/api/v2/ops/pods/logs` | read / pod | `OpsHandler.PodLogs` |
| `cluster.operate` | `/api/v2/commands` | execute / command | `CommandHandler.Create` |
| `cluster.operate` | `/api/v2/ops/pods/restart` | execute / pod | `OpsHandler.PodRestart` |
| `cluster.operate` | `/api/v2/ops/deployments/{scale,restart,image}` | execute / deployment | `OpsHandler.Deployment*` |
| `cluster.operate` | `/api/v2/ops/nodes/{cordon,uncordon}` | execute / node | `OpsHandler.NodeCordon` / `NodeUncordon` |
| `audit.read` | `/api/v2/audit/logs` | — | `AuditHandler.List` |
| `notify.manage` | `/api/v2/notify/channels` | — | `NotifyHandler.ListChannels` |
| `notify.manage` | `/api/v2/notify/channels/` | update / notify | `NotifyHandler.ChannelHandler` |
| `ai.analyze` | `/api/v2/aiops/ai/{summarize,recommend,reports}` | — | `AIOpsAIHandler.*` |
| `ai.analyze` | `/api/v2/aiops/ai/reports/` | read / ai_report | `AIOpsAIHandler.ReportDetailHandler` |
| `ai.analyze` | `/api/v2/aiops/ai/analyze` | execute / ai_analysis | `AIOpsAIHandler.AnalyzeHandler` |
| `ai.analyze` | `/api/v2/aiops/ai/analyze/status` | — | `AIOpsAIHandler.AnalyzeStatus` |
| `ai.settings.read` | `/api/v2/ai/{providers,settings,roles,budgets,reports}` | — | `AIProviderHandler.*` |

### 3.5 3 级：系统配置（Admin）

| 权限 | 路径 | 审计 | Handler |
|------|------|------|---------|
| `user.manage` | `/api/v2/user/list` | — | `UserHandler.List` |
| `user.manage` | `/api/v2/user/register` | create / user | `UserHandler.Register` |
| `user.manage` | `/api/v2/user/{update-role,update-status}` | update / user | `UserHandler.UpdateRole` / `UpdateStatus` |
| `user.manage` | `/api/v2/user/delete` | delete / user | `UserHandler.Delete` |
| `ai.settings.write` | `/api/v2/ai/providers/` | update / ai_provider | `AIProviderHandler.ProviderHandler` |
| `ai.settings.write` | `/api/v2/ai/settings/` | update / ai_settings | `AIProviderHandler.SettingsHandler` |
| `ai.settings.write` | `/api/v2/ai/budgets/` | update / ai_budget | `AIProviderHandler.BudgetHandler` |
| `github.manage` | `/api/github/connect` | create / github | `GitHubHandler.Connect` |
| `github.manage` | `/api/github/disconnect` | delete / github | `GitHubHandler.Disconnect` |
| `github.manage` | `/api/github/repos` · `/api/github/repos/` | — | `GitHubHandler.Repos` / `RepoSubRoute` |
| `deploy.manage` | `/api/deploy/{config,kustomize-paths,test-connection,status,sync,rollback}` | — | `DeployHandler.*` |

> GitHub / 部署类路由仅在配置了 GitHub 客户端时注册；AI 对话路由仅在 AI 服务启用时注册。

---

## 4. 审计覆盖

审计记录无论成功还是权限不足都会写入。覆盖范围 = 上表「审计」列非空的路由：

- **登录**：本地登录、OIDC 回调
- **敏感读**：ConfigMap / Secret 内容、Pod 日志、AI 报告详情
- **写操作**：指令下发、Pod / Deployment / Node 操作、通知渠道、AI 分析触发
- **系统配置**：用户管理、AI Provider / 设置 / 预算、GitHub 连接（含 OAuth 回跳）

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

HTTP 状态码：200 成功，400 参数错误，401 未登录且所需权限高于 0 级，403 已登录但权限不足，404 未找到，409 配置冲突（如 AI 角色未分配 Provider），500 内部错误，502 上游失败，504 超时。

AI 类接口失败时另带稳定错误码 `code`（`ai_role_unassigned` / `ai_budget_exhausted` / `ai_timeout` / `ai_upstream`），前端按码显示文案。

### 路由匹配规则

Go `http.ServeMux` 的匹配规则：
- `/api/v2/pods` — 精确匹配列表路由
- `/api/v2/pods/` — 前缀匹配，捕获 `/api/v2/pods/{name}`

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
| `admin/` | 用户、指令、通知、AI Provider / 设置 / 预算 / 报告、审计、部署 |
| `settings/` | GitHub 连接与仓库 |

OIDC 端点的实现在 `atlhyper_master_v2/auth/oidc/`，由 Router 注入。
