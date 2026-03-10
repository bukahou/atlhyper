# AI 工具增强 — OTel 查询能力与事件上下文丰富

> 状态：详细设计完成
> 创建：2026-02-27
> 更新：2026-03-09
> 前置：AI Chat 模块（已完成）、跨信号关联（已完成）

---

## 1. 问题陈述

AtlHyper AI 助手当前有 4 个工具，覆盖 K8s 资源查询和 AIOps 事件分析，
但**完全无法访问 APM Trace、OTel 日志、SLO 趋势和实体因果图数据**。

### 现有工具

| 工具 | 能力 | 数据来源 | 执行路径 |
|------|------|----------|----------|
| `query_cluster` | 查询 K8s 资源（get/list/describe/get_logs/get_events/get_configmap） | K8s API | Command → Agent → K8s API |
| `analyze_incident` | LLM 分析事件根因 | AIOps IncidentStore | 内存直读 + LLM |
| `get_cluster_risk` | 获取集群风险评分 + Top N 高风险实体 | AIOps Scorer | 内存直读 |
| `get_recent_incidents` | 获取最近事件列表 | AIOps IncidentStore | 内存直读 |

### 能力缺口

| 用户问题 | 期望行为 | 当前结果 |
|----------|---------|---------|
| "api-server 最近为什么延迟高？" | 查询 APM Traces，找到慢 Span | 只能 kubectl logs，看不到 Trace |
| "有哪些 ERROR 日志？" | 查询 OTel 结构化日志 | 只能 kubectl logs，无结构化搜索 |
| "geass-gateway 最近 7 天的 SLO？" | 查询 SLO 历史数据 | 无此工具 |
| "这个 Pod 为什么异常？" | 查看实体因果树 + 关联异常指标 | 只能看 Top N 列表，无单实体详情 |
| "分析这个事件的根因" | 上下文含 Trace/Log/SLO | 上下文只有 K8s 事件和风险数值 |

---

## 2. 架构设计

### 2.1 两条数据查询路径

```
路径 A（内存直读）— 纳秒延迟，数据范围受限于快照
┌──────────────────────────────────────────────────────┐
│  AI Tool Handler                                      │
│    ↓                                                  │
│  AIOps Engine / DataHub Store（Master 内存）           │
│    └─ OTelSnapshot: SLOWindows, RecentTraces, etc.    │
│    └─ Scorer: EntityRisk, CausalTree                  │
└──────────────────────────────────────────────────────┘

路径 B（Command 机制）— 毫秒延迟，支持任意时间范围
┌──────────────────────────────────────────────────────┐
│  AI Tool Handler                                      │
│    ↓ CreateCommand()                                  │
│  MQ (CommandBus)                                      │
│    ↓ WaitCommandResult()                              │
│  Agent ← Poll ─ Master                               │
│    ↓                                                  │
│  Agent CommandService.handleQuery*()                  │
│    ↓                                                  │
│  ClickHouse (otel_traces / otel_logs / node_metrics)  │
└──────────────────────────────────────────────────────┘
```

### 2.2 工具与路径对应

| 工具 | 查询路径 | 理由 |
|------|---------|------|
| `query_traces` | **路径 B** (Command) | 需要按条件过滤，数据在 ClickHouse |
| `query_logs` | **路径 B** (Command) | 需要全文搜索、级别/服务过滤 |
| `query_slo` | **路径 A** (内存) | OTelSnapshot.SLOWindows 已有 1d/7d/30d 预聚合 |
| `get_entity_detail` | **路径 A** (内存) | AIOps Engine 内存图已有完整因果数据 |
| 事件上下文丰富 | **路径 A** (内存) | 从 OTelSnapshot.RecentTraces/RecentLogs 过滤 |

### 2.3 复用关系

Agent 端 **零改动** — 5 个 ClickHouse 查询 Command 已全部实现：

| Action 常量 | Agent Handler | 位置 |
|-------------|--------------|------|
| `ActionQueryTraces` | `handleQueryTraces()` | `ch_query.go` |
| `ActionQueryTraceDetail` | `handleQueryTraceDetail()` | `ch_query.go` |
| `ActionQueryLogs` | `handleQueryLogs()` | `ch_query.go` |
| `ActionQueryMetrics` | `handleQueryMetrics()` | `ch_query.go` |
| `ActionQuerySLO` | `handleQuerySLO()` | `ch_query.go` |

AI 工具只需在 Master 端：
1. 定义 Tool JSON Schema（`prompts.go`）
2. 注册 ToolHandler（`master.go`）
3. ToolHandler 内构造 Command → `WaitCommandResult()` → 精简返回

---

## 3. 新增工具详细设计

### 3.1 Tool: `query_traces`

**用途**：查询 APM 分布式追踪数据，找到慢请求或错误请求。

**Tool 定义**（写入 `prompts.go` 的 `toolsJSON`）：

