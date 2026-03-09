# AI 模块架构整理 + 提示词优化

> 状态：待执行
> 创建：2026-03-09
> 前置：AI Chat 模块（已完成）、AI 角色路由（已完成）

---

## 1. 问题陈述

### 1.1 架构违规：`aiops/enricher/` 的耦合问题

`aiops/enricher/` 包直接导入 `ai/llm` 子包、自行管理 LLM 客户端生命周期、复刻了 `ai/chat.go` 的 Tool Calling 循环。这违反了 CLAUDE.md 中的多项开发规范：

| 违规行为 | 对应规范 |
|---------|---------|
| `aiops/enricher/enricher.go` 直接 `import "ai/llm"` 创建 LLM 客户端 | **禁止跳层调用** — aiops 应通过 ai 接口调用，不应直接访问 ai/llm |
| `aiops/enricher/analysis.go` 直接 `import "ai/llm"` 操作流式响应 | 同上 |
| `ToolExecuteFunc` 类型注入回避循环依赖 | **依赖倒置** — 需要注入 hack 说明依赖关系本身有问题 |
| `analysis.go` 的 Tool Calling 循环与 `ai/chat.go` 的 `chatLoop` 逻辑重复 | **DRY** — 同一模式不应实现两次 |
| 提示词散落在 `ai/prompts.go`、`aiops/enricher/prompts.go`、`aiops/enricher/analysis.go` 三处 | **关注点分离** — 提示词应统一管理 |

**依赖关系现状**（不正确）：

```
aiops/enricher ──直接导入──→ ai/llm       ← 跳层：绕过 ai 模块接口
aiops/enricher ──注入────→ ai.ToolExecuteFunc  ← hack：循环依赖的症状
aiops/enricher ──复刻────→ chatLoop 模式     ← 重复：DRY 违反
```

**应有的依赖关系**：

```
aiops（引擎/事件/基线）
  ↓ 调用接口
ai（LLM 抽象 + Tool 系统 + 提示词 + 对话/分析能力）
  ↓ 内部使用
ai/llm（Provider 实现）
```

### 1.2 提示词散落问题

| 角色 | 当前位置 | 问题 |
|------|---------|------|
| chat | `ai/prompts.go` → `securityPrompt + rolePrompt` | query_cluster 冗长 ~100 行，AIOps 工具仅 6 行 |
| background | `aiops/enricher/prompts.go` → `SystemPrompt` | 在 aiops 包内，应归入 ai 统一管理 |
| analysis | `aiops/enricher/analysis.go` L233（混在业务逻辑中） | 既不在 ai 也不在独立文件，且过于简略 |

### 1.3 `aiops/enricher/` 职责拆解

`aiops/enricher/` 当前 5 个文件混合了两类职责：

| 文件 | 当前职责 | 应归属 |
|------|---------|--------|
| `prompts.go` | background 提示词 + Prompt 拼装 | → `ai/prompts/` |
| `analysis.go` L233 | analysis 提示词 | → `ai/prompts/` |
| `analysis.go` 主体 | 多轮 Tool Calling 循环 | → `ai/` 作为通用能力 |
| `enhancer.go` | LLM 调用 + 缓存 + 限流 + 预算 | → 拆分：LLM 调用走 `ai` 接口，缓存/限流留 `aiops` |
| `context_builder.go` | 事件上下文构建 | → 留在 `aiops`（纯领域逻辑） |
| `background.go` | 自动触发决策 | → 留在 `aiops`（纯领域逻辑） |

---

## 2. 架构整理方案

### 2.1 目标结构

