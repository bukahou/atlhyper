# AIOps Phase 4 — AI 增强层

## 概要

在已有的算法层（依赖图 + 基线 + 风险评分 + 状态机 + 事件存储）基础上，增加 **AI 增强层**：将结构化事件数据转换为自然语言摘要、生成根因分析建议、匹配历史相似事件，并集成到现有 AI Chat 模块的 Tool Calling 机制中。AI 增强是锦上添花，算法层独立工作不依赖 AI。

**前置依赖**: Phase 2b（事件存储已就绪）+ Phase 3（前端页面已完成）

**中心文档**: [`aiops-engine-design.md`](./aiops-engine-design.md) §7 (Phase 4)

**关联设计**:
- [`aiops-phase2-statemachine-incident.md`](./aiops-phase2-statemachine-incident.md) — 事件数据来源
- [`aiops-phase2-risk-scorer.md`](./aiops-phase2-risk-scorer.md) — 风险评分数据来源
- [`aiops-phase3-frontend.md`](./aiops-phase3-frontend.md) — 前端 IncidentDetailModal 需新增 AI 分析按钮

---

## 1. 文件夹结构

```
atlhyper_master_v2/
├── aiops/
│   ├── interfaces.go                        (Phase 1) <- 修改: +AI 增强方法
│   ├── engine.go                            (Phase 2a) 不动
│   │
│   └── ai/                                            <- NEW (整个目录)
│       ├── enhancer.go                                <- NEW  AI 增强服务（摘要/建议/相似事件）
│       ├── prompts.go                                 <- NEW  AIOps 专用 Prompt 模板
│       └── context_builder.go                         <- NEW  LLM 输入构建（结构化数据 → 文本）
│
├── ai/
│   ├── prompts.go                           (现有)  <- 修改: toolsJSON 追加 3 个 AIOps Tool
│   └── tool.go                              (现有)  <- 修改: Execute() 追加 AIOps Tool 分支
│
├── gateway/
│   ├── routes.go                            (现有)  <- 修改: +2 路由 (Operator 权限)
│   └── handler/
│       └── aiops_ai.go                                <- NEW  AI 增强 API Handler
│
├── service/
│   ├── interfaces.go                        (现有)  <- 修改: Query 接口 +2 方法
│   └── query/
│       └── aiops.go                         (Phase 2b) <- 修改: +AI 增强查询实现
│
└── atlhyper_web/src/                        (前端)
    ├── api/
    │   └── aiops.ts                         (Phase 3) <- 修改: +AI 增强 API 方法
    ├── app/monitoring/incidents/components/
    │   └── IncidentDetailModal.tsx           (Phase 3) <- 修改: +AI 分析按钮/面板
    └── i18n/
        ├── locales/zh.ts                    (现有)  <- 修改: +AI 增强翻译 (~15 个键)
        ├── locales/ja.ts                    (现有)  <- 修改: +AI 增强翻译 (~15 个键)
        └── types/i18n.ts                    (现有)  <- 修改: AIOpsTranslations +ai 子接口
```

### 变更统计

| 操作 | 文件数 | 文件 |
|------|--------|------|
| **新建** | 4 | `aiops/ai/` 下 3 个 + `handler/aiops_ai.go` |
| **修改** | 9 | `aiops/interfaces.go`, `ai/prompts.go`, `ai/tool.go`, `gateway/routes.go`, `service/interfaces.go`, `query/aiops.go`, `api/aiops.ts`, `IncidentDetailModal.tsx`, `zh.ts`, `ja.ts`, `i18n.ts` |

---

## 2. 调用链路

### 2.1 API 方式（前端按钮触发）

```
用户点击 IncidentDetailModal 的「AI 分析」按钮
    ↓
POST /api/v2/aiops/ai/summarize { incidentId }
    ↓
Gateway Handler (aiops_ai.go)
    │
    ├── 1. 验证 Operator 权限
    ├── 2. 调用 service.Query.GetIncidentDetail(id)
    │       → Incident + Entities + Timeline
    ├── 3. 调用 service.Query.GetEntityRisks(clusterID)
    │       → 当前风险评分上下文（可选）
    ├── 4. 调用 aiopsAI.Summarize(ctx, incident, entities, timeline, risks)
    │       │
    │       ├── context_builder.BuildIncidentContext(...)
    │       │       → 结构化数据 → 文本描述
    │       ├── prompts.SummarizePrompt(context)
    │       │       → 完整 Prompt
    │       └── llmClient.ChatStream(prompt)
    │               → AI 生成摘要 + 建议
    │
    └── 5. 返回 { summary, recommendations, similarIncidents }
```

### 2.2 Chat Tool 方式（用户在 AI Chat 中询问事件）

```
用户: "分析一下最近的事件 INC-2025-0042"
    ↓
AI Chat → LLM 判断调用 tool: analyze_incident
    ↓
toolExecutor.Execute(ctx, clusterID, toolCall)
    │
    ├── action = "analyze_incident"
    ├── 跳过黑名单（只读操作）
    ├── 调用 aiopsAI.Summarize(ctx, incident, ...)
    └── 返回结构化分析结果给 LLM 继续对话
```

