# Master V2 API 参考文档

> 本文档基于 `atlhyper_master_v2/gateway/routes.go` 源码梳理，记录 Master 对 Web 前端提供的所有 HTTP API。
> 数据来源为 routes.go 中实际注册的路由 + handler 文件中的实现。

---

## 目录

1. [认证与权限模型](#1-认证与权限模型)
2. [中间件栈](#2-中间件栈)
3. [API 端点完整清单](#3-api-端点完整清单)
4. [审计覆盖](#4-审计覆盖)
5. [通用约定](#5-通用约定)
6. [Handler 文件映射](#6-handler-文件映射)

---

## 1. 认证与权限模型

### 权限级别

| 级别 | Role 值 | 说明 |
|------|---------|------|
| **Public** | — | 无需登录，所有只读查询对外开放 |
| **Viewer** | 1 | 等同游客，可使用 AI 对话 |
| **Operator** | 2 | 敏感信息查看、指令下发、运维操作 |
| **Admin** | 3 | 用户管理、系统配置 |

### 认证机制

- Token 通过 `Authorization: Bearer {token}` 传递
- 公开路由走 `publicMux`，不经过 `AuthRequired` 中间件
- 需认证路由走 `mux`，先经过 `AuthRequired`，再经过 `RequireMinRole`

---

## 2. 中间件栈

```
请求 → Logging → CORS → publicMux (匹配则直接处理)
                      → AuthRequired → mux → RequireMinRole → [Audit →] Handler
```

| 中间件 | 职责 |
|--------|------|
| `Logging` | 请求日志 |
| `CORS` | 跨域支持 |
| `AuthRequired` | Token 验证（仅非公开路由） |
| `RequireMinRole(n)` | 角色最低要求检查 |
| `Audit(action, resource)` | 审计记录（标记的路由） |

**审计中间件包装在权限检查之外**：无论认证成功或失败，都会记录审计日志。

---

## 3. API 端点完整清单

### 3.1 健康检查

| 方法 | 路径 | 权限 | 审计 | Handler |
|------|------|------|------|---------|
| GET | `/health` | Public | — | `healthCheck` |

---

### 3.2 用户认证

| 方法 | 路径 | 权限 | 审计 | Handler |
|------|------|------|------|---------|
| POST | `/api/v2/user/login` | Public | login / user | `UserHandler.Login` |

---

### 3.3 集群管理

| 方法 | 路径 | 权限 | 审计 | Handler |
|------|------|------|------|---------|
| GET | `/api/v2/overview` | Public | — | `OverviewHandler.Get` |
| GET | `/api/v2/clusters` | Public | — | `ClusterHandler.List` |
| GET | `/api/v2/clusters/{id}` | Public | — | `ClusterHandler.Get` |

---

### 3.4 K8s 资源查询（全部 Public）

#### 工作负载

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| GET | `/api/v2/pods` | `PodHandler.List` | Pod 列表 |
| GET | `/api/v2/pods/{name}` | `PodHandler.Get` | Pod 详情 |
| GET | `/api/v2/nodes` | `NodeHandler.List` | Node 列表 |
| GET | `/api/v2/nodes/{name}` | `NodeHandler.Get` | Node 详情 |
| GET | `/api/v2/deployments` | `DeploymentHandler.List` | Deployment 列表 |
| GET | `/api/v2/deployments/{name}` | `DeploymentHandler.Get` | Deployment 详情 |
| GET | `/api/v2/daemonsets` | `DaemonSetHandler.List` | DaemonSet 列表 |
| GET | `/api/v2/daemonsets/{name}` | `DaemonSetHandler.Get` | DaemonSet 详情 |
| GET | `/api/v2/statefulsets` | `StatefulSetHandler.List` | StatefulSet 列表 |
| GET | `/api/v2/statefulsets/{name}` | `StatefulSetHandler.Get` | StatefulSet 详情 |
| GET | `/api/v2/jobs` | `JobHandler.List` | Job 列表 |
| GET | `/api/v2/jobs/{name}` | `JobHandler.Get` | Job 详情 |
| GET | `/api/v2/cronjobs` | `CronJobHandler.List` | CronJob 列表 |
| GET | `/api/v2/cronjobs/{name}` | `CronJobHandler.Get` | CronJob 详情 |

#### 网络

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| GET | `/api/v2/services` | `ServiceHandler.List` | Service 列表 |
| GET | `/api/v2/services/{name}` | `ServiceHandler.Get` | Service 详情 |
| GET | `/api/v2/ingresses` | `IngressHandler.List` | Ingress 列表 |
| GET | `/api/v2/ingresses/{name}` | `IngressHandler.Get` | Ingress 详情 |

#### 存储

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| GET | `/api/v2/pvs` | `PVHandler.List` | PV 列表 |
| GET | `/api/v2/pvs/{name}` | `PVHandler.Get` | PV 详情 |
| GET | `/api/v2/pvcs` | `PVCHandler.List` | PVC 列表 |
| GET | `/api/v2/pvcs/{name}` | `PVCHandler.Get` | PVC 详情 |

#### 策略与配额

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| GET | `/api/v2/network-policies` | `NetworkPolicyHandler.List` | NetworkPolicy 列表 |
| GET | `/api/v2/network-policies/{name}` | `NetworkPolicyHandler.Get` | NetworkPolicy 详情 |
| GET | `/api/v2/resource-quotas` | `ResourceQuotaHandler.List` | ResourceQuota 列表 |
| GET | `/api/v2/resource-quotas/{name}` | `ResourceQuotaHandler.Get` | ResourceQuota 详情 |
| GET | `/api/v2/limit-ranges` | `LimitRangeHandler.List` | LimitRange 列表 |
| GET | `/api/v2/limit-ranges/{name}` | `LimitRangeHandler.Get` | LimitRange 详情 |
| GET | `/api/v2/service-accounts` | `ServiceAccountHandler.List` | ServiceAccount 列表 |
| GET | `/api/v2/service-accounts/{name}` | `ServiceAccountHandler.Get` | ServiceAccount 详情 |

#### 配置与命名空间

| 方法 | 路径 | 权限 | Handler | 说明 |
|------|------|------|---------|------|
| GET | `/api/v2/configmaps` | Public | `ConfigMapHandler.List` | ConfigMap 列表 |
| GET | `/api/v2/configmaps/{uid}` | **Operator** | `ConfigMapHandler.Get` | ConfigMap 详情 |
| GET | `/api/v2/secrets` | **Operator** | `SecretHandler.List` | Secret 列表 |
| GET | `/api/v2/namespaces` | Public | `NamespaceHandler.List` | Namespace 列表 |
| GET | `/api/v2/namespaces/{name}` | Public | `NamespaceHandler.Get` | Namespace 详情 |

---

### 3.5 事件查询（Public）

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| GET | `/api/v2/events` | `EventHandler.List` | 事件列表（实时 / 历史） |
| GET | `/api/v2/events/by-resource` | `EventHandler.ListByResource` | 按资源查询事件 |

---

### 3.6 指令系统

| 方法 | 路径 | 权限 | 审计 | Handler | 说明 |
|------|------|------|------|---------|------|
| GET | `/api/v2/commands/history` | Public | — | `CommandHandler.ListHistory` | 指令历史 |
| GET | `/api/v2/commands/{id}` | Public | — | `CommandHandler.GetStatus` | 指令状态 |
| POST | `/api/v2/commands` | Operator | execute / command | `CommandHandler.Create` | 下发指令 |

---

### 3.7 运维操作（Operator + 审计）

#### Pod 操作

| 方法 | 路径 | 审计 action | Handler | 说明 |
|------|------|------------|---------|------|
| POST | `/api/v2/ops/pods/logs` | read / pod | `OpsHandler.PodLogs` | 获取日志（同步等待） |
| POST | `/api/v2/ops/pods/restart` | execute / pod | `OpsHandler.PodRestart` | 重启 Pod |

#### Deployment 操作

| 方法 | 路径 | 审计 action | Handler | 说明 |
|------|------|------------|---------|------|
| POST | `/api/v2/ops/deployments/scale` | execute / deployment | `OpsHandler.DeploymentScale` | 扩缩容 |
| POST | `/api/v2/ops/deployments/restart` | execute / deployment | `OpsHandler.DeploymentRestart` | 滚动重启 |
| POST | `/api/v2/ops/deployments/image` | execute / deployment | `OpsHandler.DeploymentImage` | 更新镜像 |

#### Node 操作

| 方法 | 路径 | 审计 action | Handler | 说明 |
|------|------|------------|---------|------|
| POST | `/api/v2/ops/nodes/cordon` | execute / node | `OpsHandler.NodeCordon` | 封锁节点 |
| POST | `/api/v2/ops/nodes/uncordon` | execute / node | `OpsHandler.NodeUncordon` | 解封节点 |

#### 敏感数据获取

| 方法 | 路径 | 审计 action | Handler | 说明 |
|------|------|------------|---------|------|
| POST | `/api/v2/ops/configmaps/data` | read / configmap | `OpsHandler.ConfigMapData` | ConfigMap 数据 |
| POST | `/api/v2/ops/secrets/data` | read / secret | `OpsHandler.SecretData` | Secret 数据 |

---

### 3.8 SLO 监控（Public）

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| GET | `/api/v2/slo/domains` | `SLOHandler.Domains` | V1: 按 service key |
| GET | `/api/v2/slo/domains/v2` | `SLOHandler.DomainsV2` | V2: 按真实域名 |
| GET | `/api/v2/slo/domains/detail` | `SLOHandler.DomainDetail` | 域名详情 |
| GET | `/api/v2/slo/domains/history` | `SLOHandler.DomainHistory` | 域名历史数据 |
| GET | `/api/v2/slo/domains/latency` | `SLOHandler.LatencyDistribution` | 延迟分布 |
| GET/PUT | `/api/v2/slo/targets` | `SLOHandler.Targets` | SLO 目标 CRUD |
| GET | `/api/v2/slo/status-history` | `SLOHandler.StatusHistory` | 状态变更历史 |

---

### 3.9 服务网格（Public）

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| GET | `/api/v2/slo/mesh/topology` | `SLOMeshHandler.MeshTopology` | 拓扑图 |
| GET | `/api/v2/slo/mesh/service/detail` | `SLOMeshHandler.ServiceDetail` | 服务详情 |

---

### 3.10 节点硬件指标（Public）

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| GET | `/api/v2/node-metrics` | `NodeMetricsHandler.Route` | 集群所有节点指标 |
| GET | `/api/v2/node-metrics/{nodeName}` | `NodeMetricsHandler.Route` | 单节点指标 |
| GET | `/api/v2/node-metrics/{nodeName}/history` | `NodeMetricsHandler.Route` | 节点历史数据 |

---

### 3.11 可观测性查询 — ClickHouse 按需（Public）

通过 Master Command 机制转发给 Agent 执行 ClickHouse 查询，结果 JSON 透传。

#### Metrics

| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/api/v2/observe/metrics/summary` | `ObserveHandler.MetricsSummary` |
| GET | `/api/v2/observe/metrics/nodes` | `ObserveHandler.MetricsNodes` |
| GET | `/api/v2/observe/metrics/nodes/{name}` | `ObserveHandler.MetricsNodeRoute` |
| GET | `/api/v2/observe/metrics/nodes/{name}/series` | `ObserveHandler.MetricsNodeRoute` |

#### Logs

| 方法 | 路径 | Handler |
|------|------|---------|
| POST | `/api/v2/observe/logs/query` | `ObserveHandler.LogsQuery` |

#### Traces

| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/api/v2/observe/traces` | `ObserveHandler.TracesList` |
| GET | `/api/v2/observe/traces/services` | `ObserveHandler.TracesServices` |
| GET | `/api/v2/observe/traces/topology` | `ObserveHandler.TracesTopology` |
| GET | `/api/v2/observe/traces/{traceId}` | `ObserveHandler.TracesDetail` |

#### SLO (ClickHouse)

| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/api/v2/observe/slo/summary` | `ObserveHandler.SLOSummary` |
| GET | `/api/v2/observe/slo/ingress` | `ObserveHandler.SLOIngress` |
| GET | `/api/v2/observe/slo/services` | `ObserveHandler.SLOServices` |
| GET | `/api/v2/observe/slo/edges` | `ObserveHandler.SLOEdges` |
| GET | `/api/v2/observe/slo/timeseries` | `ObserveHandler.SLOTimeSeries` |

---

### 3.12 AIOps（Public 查询 + Operator AI 增强）

#### 依赖图（Public）

| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/api/v2/aiops/graph` | `AIOpsGraphHandler.Graph` |
| GET | `/api/v2/aiops/graph/trace` | `AIOpsGraphHandler.Trace` |

#### 基线（Public）

| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/api/v2/aiops/baseline` | `AIOpsBaselineHandler.Baseline` |

#### 风险评分（Public）

| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/api/v2/aiops/risk/cluster` | `AIOpsRiskHandler.ClusterRisk` |
| GET | `/api/v2/aiops/risk/entities` | `AIOpsRiskHandler.EntityRisks` |
| GET | `/api/v2/aiops/risk/entity` | `AIOpsRiskHandler.EntityRisk` |

#### 事件（Public）

| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/api/v2/aiops/incidents` | `AIOpsIncidentHandler.List` |
| GET | `/api/v2/aiops/incidents/stats` | `AIOpsIncidentHandler.Stats` |
| GET | `/api/v2/aiops/incidents/patterns` | `AIOpsIncidentHandler.Patterns` |
| GET | `/api/v2/aiops/incidents/{id}` | `AIOpsIncidentHandler.Detail` |

#### AI 增强（Operator）

| 方法 | 路径 | 权限 | Handler |
|------|------|------|---------|
| POST | `/api/v2/aiops/ai/summarize` | Operator | `AIOpsAIHandler.Summarize` |
| POST | `/api/v2/aiops/ai/recommend` | Operator | `AIOpsAIHandler.Recommend` |

---

### 3.13 AI 对话（需认证，Viewer+）

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| GET | `/api/v2/ai/conversations` | `AIHandler.Conversations` | 对话列表 |
| POST | `/api/v2/ai/conversations` | `AIHandler.Conversations` | 创建对话 |
| GET | `/api/v2/ai/conversations/{id}/messages` | `AIHandler.ConversationByID` | 消息历史 |
| DELETE | `/api/v2/ai/conversations/{id}` | `AIHandler.ConversationByID` | 删除对话 |
| POST | `/api/v2/ai/chat` | `AIHandler.Chat` | SSE 流式对话 |

注：AI 对话路由通过 `r.mux.HandleFunc` 直接注册，需要 `AuthRequired` 但不限制特定 Role（Viewer+ 可用）。`aiService != nil` 时才注册。

---

### 3.14 通知渠道（Operator）

| 方法 | 路径 | 权限 | 审计 | Handler | 说明 |
|------|------|------|------|---------|------|
| GET | `/api/v2/notify/channels` | Operator | — | `NotifyHandler.ListChannels` | 渠道列表 |
| GET | `/api/v2/notify/channels/{type}` | Operator | update / notify | `NotifyHandler.ChannelHandler` | 渠道详情 |
| PUT | `/api/v2/notify/channels/{type}` | Operator | update / notify | `NotifyHandler.ChannelHandler` | 更新渠道 |
| POST | `/api/v2/notify/channels/{type}/test` | Operator | update / notify | `NotifyHandler.ChannelHandler` | 测试渠道 |

注：通配路由 `/api/v2/notify/channels/` 处理 GET/PUT/POST 三种方法，Handler 内部按 `r.Method` 分发。

---

### 3.15 审计日志（Operator）

| 方法 | 路径 | 权限 | Handler |
|------|------|------|---------|
| GET | `/api/v2/audit/logs` | Operator | `AuditHandler.List` |

---

### 3.16 AI 配置（Operator 查看 / Admin 修改）

| 方法 | 路径 | 权限 | 审计 | Handler |
|------|------|------|------|---------|
| GET | `/api/v2/settings/ai` | Operator | — | `SettingsHandler.AIConfigHandler` |
| PUT | `/api/v2/settings/ai/` | Admin | update / ai_config | `SettingsHandler.AIConfigHandler` |
| POST | `/api/v2/settings/ai/test` | Admin | update / ai_config | `SettingsHandler.AIConfigHandler` |

---

### 3.17 AI Provider 管理（Operator 查看 / Admin 修改）

| 方法 | 路径 | 权限 | 审计 | Handler |
|------|------|------|------|---------|
| GET | `/api/v2/ai/providers` | Operator | — | `AIProviderHandler.ProvidersHandler` |
| POST | `/api/v2/ai/providers` | Admin | — | `AIProviderHandler.ProvidersHandler` |
| GET | `/api/v2/ai/providers/{id}` | Admin | update / ai_provider | `AIProviderHandler.ProviderHandler` |
| PUT | `/api/v2/ai/providers/{id}` | Admin | update / ai_provider | `AIProviderHandler.ProviderHandler` |
| DELETE | `/api/v2/ai/providers/{id}` | Admin | update / ai_provider | `AIProviderHandler.ProviderHandler` |
| GET | `/api/v2/ai/active` | Operator | — | `AIProviderHandler.ActiveConfigHandler` |
| PUT | `/api/v2/ai/active` | Admin | update / ai_provider | `AIProviderHandler.ActiveConfigHandler` |

注：
- `GET /api/v2/ai/providers` 和 `POST /api/v2/ai/providers` 在同一个 Handler 中按 Method 分发
- `ProvidersHandler` 在 Operator 块注册（GET 读取），但 POST 创建在 Handler 内部检查 Admin 权限
- `ProviderHandler` 在 Admin 审计块注册，GET/PUT/DELETE 都走 Admin 路径

---

### 3.18 用户管理（Admin）

| 方法 | 路径 | 审计 | Handler | 说明 |
|------|------|------|---------|------|
| GET | `/api/v2/user/list` | — | `UserHandler.List` | 用户列表 |
| POST | `/api/v2/user/register` | create / user | `UserHandler.Register` | 注册用户 |
| POST | `/api/v2/user/update-role` | update / user | `UserHandler.UpdateRole` | 更新角色 |
| POST | `/api/v2/user/update-status` | update / user | `UserHandler.UpdateStatus` | 更新状态 |
| POST | `/api/v2/user/delete` | delete / user | `UserHandler.Delete` | 删除用户 |

---

## 4. 审计覆盖

所有标记审计的操作，**无论认证成功或失败都会记录**。

| 路径 | action | resource |
|------|--------|----------|
| `/api/v2/user/login` | login | user |
| `/api/v2/commands` | execute | command |
| `/api/v2/ops/pods/logs` | read | pod |
| `/api/v2/ops/pods/restart` | execute | pod |
| `/api/v2/ops/deployments/scale` | execute | deployment |
| `/api/v2/ops/deployments/restart` | execute | deployment |
| `/api/v2/ops/deployments/image` | execute | deployment |
| `/api/v2/ops/nodes/cordon` | execute | node |
| `/api/v2/ops/nodes/uncordon` | execute | node |
| `/api/v2/ops/configmaps/data` | read | configmap |
| `/api/v2/ops/secrets/data` | read | secret |
| `/api/v2/notify/channels/{type}` | update | notify |
| `/api/v2/settings/ai/` | update | ai_config |
| `/api/v2/ai/providers/{id}` | update | ai_provider |
| `/api/v2/ai/active/` | update | ai_provider |
| `/api/v2/user/register` | create | user |
| `/api/v2/user/update-role` | update | user |
| `/api/v2/user/update-status` | update | user |
| `/api/v2/user/delete` | delete | user |

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

HTTP 状态码：200 成功，400 参数错误，401 未认证，403 权限不足，404 未找到，500 内部错误，504 超时。

### 路由匹配规则

Go `http.ServeMux` 的匹配规则：
- `/api/v2/pods` — 精确匹配列表路由
- `/api/v2/pods/` — 前缀匹配，捕获 `/api/v2/pods/{name}`

Handler 内部从 URL 路径提取资源名称：`strings.TrimPrefix(r.URL.Path, "/api/v2/pods/")`

---

## 6. Handler 文件映射

| 文件 | 路由数 | 职责 |
|------|--------|------|
| `overview.go` | 1 | 集群概览 Dashboard |
| `cluster.go` | 2 | 集群列表/详情 |
| `pod.go` | 2 | Pod 列表/详情 |
| `node.go` | 2 | Node 列表/详情 |
| `deployment.go` | 2 | Deployment 列表/详情 |
| `daemonset.go` | 2 | DaemonSet 列表/详情 |
| `statefulset.go` | 2 | StatefulSet 列表/详情 |
| `service.go` | 2 | Service 列表/详情 |
| `ingress.go` | 2 | Ingress 列表/详情 |
| `job.go` | 2 | Job 列表/详情 |
| `cronjob.go` | 2 | CronJob 列表/详情 |
| `pv.go` | 2 | PV 列表/详情 |
| `pvc.go` | 2 | PVC 列表/详情 |
| `network_policy.go` | 2 | NetworkPolicy 列表/详情 |
| `resource_quota.go` | 2 | ResourceQuota 列表/详情 |
| `limit_range.go` | 2 | LimitRange 列表/详情 |
| `service_account.go` | 2 | ServiceAccount 列表/详情 |
| `namespace.go` | 2 | Namespace 列表/详情 |
| `configmap.go` | 2 | ConfigMap 列表(Public)/详情(Operator) |
| `secret.go` | 1 | Secret 列表(Operator) |
| `event.go` | 2 | 事件列表/按资源查询 |
| `command.go` | 3 | 指令历史/状态/创建 |
| `ops.go` | 9 | Pod/Deployment/Node/ConfigMap/Secret 操作 |
| `slo.go` | 7 | SLO 域名查询/目标管理 |
| `slo_mesh.go` | 2 | 服务网格拓扑/详情 |
| `node_metrics.go` | 3 | 节点硬件指标 |
| `observe.go` | 13 | ClickHouse: Metrics/Logs/Traces/SLO |
| `aiops_graph.go` | 2 | 依赖图/追踪 |
| `aiops_baseline.go` | 1 | 基线查询 |
| `aiops_risk.go` | 3 | 风险评分 |
| `aiops_incident.go` | 4 | 事件管理 |
| `aiops_ai.go` | 2 | AI 总结/建议 |
| `ai.go` | 4 | AI 对话 (含 SSE) |
| `notify.go` | 4 | 通知渠道管理 |
| `settings.go` | 3 | AI 配置管理 |
| `ai_provider.go` | 7 | AI Provider CRUD |
| `audit.go` | 1 | 审计日志 |
| `user.go` | 6 | 用户认证/管理 |

**总计：约 105 个端点**（含同路径不同 Method 的计为多个）
