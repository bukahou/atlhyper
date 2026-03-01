# AI 工具增强 — Trace/Log 查询能力与事件上下文丰富

> 状态：未来规划
> 创建：2026-02-27
> 前置：AI Chat 模块（已完成）、跨信号关联（已完成）

## 1. 问题陈述

AtlHyper AI 助手当前有 4 个工具，覆盖 K8s 资源查询和 AIOps 事件分析，
但**完全无法访问 APM Trace 和 OTel 日志数据**。

### 现有工具

| 工具 | 能力 | 数据来源 |
|------|------|----------|
| `query_cluster` | 查询 K8s 资源（get/list/describe/get_logs/get_events/get_configmap） | K8s API（通过 Agent Command） |
| `analyze_incident` | LLM 分析事件根因 | AIOps IncidentStore + Enhancer |
| `get_cluster_risk` | 获取集群风险评分 + Top N 高风险实体 | AIOps Engine |
| `get_recent_incidents` | 获取最近事件列表 | AIOps IncidentStore |

### 能力缺口

| 用户问题 | 期望行为 | 当前结果 |
|----------|---------|---------|
| "api-server 最近为什么延迟高？" | 查询 APM Traces，找到慢 Span | 只能查 Pod 日志（kubectl logs），看不到 Trace |
| "有哪些 ERROR 日志？" | 查询 OTel 结构化日志（ClickHouse） | 只能 kubectl logs（无结构化搜索） |
| "分析这个事件的根因" | 事件上下文包含关联 Trace 错误和日志 | 上下文只有 K8s 事件和风险数值，无 APM/Log |
| "geass-gateway 服务最近 7 天的 SLO 趋势？" | 查询 SLO 历史数据 | 无此工具 |
| "这个 Pod 为什么重启了？" | 自动关联 Pod 日志 + 对应 Trace + K8s Events | 只看到 K8s Events |

## 2. 增强方案

### 2.1 新增 AI 工具

#### Tool 1: `query_traces`（查询 APM Traces）

```json
{
  "name": "query_traces",
  "description": "查询 APM 分布式追踪数据。可按服务名、操作名、错误状态过滤。返回 Trace 列表含耗时、错误信息。",
  "parameters": {
    "cluster_id": { "type": "string", "required": true },
    "service": { "type": "string", "description": "服务名过滤" },
    "operation": { "type": "string", "description": "操作名过滤" },
    "error_only": { "type": "boolean", "description": "仅返回包含错误的 Trace" },
    "min_duration_ms": { "type": "number", "description": "最小耗时（毫秒），用于查找慢请求" },
    "since": { "type": "string", "description": "时间范围，如 1h、24h" },
    "limit": { "type": "number", "default": 10 }
  }
}
```

**实现方式**：复用现有 Command 机制 → Agent ClickHouse 查询 otel_traces 表。
需要在 `ch_query.go` 新增 `handleQueryTraces()` 处理函数。

#### Tool 2: `query_logs`（查询 OTel 结构化日志）

```json
{
  "name": "query_logs",
  "description": "查询 OpenTelemetry 结构化日志。支持全文搜索、按服务/级别/TraceId 过滤。",
  "parameters": {
    "cluster_id": { "type": "string", "required": true },
    "query": { "type": "string", "description": "全文搜索关键词" },
    "service": { "type": "string", "description": "服务名过滤" },
    "level": { "type": "string", "enum": ["DEBUG", "INFO", "WARN", "ERROR"] },
    "trace_id": { "type": "string", "description": "按 TraceId 过滤" },
    "since": { "type": "string", "description": "时间范围" },
    "limit": { "type": "number", "default": 20 }
  }
}
```

**实现方式**：复用现有 `ActionQueryLogs` Command。后端已完整支持这些参数。
只需在 AI 工具注册时添加定义，工具执行时构造 Command 即可。

#### Tool 3: `query_slo_trend`（查询 SLO 趋势）

```json
{
  "name": "query_slo_trend",
  "description": "查询服务或域名的 SLO 指标趋势（可用性、延迟、错误率）。",
  "parameters": {
    "cluster_id": { "type": "string", "required": true },
    "service": { "type": "string", "description": "服务名" },
    "domain": { "type": "string", "description": "域名" },
    "window": { "type": "string", "enum": ["1d", "7d", "30d"], "default": "7d" }
  }
}
```

**实现方式**：从 OTelSnapshot.SLOWindows 或 SLOTimeSeries 读取。
可通过快照直读或 Command 路径实现。

#### Tool 4: `get_entity_detail`（获取实体风险详情）

```json
{
  "name": "get_entity_detail",
  "description": "获取特定实体的风险详情：当前指标值、因果树、关联上下游实体。",
  "parameters": {
    "cluster_id": { "type": "string", "required": true },
    "entity_type": { "type": "string", "enum": ["pod", "service", "node", "ingress"] },
    "entity_name": { "type": "string" },
    "namespace": { "type": "string" }
  }
}
```

**实现方式**：从 AIOps Engine 的内存图中读取实体的 Metrics、RiskScore、
关联边、因果树（CausalTree）等信息。