### 2.3 初始化链路

```
master.go 现有初始化流程中:

    aiopsEngine := aiops.NewAIOpsEngine(...)         ← Phase 1 已完成
    aiopsAI := aiopsai.NewEnhancer(                  ← NEW
        aiopsEngine,
        aiService,    // 复用现有 AI 模块的 LLM 配置
        db,           // 事件查询
    )

    // Tool 执行器追加 aiopsAI 引用
    toolExecutor.SetAIOpsEnhancer(aiopsAI)            ← NEW
```

---

## 3. 数据模型

### 3.1 AI 增强请求/响应类型

```go
// aiops/ai/enhancer.go

// SummarizeRequest 事件摘要请求
type SummarizeRequest struct {
    IncidentID string `json:"incidentId"`
}

// SummarizeResponse 事件摘要响应
type SummarizeResponse struct {
    IncidentID      string           `json:"incidentId"`
    Summary         string           `json:"summary"`          // 自然语言摘要
    RootCauseAnalysis string         `json:"rootCauseAnalysis"` // 根因分析
    Recommendations []Recommendation `json:"recommendations"`   // 处置建议
    SimilarIncidents []SimilarMatch  `json:"similarIncidents"`  // 相似历史事件
    GeneratedAt     int64            `json:"generatedAt"`       // 生成时间 (Unix ms)
}

// Recommendation 处置建议
type Recommendation struct {
    Priority    int    `json:"priority"`    // 1=最高
    Action      string `json:"action"`      // 建议操作
    Reason      string `json:"reason"`      // 理由
    Impact      string `json:"impact"`      // 预期影响
    IsAutomatic bool   `json:"isAutomatic"` // 是否可自动执行（Phase 4 全部为 false）
}

// SimilarMatch 相似事件匹配
type SimilarMatch struct {
    IncidentID  string  `json:"incidentId"`
    Similarity  float64 `json:"similarity"`  // [0, 1]
    RootCause   string  `json:"rootCause"`
    Resolution  string  `json:"resolution"`  // 当时的解决方式
    OccurredAt  string  `json:"occurredAt"`
}
```

### 3.2 aiops/interfaces.go 新增方法

```go
// aiops/interfaces.go — Phase 4 新增

type AIOpsEngine interface {
    // ... Phase 1~2b 已有方法 ...

    // Phase 4: AI 增强
    // SummarizeIncident 生成事件的 AI 摘要、根因分析和处置建议
    SummarizeIncident(ctx context.Context, incidentID string) (*ai.SummarizeResponse, error)
}
```

---

## 4. 详细设计

### 4.1 Enhancer 服务 (aiops/ai/enhancer.go)

```go
package ai

// Enhancer AIOps AI 增强服务
type Enhancer struct {
    incidentRepo database.AIOpsIncidentRepository   // 事件查询
    aiopsEngine  aiops.AIOpsEngine                  // 风险/图/基线查询
    llmProvider  func() (llm.LLMClient, error)      // 动态获取 LLM 客户端
}

// NewEnhancer 创建 AI 增强服务
// llmProvider 复用现有 AI 模块的配置加载逻辑，每次调用动态创建 LLM 客户端
func NewEnhancer(
    incidentRepo database.AIOpsIncidentRepository,
    aiopsEngine aiops.AIOpsEngine,
    llmProvider func() (llm.LLMClient, error),
) *Enhancer
```

**核心方法 Summarize 流程：**

```go
func (e *Enhancer) Summarize(ctx context.Context, incidentID string) (*SummarizeResponse, error) {
    // 1. 查询事件数据
    incident, _ := e.incidentRepo.GetByID(ctx, incidentID)
    entities, _ := e.incidentRepo.GetEntities(ctx, incidentID)
    timeline, _ := e.incidentRepo.GetTimeline(ctx, incidentID)

    // 2. 查询相似历史事件（基于根因实体 + 时间窗口）
    patterns, _ := e.incidentRepo.GetPatterns(ctx, incident.RootCause, "90d")

    // 3. 构建 LLM 上下文
    context := BuildIncidentContext(incident, entities, timeline, patterns)

    // 4. 生成 Prompt
    prompt := SummarizePrompt(context)

    // 5. 调用 LLM
    client, _ := e.llmProvider()
    defer client.Close()

    chunks, _ := client.ChatStream(ctx, &llm.Request{
        SystemPrompt: prompt.System,
        Messages:     []llm.Message{{Role: "user", Content: prompt.User}},
    })

    // 6. 收集完整响应
    fullText := collectResponse(chunks)

    // 7. 解析结构化输出
    return parseResponse(fullText, incidentID, patterns)
}
```

### 4.2 Context Builder (aiops/ai/context_builder.go)