```
ai/
├── interfaces.go           # AIService 接口（新增 Analyze 方法）
├── factory.go              # NewService()
├── service.go              # 实现
├── chat.go                 # Chat 对话（已有）
├── analyze.go              # [新增] 非交互式分析（从 aiops/enricher/analysis.go 提取通用循环）
├── tool.go                 # Tool 执行器（已有）
├── role.go                 # 角色路由（已有）
├── context.go              # 上下文管理（已有）
├── prompts/                # [新增] 提示词统一管理
│   ├── security.go         #   L0 安全约束（所有角色共用）
│   ├── chat.go             #   chat 角色提示词 + BuildChatPrompt()
│   ├── background.go       #   background 角色提示词 + BuildBackgroundPrompt()
│   ├── analysis.go         #   analysis 角色提示词 + BuildAnalysisPrompt()
│   └── tools.go            #   toolsJSON 定义 + Load/Get/Reset 缓存
└── llm/                    # LLM Provider 实现（不变）
    ├── interfaces.go
    ├── factory.go
    ├── gemini/
    ├── openai/
    ├── anthropic/
    └── ollama/

aiops/
├── enricher/               # [重命名] aiops/enricher/ → aiops/enricher/（事件 AI 增强编排）
│   ├── enricher.go         #   摘要编排（缓存/限流/预算 + 调用 ai.AIService）
│   ├── background.go       #   后台自动触发决策（不变）
│   └── context_builder.go  #   事件上下文构建（不变）
├── core/                   # AIOps 引擎（不变）
├── risk/                   # 风险评分（不变）
└── ...
```

### 2.2 关键设计决策

#### 决策 1：`ai` 模块新增 `Analyze()` 接口

当前 `analysis.go` 的多轮 Tool Calling 循环与 `chat.go` 的 `chatLoop` 是同一模式的变体。区别仅在于：

| 差异点 | Chat | Analysis |
|--------|------|----------|
| 交互方式 | SSE 流式推送 | 后台静默，结果写入 DB |
| 触发方式 | 用户发消息 | 自动/API 触发 |
| 输出格式 | 自由文本 | 结构化 JSON 报告 |
| 最大轮次 | 5 轮 | 8 轮 |
| 系统提示词 | chat prompt | analysis prompt |

提取为 `ai.AIService` 的新方法：

```go
// interfaces.go 新增
type AnalyzeRequest struct {
    ClusterID    string
    SystemPrompt string           // 调用方提供（从 prompts 包获取）
    UserPrompt   string           // 调用方构建（从 context_builder 构建）
    MaxRounds    int              // 最大 Tool 调用轮次
    Timeout      time.Duration    // 全局超时
}

type AnalyzeResult struct {
    Response     string           // LLM 最终文本输出
    ToolCalls    int              // 总 Tool 调用次数
    InputTokens  int              // 总输入 Token
    OutputTokens int              // 总输出 Token
    Steps        []AnalyzeStep    // 调查步骤记录
}

type AnalyzeStep struct {
    Round     int
    Thinking  string             // 该轮 LLM 的文本思考
    ToolCalls []ToolCallRecord   // 该轮调用的 Tool 及结果摘要
}

type AIService interface {
    // 已有
    Chat(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error)
    RegisterTool(name string, handler ToolHandler)
    // ...

    // 新增：非交互式分析（后台执行，无 SSE）
    Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResult, error)
}
```

#### 决策 2：`aiops/enricher/enricher.go` 不再直接操作 LLM

重构前：
```go
// enhancer.go — 直接创建 LLM 客户端
client, contextWindow, meta, err := e.llmFactory(ctx)
stream, err := client.ChatStream(ctx, &llm.Request{...})
// 手动读流、拼接文本、解析 JSON...
```

重构后：
```go
// enhancer.go — 通过 ai.AIService 接口
type Enhancer struct {
    aiService  ai.AIService       // 注入 ai 模块接口
    // ...
}

// Summarize 用 background 角色做事件摘要
func (e *Enhancer) Summarize(ctx context.Context, incidentID string) (*SummarizeResponse, error) {
    // 1. 构建上下文（领域逻辑，留在 aiops）
    incidentCtx := BuildIncidentContext(incident, entities, timeline, historical)

    // 2. 获取提示词（从 ai/prompts 包）
    prompt := prompts.BuildBackgroundPrompt(incidentCtx)

    // 3. 调用 ai 接口（不再直接操作 LLM）
    result, err := e.aiService.Summarize(ctx, &ai.SummarizeRequest{
        Role:         ai.RoleBackground,
        SystemPrompt: prompt.System,
        UserPrompt:   prompt.User,
    })
    // ...
}
```