```json
{
  "name": "query_traces",
  "description": "查询 APM 分布式追踪数据。可按服务名、操作名、错误状态过滤。返回 Trace 摘要列表含耗时、Span 数、错误信息。最多返回 10 条。",
  "parameters": {
    "type": "object",
    "properties": {
      "service": {
        "type": "string",
        "description": "服务名过滤（如 geass-gateway）"
      },
      "operation": {
        "type": "string",
        "description": "操作名过滤（如 GET /api/v1/users）"
      },
      "min_duration_ms": {
        "type": "number",
        "description": "最小耗时（毫秒），用于查找慢请求"
      },
      "status_code": {
        "type": "string",
        "description": "HTTP 状态码过滤（如 500、404）"
      },
      "since": {
        "type": "string",
        "description": "时间范围，如 5m、1h、24h。默认 1h",
        "default": "1h"
      }
    },
    "required": []
  }
}
```

**执行逻辑**（ToolHandler）：

```go
// master.go — 注册 query_traces 工具
aiService.RegisterTool("query_traces", func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error) {
    // 1. 构造 Command 参数
    cmdParams := map[string]interface{}{
        "sub_action": "list_traces",
        "limit":      10,  // 硬性上限
    }
    if s := getStringParam(params, "service"); s != "" {
        cmdParams["service"] = s
    }
    if s := getStringParam(params, "operation"); s != "" {
        cmdParams["operation"] = s
    }
    if d := getFloat64Param(params, "min_duration_ms"); d > 0 {
        cmdParams["min_duration_ms"] = d
    }
    if s := getStringParam(params, "status_code"); s != "" {
        cmdParams["status_code"] = s
    }
    since := getStringParam(params, "since")
    if since == "" {
        since = "1h"
    }
    cmdParams["since"] = since

    // 2. 创建 Command
    resp, err := ops.CreateCommand(&model.CreateCommandRequest{
        ClusterID: clusterID,
        Action:    command.ActionQueryTraces,
        Source:    "ai",
        Params:    cmdParams,
    })
    if err != nil {
        return fmt.Sprintf("创建查询命令失败: %v", err), nil
    }

    // 3. 等待结果
    result, err := bus.WaitCommandResult(ctx, resp.CommandID, toolTimeout)
    if err != nil {
        return fmt.Sprintf("查询超时: %v", err), nil
    }
    if !result.Success {
        return fmt.Sprintf("查询失败: %s", result.Error), nil
    }

    // 4. 精简输出（TraceSummary 已是摘要格式，直接返回）
    return truncateToolResult(result.Output, "traces"), nil
})
```

**返回数据格式**（`apm.TraceSummary` 已有，无需新建）：

```json
[
  {
    "traceId": "abc123...",
    "rootService": "geass-gateway",
    "rootOperation": "GET /api/v1/users",
    "durationMs": 1523.45,
    "spanCount": 12,
    "serviceCount": 3,
    "hasError": true,
    "statusCode": 500,
    "timestamp": "2026-03-09T10:30:00Z"
  }
]
```

**约束**：
- `limit` 硬上限 10，Tool 定义中不暴露 limit 参数，LLM 无法突破
- 默认 `since=1h`，LLM 可指定最大 `24h`
- 不返回完整 Span 树，只返回 TraceSummary

---

### 3.2 Tool: `query_logs`

**用途**：查询 OTel 结构化日志，支持全文搜索和多维过滤。

**Tool 定义**：

```json
{
  "name": "query_logs",
  "description": "查询 OpenTelemetry 结构化日志。支持全文搜索、按服务/级别/TraceId 过滤。最多返回 20 条，日志 Body 截断为 200 字符。",
  "parameters": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "全文搜索关键词（模糊匹配日志 Body）"
      },
      "service": {
        "type": "string",
        "description": "服务名过滤"
      },
      "level": {
        "type": "string",
        "enum": ["DEBUG", "INFO", "WARN", "ERROR"],
        "description": "日志级别过滤"
      },
      "trace_id": {
        "type": "string",
        "description": "按 TraceId 过滤（跨信号关联）"
      },
      "since": {
        "type": "string",
        "description": "时间范围，如 15m、1h、24h。默认 1h",
        "default": "1h"
      }
    },
    "required": []
  }
}
```

**执行逻辑**：

```go
aiService.RegisterTool("query_logs", func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error) {
    cmdParams := map[string]interface{}{
        "limit": 20,  // 硬性上限
    }
    if s := getStringParam(params, "query"); s != "" {
        cmdParams["query"] = s
    }
    if s := getStringParam(params, "service"); s != "" {
        cmdParams["service"] = s
    }
    if s := getStringParam(params, "level"); s != "" {
        cmdParams["level"] = s
    }
    if s := getStringParam(params, "trace_id"); s != "" {
        cmdParams["trace_id"] = s
    }
    since := getStringParam(params, "since")
    if since == "" {
        since = "1h"
    }
    cmdParams["since"] = since

    resp, err := ops.CreateCommand(&model.CreateCommandRequest{
        ClusterID: clusterID,
        Action:    command.ActionQueryLogs,
        Source:    "ai",
        Params:    cmdParams,
    })
    if err != nil {
        return fmt.Sprintf("创建查询命令失败: %v", err), nil
    }

    result, err := bus.WaitCommandResult(ctx, resp.CommandID, toolTimeout)
    if err != nil {
        return fmt.Sprintf("查询超时: %v", err), nil
    }
    if !result.Success {
        return fmt.Sprintf("查询失败: %s", result.Error), nil
    }

    // 精简输出：截断每条日志 Body 为 200 字符
    return truncateToolResult(result.Output, "logs"), nil
})
```