**职责**：将结构化事件数据转换为 LLM 可理解的文本描述。

```go
package ai

// IncidentContext LLM 输入上下文
type IncidentContext struct {
    IncidentSummary   string   // 事件基本信息（ID、状态、严重度、持续时间）
    TimelineText      string   // 时间线叙述
    AffectedEntities  string   // 受影响实体及其风险评分
    RootCauseEntity   string   // 根因实体详情
    HistoricalContext string   // 历史相似事件（如有）
}

// BuildIncidentContext 从结构化数据构建 LLM 上下文
func BuildIncidentContext(
    incident *database.Incident,
    entities []database.IncidentEntity,
    timeline []database.IncidentTimeline,
    patterns []database.IncidentPattern,
) *IncidentContext
```

**生成的上下文示例：**

```
事件概要:
  ID: INC-2025-0042
  状态: Incident | 严重度: High | 持续: 23 分钟
  集群: production-cluster-1

根因实体:
  node/worker-3 (角色: root_cause)
  R_local: 0.90 | R_final: 0.90

受影响实体 (3 个):
  1. node/worker-3         root_cause  R=0.90
  2. default/pod/api-abc   affected    R=0.78
  3. default/service/api   symptom     R=0.85

时间线:
  14:02:15 [异常检测] Node worker-3 内存使用率超过基线 3.2σ
  14:03:45 [状态变更] Node worker-3: Healthy → Warning
  14:04:10 [指标飙升] Pod api-server-abc 内存达到 limit 的 95%
  14:05:22 [异常检测] Service api-server 错误率 3.2% (基线 0.3%)
  14:06:00 [根因识别] 根因链: Node(memory) → Pod(OOM) → Service(errors)
  14:08:15 [状态变更] 集群: Warning → Incident

历史相似事件 (2 个):
  1. INC-2025-0031 (2025-01-15) — node/worker-3 内存压力, 持续 45 分钟
  2. INC-2025-0019 (2024-12-28) — node/worker-3 内存压力, 持续 18 分钟
```

### 4.3 Prompt 模板 (aiops/ai/prompts.go)

```go
package ai

// SystemPrompt AIOps 事件分析系统提示词
const SystemPrompt = `你是 AtlHyper 平台的 AIOps 分析引擎。你的任务是分析 Kubernetes 集群的运维事件，
提供根因分析、处置建议和历史模式匹配。

要求:
1. 根因分析必须基于提供的数据，不要臆测
2. 处置建议必须具体可执行，按优先级排列
3. 如果有历史相似事件，指出模式和趋势
4. 使用技术精确的语言，避免模糊表述
5. 输出格式严格遵循 JSON Schema

输出格式:
{
  "summary": "一段话概述事件（什么时间，什么实体，什么问题，什么影响）",
  "rootCauseAnalysis": "详细分析根因链路（从源头到影响面）",
  "recommendations": [
    {
      "priority": 1,
      "action": "具体操作步骤",
      "reason": "为什么这样做",
      "impact": "预期效果"
    }
  ],
  "similarPattern": "如果有历史相似事件，描述模式和建议"
}`

// UserPromptTemplate 用户消息模板
const UserPromptTemplate = `请分析以下 Kubernetes 集群事件:

%s

请按照指定的 JSON 格式输出分析结果。`
```

**Prompt 组装：**

```go
// SummarizePrompt 组装完整 Prompt
func SummarizePrompt(ctx *IncidentContext) *PromptPair {
    userContent := fmt.Sprintf(UserPromptTemplate,
        ctx.IncidentSummary + "\n\n" +
        ctx.RootCauseEntity + "\n\n" +
        ctx.AffectedEntities + "\n\n" +
        ctx.TimelineText + "\n\n" +
        ctx.HistoricalContext,
    )
    return &PromptPair{
        System: SystemPrompt,
        User:   userContent,
    }
}

type PromptPair struct {
    System string
    User   string
}
```

### 4.4 AI Chat Tool 集成

#### 4.4.1 新增 3 个 AIOps Tool 定义

在现有 `ai/prompts.go` 的 `toolsJSON` 中追加：

```json
{
  "name": "analyze_incident",
  "description": "分析指定事件的根因、影响面和处置建议。输入事件 ID，返回 AI 分析结果。",
  "input_schema": {
    "type": "object",
    "properties": {
      "incident_id": {
        "type": "string",
        "description": "事件 ID，格式如 INC-2025-0042"
      }
    },
    "required": ["incident_id"]
  }
},
{
  "name": "get_cluster_risk",
  "description": "获取集群当前的风险评分和高风险实体。返回 ClusterRisk 分数 (0-100) 和 Top N 风险实体列表。",
  "input_schema": {
    "type": "object",
    "properties": {
      "top_n": {
        "type": "integer",
        "description": "返回前 N 个高风险实体，默认 10"
      }
    }
  }
},
{
  "name": "get_recent_incidents",
  "description": "获取最近的事件列表。可按状态过滤，返回事件摘要。",
  "input_schema": {
    "type": "object",
    "properties": {
      "state": {
        "type": "string",
        "enum": ["warning", "incident", "recovery", "stable"],
        "description": "按状态过滤，不填则返回所有状态"
      },
      "limit": {
        "type": "integer",
        "description": "返回数量，默认 10"
      }
    }
  }
}
```

