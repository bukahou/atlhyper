# AtlHyper AI 系统设计方案

> 版本: v2.0 (WebChat Only)
> 日期: 2025-01-24
> 状态: 待实施

---

## 一、设计目标

### 1.1 核心功能
- **AI 对话**: 用户通过 Web 聊天界面与 AI 交互，查询集群状态、分析问题
- **会话管理**: 多轮对话、历史记录、上下文保持
- **Tool Calling**: AI 通过 MQ 直接与集群 apiserver 交互获取实时数据
- **行为审计**: 所有 AI 指令持久化到 DB（已完成）

### 1.2 安全约束
- **黑名单限制**: 禁止写操作、禁止访问敏感资源
- **Tool 级校验**: 在 Tool 执行前校验，而非用户输入时
- **提示词分级**: 安全约束层不可被覆盖

### 1.3 简化决策
- **不做**: 定时自律型 AI（Scheduled/Autonomous）
- **不做**: DataHub 依赖（AI 只通过 MQ 获取实时数据）
- **不做**: 多 LLM 同时使用（工厂模式预留，首期只实现 Gemini）

---

## 二、整体架构

```
┌─────────────────────────────────────────────────────┐
│                    Web Frontend                       │
│  ┌─────────────────────────────────────────────────┐│
│  │              AI Chat Page (SSE)                  ││
│  └─────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│                   Gateway Layer                       │
│  ┌──────────┐  ┌─────────────┐  ┌───────────────┐  │
│  │Auth (JWT)│→ │ AI Handler  │  │ SSE Writer    │  │
│  └──────────┘  └─────────────┘  └───────────────┘  │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│                    AI Module                          │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │              AIService.Chat()                 │   │
│  │  1. Blacklist 校验                            │   │
│  │  2. Prompt 构建 (security + role + history)   │   │
│  │  3. LLM StreamChat (Function Calling)         │   │
│  │  4. Tool 执行循环                             │   │
│  │  5. 消息持久化                                │   │
│  └──────────────────────────────────────────────┘   │
│       │              │              │                │
│       ▼              ▼              ▼                │
│  ┌─────────┐  ┌───────────┐  ┌──────────────┐      │
│  │Blacklist│  │  Prompt   │  │ Tool Executor│      │
│  └─────────┘  └───────────┘  └──────────────┘      │
│                                     │                │
│                     ┌───────────────┤                │
│                     ▼               ▼                │
│              ┌───────────┐   ┌───────────┐           │
│              │ LLMClient │   │CommandSvc │           │
│              │ (Gemini)  │   │  + MQ     │           │
│              └───────────┘   └───────────┘           │
└─────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────┐
│                   Agent V2                            │
│        (通过 MQ TopicAI 接收指令，执行后返回结果)       │
└─────────────────────────────────────────────────────┘
```

---

## 三、AI 数据流

### 3.1 AI 与底层设施的关系

```
AI 只依赖:
  ├── MQ (Producer)  → 下发指令 + 等待结果
  └── DB             → 会话/消息持久化 + 指令审计(已自动完成)

AI 不依赖:
  └── DataHub        → AI 不读快照，直接通过 MQ 查实时数据
```

### 3.2 Chat 完整流程

```
用户: "Pod nginx-abc 为什么 CrashLoopBackOff?"
    │
    ├─→ 1. Blacklist: 检查意图（只读查询，通过）
    │
    ├─→ 2. Prompt 构建:
    │     [L0] security.txt (不可覆盖的安全约束)
    │     [L1] role.txt (身份定义)
    │     [L2] tools.json (可用工具定义)
    │     [L3] conversation history (多轮上下文)
    │     [L4] user input
    │
    ├─→ 3. LLM.StreamChat() → 流式响应开始
    │     SSE: {"type":"text","content":"让我查看一下这个 Pod 的日志..."}
    │
    │     LLM 返回 tool_call:
    │     SSE: {"type":"tool_call","name":"get_logs","params":{...}}
    │
    ├─→ 4. Tool 执行:
    │     Blacklist.Check(get_logs, Pod, default) → 通过
    │     CommandService.CreateCommand(Source="ai") → MQ 入队
    │     bus.WaitCommandResult(cmdID, 30s) → 日志内容
    │     SSE: {"type":"tool_result","content":"<日志内容>"}
    │
    ├─→ 5. 将 tool_result 反馈给 LLM，继续生成
    │     LLM 可能再次 tool_call（如需更多信息）
    │     或直接输出分析文本:
    │     SSE: {"type":"text","content":"根据日志分析，原因是..."}
    │
    ├─→ 6. 循环直到 LLM 完成输出
    │     SSE: {"type":"done"}
    │
    └─→ 7. 持久化:
          保存 user message + assistant message(含 tool_calls) 到 DB
          更新 conversation.message_count
```

---

## 四、模块设计

### 4.1 目录结构

