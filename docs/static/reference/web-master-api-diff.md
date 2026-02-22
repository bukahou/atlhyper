# Web ↔ Master API 差异分析

> 基于 `web-api-reference.md`（前端）和 `master-api-reference.md`（后端）独立对比。
> 生成日期：2026-02-22

---

## 目录

1. [Web 调用但 Master 未提供](#1-web-调用但-master-未提供)
2. [Master 提供但 Web 未使用](#2-master-提供但-web-未使用)
3. [权限标注不一致](#3-权限标注不一致)
4. [可观测性两套路径对比](#4-可观测性两套路径对比)
5. [细节差异](#5-细节差异)
6. [总结](#6-总结)

---

## 1. Web 调用但 Master 未提供

前端定义了 API 调用函数，但 Master `routes.go` 中无对应路由。**调用会返回 404。**

| Web 路径 | 方法 | Web 文件 | 函数名 | 说明 |
|----------|------|---------|--------|------|
| `/api/v2/apm/services` | GET | `api/apm.ts` | `getAPMServices()` | APM 服务列表 |
| `/api/v2/apm/traces/query` | POST | `api/apm.ts` | `queryTraces()` | APM Trace 查询 |
| `/api/v2/apm/traces/{traceId}` | GET | `api/apm.ts` | `getTraceDetail()` | APM Trace 详情 |
| `/api/v2/apm/topology` | GET | `api/apm.ts` | `getTopology()` | APM 拓扑图 |
| `/api/v2/logs/query` | POST | `api/log.ts` | `queryLogs()` | 独立日志查询 |

**共 5 个端点（2 个文件）。**

说明：前端同时通过 `api/observe.ts` 调用 `/api/v2/observe/traces/*` 和 `/api/v2/observe/logs/query`，这些 observe 路径 Master 已实现。`apm.ts` / `log.ts` 是另一套独立路径，Master 未注册。

---

## 2. Master 提供但 Web 未使用

Master 注册了路由，但前端 `api/` 和 `datasource/` 中无对应调用。

| Master 路径 | 方法 | 权限 | Handler | 说明 |
|------------|------|------|---------|------|
| `/health` | GET | Public | `healthCheck` | 健康检查（基础设施用） |
| `/api/v2/events/by-resource` | GET | Public | `EventHandler.ListByResource` | 按资源查询事件 |
| `/api/v2/commands` | POST | Operator | `CommandHandler.Create` | 通用指令下发 |
| `/api/v2/notify/channels/{type}/test` | POST | Operator | `NotifyHandler.ChannelHandler` | 测试通知渠道 |
| `/api/v2/configmaps/{uid}` | GET | Operator | `ConfigMapHandler.Get` | ConfigMap 详情 |
| `/api/v2/aiops/graph/trace` | GET | Public | `AIOpsGraphHandler.Trace` | 依赖图追踪 |
| `/api/v2/aiops/baseline` | GET | Public | `AIOpsBaselineHandler.Baseline` | 基线查询 |
| `/api/v2/aiops/incidents/patterns` | GET | Public | `AIOpsIncidentHandler.Patterns` | 事件模式识别 |

**共 8 个端点。**

备注：
- `/health` 属于基础设施探针，前端无需调用
- `POST /commands` 是通用指令入口，前端通过 `/api/v2/ops/*` 端点间接创建指令
- AIOps 的 3 个端点（graph/trace、baseline、incidents/patterns）可能是前端尚未对接

---

## 3. 权限标注不一致

| 端点 | Web 文档标注 | Master 实际权限 | 差异说明 |
|------|-------------|----------------|---------|
| `GET /api/v2/audit/logs` | Admin | **Operator** | Web 标记为 Admin，Master 实际 Operator 即可访问 |
| `GET /api/v2/secrets` | 无标注（与 configmaps 并列） | **Operator** | Web 将其与 Public 的 configmaps 放在同一节，未标注权限要求 |
| `GET /api/v2/notify/channels` | 无标注 | **Operator** | Web 未说明需要认证 |
| `GET /api/v2/commands/history` | 无标注 | **Public** | 一致，但 Web 未显式说明 |

---

## 4. 可观测性两套路径对比

前端存在两套可观测性 API，分属不同文件：

| 功能 | observe.ts 路径 (ClickHouse 按需) | 独立 API 路径 | Master 实现 |
|------|----------------------------------|--------------|------------|
| **Traces** | `GET /api/v2/observe/traces/*`（4 端点） | `GET/POST /api/v2/apm/*`（4 端点） | observe ✅ / apm ❌ |
| **Logs** | `POST /api/v2/observe/logs/query` | `POST /api/v2/logs/query` | observe ✅ / logs ❌ |
| **SLO** | `GET /api/v2/observe/slo/*`（5 端点） | `GET/PUT /api/v2/slo/*`（8 端点） | observe ✅ / slo ✅ |

- `observe.ts` 走 Master Command → Agent → ClickHouse 查询路径，Master 全部实现
- `apm.ts` / `log.ts` 的独立路径 Master 未实现（调用会 404）
- `slo.ts` 两套路径 Master 均已实现（observe 是 ClickHouse 实时查询，slo 是数据库持久化数据）

---

## 5. 细节差异

### 5.1 ConfigMap 详情路径参数

- Master 使用 `{uid}` 作为 ConfigMap 详情的路径参数：`GET /api/v2/configmaps/{uid}`
- 其他所有资源统一使用 `{name}` 作为路径参数
- Web 未定义 ConfigMap 详情调用（通过 `POST /ops/configmaps/data` 获取数据）

### 5.2 Secret 无详情端点

- Master 只提供 `GET /api/v2/secrets`（列表），无 `GET /api/v2/secrets/{name}`
- 实际 Secret 数据通过 `POST /api/v2/ops/secrets/data` 获取（Operator + 审计）
- Web 侧同样无 Secret 详情调用，两端一致

### 5.3 Event 查询方式差异

- Web 使用查询参数区分：`GET /api/v2/events?source=history`（历史告警）
- Master 另有独立端点：`GET /api/v2/events/by-resource`（按资源查询）
- Web 未使用 `by-resource` 端点

---

## 6. 总结

| 类别 | 数量 | 影响 |
|------|------|------|
| **Web 有但 Master 没有** | 5 个端点 | 高 — 前端调用返回 404 |
| **Master 有但 Web 没用** | 8 个端点 | 低 — 功能可用但前端未对接 |
| **权限标注不一致** | 2 处关键差异 | 中 — 前端权限感知与后端实际不匹配 |
| **两套可观测性路径** | Traces/Logs 独立路径未实现 | 中 — 需决定实现或删除 |
| **细节差异** | 3 处 | 低 — 不影响现有功能 |