#### 4.4.2 Tool 执行器扩展 (ai/tool.go)

```go
// tool.go — Execute() 中新增分支

func (e *toolExecutor) Execute(ctx context.Context, clusterID string, tc llm.ToolCall) (string, error) {
    params := parseParams(tc.Arguments)
    action := params["action"]

    // 现有 query_cluster tool 处理...

    switch tc.Name {
    case "analyze_incident":
        // 直接调用 AIOps AI Enhancer
        incidentID := params["incident_id"]
        result, err := e.aiopsEnhancer.Summarize(ctx, incidentID)
        if err != nil {
            return fmt.Sprintf("分析事件失败: %v", err), nil
        }
        return marshalJSON(result), nil

    case "get_cluster_risk":
        topN := getIntParam(params, "top_n", 10)
        risk, err := e.aiopsEngine.GetClusterRisk(ctx, clusterID)
        if err != nil {
            return fmt.Sprintf("获取风险评分失败: %v", err), nil
        }
        entities, _ := e.aiopsEngine.GetEntityRisks(ctx, clusterID, "r_final", topN)
        return formatRiskResult(risk, entities), nil

    case "get_recent_incidents":
        state := params["state"]
        limit := getIntParam(params, "limit", 10)
        incidents, err := e.aiopsEngine.GetIncidents(ctx, clusterID, state, limit)
        if err != nil {
            return fmt.Sprintf("获取事件列表失败: %v", err), nil
        }
        return formatIncidentList(incidents), nil
    }

    // 原有 query_cluster 处理 ...
}
```

#### 4.4.3 角色提示词追加

在 `ai/prompts.go` 的 `rolePrompt` 中追加 AIOps 相关指引：

```
## AIOps 工具

你还可以使用以下 AIOps 工具来分析集群风险和事件：

- analyze_incident: 深度分析事件（根因、建议、相似历史）。当用户询问某个事件时使用。
- get_cluster_risk: 获取当前集群风险概况。当用户问"集群状态如何"、"有什么风险"时使用。
- get_recent_incidents: 获取最近的事件列表。当用户问"最近有什么事件"、"有什么告警"时使用。

使用建议：
- 用户提到事件 ID 时，优先使用 analyze_incident
- 用户询问集群健康状况时，先用 get_cluster_risk 获取概况
- 结合 AIOps 工具和 query_cluster 工具可以提供更全面的分析
```

### 4.5 响应解析逻辑

```go
// enhancer.go — parseResponse

func parseResponse(raw string, incidentID string, patterns []database.IncidentPattern) (*SummarizeResponse, error) {
    // 尝试从 LLM 输出中提取 JSON
    jsonStr := extractJSON(raw)

    var parsed struct {
        Summary           string `json:"summary"`
        RootCauseAnalysis string `json:"rootCauseAnalysis"`
        Recommendations   []struct {
            Priority int    `json:"priority"`
            Action   string `json:"action"`
            Reason   string `json:"reason"`
            Impact   string `json:"impact"`
        } `json:"recommendations"`
        SimilarPattern string `json:"similarPattern"`
    }

    if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
        // 降级：整段文本作为摘要
        return &SummarizeResponse{
            IncidentID:  incidentID,
            Summary:     raw,
            GeneratedAt: time.Now().UnixMilli(),
        }, nil
    }

    // 构建 Recommendations
    recommendations := make([]Recommendation, len(parsed.Recommendations))
    for i, r := range parsed.Recommendations {
        recommendations[i] = Recommendation{
            Priority:    r.Priority,
            Action:      r.Action,
            Reason:      r.Reason,
            Impact:      r.Impact,
            IsAutomatic: false, // Phase 4 全部手动
        }
    }

    // 构建 SimilarIncidents（从结构化 patterns 数据生成，非 LLM 输出）
    similarIncidents := buildSimilarMatches(patterns)

    return &SummarizeResponse{
        IncidentID:        incidentID,
        Summary:           parsed.Summary,
        RootCauseAnalysis: parsed.RootCauseAnalysis,
        Recommendations:   recommendations,
        SimilarIncidents:  similarIncidents,
        GeneratedAt:       time.Now().UnixMilli(),
    }, nil
}

// buildSimilarMatches 从结构化 patterns 数据构建相似事件列表
// 相似度基于：根因实体匹配 + 指标重叠度
func buildSimilarMatches(patterns []database.IncidentPattern) []SimilarMatch {
    if len(patterns) == 0 {
        return []SimilarMatch{}
    }

    matches := make([]SimilarMatch, 0, len(patterns))
    for _, p := range patterns {
        for _, inc := range p.Incidents {
            matches = append(matches, SimilarMatch{
                IncidentID:  inc.ID,
                Similarity:  calculateSimilarity(p),
                RootCause:   inc.RootCause,
                Resolution:  "", // 当前无结构化 resolution 字段，后续可扩展
                OccurredAt:  inc.StartedAt.Format(time.RFC3339),
            })
        }
    }
    return matches
}
```