**返回数据格式**（Agent 返回 `log.QueryResult`，AI Handler 精简后）：

```json
{
  "total": 1523,
  "showing": 20,
  "hint": "共 1523 条日志，显示最近 20 条",
  "logs": [
    {
      "timestamp": "2026-03-09T10:30:15Z",
      "severity": "ERROR",
      "service": "geass-auth",
      "body": "failed to validate token: jwt expired at 2026-03-09T...(truncated)",
      "traceId": "abc123...",
      "spanId": "def456..."
    }
  ]
}
```

**约束**：
- `limit` 硬上限 20
- `body` 截断为 200 字符，附 `(truncated)` 标记
- 返回 `total` 总数 + `showing` 实际条数 + `hint` 提示文本
- 安全提示：日志可能包含敏感信息（Token 泄漏等），AI System Prompt 中标注

---

### 3.3 Tool: `query_slo`

**用途**：查询服务或域名的 SLO 指标（可用性、延迟、错误率）。

**Tool 定义**：

```json
{
  "name": "query_slo",
  "description": "查询 SLO 指标数据。返回服务/域名的可用性、延迟分位数（P50/P90/P99）、错误率、RPS。支持 1 天/7 天/30 天窗口。",
  "parameters": {
    "type": "object",
    "properties": {
      "service": {
        "type": "string",
        "description": "服务名（Linkerd mesh 服务）"
      },
      "domain": {
        "type": "string",
        "description": "域名（Traefik ingress 域名）"
      },
      "window": {
        "type": "string",
        "enum": ["1d", "7d", "30d"],
        "description": "时间窗口，默认 7d",
        "default": "7d"
      }
    },
    "required": []
  }
}
```

**执行逻辑**（内存直读，无 Command）：

```go
aiService.RegisterTool("query_slo", func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error) {
    window := getStringParam(params, "window")
    if window == "" {
        window = "7d"
    }
    service := getStringParam(params, "service")
    domain := getStringParam(params, "domain")

    // 从 DataHub Store 读取最新 OTelSnapshot
    snapshot := store.GetLatestSnapshot(clusterID)
    if snapshot == nil || snapshot.OTel == nil {
        return "当前无 OTel 数据", nil
    }

    otel := snapshot.OTel

    // 1. 尝试从 SLOWindows 获取预聚合数据
    if windowData, ok := otel.SLOWindows[window]; ok && windowData != nil {
        result := buildSLOResult(windowData, service, domain, window)
        data, _ := json.Marshal(result)
        return string(data), nil
    }

    // 2. 降级：从实时 SLO 列表获取（只有当前窗口数据）
    result := buildSLOFromCurrent(otel, service, domain)
    data, _ := json.Marshal(result)
    return string(data), nil
})
```

**返回数据格式**：

```json
{
  "window": "7d",
  "services": [
    {
      "name": "geass-gateway",
      "type": "ingress",
      "rps": 45.2,
      "successRate": 0.9987,
      "errorRate": 0.0013,
      "p50Ms": 12.3,
      "p90Ms": 45.6,
      "p99Ms": 120.8
    }
  ],
  "hint": "数据来源: 7 天预聚合窗口"
}
```

**约束**：
- 内存直读，纳秒级延迟
- 如指定 `service` 或 `domain`，只返回匹配项
- 未指定时返回所有服务（不超过 50 个）

---

### 3.4 Tool: `get_entity_detail`

**用途**：获取特定实体的风险详情、异常指标、因果树和关联实体。

**Tool 定义**：

```json
{
  "name": "get_entity_detail",
  "description": "获取特定实体（Pod/Service/Node/Ingress）的风险详情。包含：风险分数、异常指标列表、因果树（上下游异常实体关系）、传播路径。用于深度分析某个实体为什么异常。",
  "parameters": {
    "type": "object",
    "properties": {
      "entity_type": {
        "type": "string",
        "enum": ["pod", "service", "node", "ingress"],
        "description": "实体类型"
      },
      "entity_name": {
        "type": "string",
        "description": "实体名称"
      },
      "namespace": {
        "type": "string",
        "description": "命名空间（Pod/Service 必需，Node/Ingress 可选）"
      }
    },
    "required": ["entity_type", "entity_name"]
  }
}
```

**执行逻辑**（内存直读，无 Command）：

