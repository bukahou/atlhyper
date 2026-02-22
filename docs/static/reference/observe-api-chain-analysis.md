# Observe API 全链路参数对齐分析

> 逐个端点追踪 Web → Master → Agent → ClickHouse SQL 的完整调用链，验证参数传递是否正确。
> 生成日期：2026-02-22

---

## 目录

1. [调用链结构](#1-调用链结构)
2. [Traces（APM）— 4 个端点](#2-tracesapm-4-个端点)
3. [Logs — 1 个端点](#3-logs-1-个端点)
4. [Metrics — 4 个端点](#4-metrics-4-个端点)
5. [SLO — 5 个端点](#5-slo-5-个端点)
6. [汇总](#6-汇总)
7. [根因分类](#7-根因分类)

---

## 1. 调用链结构

```
Web (api/observe.ts)
  │ GET/POST query params 或 JSON body
  ↓
Master (gateway/handler/observe.go)
  │ 提取参数 → 构建 Command{Action, Params}
  │ → MQ 入队 → 同步等待结果 (30s)
  ↓
Agent (service/command/ch_query.go)
  │ 解析 Command.Params (getStringParam / getIntParam / getFloat64Param / getDurationParam)
  │ → 调用 Repository 方法
  ↓
ClickHouse (repository/ch/query/*.go)
  │ 执行 SQL
  ↓
JSON 透传回 Web
```

**关键类型转换节点：**
- GET query params → Master 读取 → **全部是 string**
- POST JSON body → Master json.Decode → **数字是 float64，字符串是 string**
- Agent `getIntParam` 只处理 `float64` 和 `int` 类型 → **string 类型会跳过，返回默认值**

---

## 2. Traces（APM）— 4 个端点

### 2.1 GET /api/v2/observe/traces — Trace 列表

| 层 | 代码位置 | 发送/接收的参数 |
|---|---------|----------------|
| **Web** | `api/observe.ts:164-178` | query params: `service?, operation?, min_duration?, max_duration?, limit?, offset?, start_time?, end_time?` (全部 string) |
| **Master** | `observe.go:294-316` | 原样转发 8 个 query param 到 `Command.Params` (string 值) |
| **Agent** | `ch_query.go:24-30` | 读取: `service`, `min_duration_ms`, `limit`, `since` |

**逐参数对照：**

| Web 发送 | Master 转发 | Agent 读取 | 匹配 | 说明 |
|---------|-----------|-----------|------|------|
| `service` (string) | `service` (string) | `getStringParam("service")` | ✅ | |
| `min_duration` (string) | `min_duration` (string) | `getFloat64Param("min_duration_ms")` | ❌ | 参数名不匹配 + 类型不匹配 |
| `max_duration` (string) | `max_duration` (string) | 不读取 | ❌ | Agent 忽略 |
| `limit` (string "50") | `limit` (string "50") | `getIntParam("limit", 50)` | ❌ | string 无法匹配 float64/int 类型断言 |
| `offset` (string) | `offset` (string) | 不读取 | ❌ | Agent 忽略 |
| `operation` (string) | `operation` (string) | 不读取 | ❌ | Agent 忽略 |
| `start_time` (string) | `start_time` (string) | 不读取 | ❌ | Agent 忽略 |
| `end_time` (string) | `end_time` (string) | 不读取 | ❌ | Agent 忽略 |
| 不发送 | 不发送 | `getDurationParam("since", 5min)` | ❌ | 永远使用默认值 5 分钟 |

**结果：** 只有 `service` 过滤有效。其余 8 个参数全部失效，但不会报错（静默使用默认值）。

---

### 2.2 GET /api/v2/observe/traces/services — 服务列表

| 层 | 参数 | 匹配 |
|---|------|------|
| Web | `cluster_id` | |
| Master | `{"sub_action": "list_services"}` | |
| Agent | `ListServices(ctx)` — 无额外参数 | ✅ |

---

### 2.3 GET /api/v2/observe/traces/topology — 拓扑图

| 层 | 参数 | 匹配 |
|---|------|------|
| Web | `cluster_id, time_range?` | |
| Master | `{"sub_action": "get_topology", "time_range": "1h"}` | |
| Agent | `GetTopology(ctx)` — 无参数，SQL 硬编码 `INTERVAL 5 MINUTE` | ⚠️ |

**结果：** `time_range` 被忽略，始终查询最近 5 分钟。

---

### 2.4 GET /api/v2/observe/traces/{traceId} — Trace 详情

| 层 | 参数 | 匹配 |
|---|------|------|
| Web | `cluster_id`, traceId (URL 路径) | |
| Master | `{"trace_id": traceID}` | |
| Agent | `getStringParam("trace_id")` → `GetTraceDetail(ctx, traceID)` | ✅ |

---

## 3. Logs — 1 个端点

### 3.1 POST /api/v2/observe/logs/query — 日志查询

| 层 | 代码位置 | 参数 |
|---|---------|------|
| **Web** | `api/observe.ts:146-157` | POST JSON body: `{ cluster_id, query?, service?, level?, scope?, limit?, offset?, since? }` |
| **Master** | `observe.go:266-287` | body JSON 解析 → 删除 `cluster_id` → 透传 |
| **Agent** | `ch_query.go:57-73` | 逐个读取所有参数 |

**逐参数对照：**

| Web 发送 | Agent 读取 | 匹配 | 说明 |
|---------|-----------|------|------|
| `query` (string) | `getStringParam("query")` | ✅ | |
| `service` (string) | `getStringParam("service")` | ✅ | |
| `level` (string) | `getStringParam("level")` | ✅ | |
| `scope` (string) | `getStringParam("scope")` | ✅ | |
| `limit` (number → float64) | `getIntParam("limit", 50)` | ✅ | JSON 数字 → float64 → getIntParam 处理 |
| `offset` (number → float64) | `getIntParam("offset", 0)` | ✅ | 同上 |
| `since` (string "15m") | `getDurationParam("since", 15min)` | ✅ | string → time.ParseDuration |

**结果：** ✅ **全部参数正确对齐。** 这是唯一全链路参数完全正确的模块。

**正常原因：** POST body 是 JSON，Go `json.Decode` 将数字反序列化为 `float64`，`getIntParam` 的 `float64` case 能正确处理。字符串保持 string 类型，`getStringParam` 和 `getDurationParam` 都能正确处理。

---

## 4. Metrics — 4 个端点

### 4.1 GET /api/v2/observe/metrics/summary — 集群概览

| 层 | 参数 | 匹配 |
|---|------|------|
| Master | `{"sub_action": "get_summary"}` | |
| Agent | `GetMetricsSummary(ctx)` → 调用 `ListAllNodeMetrics` 后聚合 | ✅ |

---

### 4.2 GET /api/v2/observe/metrics/nodes — 所有节点

| 层 | 参数 | 匹配 |
|---|------|------|
| Master | `{"sub_action": "list_all"}` | |
| Agent | `ListAllNodeMetrics(ctx)` | ✅ |

---

### 4.3 GET /api/v2/observe/metrics/nodes/{name} — 单节点

| 层 | 参数 | 匹配 |
|---|------|------|
| Master | `{"sub_action": "get_node", "node_name": "k8s-worker-1"}` | |
| Agent | `getStringParam("node_name")` → `GetNodeMetrics(ctx, nodeName)` | ✅ |

---

### 4.4 GET /api/v2/observe/metrics/nodes/{name}/series — 时序数据

| 层 | 代码位置 | 参数 |
|---|---------|------|
| **Web** | `api/observe.ts:134-139` | query params: `cluster_id`, `minutes?` |
| **Master** | `observe.go:239-250` | `{"sub_action": "get_series", "node_name": X}` + `minutes` (int, 通过 strconv.Atoi) |
| **Agent** | `ch_query.go:94-101` | 读取 `node_name`, `metric`, `since` |

**逐参数对照：**

| Web 发送 | Master 转发 | Agent 读取 | 匹配 | 说明 |
|---------|-----------|-----------|------|------|
| nodeName (URL 路径) | `node_name` (string) | `getStringParam("node_name")` | ✅ | |
| 不发送 | 不发送 | `getStringParam("metric")` — **必需** | ❌ 致命 | 永远为空 → 报错 |
| `minutes=30` (string) | `minutes` (int, Atoi 后) | `getDurationParam("since", 30min)` | ❌ | 参数名不匹配 |

**结果：** ❌ **致命 BUG — 此端点永远返回错误。**

Agent 要求 `metric` 和 `node_name` 都非空：
```go
if nodeName == "" || metric == "" {
    return nil, fmt.Errorf("node_name and metric are required")
}
```

Web 和 Master 都不知道要传 `metric` 参数（指定查询哪个指标的时序数据），导致 `metric` 永远为空。

---

## 5. SLO — 5 个端点

### 5.1 GET /api/v2/observe/slo/summary — SLO 摘要

| 层 | 参数 | 匹配 |
|---|------|------|
| Web | `cluster_id, time_range?` | |
| Master | `{"sub_action": "get_summary", "time_range": "1h"}` | |
| Agent | `GetSLOSummary(ctx)` — 无参数，内部硬编码调用 `ListIngressSLO(5min)` + `ListServiceSLO(5min)` | ⚠️ |

**结果：** `time_range` 被忽略，始终查询最近 5 分钟。

---

### 5.2 GET /api/v2/observe/slo/ingress — Ingress SLO

| 层 | 代码位置 | 参数 |
|---|---------|------|
| **Web** | `api/observe.ts:216-221` | `cluster_id, time_range?` |
| **Master** | `observe.go:390-410` | `{"sub_action": "list_ingress", "time_range": "1h"}` |
| **Agent** | `ch_query.go:118,121-122` | `getDurationParam("since", 5min)` → `ListIngressSLO(ctx, since)` |

**参数对照：**

| Master 转发 | Agent 读取 | 匹配 | 说明 |
|-----------|-----------|------|------|
| `time_range` (string) | `getDurationParam("since", 5min)` | ❌ | 参数名不匹配，永远 5 分钟 |

---

### 5.3 GET /api/v2/observe/slo/services — Service SLO

同 5.2 模式。Master 发 `time_range`，Agent 读 `since`。❌ 永远 5 分钟。

---

### 5.4 GET /api/v2/observe/slo/edges — 服务调用拓扑

同 5.2 模式。❌ 永远 5 分钟。

---

### 5.5 GET /api/v2/observe/slo/timeseries — SLO 时序

| 层 | 代码位置 | 参数 |
|---|---------|------|
| **Web** | `api/observe.ts:240-249` | `cluster_id, service?, time_range?, interval?` |
| **Master** | `observe.go:457-478` | `{"sub_action": "get_time_series", "service": X, "time_range": Y, "interval": Z}` |
| **Agent** | `ch_query.go:130-135` | 读取 `name` (必需), `since` |

**逐参数对照：**

| Master 转发 | Agent 读取 | 匹配 | 说明 |
|-----------|-----------|------|------|
| `service` (string) | `getStringParam("name")` — **必需** | ❌ 致命 | 参数名不匹配，永远为空 → 报错 |
| `time_range` (string) | `getDurationParam("since", 5min)` | ❌ | 参数名不匹配，永远 5 分钟 |
| `interval` (string) | 不读取 | ❌ | Agent 忽略 |

**结果：** ❌ **致命 BUG — 此端点永远返回错误。**

Agent 要求 `name` 非空：
```go
name := getStringParam(cmd.Params, "name")
if name == "" {
    return nil, fmt.Errorf("name is required")
}
```

Master 发的是 `service`，Agent 读的是 `name`，匹配不上。

---

## 6. 汇总

| # | 端点 | 状态 | 严重程度 |
|---|------|------|---------|
| 1 | `GET /observe/traces` | service 有效，其余 8 参数全失效 | 中 |
| 2 | `GET /observe/traces/services` | ✅ 正常 | — |
| 3 | `GET /observe/traces/topology` | time_range 被忽略 | 低 |
| 4 | `GET /observe/traces/{traceId}` | ✅ 正常 | — |
| 5 | `POST /observe/logs/query` | ✅ **全部正常** | — |
| 6 | `GET /observe/metrics/summary` | ✅ 正常 | — |
| 7 | `GET /observe/metrics/nodes` | ✅ 正常 | — |
| 8 | `GET /observe/metrics/nodes/{name}` | ✅ 正常 | — |
| 9 | `GET /observe/metrics/nodes/{name}/series` | **永远报错**（缺 metric） | 致命 |
| 10 | `GET /observe/slo/summary` | time_range 被忽略 | 低 |
| 11 | `GET /observe/slo/ingress` | time_range 永远失效 | 中 |
| 12 | `GET /observe/slo/services` | time_range 永远失效 | 中 |
| 13 | `GET /observe/slo/edges` | time_range 永远失效 | 中 |
| 14 | `GET /observe/slo/timeseries` | **永远报错**（service vs name） | 致命 |

**统计：**

| 分类 | 端点数 |
|------|--------|
| ✅ 全链路正常 | 7 / 14 |
| ⚠️ 部分参数失效（功能可用但降级） | 5 / 14 |
| ❌ 致命（端点完全不可用） | 2 / 14 |

---

## 7. 根因分类

### 7.1 参数名不匹配（Master 发 A，Agent 读 B）

| Master 发送 | Agent 读取 | 影响端点 |
|-----------|-----------|---------|
| `min_duration` | `min_duration_ms` | traces list |
| `minutes` | `since` | metrics series |
| `time_range` | `since` | SLO ingress / services / edges / timeseries / summary |
| `service` | `name` | SLO timeseries |

### 7.2 缺少参数（三层都没发）

| 缺少的参数 | 影响端点 | 说明 |
|-----------|---------|------|
| `metric` | metrics series | Agent 需要知道查询哪个指标的时序数据，但 API 设计上缺失此参数 |
| `since` | traces list | 时间范围参数从未发送，永远默认 5 分钟 |

### 7.3 类型不匹配（GET query params 全是 string）

| 参数 | Master 发送类型 | Agent 期望类型 | 影响端点 |
|------|---------------|--------------|---------|
| `limit` | string `"50"` | float64 / int | traces list |

**根因：** GET 请求的 query params 经 `r.URL.Query().Get()` 读取后是 string，直接放入 `map[string]any`。Agent 的 `getIntParam` / `getFloat64Param` 用类型断言（`v.(float64)`）匹配，string 不会命中任何 case，返回默认值。

POST body（如 Logs）不受影响，因为 `json.Decode` 将 JSON number 反序列化为 Go float64。

### 7.4 Agent 未读取的参数

| Master 发送但 Agent 忽略 | 影响端点 |
|------------------------|---------|
| `operation`, `max_duration`, `offset`, `start_time`, `end_time` | traces list |
| `time_range` (traces topology) | traces topology |
| `interval` | SLO timeseries |