---

## 5. API 端点

### 5.1 路由

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | `/api/v2/aiops/ai/summarize` | Operator | 生成事件 AI 摘要 |
| POST | `/api/v2/aiops/ai/recommend` | Operator | 生成处置建议（独立调用） |

**注意**：使用 POST 而非 GET，因为 AI 生成是计算密集型操作，且可能有请求体参数。

### 5.2 请求/响应格式

#### POST /api/v2/aiops/ai/summarize

**请求：**
```json
{
  "incidentId": "INC-2025-0042"
}
```

**响应：**
```json
{
  "message": "分析完成",
  "data": {
    "incidentId": "INC-2025-0042",
    "summary": "2025-01-20 14:02 起，production 集群 worker-3 节点内存使用率持续超过基线 3.2σ（94%），导致运行在该节点上的 api-server Pod 内存接近 limit (95%)，进而引发 api-server Service 错误率从基线 0.3% 飙升至 3.2%。事件持续 23 分钟后开始恢复。",
    "rootCauseAnalysis": "根因链路: Node worker-3 内存压力 → Pod api-server-abc 内存溢出风险 → Service api-server 错误率异常。根本原因是 worker-3 节点上的工作负载内存需求超过节点容量，可能由近期部署变更或流量增长引起。",
    "recommendations": [
      {
        "priority": 1,
        "action": "检查 worker-3 节点上的 Pod 内存 request/limit 设置，确认是否有 Pod 未设置内存限制",
        "reason": "未设置内存限制的 Pod 可能无限制消耗内存，导致节点压力",
        "impact": "防止内存争用，避免 OOM Kill",
        "isAutomatic": false
      },
      {
        "priority": 2,
        "action": "考虑为 api-server Deployment 配置 Pod Disruption Budget (PDB)，并添加 anti-affinity 分散到多个节点",
        "reason": "当前 api-server Pod 集中在单一节点，节点故障影响面大",
        "impact": "提高服务高可用性",
        "isAutomatic": false
      },
      {
        "priority": 3,
        "action": "为 worker-3 节点配置资源预警 (内存 > 80%) 的告警规则",
        "reason": "本次事件在内存达到 94% 时才触发，预警阈值应更低",
        "impact": "提前预警，留出处理时间",
        "isAutomatic": false
      }
    ],
    "similarIncidents": [
      {
        "incidentId": "INC-2025-0031",
        "similarity": 0.85,
        "rootCause": "node/worker-3",
        "resolution": "",
        "occurredAt": "2025-01-15T10:23:00Z"
      },
      {
        "incidentId": "INC-2025-0019",
        "similarity": 0.72,
        "rootCause": "node/worker-3",
        "resolution": "",
        "occurredAt": "2024-12-28T16:45:00Z"
      }
    ],
    "generatedAt": 1737364200000
  }
}
```

#### POST /api/v2/aiops/ai/recommend

**请求：**
```json
{
  "incidentId": "INC-2025-0042"
}
```

**响应：** 与 summarize 相同格式，但 Prompt 专注于处置建议，返回更详细的 `recommendations` 列表。

---

## 6. Service 层接口变更

### 6.1 service/interfaces.go 新增方法

```go
// service/interfaces.go — Phase 4 新增

type Query interface {
    // ... Phase 1~2b 已有方法 ...

    // Phase 4: AI 增强
    // SummarizeIncident 调用 AI 分析事件
    SummarizeIncident(ctx context.Context, incidentID string) (*aiopsai.SummarizeResponse, error)
}
```

### 6.2 service/query/aiops.go 新增实现

```go
// query/aiops.go — Phase 4 新增

func (s *QueryService) SummarizeIncident(ctx context.Context, incidentID string) (*aiopsai.SummarizeResponse, error) {
    return s.aiopsEngine.SummarizeIncident(ctx, incidentID)
}
```

---

## 7. Gateway Handler

### 7.1 handler/aiops_ai.go

```go
package handler

type AIOpsAIHandler struct {
    svc service.Query
}

func NewAIOpsAIHandler(svc service.Query) *AIOpsAIHandler {
    return &AIOpsAIHandler{svc: svc}
}

// Summarize POST /api/v2/aiops/ai/summarize
func (h *AIOpsAIHandler) Summarize(w http.ResponseWriter, r *http.Request) {
    // 1. 检查 HTTP 方法
    if r.Method != http.MethodPost {
        writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    // 2. 解析请求体
    var req struct {
        IncidentID string `json:"incidentId"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    if req.IncidentID == "" {
        writeError(w, http.StatusBadRequest, "incidentId is required")
        return
    }

    // 3. 调用 Service
    result, err := h.svc.SummarizeIncident(r.Context(), req.IncidentID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "AI analysis failed")
        return
    }

    // 4. 统一 JSON 响应
    writeJSON(w, http.StatusOK, map[string]interface{}{
        "message": "分析完成",
        "data":    result,
    })
}
```

### 7.2 路由注册 (gateway/routes.go)

```go
// routes.go — Phase 4 新增路由