```go
aiService.RegisterTool("get_entity_detail", func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error) {
    entityType := getStringParam(params, "entity_type")
    entityName := getStringParam(params, "entity_name")
    namespace := getStringParam(params, "namespace")

    if entityType == "" || entityName == "" {
        return "缺少参数 entity_type 和 entity_name", nil
    }

    // 构造 entityKey（与 AIOps Engine 内部格式一致）
    entityKey := buildEntityKey(entityType, namespace, entityName)

    // 从 AIOps Engine 获取实体风险详情
    detail := aiopsEngine.GetEntityRisk(clusterID, entityKey)
    if detail == nil {
        return fmt.Sprintf("未找到实体 %s（可能不在依赖图中或无异常数据）", entityKey), nil
    }

    // 精简输出：因果树只保留 2 层，指标只保留异常的
    result := simplifyEntityDetail(detail)
    data, _ := json.Marshal(result)
    return string(data), nil
})
```

**返回数据格式**（精简版 `EntityRiskDetail`）：

```json
{
  "entityKey": "service:default/geass-gateway",
  "entityType": "service",
  "rFinal": 0.85,
  "riskLevel": "critical",
  "metrics": [
    {
      "metricKey": "slo:error_rate",
      "currentValue": 0.15,
      "baseline": 0.002,
      "deviation": 74.0,
      "isAnomaly": true
    }
  ],
  "causalTree": [
    {
      "entityKey": "pod:default/geass-auth-xxx",
      "direction": "upstream",
      "rFinal": 0.72,
      "metrics": [{"metricKey": "k8s:restart_count", "isAnomaly": true}]
    }
  ],
  "propagation": [
    {"from": "pod:default/geass-auth-xxx", "to": "service:default/geass-gateway", "edgeType": "pod_to_service"}
  ]
}
```

**约束**：
- 内存直读，纳秒级延迟
- 因果树深度 ≤ 2（已由 Engine.buildCausalTree 保证）
- 只返回有异常的指标（`isAnomaly=true`）

---

## 4. 事件上下文丰富

### 4.1 现有上下文结构

```go
// ai/prompts/background.go（IncidentPromptContext 定义在 ai/prompts 包）
type IncidentPromptContext struct {
    IncidentSummary  string  // 事件基本信息
    TimelineText     string  // 时间线叙述
    AffectedEntities string  // 受影响实体及其风险评分
    RootCauseEntity  string  // 根因实体详情
    HistoricalContext string // 历史相似事件
}
```

### 4.2 新增字段

```go
type IncidentPromptContext struct {
    // 已有字段
    IncidentSummary  string
    TimelineText     string
    AffectedEntities string
    RootCauseEntity  string
    HistoricalContext string

    // 新增字段
    RecentErrorTraces string  // 受影响服务的最近错误 Traces（Top 5）
    RecentErrorLogs   string  // 受影响服务的最近 ERROR 日志（Top 10）
    SLOContext        string  // 受影响服务的 SLO 变化摘要
}
```

### 4.3 数据来源（内存直读）

从 `OTelSnapshot` 中过滤：

```go
func buildOTelContext(otel *cluster.OTelSnapshot, affectedServices []string) (traces, logs, slo string) {
    // 1. 错误 Traces — 从 otel.RecentTraces 过滤
    var errorTraces []apm.TraceSummary
    for _, t := range otel.RecentTraces {
        if t.HasError && containsService(affectedServices, t.RootService) {
            errorTraces = append(errorTraces, t)
            if len(errorTraces) >= 5 {
                break
            }
        }
    }

    // 2. ERROR 日志 — 从 otel.RecentLogs 过滤
    var errorLogs []log.Entry
    for _, l := range otel.RecentLogs {
        if l.SeverityText == "ERROR" && containsService(affectedServices, l.ServiceName) {
            errorLogs = append(errorLogs, l)
            if len(errorLogs) >= 10 {
                break
            }
        }
    }

    // 3. SLO 摘要 — 从 otel.SLOIngress + otel.SLOServices 匹配
    // 构建 "ServiceA: SuccessRate=99.8%, P99=120ms" 格式文本

    return formatTraces(errorTraces), formatLogs(errorLogs), formatSLO(...)
}
```

**选择内存直读而非 Command 的理由**：
- 事件上下文构建在 `Enricher.summarizeCore()` 内，同步执行
- Command 机制需要等待 Agent 响应（秒级延迟），对 background 角色来说过慢
- OTelSnapshot.RecentTraces（无过滤，Dashboard 首屏用）和 RecentLogs（5 分钟窗口，最多 500 条）数据量足够覆盖事件上下文需求
- 如果受影响服务在最近快照中无匹配数据，上下文字段返回 "无相关数据"

### 4.4 Background Prompt 更新

在 `ai/prompts/background.go` 的 system prompt 中新增指引：

```
如果上下文中包含「最近错误 Traces」或「最近 ERROR 日志」，请优先分析这些信号与事件的关联：
- Trace 中的高耗时 Span 可能指示瓶颈位置
- ERROR 日志中的异常堆栈可能揭示根因
- TraceId 可用于关联 Trace 和 Log
- SLO 变化可以量化事件对服务质量的影响
```

---

## 5. 辅助函数设计