```
atlhyper_master_v2/ai/
├── interfaces.go              # AIService 接口 + 请求/响应类型（对外）
├── service.go                 # aiServiceImpl + NewAIService 工厂
├── chat.go                    # Chat 核心逻辑（多轮 tool calling 循环）
├── tool.go                    # Tool 执行器（CommandService + WaitCommandResult）
├── blacklist.go               # 黑名单校验
├── prompt.go                  # 提示词构建
├── llm/                       # LLM 抽象层
│   ├── interfaces.go          # LLMClient 接口 + 类型定义
│   ├── factory.go             # NewLLMClient 工厂
│   └── gemini/
│       └── client.go          # Gemini Function Calling + Stream 实现
└── prompts/                   # 提示词文件（go:embed）
    ├── security.txt           # L0: 安全约束
    ├── role.txt               # L1: 角色定义
    └── tools.json             # L2: Function Calling Schema
```

### 4.2 接口定义

#### AIService（对外接口）

```go
// ai/interfaces.go

type AIService interface {
    CreateConversation(ctx context.Context, userID int64, clusterID, title string) (*Conversation, error)
    Chat(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error)
    GetConversations(ctx context.Context, userID int64) ([]*Conversation, error)
    GetMessages(ctx context.Context, conversationID int64) ([]*Message, error)
    DeleteConversation(ctx context.Context, conversationID int64) error
}

type ChatRequest struct {
    ConversationID int64  // 对话 ID
    ClusterID      string // 目标集群
    UserID         int64  // 用户 ID
    Message        string // 用户消息
}

type ChatChunk struct {
    Type    string `json:"type"`    // text / tool_call / tool_result / done / error
    Content string `json:"content"` // 文本内容
    Tool    string `json:"tool,omitempty"`   // tool 名称
    Params  string `json:"params,omitempty"` // tool 参数 JSON
}

type Conversation struct {
    ID           int64
    UserID       int64
    ClusterID    string
    Title        string
    MessageCount int
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type Message struct {
    ID             int64
    ConversationID int64
    Role           string // user / assistant / tool
    Content        string
    ToolCalls      string // JSON
    CreatedAt      time.Time
}
```

#### LLMClient（内部接口）

```go
// ai/llm/interfaces.go

type LLMClient interface {
    ChatStream(ctx context.Context, req *Request) (<-chan *Chunk, error)
}

type Request struct {
    SystemPrompt string
    Messages     []Message        // 历史消息
    Tools        []ToolDefinition // Function Calling 定义
}

type Message struct {
    Role       string // user / assistant / tool
    Content    string
    ToolCalls  []ToolCall   // assistant 发起的 tool calls
    ToolResult *ToolResult  // tool 执行结果
}

type ToolCall struct {
    ID     string // tool call ID
    Name   string // 函数名
    Params string // JSON 参数
}

type ToolResult struct {
    CallID  string // 对应的 tool call ID
    Content string // 执行结果
}

type ToolDefinition struct {
    Name        string
    Description string
    Parameters  json.RawMessage // JSON Schema
}

type Chunk struct {
    Type     string    // text / tool_call / done / error
    Content  string    // 文本片段
    ToolCall *ToolCall // tool call 信息
    Error    error     // 错误信息
}
```

### 4.3 AIService 依赖

```go
type aiServiceImpl struct {
    llm      llm.LLMClient                    // LLM API
    ops      service.Ops                       // CreateCommand
    bus      mq.Producer                       // WaitCommandResult
    convRepo database.AIConversationRepository // 会话持久化
    msgRepo  database.AIMessageRepository      // 消息持久化
}
```

### 4.4 Tool 定义

AI 可用的 Tools（对应 Agent 的 Actions）：

| Tool 名称 | 对应 Action | 说明 |
|-----------|-------------|------|
| get_pod_logs | get_logs | 获取 Pod 日志 |
| get_pod_describe | dynamic | kubectl describe pod |
| get_deployment_status | dynamic | 查询 Deployment 状态 |
| get_events | dynamic | 查询集群事件 |
| get_configmap | get_configmap | 获取 ConfigMap 数据 |
| get_node_status | dynamic | 查询 Node 状态 |
| list_pods | dynamic | 列出 Pod |

所有 Tool 执行前经过 Blacklist 校验。

---

## 五、黑名单设计

### 5.1 校验时机

在 **Tool 执行前**校验，不阻拦用户输入：

```
用户输入 → LLM 分析 → tool_call → Blacklist.Check() → 通过/拒绝
                                                          │
                                            拒绝 → 告知 LLM "无权限"
                                            LLM 重新生成回复告知用户
```

### 5.2 黑名单规则

```go
// 禁止的写操作
var forbiddenActions = map[string]bool{
    "scale": true, "restart": true, "delete": true,
    "delete_pod": true, "exec": true, "cordon": true,
    "uncordon": true, "drain": true, "update_image": true,
}

// 禁止的命名空间
var forbiddenNamespaces = map[string]bool{
    "kube-system": true, "kube-public": true, "kube-node-lease": true,
}

// 禁止的资源类型
var forbiddenResources = map[string]bool{
    "Secret": true,
}
```

### 5.3 权限对比