### 2.2 事件分析上下文丰富

当前 `aiops/ai/context_builder.go` 构建的 `IncidentContext` 缺少 APM 和日志数据。

#### 改进：自动附加关联信号

当 AI 调用 `analyze_incident` 时，IncidentContext 应自动包含：

```
当前上下文（已有）：
├── 事件基本信息（状态、持续时间、影响实体）
├── 受影响实体的风险数值
├── K8s Events 时间线
└── 历史相似事件

新增上下文：
├── 受影响服务的最近错误 Traces（Top 5）
│   └── 每个 Trace: rootSpan、错误信息、耗时
├── 受影响服务的最近 ERROR 日志（Top 10）
│   └── 每条日志: 时间、Severity、Body、TraceId
└── 受影响服务的 SLO 变化摘要
    └── 事件前后的 SuccessRate/P99 对比
```

**实现方式**：
1. `context_builder.go` 在构建上下文时，通过 Command 机制查询 ClickHouse
2. 或直接从 OTelSnapshot 的 RecentTraces/RecentLogs 中过滤
3. 方案 2 更快（内存读取），但数据量受限于快照缓存大小

## 3. 实现架构

### 3.1 AI 工具执行路径

```
用户提问 → AI Chat → LLM 选择工具
  → query_traces  → Command(ActionQueryTraces)  → Agent → ClickHouse → 返回
  → query_logs    → Command(ActionQueryLogs)     → Agent → ClickHouse → 返回
  → query_slo     → 快照直读或 Command           → 返回
  → get_entity    → AIOps Engine 内存图           → 返回
```

### 3.2 文件改动

| 文件 | 改动 |
|------|------|
| `atlhyper_master_v2/ai/prompts.go` | 新增 4 个工具定义（JSON Schema） |
| `atlhyper_master_v2/ai/master.go` | 工具执行函数（handleQueryTraces/handleQueryLogs/...） |
| `atlhyper_agent_v2/service/command/ch_query.go` | 新增 handleQueryTraces()（如不存在） |
| `atlhyper_master_v2/aiops/ai/context_builder.go` | IncidentContext 增加 APM/Log 数据 |
| `atlhyper_master_v2/aiops/ai/enhancer.go` | System prompt 更新以利用新上下文 |

## 4. 实施路线

### Phase 1：query_logs 工具（最容易，后端已全面支持）

1. `prompts.go` 新增工具定义
2. `master.go` 新增执行函数（构造 Command → 等待响应 → 返回）
3. 验证：在 AI Chat 中问"查看最近的 ERROR 日志"

### Phase 2：query_traces 工具

1. 确认 Agent 端是否已有 `ActionQueryTraces` Command
2. 如无，在 `ch_query.go` 新增 Trace 查询处理
3. `prompts.go` + `master.go` 新增工具
4. 验证：在 AI Chat 中问"api-server 最近的慢请求"

### Phase 3：事件上下文丰富

1. `context_builder.go` 增加 APM/Log 关联数据
2. 更新 Enhancer 的 System prompt
3. 验证：触发一个事件，观察 AI 分析是否包含 Trace/Log 信息

### Phase 4：query_slo_trend + get_entity_detail

1. 实现快照直读路径
2. 验证端到端

## 5. 设计考量

### 5.1 算法 + AI + 人的分工

AtlHyper 的理念是"次世代 SRE 平台"，不是纯 AI 替代人。分工模型：

| 角色 | 职责 | 工具 |
|------|------|------|
| **算法** | 自动检测异常、评估风险、管理事件生命周期 | AIOps Engine（EMA+3σ + Risk + StateMachine） |
| **AI** | 根因分析、关联信号、自然语言交互、建议修复方案 | AI Chat + 增强后的工具集 |
| **人** | 最终决策、执行修复、验证恢复 | AtlHyper UI + kubectl |

AI 工具增强的目标是让 AI 能「看到」算法看到的一切 + 更多原始信号，
从而提供更准确的分析。但 AI **不做**自动修复决策。

### 5.2 工具输出格式

AI 工具返回的数据需要精简，避免 LLM 上下文爆炸：
- Traces：只返回摘要（duration、error、serviceName），不返回完整 Span 树
- Logs：只返回 Body 前 200 字符 + Severity + Timestamp
- 附加截断提示："共 150 条日志，显示最近 20 条"

### 5.3 安全考虑

- `query_logs` 可能返回敏感日志内容（密码、Token 泄漏到日志中）
- 对策：AI 工具应标注"日志内容可能包含敏感信息"的警告
- 不在 AI 响应中缓存或持久化日志原文

## 6. 文件变更预估

| 文件 | 改动 | Phase |
|------|------|-------|
| `ai/prompts.go` | 新增 4 个工具定义 | 1-4 |
| `ai/master.go` | 新增工具执行函数 | 1-4 |
| `service/command/ch_query.go` | 新增 handleQueryTraces（如需） | 2 |
| `aiops/ai/context_builder.go` | 事件上下文增加 APM/Log | 3 |
| `aiops/ai/enhancer.go` | System prompt 更新 | 3 |
| **合计** | | ~4-6 |