// AIOps AI 增强 (Operator 权限)
aiopsAIHandler := handler.NewAIOpsAIHandler(svc)
r.Handle("/api/v2/aiops/ai/summarize", r.auth(aiopsAIHandler.Summarize, PermOperator))
r.Handle("/api/v2/aiops/ai/recommend", r.auth(aiopsAIHandler.Recommend, PermOperator))
```

**权限级别选择：Operator**

| 理由 | 说明 |
|------|------|
| 调用 LLM API | 有成本（API 调用费用），不应开放给所有用户 |
| 数据敏感度 | AI 分析可能暴露基础设施详情 |
| 一致性 | 与现有 AI Chat 功能权限一致 |

---

## 8. 前端变更

### 8.1 api/aiops.ts 新增方法

```typescript
// api/aiops.ts — Phase 4 新增

// AI 增强类型
export interface SummarizeResponse {
  incidentId: string
  summary: string
  rootCauseAnalysis: string
  recommendations: Recommendation[]
  similarIncidents: SimilarMatch[]
  generatedAt: number
}

export interface Recommendation {
  priority: number
  action: string
  reason: string
  impact: string
  isAutomatic: boolean
}

export interface SimilarMatch {
  incidentId: string
  similarity: number
  rootCause: string
  resolution: string
  occurredAt: string
}

// AI 增强 API
export function summarizeIncident(incidentId: string) {
  return request.post<SummarizeResponse>('/api/v2/aiops/ai/summarize', { incidentId })
}

export function recommendActions(incidentId: string) {
  return request.post<SummarizeResponse>('/api/v2/aiops/ai/recommend', { incidentId })
}
```

### 8.2 IncidentDetailModal.tsx 变更

在现有事件详情弹窗中新增 AI 分析面板：

```
IncidentDetailModal (Phase 3 已有)
  ┌─────────────────────────────────────────────────────┐
  │  事件 #INC-xxxx                            [关闭]  │
  │  状态/严重度/持续时间                               │
  │                                                     │
  │  ┌── RootCauseCard ─────────────────────────────── │
  │  │ ...                                             │ │
  │  └──────────────────────────────────────────────── │
  │                                                     │
  │  ┌── 受影响实体 ────────────────────────────────── │
  │  │ ...                                             │ │
  │  └──────────────────────────────────────────────── │
  │                                                     │
  │  ┌── TimelineView ─────────────────────────────── │
  │  │ ...                                             │ │
  │  └──────────────────────────────────────────────── │
  │                                                     │
  │  ┌── AI 分析 ★ NEW ───────────────────────────── │  │
  │  │  [🤖 AI 分析] 按钮                              │ │
  │  │                                                 │ │
  │  │  (点击后加载):                                   │ │
  │  │  📋 摘要: ...                                   │ │
  │  │  🔍 根因分析: ...                               │ │
  │  │  💡 处置建议:                                    │ │
  │  │    1. [P1] 检查 Pod 内存设置...                  │ │
  │  │    2. [P2] 配置 anti-affinity...                │ │
  │  │  📊 相似历史事件:                                │ │
  │  │    - INC-xxx (85% 相似) 2025-01-15              │ │
  │  └──────────────────────────────────────────────── │
  └─────────────────────────────────────────────────────┘
```

**实现逻辑：**

```typescript
// IncidentDetailModal.tsx — Phase 4 新增状态和交互

// 新增状态
const [aiAnalysis, setAiAnalysis] = useState<SummarizeResponse | null>(null)
const [aiLoading, setAiLoading] = useState(false)
const [aiError, setAiError] = useState<string | null>(null)

// AI 分析按钮点击处理
const handleAiAnalyze = async () => {
  setAiLoading(true)
  setAiError(null)
  try {
    const res = await summarizeIncident(incidentId)
    setAiAnalysis(res.data)
  } catch (err) {
    setAiError(t.aiops.ai.analysisFailed)
  } finally {
    setAiLoading(false)
  }
}

// 渲染 AI 分析面板
// 按钮: disabled={aiLoading}, 加载中显示 spinner
// 分析结果: summary → rootCauseAnalysis → recommendations 列表 → similarIncidents 列表
// 权限: 仅 Operator+ 可见 AI 分析按钮（通过 useAuth() 检查）
```

### 8.3 i18n 新增翻译键

#### types/i18n.ts 新增

```typescript
// AIOpsTranslations 中追加 ai 子接口
export interface AIOpsTranslations {
  // ... Phase 3 已有 ...