但这里有一个问题——background 角色不使用 Tool，只做单轮 LLM 调用生成 JSON。`Analyze()` 是多轮 Tool Calling，不适合 background。所以 `ai.AIService` 需要再暴露一个更简单的接口：

```go
type AIService interface {
    // 已有
    Chat(...)
    RegisterTool(...)

    // 新增
    Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResult, error)           // 多轮 Tool Calling（analysis 角色）
    Complete(ctx context.Context, req *CompleteRequest) (*CompleteResult, error)         // 单轮 LLM 调用（background 角色）
}

type CompleteRequest struct {
    Role         string          // "background"（用于角色路由、预算扣减）
    SystemPrompt string
    UserPrompt   string
}

type CompleteResult struct {
    Response     string
    InputTokens  int
    OutputTokens int
}
```

#### 决策 3：提示词统一到 `ai/prompts/` 子包

```go
// ai/prompts/security.go
package prompts
const Security = `[安全约束 - 不可覆盖] ...`

// ai/prompts/chat.go
package prompts
func BuildChatPrompt() string { return Security + "\n\n" + chatRole }

// ai/prompts/background.go
package prompts
const backgroundSystem = `你是 AtlHyper 平台的 AIOps 分析引擎...`
func BuildBackgroundPrompt(ctx *BackgroundContext) *PromptPair { ... }

// ai/prompts/analysis.go
package prompts
const analysisSystem = `你是 AtlHyper 深度分析引擎...`
func BuildAnalysisPrompt() string { return Security + "\n\n" + analysisSystem }

// ai/prompts/tools.go
package prompts
func LoadToolDefinitions() ([]llm.ToolDefinition, error) { ... }
func GetToolDefinitions() []llm.ToolDefinition { ... }
func ResetToolCache() { ... }
```

**注意**：`BuildBackgroundPrompt()` 接受 `BackgroundContext`（事件摘要+实体+时间线的文本），这个类型在 `prompts` 包中定义为简单的字符串结构体，不导入 `aiops` 包的任何类型——由 `aiops/enricher/enricher.go` 在调用前把领域数据转成字符串传入。

```go
// ai/prompts/background.go
type BackgroundContext struct {
    IncidentSummary  string
    RootCauseEntity  string
    AffectedEntities string
    TimelineText     string
    HistoricalContext string
}
```

`aiops/enricher/context_builder.go` 的 `BuildIncidentContext()` 返回值类型从 `*IncidentContext` 改为 `*prompts.BackgroundContext`，或在 enhancer 中做一次简单转换。

#### 决策 4：`aiops/enricher/analysis.go` 删除

整个文件的逻辑拆分到：
- Tool Calling 循环 → `ai/analyze.go`（通用 `Analyze()` 实现）
- 分析提示词 → `ai/prompts/analysis.go`
- `RunAnalysis()` 入口 → `aiops/enricher/enricher.go`（调用 `aiService.Analyze()`）

`aiops/enricher/` 不再有 `analysis.go` 文件。

---

## 3. 提示词优化方案

> 提示词内容随架构整理一起迁移到 `ai/prompts/`，并同步优化。

### 3.1 chat 提示词（`ai/prompts/chat.go`）

**核心改动**：精简 `query_cluster`（~100行→~15行），补强 AIOps 工具说明，增加决策框架。