### 5.1 `truncateToolResult`

```go
// truncateToolResult 精简 AI Tool 返回数据，避免 LLM 上下文爆炸
func truncateToolResult(raw string, dataType string) string {
    switch dataType {
    case "traces":
        // TraceSummary 已是摘要格式，只需确保总量不超过 10
        var traces []json.RawMessage
        json.Unmarshal([]byte(raw), &traces)
        if len(traces) > 10 {
            traces = traces[:10]
        }
        data, _ := json.Marshal(map[string]interface{}{
            "traces":  traces,
            "showing": len(traces),
        })
        return string(data)

    case "logs":
        // 解析 QueryResult，截断 Body，限制条数
        var qr struct {
            Logs  []json.RawMessage `json:"logs"`
            Total int64             `json:"total"`
        }
        json.Unmarshal([]byte(raw), &qr)
        // 截断每条日志 Body 为 200 字符
        truncated := truncateLogBodies(qr.Logs, 200)
        if len(truncated) > 20 {
            truncated = truncated[:20]
        }
        data, _ := json.Marshal(map[string]interface{}{
            "logs":    truncated,
            "total":   qr.Total,
            "showing": len(truncated),
            "hint":    fmt.Sprintf("共 %d 条日志，显示最近 %d 条", qr.Total, len(truncated)),
        })
        return string(data)
    }
    return raw
}
```

### 5.2 `buildEntityKey`

```go
// buildEntityKey 构造 AIOps Engine 内部的 entityKey 格式
func buildEntityKey(entityType, namespace, name string) string {
    switch entityType {
    case "node":
        return fmt.Sprintf("node:%s", name)
    case "ingress":
        if namespace != "" {
            return fmt.Sprintf("ingress:%s/%s", namespace, name)
        }
        return fmt.Sprintf("ingress:%s", name)
    default:
        // pod, service
        if namespace != "" {
            return fmt.Sprintf("%s:%s/%s", entityType, namespace, name)
        }
        return fmt.Sprintf("%s:default/%s", entityType, name)
    }
}
```

### 5.3 `simplifyEntityDetail`

```go
// simplifyEntityDetail 精简 EntityRiskDetail 用于 AI 返回
func simplifyEntityDetail(detail *aiops.EntityRiskDetail) map[string]interface{} {
    // 只保留异常指标
    var anomalyMetrics []map[string]interface{}
    for _, m := range detail.Metrics {
        if m.IsAnomaly {
            anomalyMetrics = append(anomalyMetrics, map[string]interface{}{
                "metricKey":    m.MetricKey,
                "currentValue": m.CurrentValue,
                "baseline":     m.BaselineValue,
                "deviation":    m.Deviation,
            })
        }
    }

    // 因果树（已限 2 层，直接使用）
    return map[string]interface{}{
        "entityKey":   detail.EntityKey,
        "entityType":  detail.EntityType,
        "namespace":   detail.Namespace,
        "name":        detail.Name,
        "rFinal":      detail.RFinal,
        "riskLevel":   detail.RiskLevel,
        "metrics":     anomalyMetrics,
        "causalTree":  detail.CausalTree,
        "propagation": detail.Propagation,
    }
}
```

---

## 6. 量控约束

| 数据类型 | 硬上限 | 截断规则 | 理由 |
|---------|--------|---------|------|
| Traces | 10 条 | 只返回 TraceSummary，不含 Span 树 | 1 条 TraceSummary ≈ 200 字符 |
| Logs | 20 条 | Body 截断 200 字符 | 1 条 ≈ 300 字符，20 条 ≈ 6KB |
| SLO | 50 个服务 | 返回聚合指标，不含原始数据点 | 1 个服务 ≈ 100 字符 |
| Entity Detail | 1 个实体 | 因果树深度 ≤ 2，只含异常指标 | 完整树 ≈ 2-5KB |
| 事件上下文 Traces | 5 条 | 摘要格式 | 附加到 LLM prompt |
| 事件上下文 Logs | 10 条 | Body 截断 200 字符 | 附加到 LLM prompt |

**总量估算**：单次 Tool Call 最大返回 ≈ 8KB 文本，远小于 `toolResultMaxLen` 的阈值（8000-32000 字符）。

---

## 7. 文件变更清单

### 7.1 目录结构与影响范围

> 注：`ai/prompts/` 子包和 `aiops/enricher/` 重命名已在前置重构中完成。
> 本设计基于重构后的目录结构。