  // AI 增强 (Phase 4)
  ai: {
    analyze: string           // "AI 分析"
    analyzing: string         // "分析中..."
    analysisFailed: string    // "AI 分析失败"
    summary: string           // "摘要"
    rootCauseAnalysis: string // "根因分析"
    recommendations: string   // "处置建议"
    similarIncidents: string  // "相似历史事件"
    priority: string          // "优先级"
    action: string            // "建议操作"
    reason: string            // "理由"
    impact: string            // "预期影响"
    similarity: string        // "相似度"
    noSimilar: string         // "暂无相似事件"
    generatedAt: string       // "生成时间"
    regenerate: string        // "重新生成"
  }
}
```

#### zh.ts 翻译

```typescript
ai: {
  analyze: 'AI 分析',
  analyzing: '分析中...',
  analysisFailed: 'AI 分析失败，请稍后重试',
  summary: '摘要',
  rootCauseAnalysis: '根因分析',
  recommendations: '处置建议',
  similarIncidents: '相似历史事件',
  priority: '优先级',
  action: '建议操作',
  reason: '理由',
  impact: '预期影响',
  similarity: '相似度',
  noSimilar: '暂无相似历史事件',
  generatedAt: '生成时间',
  regenerate: '重新生成',
}
```

#### ja.ts 翻訳

```typescript
ai: {
  analyze: 'AI 分析',
  analyzing: '分析中...',
  analysisFailed: 'AI 分析に失敗しました。後でもう一度お試しください',
  summary: '概要',
  rootCauseAnalysis: '根本原因分析',
  recommendations: '対処提案',
  similarIncidents: '類似インシデント',
  priority: '優先度',
  action: '推奨アクション',
  reason: '理由',
  impact: '想定される効果',
  similarity: '類似度',
  noSimilar: '類似のインシデントはありません',
  generatedAt: '生成日時',
  regenerate: '再生成',
}
```

---

## 9. 实现阶段

```
P1: AI 增强核心
  ├── aiops/ai/context_builder.go — 结构化数据 → 文本
  ├── aiops/ai/prompts.go — Prompt 模板
  ├── aiops/ai/enhancer.go — Enhancer 服务
  ├── aiops/interfaces.go — +SummarizeIncident 方法
  └── 单元测试:
      ├── context_builder_test.go — 上下文构建正确性
      └── enhancer_test.go — Mock LLM 客户端测试

P2: API 端点
  ├── gateway/handler/aiops_ai.go — Handler
  ├── gateway/routes.go — +2 路由
  ├── service/interfaces.go — +1 方法
  ├── service/query/aiops.go — +AI 查询实现
  └── 集成测试: API 调用完整流程

P3: AI Chat Tool 集成
  ├── ai/prompts.go — toolsJSON 追加 3 个 Tool
  ├── ai/tool.go — Execute() 追加 AIOps 分支
  ├── ai/prompts.go (rolePrompt) — 追加 AIOps 工具说明
  └── 测试: Tool 执行 + LLM 交互

P4: 前端集成
  ├── api/aiops.ts — +AI 增强 API 方法
  ├── IncidentDetailModal.tsx — +AI 分析面板
  ├── i18n (types + zh + ja) — +~15 个翻译键
  └── 构建验证: next build

P5: 集成测试
  ├── 完整流程: 事件创建 → AI 分析 → 前端展示
  ├── Chat 流程: 用户提问 → Tool 调用 → 分析结果
  └── 错误处理: LLM 不可用时的降级