| 操作 | AI | Web |
|------|-----|-----|
| 查询 Pod/Node/Deployment 状态 | ✅ | ✅ |
| 查看 Pod 日志 | ✅ | ✅ |
| 查看 ConfigMap | ✅ | ✅ |
| 查看 Secret | ❌ | ✅ |
| 访问 kube-system | ❌ | ✅ |
| 扩缩容/重启/删除 | ❌ | ✅ |

---

## 六、数据库设计

### 6.1 新增表

```sql
-- AI 对话表
CREATE TABLE ai_conversations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    cluster_id VARCHAR(100) NOT NULL,
    title VARCHAR(200),
    message_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- AI 消息表
CREATE TABLE ai_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id INTEGER NOT NULL,
    role VARCHAR(20) NOT NULL,       -- user / assistant / tool
    content TEXT NOT NULL,
    tool_calls TEXT,                  -- JSON: [{id, name, params, result}]
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (conversation_id) REFERENCES ai_conversations(id) ON DELETE CASCADE
);

CREATE INDEX idx_ai_conv_user ON ai_conversations(user_id);
CREATE INDEX idx_ai_conv_cluster ON ai_conversations(cluster_id);
CREATE INDEX idx_ai_msg_conv ON ai_messages(conversation_id);
```

### 6.2 Repository 接口

```go
// database/interfaces.go 新增

type AIConversationRepository interface {
    Create(ctx context.Context, conv *AIConversation) error
    Update(ctx context.Context, conv *AIConversation) error
    Delete(ctx context.Context, id int64) error
    GetByID(ctx context.Context, id int64) (*AIConversation, error)
    ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*AIConversation, error)
}

type AIMessageRepository interface {
    Create(ctx context.Context, msg *AIMessage) error
    ListByConversation(ctx context.Context, convID int64) ([]*AIMessage, error)
    DeleteByConversation(ctx context.Context, convID int64) error
}
```

### 6.3 已有设施（无需新建）

- `command_history` 表: 已有，AI 指令自动持久化（Source="ai"）
- `audit_logs` 表: 已有，Gateway 层自动记录

---

## 七、提示词设计

### 7.1 层级

```
L0: security.txt   — 安全约束（不可覆盖）
L1: role.txt       — 角色定义 + 能力说明
L2: tools.json     — Function Calling Schema
L3: history        — 会话历史（从 DB 加载）
L4: user input     — 用户当前消息
```

### 7.2 安全约束要点 (security.txt)

- 你是只读分析助手，禁止执行任何写操作
- 禁止查询 Secret 资源
- 禁止访问 kube-system/kube-public/kube-node-lease 命名空间
- 禁止输出密码、Token、API Key 等敏感信息
- 只回答与 Kubernetes 集群相关的问题
- 无法回答时明确告知用户

### 7.3 角色定义要点 (role.txt)

- 你是 AtlHyper 集群运维助手
- 擅长 Kubernetes 问题诊断、状态分析、日志解读
- 通过 Tool 获取集群实时数据进行分析
- 回复使用中文，技术术语保留英文

---

## 八、Gateway API

### 8.1 路由

| Method | Path | 说明 | 权限 |
|--------|------|------|------|
| POST | `/api/v2/ai/conversations` | 创建对话 | Viewer+ |
| GET | `/api/v2/ai/conversations` | 获取对话列表 | Viewer+ |
| POST | `/api/v2/ai/chat` | 发送消息 (SSE) | Viewer+ |
| GET | `/api/v2/ai/conversations/{id}/messages` | 获取历史消息 | Viewer+ |
| DELETE | `/api/v2/ai/conversations/{id}` | 删除对话 | Viewer+ |

### 8.2 SSE 响应格式

```
event: message
data: {"type":"text","content":"让我查看一下..."}

event: message
data: {"type":"tool_call","tool":"get_pod_logs","params":"{\"pod\":\"nginx-abc\"}"}

event: message
data: {"type":"tool_result","content":"Error: container exited with code 1..."}

event: message
data: {"type":"text","content":"根据日志分析，Pod 崩溃的原因是..."}

event: message
data: {"type":"done"}
```

---

## 九、配置项

```go
// config/types.go 新增

type AIConfig struct {
    Enabled  bool   // 是否启用 AI 功能
    Provider string // LLM 提供商: gemini / openai / claude
    Gemini   GeminiConfig
}

type GeminiConfig struct {
    APIKey string // Gemini API Key
    Model  string // 模型名称 (gemini-2.0-flash)
}
```

环境变量:
```
MASTER_AI_ENABLED=true
MASTER_AI_PROVIDER=gemini
MASTER_AI_GEMINI_API_KEY=xxx
MASTER_AI_GEMINI_MODEL=gemini-2.0-flash
```

---

## 十、安全检查清单

- [ ] Blacklist: 禁止写操作 (scale/restart/delete/exec/cordon/drain/update_image)
- [ ] Blacklist: 禁止访问 Secret
- [ ] Blacklist: 禁止访问 kube-system/kube-public/kube-node-lease
- [ ] 所有 AI 指令关联 UserID + Source="ai"
- [ ] security.txt 不可被用户提示词覆盖
- [ ] Tool 执行结果中的敏感信息过滤
- [ ] 单次对话 Token 限制
- [ ] Tool 执行超时处理 (30s)
