# Web 前端 API 参考文档

> 本文档基于 `atlhyper_web/src/` 源码梳理，记录前端实际调用的所有 API 端点、数据源代理机制和文件结构。
> 不包含后端视角的信息，仅反映前端代码现状。

---

## 目录

1. [架构概览](#1-架构概览)
2. [请求框架 (request.ts)](#2-请求框架)
3. [数据源代理层 (datasource/)](#3-数据源代理层)
4. [API 端点完整清单](#4-api-端点完整清单)
5. [前端 API 文件索引](#5-前端-api-文件索引)
6. [前端数据处理现状](#6-前端数据处理现状)

---

## 1. 架构概览

```
页面组件
   │
   ├── 有 datasource 代理的模块 ──→ datasource/ ──→ mock 或 api/
   │                                   │
   │                                   ├─ getDataSourceMode(key) === "mock" → mock/
   │                                   └─ getDataSourceMode(key) === "api"  → api/
   │
   └── 无 datasource 代理的模块 ──→ 直接调用 api/
```

### 三层结构

| 层 | 路径 | 职责 |
|----|------|------|
| **config** | `config/data-source.ts` | 模块注册表 + localStorage 持久化，控制 mock/api 切换 |
| **datasource** | `datasource/*.ts` | 代理层，根据配置透明路由到 mock 或 api |
| **api** | `api/*.ts` | 实际的 HTTP 请求，调用后端 REST API |

### 有 datasource 代理的模块

| datasource 文件 | 模块 key | 代理的 API 文件 |
|-----------------|----------|----------------|
| `datasource/cluster.ts` | pod, node, deployment, service, namespace, ingress, event, daemonset, statefulset, job, cronjob, pv, pvc, netpol, quota, limit, sa | `api/pod.ts`, `api/node.ts`, `api/deployment.ts`, `api/service.ts`, `api/namespace.ts`, `api/ingress.ts`, `api/event.ts`, `api/workload.ts`, `api/cluster-resources.ts` |
| `datasource/metrics.ts` | metrics | `api/node-metrics.ts` |
| `datasource/logs.ts` | logs | `api/observe.ts` (queryLogs) |
| `datasource/apm.ts` | apm | `api/observe.ts` (traces) |
| `datasource/slo.ts` | slo | `api/observe.ts` (SLO) |
| `datasource/mesh.ts` | — | `api/mesh.ts` |
| `datasource/overview.ts` | overview | `api/overview.ts` |

### 无 datasource 代理的模块（直接调用 api/）

`api/auth.ts`, `api/ai.ts`, `api/ai-provider.ts`, `api/settings.ts`, `api/notify.ts`, `api/commands.ts`, `api/cluster.ts`, `api/slo.ts`, `api/apm.ts`, `api/log.ts`, `api/aiops.ts`

---

## 2. 请求框架

**文件：** `api/request.ts`

### Axios 配置

| 项目 | 值 |
|------|-----|
| baseURL | `env.apiUrl`（环境变量） |
| timeout | 30000ms |
| Content-Type | application/json |

### 拦截器

**请求拦截器：** 从 `localStorage.getItem("token")` 注入 `Authorization: Bearer {token}`

**响应拦截器：**

| HTTP 状态码 | 处理 |
|------------|------|
| 2xx | 直接返回 response |
| 401 | `authErrorManager.emit({ type: "unauthorized" })` |
| 403 | `authErrorManager.emit({ type: "forbidden" })` |
| 其他 4xx/5xx | `Promise.reject(error)`，由调用方处理 |

### 便捷方法

```typescript
export const get  = <T>(url, params?) => request.get(url, { params });
export const post = <T>(url, data?)   => request.post(url, data);
export const put  = <T>(url, data?)   => request.put(url, data);
export const del  = <T>(url, params?) => request.delete(url, { params });
```

### 通用响应格式

```typescript
// 大多数 API
interface Response<T> { message: string; data: T; total?: number; }

// 可观测性 API
interface ObserveResponse<T> { message: string; data: T; }
```

---

## 3. 数据源代理层

### 模块注册表 (data-source.ts)

#### Observe 类 (5 个)

| key | category | hasMock | defaultMode |
|-----|----------|---------|-------------|
| metrics | observe | true | mock |
| logs | observe | true | mock |
| apm | observe | true | mock |
| slo | observe | true | mock |
| overview | observe | true | mock |

#### Cluster 类 (16 个)

| key | category | hasMock | defaultMode |
|-----|----------|---------|-------------|
| pod | cluster | true | mock |
| node | cluster | true | mock |
| deployment | cluster | true | mock |
| service | cluster | true | mock |
| namespace | cluster | true | mock |
| ingress | cluster | true | mock |
| event | cluster | true | mock |
| daemonset | cluster | true | mock |
| statefulset | cluster | true | mock |
| job | cluster | true | mock |
| cronjob | cluster | true | mock |
| pv | cluster | true | mock |
| pvc | cluster | true | mock |
| netpol | cluster | true | mock |
| quota | cluster | true | mock |
| limit | cluster | true | mock |
| sa | cluster | true | mock |

#### Admin 类 (4 个)

| key | category | hasMock | defaultMode |
|-----|----------|---------|-------------|
| users | admin | false | api |
| roles | admin | false | api |
| audit | admin | false | api |
| commands | admin | false | api |

#### Settings 类 (2 个)

| key | category | hasMock | defaultMode |
|-----|----------|---------|-------------|
| aiSettings | settings | false | api |
| notifications | settings | false | api |

#### AIOps 类 (4 个)

| key | category | hasMock | defaultMode |
|-----|----------|---------|-------------|
| risk | aiops | false | api |
| incidents | aiops | false | api |
| topology | aiops | false | api |
| chat | aiops | false | api |

### 切换机制

```typescript
// 读取（localStorage 持久化）
getDataSourceMode(key: string): "mock" | "api"

// 设置
setDataSourceMode(key: string, mode: "mock" | "api"): void
```

---

## 4. API 端点完整清单

### 4.1 集群管理

**文件：** `api/cluster.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/clusters` | `getClusterList()` | 集群列表 |
| GET | `/api/v2/clusters/{id}` | `getClusterDetail(id)` | 集群详情 |

**文件：** `api/overview.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/overview` | `getClusterOverview({ cluster_id })` | 集群概览 (Dashboard) |

---

### 4.2 K8s 资源查询

#### Pod — `api/pod.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/pods` | `getPodList(params)` | Pod 列表 |
| GET | `/api/v2/pods/{name}` | `getPodDetail({ ClusterID, Namespace, PodName })` | Pod 详情 |
| — | — | `getPodOverview({ ClusterID })` | 前端聚合：调用 getPodList 后统计 running/pending/failed/unknown |

查询参数：`cluster_id`, `namespace?`, `node?`, `phase?`, `limit?`, `offset?`

#### Node — `api/node.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/nodes` | `getNodeList({ cluster_id })` | Node 列表 |
| GET | `/api/v2/nodes/{name}` | `getNodeDetail({ ClusterID, NodeName })` | Node 详情 |
| — | — | `getNodeOverview({ ClusterID })` | 前端聚合：调用 getNodeList 后统计 ready/notReady |

#### Deployment — `api/deployment.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/deployments` | `getDeploymentList(params)` | Deployment 列表 |
| GET | `/api/v2/deployments/{name}` | `getDeploymentDetail({ ClusterID, Namespace, Name })` | Deployment 详情 |
| — | — | `getDeploymentOverview({ ClusterID })` | 前端聚合统计 |

查询参数：`cluster_id`, `namespace?`

#### Service — `api/service.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/services` | `getServiceList(params)` | Service 列表 |
| GET | `/api/v2/services/{name}` | `getServiceDetail({ ClusterID, Namespace, Name })` | Service 详情 |
| — | — | `getServiceOverview({ ClusterID })` | 前端聚合统计 |

#### Namespace — `api/namespace.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/namespaces` | `getNamespaceList({ cluster_id })` | Namespace 列表 |
| GET | `/api/v2/namespaces/{name}` | `getNamespaceDetail({ ClusterID, Name })` | Namespace 详情 |
| GET | `/api/v2/configmaps` | `getConfigMapList({ cluster_id, namespace? })` | ConfigMap 列表 |
| GET | `/api/v2/secrets` | `getSecretList({ cluster_id, namespace? })` | Secret 列表 |
| — | — | `getNamespaceOverview({ ClusterID })` | 前端聚合统计 |

#### Ingress — `api/ingress.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/ingresses` | `getIngressList(params)` | Ingress 列表（已按 host x path 展开多行） |
| GET | `/api/v2/ingresses/{name}` | `getIngressDetail({ ClusterID, Namespace, Name })` | Ingress 详情 |
| — | — | `getIngressOverview({ ClusterID })` | 前端聚合：统计 host、TLS、路径等 |

#### Event — `api/event.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/events` | `getEventList({ cluster_id, namespace?, type? })` | 实时事件列表 |
| GET | `/api/v2/events?source=history` | `getAlertList({ cluster_id, source: "history" })` | 历史告警列表 |
| — | — | `getEventOverview({ ClusterID })` | 前端聚合：统计 warning/error/info |

#### StatefulSet / DaemonSet — `api/workload.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/statefulsets` | `getStatefulSetList(params)` | StatefulSet 列表 |
| GET | `/api/v2/statefulsets/{name}` | `getStatefulSetDetail({ ClusterID, Namespace, Name })` | StatefulSet 详情 |
| GET | `/api/v2/daemonsets` | `getDaemonSetList(params)` | DaemonSet 列表 |
| GET | `/api/v2/daemonsets/{name}` | `getDaemonSetDetail({ ClusterID, Namespace, Name })` | DaemonSet 详情 |

#### Job / CronJob / PV / PVC / NetworkPolicy / ResourceQuota / LimitRange / ServiceAccount — `api/cluster-resources.ts`

| 方法 | 路径 | 函数名 |
|------|------|--------|
| GET | `/api/v2/jobs` | `getJobList(params)` |
| GET | `/api/v2/jobs/{name}` | `getJobDetail(params)` |
| GET | `/api/v2/cronjobs` | `getCronJobList(params)` |
| GET | `/api/v2/cronjobs/{name}` | `getCronJobDetail(params)` |
| GET | `/api/v2/pvs` | `getPVList(params)` |
| GET | `/api/v2/pvs/{name}` | `getPVDetail(params)` |
| GET | `/api/v2/pvcs` | `getPVCList(params)` |
| GET | `/api/v2/pvcs/{name}` | `getPVCDetail(params)` |
| GET | `/api/v2/network-policies` | `getNetworkPolicyList(params)` |
| GET | `/api/v2/network-policies/{name}` | `getNetworkPolicyDetail(params)` |
| GET | `/api/v2/resource-quotas` | `getResourceQuotaList(params)` |
| GET | `/api/v2/resource-quotas/{name}` | `getResourceQuotaDetail(params)` |
| GET | `/api/v2/limit-ranges` | `getLimitRangeList(params)` |
| GET | `/api/v2/limit-ranges/{name}` | `getLimitRangeDetail(params)` |
| GET | `/api/v2/service-accounts` | `getServiceAccountList(params)` |
| GET | `/api/v2/service-accounts/{name}` | `getServiceAccountDetail(params)` |

所有详情 API 查询参数：`cluster_id`, `namespace?`（PV 无 namespace）

---

### 4.3 操作接口 (Ops)

#### Pod 操作 — `api/pod.ts`

| 方法 | 路径 | 函数名 | 请求体 |
|------|------|--------|--------|
| POST | `/api/v2/ops/pods/logs` | `getPodLogs(data)` | `{ cluster_id, namespace, name, container?, tail_lines? }` |
| POST | `/api/v2/ops/pods/restart` | `restartPod(data)` | `{ cluster_id, namespace, name }` |

#### Deployment 操作 — `api/deployment.ts`

| 方法 | 路径 | 函数名 | 请求体 |
|------|------|--------|--------|
| POST | `/api/v2/ops/deployments/scale` | `scaleDeployment(data)` | `{ cluster_id, namespace, name, replicas }` |
| POST | `/api/v2/ops/deployments/restart` | `restartDeployment(data)` | `{ cluster_id, namespace, name }` |
| POST | `/api/v2/ops/deployments/image` | `updateDeploymentImage(data)` | `{ cluster_id, namespace, name, container, image }` |

#### Node 操作 — `api/node.ts`

| 方法 | 路径 | 函数名 | 请求体 |
|------|------|--------|--------|
| POST | `/api/v2/ops/nodes/cordon` | `cordonNode(data)` | `{ cluster_id, name }` |
| POST | `/api/v2/ops/nodes/uncordon` | `uncordonNode(data)` | `{ cluster_id, name }` |

#### 敏感数据获取 — `api/namespace.ts`

| 方法 | 路径 | 函数名 | 请求体 |
|------|------|--------|--------|
| POST | `/api/v2/ops/configmaps/data` | `getConfigMapData(data)` | `{ cluster_id, namespace, name }` |
| POST | `/api/v2/ops/secrets/data` | `getSecretData(data)` | `{ cluster_id, namespace, name }` |

---

### 4.4 可观测性 — ClickHouse 按需查询

**文件：** `api/observe.ts`

通过 Master Command 机制转发给 Agent 执行 ClickHouse 查询。

#### Metrics

| 方法 | 路径 | 函数名 |
|------|------|--------|
| GET | `/api/v2/observe/metrics/summary` | `getMetricsSummary(clusterId)` |
| GET | `/api/v2/observe/metrics/nodes` | `getMetricsNodes(clusterId)` |
| GET | `/api/v2/observe/metrics/nodes/{name}` | `getMetricsNode(clusterId, nodeName)` |
| GET | `/api/v2/observe/metrics/nodes/{name}/series` | `getMetricsNodeSeries(clusterId, nodeName, minutes?)` |

#### Logs

| 方法 | 路径 | 函数名 | 请求体 |
|------|------|--------|--------|
| POST | `/api/v2/observe/logs/query` | `queryLogs(params)` | `{ cluster_id, query?, service?, level?, scope?, limit?, offset?, since? }` |

#### Traces

| 方法 | 路径 | 函数名 |
|------|------|--------|
| GET | `/api/v2/observe/traces` | `getTracesList(clusterId, params?)` |
| GET | `/api/v2/observe/traces/services` | `getTracesServices(clusterId)` |
| GET | `/api/v2/observe/traces/topology` | `getTracesTopology(clusterId, timeRange?)` |
| GET | `/api/v2/observe/traces/{traceId}` | `getTraceDetail(clusterId, traceId)` |

Traces 查询参数：`service?`, `operation?`, `min_duration?`, `max_duration?`, `limit?`, `offset?`, `start_time?`, `end_time?`

#### SLO

| 方法 | 路径 | 函数名 |
|------|------|--------|
| GET | `/api/v2/observe/slo/summary` | `getSLOSummary(clusterId, timeRange?)` |
| GET | `/api/v2/observe/slo/ingress` | `getSLOIngress(clusterId, timeRange?)` |
| GET | `/api/v2/observe/slo/services` | `getSLOServices(clusterId, timeRange?)` |
| GET | `/api/v2/observe/slo/edges` | `getSLOEdges(clusterId, timeRange?)` |
| GET | `/api/v2/observe/slo/timeseries` | `getSLOTimeSeries(clusterId, { service?, time_range?, interval? })` |

---

### 4.5 节点硬件指标 (Agent 直推)

**文件：** `api/node-metrics.ts`

| 方法 | 路径 | 函数名 |
|------|------|--------|
| GET | `/api/v2/node-metrics` | `getClusterNodeMetrics(clusterId)` |
| GET | `/api/v2/node-metrics/{nodeName}` | `getNodeMetricsDetail(clusterId, nodeName)` |
| GET | `/api/v2/node-metrics/{nodeName}/history` | `getNodeMetricsHistory(clusterId, nodeName, hours?)` |

---

### 4.6 SLO 管理 (数据库持久化)

**文件：** `api/slo.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/slo/domains` | `getSLODomains(params)` | V1 — 按 service key |
| GET | `/api/v2/slo/domains/v2` | `getSLODomainsV2(params)` | V2 — 按真实域名分组 |
| GET | `/api/v2/slo/domains/detail` | `getSLODomainDetail({ clusterId, host, timeRange })` | 域名 SLO 详情 |
| GET | `/api/v2/slo/domains/history` | `getSLODomainHistory({ clusterId, host, timeRange })` | 域名历史数据 |
| GET | `/api/v2/slo/domains/latency` | `getSLOLatencyDistribution({ clusterId, domain, timeRange })` | 延迟分布 |
| GET | `/api/v2/slo/targets` | `getSLOTargets(clusterId?)` | SLO 目标列表 |
| PUT | `/api/v2/slo/targets` | `upsertSLOTarget(params)` | 创建/更新 SLO 目标 |
| GET | `/api/v2/slo/status-history` | `getSLOStatusHistory(params)` | 状态变更历史 |

---

### 4.7 服务网格

**文件：** `api/mesh.ts`

| 方法 | 路径 | 函数名 |
|------|------|--------|
| GET | `/api/v2/slo/mesh/topology` | `getMeshTopology({ cluster_id, time_range? })` |
| GET | `/api/v2/slo/mesh/service/detail` | `getMeshServiceDetail({ cluster_id, namespace, name, time_range? })` |

---

### 4.8 APM (独立 API，非 observe)

**文件：** `api/apm.ts`

| 方法 | 路径 | 函数名 |
|------|------|--------|
| GET | `/api/v2/apm/services` | `getAPMServices({ cluster_id, namespace? })` |
| POST | `/api/v2/apm/traces/query` | `queryTraces({ cluster_id, service?, statusCode?, minDurationMs?, maxDurationMs?, limit? })` |
| GET | `/api/v2/apm/traces/{traceId}` | `getTraceDetail({ cluster_id, traceId })` |
| GET | `/api/v2/apm/topology` | `getTopology({ cluster_id, namespace? })` |

---

### 4.9 日志查询 (独立 API，非 observe)

**文件：** `api/log.ts`

| 方法 | 路径 | 函数名 | 请求体 |
|------|------|--------|--------|
| POST | `/api/v2/logs/query` | `queryLogs(params)` | `{ cluster_id, search?, services?, severities?, scopes?, limit?, offset? }` |

---

### 4.10 用户认证与管理

**文件：** `api/auth.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| POST | `/api/v2/user/login` | `login(data)` | 登录 |
| GET | `/api/v2/user/list` | `getUserList()` | 用户列表 (Admin) |
| POST | `/api/v2/user/register` | `registerUser(data)` | 注册用户 (Admin) |
| POST | `/api/v2/user/update-role` | `updateUserRole(data)` | 更新角色 (Admin) |
| POST | `/api/v2/user/update-status` | `updateUserStatus(data)` | 更新状态 (Admin) |
| POST | `/api/v2/user/delete` | `deleteUser(id)` | 删除用户 (Admin) |
| GET | `/api/v2/audit/logs` | `getAuditLogs(params)` | 审计日志 (Admin) |

审计日志查询参数：`user_id?`, `source?`, `action?`, `since?`, `until?`, `limit?`, `offset?`

---

### 4.11 AI 对话

**文件：** `api/ai.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/ai/conversations` | `getConversations(limit, offset)` | 对话列表 |
| POST | `/api/v2/ai/conversations` | `createConversation(clusterId, title?)` | 创建对话 |
| DELETE | `/api/v2/ai/conversations/{id}` | `deleteConversation(id)` | 删除对话 |
| GET | `/api/v2/ai/conversations/{id}/messages` | `getMessages(conversationId)` | 消息历史 |
| POST | `/api/v2/ai/chat` | `streamChat(params, onChunk, onDone, onError, signal)` | **SSE 流式对话** |

SSE 流式对话使用原生 `fetch` + `ReadableStream`（Axios 不支持流式），支持 `AbortSignal` 中断。

SSE 事件类型：`text`, `tool_call`, `tool_result`, `done`, `error`

---

### 4.12 AI Provider 管理

**文件：** `api/ai-provider.ts`

| 方法 | 路径 | 函数名 |
|------|------|--------|
| GET | `/api/v2/ai/providers` | `listProviders()` |
| POST | `/api/v2/ai/providers` | `createProvider(data)` |
| GET | `/api/v2/ai/providers/{id}` | `getProvider(id)` |
| PUT | `/api/v2/ai/providers/{id}` | `updateProvider(id, data)` |
| DELETE | `/api/v2/ai/providers/{id}` | `deleteProvider(id)` |
| GET | `/api/v2/ai/active` | `getActiveConfig()` |
| PUT | `/api/v2/ai/active` | `updateActiveConfig(data)` |

---

### 4.13 AI 配置

**文件：** `api/settings.ts`

| 方法 | 路径 | 函数名 |
|------|------|--------|
| GET | `/api/v2/settings/ai` | `getAIConfig()` |
| PUT | `/api/v2/settings/ai/` | `updateAIConfig(data)` |
| POST | `/api/v2/settings/ai/test` | `testAIConnection()` |

---

### 4.14 通知渠道

**文件：** `api/notify.ts`

| 方法 | 路径 | 函数名 |
|------|------|--------|
| GET | `/api/v2/notify/channels` | `listChannels()` |
| GET | `/api/v2/notify/channels/{type}` | `getChannel(type)` |
| PUT | `/api/v2/notify/channels/slack` | `updateSlack({ enabled?, webhook_url? })` |
| PUT | `/api/v2/notify/channels/email` | `updateEmail(data)` |

---

### 4.15 指令历史

**文件：** `api/commands.ts`

| 方法 | 路径 | 函数名 |
|------|------|--------|
| GET | `/api/v2/commands/history` | `getCommandHistory(params)` |
| GET | `/api/v2/commands/{commandId}` | `getCommandStatus(commandId)` |

查询参数：`cluster_id?`, `source?`, `status?`, `action?`, `search?`, `limit?`, `offset?`

---

### 4.16 AIOps

**文件：** `api/aiops.ts`

| 方法 | 路径 | 函数名 | 说明 |
|------|------|--------|------|
| GET | `/api/v2/aiops/risk/cluster` | `getClusterRisk(cluster)` | 集群风险 |
| GET | `/api/v2/aiops/risk/entities` | `getEntityRisks(cluster, sort, limit)` | 实体风险列表 |
| GET | `/api/v2/aiops/risk/entity` | `getEntityRiskDetail(cluster, entityKey)` | 实体风险详情 |
| GET | `/api/v2/aiops/graph` | `getGraph(cluster)` | 依赖图 |
| GET | `/api/v2/aiops/incidents` | `getIncidents(params)` | 事件列表 |
| GET | `/api/v2/aiops/incidents/{id}` | `getIncidentDetail(id)` | 事件详情 |
| GET | `/api/v2/aiops/incidents/stats` | `getIncidentStats(cluster, period)` | 事件统计 |
| POST | `/api/v2/aiops/ai/summarize` | `summarizeIncident(incidentId)` | AI 事件总结 |
| POST | `/api/v2/aiops/ai/recommend` | `recommendActions(incidentId)` | AI 行动建议 |

---

## 5. 前端 API 文件索引

### api/ 目录 (22 个文件)

| 文件 | 端点数 | 说明 |
|------|--------|------|
| `request.ts` | — | Axios 封装、拦截器、便捷方法 |
| `auth.ts` | 7 | 登录、用户管理、审计日志 |
| `cluster.ts` | 2 | 集群列表/详情 |
| `overview.ts` | 1 | 集群概览 |
| `pod.ts` | 4+1 | Pod CRUD + 前端聚合 |
| `node.ts` | 4+1 | Node CRUD + 前端聚合 |
| `deployment.ts` | 5+1 | Deployment CRUD + 前端聚合 |
| `service.ts` | 2+1 | Service 查询 + 前端聚合 |
| `namespace.ts` | 6+1 | Namespace/ConfigMap/Secret + 前端聚合 |
| `ingress.ts` | 2+1 | Ingress 查询 + 前端聚合 |
| `event.ts` | 2+1 | Event 查询 + 前端聚合 |
| `workload.ts` | 4 | StatefulSet/DaemonSet |
| `cluster-resources.ts` | 16 | Job/CronJob/PV/PVC/NetworkPolicy/ResourceQuota/LimitRange/ServiceAccount |
| `commands.ts` | 2 | 指令历史/状态 |
| `node-metrics.ts` | 3 | 节点硬件指标 |
| `observe.ts` | 10 | ClickHouse: Metrics/Logs/Traces/SLO |
| `slo.ts` | 8 | SLO 管理 (数据库) |
| `mesh.ts` | 2 | 服务网格 |
| `apm.ts` | 4 | APM |
| `log.ts` | 1 | 日志查询 |
| `ai.ts` | 5 | AI 对话 (含 SSE 流式) |
| `ai-provider.ts` | 7 | AI Provider CRUD |
| `settings.ts` | 3 | AI 配置 |
| `notify.ts` | 4 | 通知渠道 |
| `aiops.ts` | 9 | AIOps 风险/事件/AI |

### datasource/ 目录 (8 个文件)

| 文件 | 代理函数数 | 说明 |
|------|-----------|------|
| `index.ts` | — | 统一导出 |
| `cluster.ts` | ~30 | 16 种 K8s 资源的 list/overview/detail + 写操作 |
| `metrics.ts` | 2 | 节点指标集群/历史 |
| `logs.ts` | 1 | 日志查询 |
| `apm.ts` | 4 | Traces 查询 |
| `slo.ts` | 3 | SLO (observe) 查询 |
| `mesh.ts` | 2 | 服务网格 |
| `overview.ts` | 1 | 集群概览 |

---

## 6. 前端数据处理现状

### 前端做了聚合统计的 API（违反大后端小前端原则）

| API 文件 | 函数 | 前端处理 | 说明 |
|---------|------|---------|------|
| `pod.ts` | `getPodOverview()` | 调用 `getPodList()` 后在前端统计 running/pending/failed/unknown | 应由后端返回聚合好的 Overview |
| `node.ts` | `getNodeOverview()` | 调用 `getNodeList()` 后在前端统计 ready/notReady | 同上 |
| `deployment.ts` | `getDeploymentOverview()` | 调用列表后前端统计 | 同上 |
| `service.ts` | `getServiceOverview()` | 调用列表后前端统计 | 同上 |
| `ingress.ts` | `getIngressOverview()` | 调用列表后前端统计 host/TLS/path | 同上 |
| `event.ts` | `getEventOverview()` | 调用列表后前端统计 warning/error/info | 同上 |

### 可观测性 API 的两套并行接口

前端存在两套可观测性相关 API，部分功能重叠：

| 功能 | observe.ts (ClickHouse 按需查询) | 独立 API 文件 |
|------|------|------|
| Traces | `getTracesList()`, `getTraceDetail()` | `apm.ts`: `queryTraces()`, `getTraceDetail()` |
| Logs | `queryLogs()` (observe) | `log.ts`: `queryLogs()` |
| SLO | `getSLOSummary()`, `getSLOIngress()` 等 | `slo.ts`: `getSLODomains()`, `getSLODomainsV2()` 等 |

- `observe.ts` 走 Master Command → Agent → ClickHouse 查询路径
- 独立 API 文件 (`apm.ts`, `log.ts`, `slo.ts`) 走 Master 直连数据库路径
- 两套 API 对应不同数据源和不同页面，并非真正重复