```

---

## 10. 文件变更清单

### 新建

| 文件 | 说明 |
|------|------|
| `aiops/ai/enhancer.go` | AI 增强服务（Summarize 主逻辑 + 响应解析） |
| `aiops/ai/prompts.go` | AIOps 专用 Prompt 模板（SystemPrompt + UserPromptTemplate） |
| `aiops/ai/context_builder.go` | LLM 上下文构建器（结构化数据 → 文本描述） |
| `gateway/handler/aiops_ai.go` | AI 增强 API Handler (Summarize + Recommend) |

### 修改

| 文件 | 变更 |
|------|------|
| `aiops/interfaces.go` | +`SummarizeIncident` 方法 |
| `ai/prompts.go` | `toolsJSON` 追加 3 个 AIOps Tool 定义 + `rolePrompt` 追加使用说明 |
| `ai/tool.go` | `Execute()` 追加 `analyze_incident` / `get_cluster_risk` / `get_recent_incidents` 分支 + `aiopsEnhancer` 字段 |
| `gateway/routes.go` | +2 路由 (`/aiops/ai/summarize`, `/aiops/ai/recommend`) |
| `service/interfaces.go` | Query 接口 +`SummarizeIncident` 方法 |
| `service/query/aiops.go` | +AI 摘要查询实现 |
| `api/aiops.ts` | +`SummarizeResponse` 等类型 + `summarizeIncident()` / `recommendActions()` |
| `IncidentDetailModal.tsx` | +AI 分析按钮 + 分析结果面板 + 加载/错误状态 |
| `i18n/types/i18n.ts` | `AIOpsTranslations` +`ai` 子接口 (~15 个键) |
| `i18n/locales/zh.ts` | +`aiops.ai` 翻译 |
| `i18n/locales/ja.ts` | +`aiops.ai` 翻译 |

### 无新增数据库表

AI 增强层不需要新的数据库表。摘要结果不持久化（按需生成），相似事件匹配复用 Phase 2b 已有的 `incidents` 表和 `GetPatterns()` 查询。

---

## 11. 测试计划

| 组件 | 测试类型 | 验证点 |
|------|---------|--------|
| `context_builder.go` | 单元测试 | 各类事件数据正确转换为文本描述 |
| `prompts.go` | 单元测试 | Prompt 组装格式正确、不超过 token 限制 |
| `enhancer.go` | 单元测试 (Mock LLM) | 正常响应解析 + JSON 提取 + 降级处理 |
| `handler/aiops_ai.go` | Handler 测试 | 参数校验 + 权限检查 + 响应格式 |
| `ai/tool.go` (新增分支) | 单元测试 | 3 个新 Tool 的执行 + 参数解析 + 错误处理 |
| `api/aiops.ts` (前端) | API 类型检查 | TypeScript 类型与后端响应对齐 |
| `IncidentDetailModal` | 交互测试 | 按钮点击 → 加载状态 → 结果展示 → 错误处理 |
| i18n | 完整性检查 | zh.ts 和 ja.ts 的 aiops.ai 键一致 |

### 关键测试场景

```go
// enhancer_test.go

func TestSummarize_NormalIncident(t *testing.T) {
    // 给定: 有根因和时间线的完整事件
    // 期望: 返回 summary + rootCauseAnalysis + recommendations
}

func TestSummarize_LLMParseError(t *testing.T) {
    // 给定: LLM 返回非 JSON 格式
    // 期望: 降级为整段文本作为 summary
}

func TestSummarize_LLMUnavailable(t *testing.T) {
    // 给定: LLM 连接失败
    // 期望: 返回错误，前端显示 analysisFailed
}

func TestSummarize_NoHistoricalPatterns(t *testing.T) {
    // 给定: 无历史相似事件
    // 期望: similarIncidents 为空数组
}

func TestToolExecute_AnalyzeIncident(t *testing.T) {
    // 给定: Chat Tool 调用 analyze_incident
    // 期望: 正确调用 Enhancer.Summarize 并返回格式化结果
}

func TestToolExecute_GetClusterRisk(t *testing.T) {
    // 给定: Chat Tool 调用 get_cluster_risk
    // 期望: 返回 ClusterRisk + TopN 实体
}
```

---

## 12. 验证命令

```bash
# 后端测试
cd atlhyper_master_v2
go test ./aiops/ai/... -v
go test ./ai/... -v -run TestToolExecute_AIOps
go test ./gateway/handler/... -v -run TestAIOpsAI

# 后端构建
go build ./...

# 前端构建
cd atlhyper_web
npm run build

# 开发模式验证
npm run dev
# 1. 打开 /monitoring/incidents
# 2. 点击任一事件 → 打开详情弹窗
# 3. 点击「AI 分析」按钮
# 4. 验证摘要/建议/相似事件展示

# AI Chat 验证
# 1. 打开 /workbench (AI Chat)
# 2. 输入: "当前集群风险如何？"
# 3. 验证 LLM 调用 get_cluster_risk 工具
# 4. 输入: "分析一下最近的事件"
# 5. 验证 LLM 调用 get_recent_incidents → analyze_incident
```

---

## 13. 设计决策记录

### 为什么 AI 增强层独立于算法层？

| 方面 | 算法层 (Phase 1~2b) | AI 增强层 (Phase 4) |
|------|---------------------|---------------------|
| 依赖 | 无外部依赖 | 依赖 LLM API |
| 可用性 | 始终可用 | LLM 不可用时降级 |
| 延迟 | 毫秒级 | 秒级 (LLM 调用) |
| 成本 | 零 | API 调用费用 |
| 确定性 | 确定性算法输出 | 每次输出可能不同 |
| 权限 | Public (只读) | Operator (有成本) |

**核心原则**：算法层提供「确定性、可解释的数据」，AI 层提供「可读性、可操作的建议」。AI 不可用不影响监控告警。

### 为什么不持久化 AI 摘要？

1. 事件数据在变化中（Warning → Incident → Recovery），缓存的摘要可能过时
2. 避免新增数据库表和迁移
3. 按需生成保证摘要基于最新数据
4. 后续可选加缓存（incident 到达 Stable 后摘要不再变化）

### 为什么用 Operator 权限？

1. LLM API 调用有成本
2. 与现有 AI Chat 权限一致
3. 避免未授权用户大量触发 AI 分析