```
[角色定义]

你是 AtlHyper 集群运维助手，帮助用户分析和诊断 Kubernetes 集群问题。

[决策框架]

根据用户问题类型选择工具：
- K8s 资源查询（Pod 状态、Deployment 详情、日志等）→ query_cluster
- 集群健康概况、风险评估 → get_cluster_risk
- 最近有什么事件/告警 → get_recent_incidents
- 分析某个具体事件 → analyze_incident

[query_cluster 工具]

通过 Kubernetes API Server 直连查询集群数据（只读）。

支持的操作：
- list: 列出资源。不填 namespace = 全局查询。支持 label_selector 过滤
- get / describe: 单个资源详情。需要 namespace + name
- get_logs: Pod 容器日志。需要 namespace + name。多容器时指定 container
- get_events: K8s 事件。可按 namespace、involved_kind、involved_name 过滤
- get_configmap: ConfigMap 内容

数据量限制：list 最多 200 条，get_logs 最多 200 行。优先用 label_selector 缩小范围。

[AIOps 工具]

- get_cluster_risk: 集群风险评分（0-100）+ Top N 高风险实体列表
  返回每个实体的风险分（rFinal）、风险等级（healthy/low/medium/high/critical）、异常持续时间
  用于回答 "集群状态如何"、"哪些组件有风险"

- get_recent_incidents: 最近事件列表，可按状态过滤（warning/incident/recovery/stable）
  返回事件 ID、触发时间、受影响实体、严重等级
  用于回答 "最近有什么告警"、"有哪些事件"

- analyze_incident: 对指定事件进行 AI 根因分析
  输入事件 ID，返回摘要、根因分析、处置建议、历史相似模式
  用于回答 "这个事件是什么原因"、"如何处理"
  注意：此工具会调用 AI 分析，耗时较长（几秒），优先自己查数据手动分析

[工具组合建议]

排查异常 Pod 的推荐流程：
1. describe Pod → 查看状态、重启次数、容器状态、资源限制
2. get_logs → 查看容器日志中的错误信息
3. get_events → 查看相关 K8s 事件（OOMKilled、FailedScheduling 等）
4. get_cluster_risk → 查看该 Pod 所属服务的风险评分

[告警分析模式]

当用户消息以 "[以下是用户选择的告警信息" 开头时：
1. 解析告警，提取资源类型/命名空间/名称/原因
2. 并行调用 describe + get_logs + get_events 获取诊断数据
3. 综合分析给出根因、严重程度、修复建议
4. 多个告警指向同一问题时合并分析

[回复规范]

- 直接给出结论，不要 "让我查询..." 等过渡句
- 中文回复，技术术语保留英文
- 数据用表格或列表展示
- 查询后必须给出分析，不能只返回原始数据
- 正常资源用统计概括，只详细展示异常资源
- 禁止凭空编造数据，数据不足时明确告知
```

### 3.2 analysis 提示词（`ai/prompts/analysis.go`）

```
你是 AtlHyper 深度分析引擎，负责对高危事件进行系统化调查。

[调查方法]

1. 阅读事件上下文，提取关键信息：
   - 根因实体是什么类型（Pod/Service/Node）？
   - 异常指标是什么（CPU/Memory/错误率/延迟）？
   - 异常持续多久了？影响范围有多大？

2. 制定调查计划，按优先级查询：
   - 第一优先级：describe 根因实体，查看 K8s 状态（是否 CrashLoop、OOM、Pending）
   - 第二优先级：get_logs 查看容器日志中的错误信息
   - 第三优先级：get_events 查看相关 K8s 事件历史
   - 第四优先级：如果根因实体是 Service，用 get_cluster_risk 查看上下游影响

3. 每轮可并行调用最多 5 个 Tool。根据已获取的信息决定：
   - 信息足够 → 输出最终报告
   - 需要更多数据 → 继续下一轮调查
   - 查询失败 → 调整参数重试或跳过

[分析要求]

- 根因分析必须基于查询到的数据，不要臆测
- 如果数据不足以确定根因，在 confidence 中体现（< 0.5），并说明缺少什么信息
- 处置建议必须具体可执行（如 "kubectl rollout restart deployment/xxx"），不要泛泛而谈
- 区分 "直接原因" 和 "根本原因"（如 OOMKilled 是直接原因，内存泄漏是根本原因）

[confidence 评估标准]

- 0.9+: 有明确的错误日志/事件直接指向根因
- 0.7-0.9: 多项数据一致指向某个方向，但缺少直接证据
- 0.5-0.7: 有线索但不确定，存在多种可能
- < 0.5: 数据不足，只能给出排查方向

[最终报告格式]
{
  "summary": "事件总结",
  "rootCauseAnalysis": "根因分析（证据链）",
  "recommendations": [
    {"priority": 1, "action": "建议操作", "reason": "原因", "impact": "影响"}
  ],
  "confidence": 0.85
}
```