```
atlhyper_master_v2/
├── ai/
│   ├── prompts/                     # 提示词子包（Phase 1 重构已创建）
│   │   ├── tools.go                 ← [修改] toolsJSON 追加 4 个工具定义
│   │   ├── chat.go                  ← [修改] chatRole 新增 [可观测性查询工具] + [工具组合使用建议]
│   │   ├── analysis.go              ← [修改] analysisSystem 新增调查策略和信号关联指引
│   │   ├── background.go            ← [修改] BuildBackgroundPrompt() 拼接 OTel 上下文字段
│   │   └── security.go              ← 不变
│   ├── tool.go                      ← [修改] 新增 truncateToolResult / truncateLogBodies / buildEntityKey / simplifyEntityDetail
│   └── ...（其他文件不变）
├── master.go                        ← [修改] 新增 4 个 RegisterTool 调用（query_traces/query_logs/query_slo/get_entity_detail）
├── aiops/
│   └── enricher/                    # AIOps AI 增强（Phase 3 重构：aiops/ai → aiops/enricher）
│       ├── context_builder.go       ← [修改] IncidentPromptContext 新增 3 个 OTel 字段 + buildOTelContext()
│       ├── enricher.go              ← [修改] summarizeCore() 传入 OTelSnapshot 参数
│       └── ...（其他文件不变）
└── ...（其他目录不变）

atlhyper_agent_v2/                   ← 零改动（所有 ClickHouse 查询 Command Handler 已存在）

model_v3/                            ← 零改动（所有数据模型已存在）
```

### 7.2 影响范围分析

| 影响层 | 涉及包 | 耦合方向 | 说明 |
|--------|--------|---------|------|
| **Tool 定义层** | `ai/prompts/` | 内聚 | 新增 4 个工具 JSON Schema + 提示词更新，独立修改 |
| **Tool 实现层** | `ai/tool.go` + `master.go` | `master.go` → `ai`, `service/operations`, `datahub`, `aiops` | ToolHandler 注册在 master.go，需注入 `ops`（Command 路径）、`store`（内存直读）、`aiopsEngine`（实体详情） |
| **事件上下文层** | `ai/prompts/background.go` + `aiops/enricher/` | `enricher` → `ai/prompts`, `datahub` | P3 新增 OTel 上下文，enricher.summarizeCore() 需读取 OTelSnapshot |

**关键依赖链**：

```
P1-P2（工具注册）:
  master.go
    ├─→ ai.RegisterTool()                    # 注册 ToolHandler
    ├─→ service/operations.CreateCommand()    # query_traces/query_logs: Command 路径
    ├─→ mq.WaitCommandResult()               # 等待 Agent 查询结果
    ├─→ datahub.Store.GetLatestSnapshot()     # query_slo: 内存直读 OTelSnapshot
    └─→ aiops.Engine.GetEntityRisk()          # get_entity_detail: 内存直读因果树

P3（事件上下文丰富）:
  aiops/enricher/enricher.go
    └─→ datahub.Store.GetLatestSnapshot()     # 新增：读取 OTelSnapshot
    └─→ enricher.buildOTelContext()           # 新增：从快照过滤错误 Traces/Logs/SLO
    └─→ ai/prompts.BuildBackgroundPrompt()    # 已有：拼接新增字段
```

### 7.3 文件变更明细

| 文件 | 变更类型 | 变更内容 | Phase |
|------|---------|---------|-------|
| `ai/prompts/tools.go` | **修改** | `toolsJSON` 追加 4 个工具 JSON Schema | P1-P2 |
| `ai/prompts/chat.go` | **修改** | `chatRole` 新增 `[可观测性查询工具]` + `[工具组合使用建议]` 区块 | P1-P2 |
| `ai/prompts/analysis.go` | **修改** | `analysisSystem` 新增调查策略和信号关联指引 | P1-P2 |
| `ai/tool.go` | **修改** | 新增 `truncateToolResult()`、`truncateLogBodies()`、`buildEntityKey()`、`simplifyEntityDetail()` | P1-P2 |
| `master.go` | **修改** | Tool 注册区块新增 4 个 `aiService.RegisterTool()` 调用 | P1-P2 |
| `ai/prompts/background.go` | **修改** | `BuildBackgroundPrompt()` 拼接 `ctx.RecentErrorTraces`/`RecentErrorLogs`/`SLOContext` | P3 |
| `aiops/enricher/context_builder.go` | **修改** | `IncidentPromptContext` 新增 3 个字段 + `buildOTelContext()` | P3 |
| `aiops/enricher/enricher.go` | **修改** | `summarizeCore()` 需注入 `datahub.Store` 读取 OTelSnapshot | P3 |

**总计：8 个文件修改，0 个新文件，Agent 端零改动。**

---

## 8. 实施路线

### Phase 1：Command 路径工具（query_traces + query_logs）

**目标**：AI 能查询 ClickHouse 中的 Trace 和 Log 数据。

**步骤**：
1. `prompts.go` — `toolsJSON` 追加 `query_traces` 和 `query_logs` 定义
2. `tool.go` — 新增 `truncateToolResult()`、`truncateLogBodies()` 辅助函数
3. `master.go` — 注册 `query_traces` 和 `query_logs` ToolHandler
4. 调用 `prompts.ResetToolCache()` 刷新工具缓存

**验证**：
- AI Chat 中问 "查看最近的 ERROR 日志" → 调用 `query_logs` → 返回结构化日志
- AI Chat 中问 "geass-gateway 最近有慢请求吗？" → 调用 `query_traces` → 返回 Trace 摘要