### 3.3 background 提示词（`ai/prompts/background.go`）

在现有 `SystemPrompt` 基础上补充边界条件：

```
（现有内容不变）

补充规则:
6. 如果提供的数据不足以判断根因（如只有风险分数无具体异常指标），在 rootCauseAnalysis 中说明 "数据不足，建议进一步调查"
7. 如果无历史相似事件，similarPattern 填写 "无匹配的历史模式"
```

---

## 4. 文件变更清单

### 4.1 目录结构标注

```
atlhyper_master_v2/
├── ai/
│   ├── interfaces.go           ← [修改] 新增 Analyze() + Complete() 方法签名
│   ├── factory.go              ← [修改] NewService 参数调整
│   ├── service.go              ← [修改] 实现 Analyze() + Complete()
│   ├── analyze.go              ← [新增] 通用多轮 Tool Calling 循环（从 aiops/enricher/analysis.go 提取）
│   ├── chat.go                 ← [修改] chatLoop 提取共用逻辑到 analyze.go
│   ├── prompts.go              ← [删除] 拆分到 prompts/ 子包
│   └── prompts/                ← [新增] 提示词统一管理
│       ├── security.go         ←   L0 安全约束
│       ├── chat.go             ←   chat 角色提示词（重写）
│       ├── background.go       ←   background 角色提示词（从 aiops/enricher/prompts.go 迁入）
│       ├── analysis.go         ←   analysis 角色提示词（从 aiops/enricher/analysis.go 迁入 + 重写）
│       └── tools.go            ←   Tool JSON 定义 + 加载/缓存函数
└── aiops/
    └── enricher/               ← [重命名] aiops/ai/ → aiops/enricher/
        ├── enricher.go         ← [修改] 原 enhancer.go，不再 import ai/llm，改用 ai.AIService 接口
        ├── background.go       ← [不变] 纯领域逻辑
        ├── context_builder.go  ← [修改] 返回类型改为 prompts.BackgroundContext
        ├── prompts.go          ← [删除] 迁入 ai/prompts/background.go
        └── analysis.go         ← [删除] 拆分到 ai/analyze.go + ai/prompts/analysis.go
```

### 4.2 文件变更明细

| 文件 | 变更 | 说明 |
|------|------|------|
| `ai/interfaces.go` | **修改** | 新增 `Analyze()` + `Complete()` 方法签名 |
| `ai/factory.go` | **修改** | 构造函数适配新方法 |
| `ai/service.go` | **修改** | 实现 `Analyze()` + `Complete()` |
| `ai/analyze.go` | **新增** | 通用多轮 Tool Calling 循环 |
| `ai/chat.go` | **修改** | `chatLoop` 复用 `analyze.go` 的循环逻辑 |
| `ai/prompts.go` | **删除** | 拆分到 `ai/prompts/` 子包 |
| `ai/prompts/security.go` | **新增** | 从 `ai/prompts.go` 迁入 `securityPrompt` |
| `ai/prompts/chat.go` | **新增** | 从 `ai/prompts.go` 迁入 `rolePrompt` 并重写 |
| `ai/prompts/background.go` | **新增** | 从 `aiops/enricher/prompts.go` 迁入 + 补充边界条件 |
| `ai/prompts/analysis.go` | **新增** | 从 `aiops/enricher/analysis.go` L233 迁入 + 重写 |
| `ai/prompts/tools.go` | **新增** | 从 `ai/prompts.go` 迁入 `toolsJSON` + 加载函数 |
| `aiops/ai/` → `aiops/enricher/` | **重命名** | 包重命名，`enhancer.go` → `enricher.go` |
| `aiops/enricher/enricher.go` | **修改** | 删除 `import "ai/llm"`，改用 `ai.AIService` 接口 |
| `aiops/enricher/context_builder.go` | **修改** | 返回类型适配 `prompts.BackgroundContext` |
| `aiops/enricher/prompts.go` | **删除** | 已迁入 `ai/prompts/background.go` |
| `aiops/enricher/analysis.go` | **删除** | 循环逻辑迁入 `ai/analyze.go`，提示词迁入 `ai/prompts/analysis.go` |
| `master.go` | **修改** | 初始化流程适配新接口 + 导入路径 `aiops/ai` → `aiops/enricher` |

**总计：8 个修改，5 个新增，3 个删除，1 个重命名。**

---

## 5. 实施路线

### Phase 1：提示词迁移 + 子包创建

1. 创建 `ai/prompts/` 子包
2. 迁移 `security.go`、`tools.go`（从 `ai/prompts.go`）
3. 迁移 `chat.go` 并重写提示词内容
4. 迁移 `background.go`（从 `aiops/enricher/prompts.go`）并补充边界条件
5. 迁移 `analysis.go`（从 `aiops/enricher/analysis.go` L233）并重写
6. 更新所有导入路径（`ai/chat.go`、`aiops/enricher/enricher.go` 等）
7. 删除 `ai/prompts.go`、`aiops/enricher/prompts.go`
8. `go build` 验证

**Phase 1 完成标志**：所有提示词在 `ai/prompts/` 统一管理，编译通过，行为不变。

### Phase 2：`ai` 接口扩展

1. `ai/interfaces.go` 新增 `Analyze()` + `Complete()` 签名
2. `ai/analyze.go` 实现通用多轮 Tool Calling 循环
3. `ai/service.go` 实现 `Complete()`（单轮 LLM 调用）
4. `go build` 验证新接口编译通过

**Phase 2 完成标志**：`ai.AIService` 暴露 `Chat` + `Analyze` + `Complete` 三个入口。

### Phase 3：`aiops/ai/` → `aiops/enricher/` 重命名 + 解耦

1. `aiops/ai/` 目录重命名为 `aiops/enricher/`，`enhancer.go` → `enricher.go`
2. `aiops/enricher/enricher.go` 改为注入 `ai.AIService` 接口
   - `Summarize()` 改用 `aiService.Complete()`
   - `SummarizeBackground()` 同上
   - 删除 `LLMClientFactory` 类型和所有直接 LLM 操作
3. `aiops/enricher/analysis.go` 删除 — `RunAnalysis()` 入口移到 enricher，调用 `aiService.Analyze()`
4. `aiops/enricher/context_builder.go` 返回类型适配
5. `master.go` 初始化流程更新 — 导入路径 `aiops/ai` → `aiops/enricher`，移除 `ToolExecuteFunc`/`AnalysisConfig` 注入
6. 全局搜索替换所有 `aiops/ai` 导入路径
7. `go build` 验证

**Phase 3 完成标志**：`aiops/ai/` 目录不再存在，`aiops/enricher/` 不 `import "ai/llm"`，无 `ToolExecuteFunc` hack。

---

## 6. 验证方法

### 6.1 架构验证

```bash
# 确认 aiops/enricher 不再直接导入 ai/llm
grep -r '"AtlHyper/atlhyper_master_v2/ai/llm"' atlhyper_master_v2/aiops/
# 预期：无输出

# 确认 ToolExecuteFunc 类型已移除
grep -r 'ToolExecuteFunc' atlhyper_master_v2/
# 预期：无输出

# 确认提示词全部在 ai/prompts/
ls atlhyper_master_v2/ai/prompts/
# 预期：security.go chat.go background.go analysis.go tools.go
```

### 6.2 功能验证

1. AI Chat 交互对话 — 行为不变
2. 事件自动摘要（background） — 行为不变
3. 高危事件深度分析（analysis） — 行为不变
4. 三个角色使用不同的系统提示词 — 通过日志确认

### 6.3 提示词效果验证

1. Chat 问 "集群状态怎么样" → 应调用 `get_cluster_risk`
2. Chat 问 "某 Pod CrashLoopBackOff" → 应按 describe → logs → events 流程
3. Analysis 深度分析 → 输出含 confidence 的结构化报告
4. Background 无历史匹配 → `similarPattern` 输出 "无匹配的历史模式"