### Phase 2：内存直读工具（query_slo + get_entity_detail）

**目标**：AI 能查询 SLO 趋势和实体因果详情。

**步骤**：
1. `prompts.go` — `toolsJSON` 追加 `query_slo` 和 `get_entity_detail` 定义
2. `tool.go` — 新增 `buildEntityKey()`、`simplifyEntityDetail()` 辅助函数
3. `master.go` — 注册 `query_slo` 和 `get_entity_detail` ToolHandler（需注入 `store` 和 `aiopsEngine`）

**验证**：
- AI Chat 中问 "geass-gateway 7 天 SLO 怎么样？" → 调用 `query_slo` → 返回聚合指标
- AI Chat 中问 "这个 Pod 为什么风险分高？" → 调用 `get_entity_detail` → 返回因果树

### Phase 3：事件上下文丰富

**目标**：`analyze_incident` 的上下文自动包含 Trace/Log/SLO 数据。

**步骤**：
1. `context_builder.go` — `IncidentContext` 新增 3 个字段 + `buildOTelContext()` 实现
2. `enhancer.go` — System prompt 追加 OTel 信号分析指引
3. `BuildIncidentContext()` 调用方需传入 `OTelSnapshot`（调整 Enhancer.Summarize 参数）

**验证**：
- 触发一个事件 → AI 自动分析 → 查看分析报告是否包含 Trace 错误和 ERROR 日志关联

---

## 9. 设计考量

### 9.1 为什么不新建 Command 类型

Agent 端已有 5 个 ClickHouse 查询 Command（`ActionQueryTraces`/`ActionQueryLogs`/`ActionQueryMetrics`/`ActionQuerySLO`/`ActionQueryTraceDetail`），且参数体系完善（支持 `sub_action`、绝对/相对时间、limit、多维过滤）。

AI 工具的查询需求是这些 Command 能力的**子集**——只需 `list_traces`（不需要 `list_services`/`get_topology` 等 sub_action），只需日志搜索（不需要 `histogram`）。

因此 AI 工具复用现有 Command，通过参数组合实现，无需在 Agent 端做任何改动。

### 9.2 为什么 query_slo 用内存直读而非 Command

- OTelSnapshot 已包含 `SLOWindows`（1d/7d/30d 预聚合数据），由 Agent 定期聚合上报
- 内存直读延迟 < 1ms，Command 路径延迟 > 1s（MQ 入队 + Agent 轮询 + ClickHouse 查询）
- SLO 工具主要返回聚合统计，不需要原始数据点，OTelSnapshot 中的数据完全满足

### 9.3 安全考虑

- `query_logs` 可能返回包含密钥泄漏的日志内容
- **对策**：
  1. Body 截断为 200 字符，降低泄漏面
  2. System prompt 中标注 "日志内容可能包含敏感信息，请勿在回复中重复密钥/Token"
  3. AI 响应不缓存或持久化日志原文（`ToolResult` 截断后存储）
- `get_entity_detail` 不涉及敏感数据（只有风险分数和指标名）

### 9.4 算法 + AI + 人的分工

| 角色 | 职责 | 工具 |
|------|------|------|
| **算法** | 自动检测异常、评估风险、管理事件生命周期 | AIOps Engine（EMA+3σ + Risk + StateMachine） |
| **AI** | 根因分析、关联信号、自然语言交互、建议修复方案 | AI Chat + 增强后的 8 个工具 |
| **人** | 最终决策、执行修复、验证恢复 | AtlHyper UI + kubectl |

AI 工具增强的目标是让 AI 能「看到」算法看到的一切 + 更多原始信号（Trace/Log），从而提供更准确的分析。但 AI **不做**自动修复决策。

---

## 10. 系统提示词变更（工具增强后）

> 本节描述新增 4 个工具后的提示词变更。
> 现有 3 角色提示词的基础优化（与本任务无关）见独立设计文档。

### 10.1 提示词架构现状

| 角色 | 提示词位置 | 用途 | 是否使用 Tool |
|------|-----------|------|-------------|
| **chat** | `ai/prompts/chat.go` → `Security + chatRole` | 用户交互式对话 | ✅ 全部 8 个 |
| **background** | `ai/prompts/background.go` → `BuildBackgroundPrompt()` | 事件摘要（JSON 输出） | ❌ 无 Tool |
| **analysis** | `ai/prompts/analysis.go` → `BuildAnalysisPrompt()` | 高危事件深度调查 | ✅ 全部 8 个 |

### 10.2 chat 提示词变更

在 `ai/prompts/chat.go` 的 `chatRole` 中，`[AIOps 工具]` 区块需扩展为包含全部工具的说明：

```
[可观测性查询工具]

除了 query_cluster 查询 K8s 资源外，你还可以查询 OTel 可观测性数据：

- query_traces: 查询 APM 分布式追踪数据。可按服务名、操作名、耗时、状态码过滤。
  返回 Trace 摘要（最多 10 条），包含耗时、Span 数、错误信息。
  适用场景：用户问"为什么延迟高"、"有没有慢请求"、"最近有 500 错误吗"
  注意：返回的是 Trace 摘要，不含完整 Span 树。如需查看单个 Trace 的详细 Span，使用 trace_id 进一步关联

- query_logs: 查询 OTel 结构化日志（ClickHouse 存储）。支持全文搜索、按服务/级别/TraceId 过滤。
  返回最多 20 条日志，Body 截断为 200 字符。
  适用场景：用户问"有没有 ERROR 日志"、"某服务最近报了什么错"
  ⚠️ 与 query_cluster 的 get_logs 区别：
    - get_logs = kubectl logs（容器标准输出，实时但无结构化搜索）
    - query_logs = OTel 日志（ClickHouse 结构化存储，支持全文搜索和跨信号关联）
  ⚠️ 安全注意：日志内容可能包含意外泄漏的密钥/Token，请勿在回复中原样复述敏感内容

- query_slo: 查询 SLO 指标趋势。返回服务的可用性、延迟分位数（P50/P90/P99）、错误率、RPS。
  支持 1d/7d/30d 时间窗口。
  适用场景：用户问"某服务 SLO 怎么样"、"最近 7 天的可用性"

- get_entity_detail: 获取实体（Pod/Service/Node/Ingress）的风险详情。
  包含：风险分数、异常指标列表、因果树（上下游异常传播关系）。
  适用场景：用户问"这个 Pod 为什么异常"、"服务风险从哪里传播来的"
  返回的因果树展示了异常的传播路径——上游实体的异常如何影响到目标实体

[工具组合使用建议]

当用户问"某服务为什么异常"时，推荐的调查流程：
1. get_entity_detail → 查看风险分和因果树，确定是自身问题还是上游传播
2. query_traces → 查看该服务的慢请求或错误请求
3. query_logs → 查看 ERROR 日志获取错误详情
4. query_cluster describe → 查看 Pod/Deployment 的 K8s 状态
5. query_slo → 量化服务质量影响

不需要每次都调用所有工具，根据问题性质选择最相关的 2-3 个即可。
```

### 10.3 analysis 提示词变更

在 `ai/prompts/analysis.go` 的 `analysisSystem` 中，增加调查策略指引：

```
调查策略:
1. 首先使用 get_entity_detail 查看根因实体的因果树，判断异常来源方向
2. 对异常实体使用 query_traces 检查是否有慢请求或错误请求（重点关注 hasError=true 和高耗时）
3. 使用 query_logs level=ERROR 查看相关服务的错误日志，寻找异常堆栈或错误消息
4. 使用 query_slo 量化异常对服务质量的影响（对比事件前后的 SuccessRate/P99）
5. 必要时使用 query_cluster describe/get_logs 查看 K8s 资源状态和容器日志
6. 如果因果树显示上游异常，递归调查上游实体

信号关联:
- TraceId 可以关联 Trace 和 Log：在 query_traces 发现错误 Trace 后，用其 traceId 调用 query_logs 获取关联日志
- 实体因果树中的 upstream 方向指示异常源头，downstream 方向指示影响范围
```

### 10.4 background 提示词 — 无 system prompt 变更

background 角色（`ai/prompts/background.go`）不使用 Tool，只做事件摘要的 JSON 结构化输出。
新增的 OTel 上下文数据（Traces/Logs/SLO）通过 `IncidentPromptContext` 字段传入 user prompt，
不需要修改 system prompt 中的格式定义。

### 10.5 文件变更补充

| 文件 | 变更内容 | Phase |
|------|---------|-------|
| `ai/prompts/chat.go` | `chatRole` 新增 `[可观测性查询工具]` + `[工具组合使用建议]` 区块 | P1-P2 |
| `ai/prompts/analysis.go` | `analysisSystem` 新增调查策略和信号关联指引 | P1-P2 |
| `ai/prompts/background.go` | `BuildBackgroundPrompt()` 拼接新增的 OTel 上下文字段 | P3 |

---

## 11. 增强后工具全景

| # | 工具 | 能力 | 数据来源 | 执行路径 | Phase |
|---|------|------|---------|---------|-------|
| 1 | `query_cluster` | K8s 资源查询 | K8s API | Command → Agent | 已有 |
| 2 | `analyze_incident` | LLM 事件根因分析 | AIOps + OTel | 内存 + LLM | 已有(P3增强) |
| 3 | `get_cluster_risk` | 集群风险评分 | AIOps Scorer | 内存直读 | 已有 |
| 4 | `get_recent_incidents` | 最近事件列表 | AIOps Store | 内存直读 | 已有 |
| 5 | **`query_traces`** | APM Trace 查询 | ClickHouse | Command → Agent | **P1** |
| 6 | **`query_logs`** | OTel 日志查询 | ClickHouse | Command → Agent | **P1** |
| 7 | **`query_slo`** | SLO 指标趋势 | OTelSnapshot | 内存直读 | **P2** |
| 8 | **`get_entity_detail`** | 实体风险详情+因果树 | AIOps Engine | 内存直读 | **P2** |
