# AI 多角色路由设计

> 状态: active | 创建: 2026-03-06

## 背景

当前 AI 模块使用单一 Provider 配置（`ai_active_config.provider_id`），所有 AI 功能共享同一个 LLM。
但不同 AI 功能对模型能力、成本、延迟的要求差异很大：

| 功能 | 场景 | 对模型的要求 | 典型选型（仅参考） |
|------|------|-------------|-------------------|
| **后台分析** | AIOps 事件摘要，24h 自动运行 | 低成本、可用即可 | Ollama / Gemini Flash |
| **Chat** | 用户交互式对话，Tool Calling | 快速响应、工具调用能力强 | Gemini Flash / GPT-4o-mini |
| **深度分析** | 复杂根因分析，需强推理 | 强推理、大上下文 | Gemini Pro / Claude / GPT-4o |

> **角色与 Provider 不硬绑定**：任何角色可分配给任何 Provider，上下文管理自动适配。
> 上表的"典型选型"仅为参考，不构成约束。

**目标**: 让每个 Provider 可以被分配不同的工作角色，并对高成本角色设置用量预算，全部通过 Web UI 配置。

---

## 设计概览

### 用户心智模型

```
1. 配置 Provider（已有功能：Ollama / Gemini / Claude / OpenAI）
2. 给 Provider 分配工作角色（新增）
3. 有角色的 Provider = 激活；无角色 = 待机
4. 一个 Provider 可承担多个角色，但一个角色只能属于一个 Provider
5. 重新分配角色前，必须先从当前 Provider 移除该角色
```

### 架构图

```
  ┌──────────────────────────────────────────────────────┐
  │                   ai_providers 表                     │
  │                                                      │
  │  Ollama  [background]          ← 激活（有角色）       │
  │  Gemini  [chat, analysis]      ← 激活（有角色）       │
  │  Claude  []                    ← 待机（无角色）       │
  │  OpenAI  []                    ← 待机（无角色）       │
  └──────┬───────────┬─────────────────────────────────────┘
         │           │
         ▼           ▼
   AIOps Enhancer  AI Chat / Deep Analysis
   role=background role=chat  role=analysis
```

### 核心原则

1. **角色绑定在 Provider 侧**: 在 Provider 卡片上分配角色，而非独立的角色配置表
2. **互斥约束**: 3 个角色（background / chat / analysis），每个角色最多被一个 Provider 持有
3. **未分配=未激活**: 没有角色的 Provider 不参与任何 AI 调用
4. **向后兼容**: 无角色分配时，退回到 `ai_active_config.provider_id`（现有行为不变）
5. **切换无需特殊设计**: 当前请求用旧 Provider 完成，下次请求自然用新 Provider
6. **零停机**: 角色配置热更新，每次调用从 DB 读取最新配置

---

## 数据模型

### 方案: `ai_providers` 新增 `roles` 列 + 独立预算表

角色分配直接存在 Provider 上（直观、UI 操作自然），预算/用量单独一张表（按角色统计）。

### `ai_providers` 表变更

```sql
-- 新增列
ALTER TABLE ai_providers ADD COLUMN roles TEXT DEFAULT '[]';
-- roles: JSON 数组，如 '["background","chat"]' 或 '[]'
```

> **注意**: `context_window` 不存在 Provider 上，而是跟着模型走。见下方「上下文窗口配置」章节。

### 新增表: `ai_role_budget`

```sql
CREATE TABLE IF NOT EXISTS ai_role_budget (
    role             TEXT PRIMARY KEY,    -- 'background' / 'chat' / 'analysis'

    -- 预算控制（0 = 无限制）
    daily_token_limit   INTEGER DEFAULT 0,
    daily_call_limit    INTEGER DEFAULT 0,

    -- 降级配置
    fallback_provider_id INTEGER,        -- 预算耗尽时的降级 Provider

    -- 自动触发配置（analysis 角色专用）
    auto_trigger_min_severity TEXT DEFAULT 'critical',
    -- 可选值: 'critical' / 'high' / 'medium' / 'low' / 'off'
    -- 'off' = 禁用自动触发，仅支持手动触发
    -- 非 analysis 角色忽略此字段

    -- 当日用量（跨日自动重置）
    daily_tokens_used   INTEGER DEFAULT 0,
    daily_calls_used    INTEGER DEFAULT 0,
    daily_reset_at      TEXT,            -- 上次重置时间 (RFC3339)

    updated_at          TEXT NOT NULL,

    FOREIGN KEY (fallback_provider_id) REFERENCES ai_providers(id)
);
```

### Go 模型

```go
// database/types.go

// AIProvider 新增字段
type AIProvider struct {
    // ... 已有字段 ...
    Roles []string // JSON 解码后的角色列表: ["background","chat"]
}

// AIProviderModel 扩展字段
type AIProviderModel struct {
    // ... 已有字段 ...
    ContextWindow int // 上下文窗口大小(tokens)，0=无限制
}

// AIRoleBudget 角色预算配置
type AIRoleBudget struct {
    Role               string
    DailyTokenLimit    int    // 0 = 无限制
    DailyCallLimit     int    // 0 = 无限制
    FallbackProviderID *int64 // 降级 Provider（可选）

    // analysis 角色专用: 自动触发的最低严重度
    // "critical" / "high" / "medium" / "low" / "off"
    // "off" = 禁用自动触发; 非 analysis 角色忽略
    AutoTriggerMinSeverity string

    DailyTokensUsed    int
    DailyCallsUsed     int
    DailyResetAt       *time.Time
    UpdatedAt          time.Time
}
```

### 角色常量与现状

```go
// ai/role.go

const (
    RoleBackground = "background"  // AIOps 后台分析（单轮，低上下文要求）
    RoleChat       = "chat"        // 用户交互 Chat（多轮 Tool Calling）
    RoleAnalysis   = "analysis"    // 深度分析（预留，无后端实现）
)

var ValidRoles = []string{RoleBackground, RoleChat, RoleAnalysis}
```

**各角色后端实现现状:**

| 角色 | 调用入口 | 现状 | 说明 |
|------|---------|------|------|
| `chat` | `ai/chat.go` → `chatLoop` | **完善** | 多轮 Tool Calling + SSE + 历史持久化 |
| `background` | `aiops/ai/enhancer.go` → `Summarize` | **基本可用** | 单轮 LLM 事件摘要，手动/被动触发 |
| `analysis` | -- | **未实现** | 无代码路径，仅作为枚举预留 |

- `chat` + `background`: 本设计直接接入角色路由，立即生效
- `analysis`: 前端 UI 显示灰色 + "即将推出" 标签，暂不可分配
- `background` 的完善（24h 自动触发等）和 `analysis` 的具体定义，需要独立设计文档

### `ai_active_config` 保留（向后兼容）

```
ai_active_config.provider_id = 兜底 Provider
当没有任何角色分配时，所有 AI 功能使用此 Provider（与现有行为一致）
```

---

## 互斥约束

### 规则

```
一个角色 → 最多一个 Provider
一个 Provider → 0~N 个角色
总共 3 个角色 → 最多 3 个 Provider 被激活
```

### 分配/移除流程

```
分配 "chat" 给 Gemini:
  1. 检查 "chat" 是否已被其他 Provider 持有
     → 是 → 返回错误 "chat 角色已被 [Ollama] 持有，请先移除"
     → 否 → Gemini.roles 追加 "chat"，保存

移除 "chat" 从 Ollama:
  1. Ollama.roles 移除 "chat"，保存
  2. 此后可将 "chat" 分配给其他 Provider
```

### API 层校验

```go
// 分配角色前检查互斥
func validateRoleAssignment(ctx context.Context, providerID int64, role string, repo AIProviderRepository) error {
    providers, _ := repo.List(ctx)
    for _, p := range providers {
        if p.ID == providerID { continue }
        for _, r := range p.Roles {
            if r == role {
                return fmt.Errorf("角色 %s 已被 [%s] 持有，请先移除", role, p.Name)
            }
        }
    }
    return nil
}
```

---

## 上下文窗口配置（基于模型）

### 设计: context_window 跟着模型走

上下文窗口是**模型属性**，不是 Provider 属性。同一个 Ollama 服务器换个模型（7B → 32B）上下文就不同。
因此 `context_window` 存在已有的 `ai_provider_models` 表上，而非 `ai_providers` 表。

### `ai_provider_models` 表变更

```sql
ALTER TABLE ai_provider_models ADD COLUMN context_window INTEGER DEFAULT 0;
-- 0 = 无限制（云端大模型默认不限制）
```

### 默认值（`initDefaultAIModels` 中填充）

```
gemini  | gemini-2.5-flash       | 1048576  (1M)
gemini  | gemini-2.5-flash-lite  | 1048576  (1M)
gemini  | gemini-2.5-pro         | 1048576  (1M)
openai  | gpt-4o                 | 128000   (128K)
openai  | gpt-4o-mini            | 128000   (128K)
openai  | o1                     | 200000   (200K)
anthropic | claude-sonnet-4      | 200000   (200K)
anthropic | claude-opus-4.5      | 200000   (200K)
anthropic | claude-3.5-sonnet    | 200000   (200K)
ollama  | qwen2.5:14b            | 8192     (8K, Ollama 默认)
ollama  | qwen2.5:7b             | 8192
ollama  | qwen2.5:32b            | 8192
ollama  | llama3.1:8b            | 8192
ollama  | deepseek-r1:14b        | 8192
```

### context_window 解析流程

```
1. 用户在 Provider 中选择模型 (e.g. qwen2.5:14b)
2. 从 ai_provider_models 查询该模型的默认 context_window → 8192
3. Provider 卡片显示 "上下文: 8,192 tokens"
4. 用户可手动覆盖（如 Ollama 启动时指定了 num_ctx=32768 → 改为 32768）
5. 手动覆盖值存在 ai_providers 表新增列: context_window_override
   → 0 = 使用模型默认值; >0 = 用户覆盖值
```

```sql
ALTER TABLE ai_providers ADD COLUMN context_window_override INTEGER DEFAULT 0;
-- 0 = 使用 ai_provider_models 中模型的默认值
-- >0 = 用户手动覆盖（如 Ollama 自定义了 num_ctx）
```

### 解析逻辑

```go
// 获取 Provider 的有效 context_window
func effectiveContextWindow(provider *AIProvider, modelRepo AIProviderModelRepository) int {
    // 用户覆盖优先
    if provider.ContextWindowOverride > 0 {
        return provider.ContextWindowOverride
    }
    // 查模型默认值
    model, _ := modelRepo.FindByProviderAndModel(provider.Provider, provider.Model)
    if model != nil && model.ContextWindow > 0 {
        return model.ContextWindow
    }
    // 兜底: 0 = 不限制
    return 0
}
```

### 对用户的价值

**系统侧（自动生效，用户无需操作）：**

| 用途 | 说明 |
|------|------|
| 防止静默截断 | Ollama 超限不报错直接丢内容。有 context_window 后，系统在发送前主动裁剪并告知用户 |
| Tool 结果截断强度 | 8K 模型的 Tool 结果只保留 2K 字符；1M 模型保留 32K |
| AIOps Prompt 截断 | Enhancer 的 MaxPromptChars 从写死 16K 改为根据模型动态调整 |

**用户侧（展示 + 决策参考）：**

| 用途 | 说明 |
|------|------|
| 模型能力一目了然 | Provider 卡片显示 "上下文: 8,192 tokens"，用户知道模型能处理多少信息 |
| 角色适配提示 | 分配 Chat 角色时提示："Chat 角色推荐 32K+ 上下文，当前模型仅 8K" |
| 对话健康度（后续） | Chat 页面可显示当前对话已消耗的 token 估算，接近上限时变色 |

**用户可管理（手动覆盖）：**

| 场景 | 操作 |
|------|------|
| Ollama 自定义 num_ctx | 用户启动 Ollama 时指定了 `num_ctx=32768`，在 Provider 编辑页覆盖为 32768 |
| 限制云端模型成本 | Gemini 有 1M 上下文但想省钱，手动设为 32K，系统会主动裁剪减少 token 用量 |

---

## 上下文管理器（ContextManager）

### 问题

不同模型的上下文窗口差异巨大，溢出行为各不相同：

| Provider | 模型 | 上下文窗口 | 溢出行为 |
|----------|------|-----------|---------|
| Gemini Flash | gemini-2.5-flash | 1M tokens | 无问题 |
| GPT-4o | gpt-4o | 128K tokens | 返回错误 |
| Claude | claude-sonnet-4 | 200K tokens | 无问题 |
| **Ollama** | **qwen2.5:14b** | **8K tokens** | **静默截断（丢弃最早内容，不报错）** |

Ollama 的静默截断最危险：HTTP 200 正常返回，但 AI 已经"忘记"被丢弃的内容，导致分析不完整或矛盾。

### 上下文预算计算

```
可用上下文 = context_window - system_prompt_tokens - output_reserve

以 Ollama qwen2.5:14b (8192 tokens) 为例:
  system_prompt  ≈ 3000 tokens (AI Chat 的 system prompt ~11KB)
  output_reserve ≈ 1500 tokens (留给模型生成回答)
  ────────────────────────────
  可用于历史消息  ≈ 3700 tokens

  一轮 Tool Calling 的消息量:
    assistant (tool_use)  ≈ 200 tokens
    tool_result (数据)    ≈ 1000~5000 tokens
  ────────────────────────────
  结论: Ollama 8K 最多支撑 1~2 轮 Tool Calling
```

### ContextManager 实现

```go
// ai/context.go

// ContextManager 上下文管理器
// 根据 Provider 的 context_window 在发送前裁剪消息，防止静默截断
type ContextManager struct {
    contextWindow int // Provider 上下文窗口 (tokens), 0 = 不裁剪
    outputReserve int // 为输出保留的 token 数
}

// NewContextManager 创建上下文管理器
func NewContextManager(contextWindow int) *ContextManager {
    if contextWindow <= 0 {
        return &ContextManager{contextWindow: 0} // 无限制
    }
    // 为输出保留 ~20% 的上下文
    reserve := contextWindow / 5
    if reserve < 1000 {
        reserve = 1000
    }
    return &ContextManager{
        contextWindow: contextWindow,
        outputReserve: reserve,
    }
}

// FitMessages 裁剪消息列表以适应上下文窗口
// 策略: 从最新消息往回填充，保证最近的对话完整
// 返回: 裁剪后的消息列表 + 是否发生了裁剪
func (cm *ContextManager) FitMessages(systemPrompt string, messages []llm.Message) ([]llm.Message, bool) {
    if cm.contextWindow <= 0 {
        return messages, false // 无限制，原样返回
    }

    budget := cm.contextWindow - cm.outputReserve - estimateTokens(systemPrompt)
    if budget <= 0 {
        // system prompt 已经超限（极端情况）
        return messages, true
    }

    // 从最新消息往前，贪心填充
    var fitted []llm.Message
    for i := len(messages) - 1; i >= 0; i-- {
        cost := estimateMessageTokens(&messages[i])
        if budget-cost < 0 && len(fitted) > 0 {
            break // 预算不够了，停止
        }
        budget -= cost
        fitted = append([]llm.Message{messages[i]}, fitted...)
    }

    truncated := len(fitted) < len(messages)
    return fitted, truncated
}

// estimateTokens 估算文本的 token 数
// 粗估: 英文 ~4 chars/token, 中文 ~1.5 chars/token
// 取保守值: 1 token ≈ 2.5 字符
func estimateTokens(text string) int {
    chars := len([]rune(text))
    return (chars*10 + 24) / 25 // 等价于 chars / 2.5 向上取整
}

// estimateMessageTokens 估算单条消息的 token 数
func estimateMessageTokens(msg *llm.Message) int {
    tokens := estimateTokens(msg.Content)
    // Tool Call 参数
    for _, tc := range msg.ToolCalls {
        tokens += estimateTokens(tc.Params) + 20 // 20 tokens overhead
    }
    // Tool Result
    if msg.ToolResult != nil {
        tokens += estimateTokens(msg.ToolResult.Content) + 20
    }
    return tokens + 5 // 消息 overhead (role 标签等)
}
```

### 各场景适配

#### 1. AI Chat 多轮 Tool Calling (`ai/chat.go`)

```diff
 func (s *aiServiceImpl) chatLoop(ctx context.Context, ...) {
     llmCfg, err := s.loadAIConfigForRole(ctx, RoleChat)
     // ...

+    // 根据 Provider 上下文窗口创建 ContextManager
+    ctxMgr := NewContextManager(llmCfg.ContextWindow)

     for round := 0; round < maxToolRounds; round++ {
+        // 发送前裁剪消息
+        fittedMsgs, truncated := ctxMgr.FitMessages(systemPrompt, messages)
+        if truncated {
+            log.Warn("上下文超限，已裁剪历史消息",
+                "original", len(messages), "fitted", len(fittedMsgs),
+                "provider", llmCfg.Provider, "contextWindow", llmCfg.ContextWindow)
+            // 通知前端
+            ch <- &ChatChunk{Type: "text", Content: "\n[系统: 历史消息已裁剪以适应模型上下文窗口]\n"}
+        }

         llmReq := &llm.Request{
             SystemPrompt: systemPrompt + roundHint,
-            Messages:     messages,
+            Messages:     fittedMsgs,
             Tools:        tools,
         }
         // ...
     }
 }
```

**关键行为**: 当 Ollama 做 Chat 时（虽然不推荐），多轮 Tool Calling 的早期结果会被裁剪。
前端会看到 `[系统: 历史消息已裁剪以适应模型上下文窗口]` 提示，用户知道信息可能不完整。

#### 2. Tool Result 截断强度按上下文调整

```go
// ai/chat.go — 现有 truncate 函数改造

// toolResultMaxLen 根据 Provider 上下文窗口决定 Tool 结果最大长度
func toolResultMaxLen(contextWindow int) int {
    if contextWindow <= 0 {
        return 32000 // 云端大模型: 32K 字符
    }
    if contextWindow <= 8192 {
        return 2000 // Ollama 8K: 2K 字符（约 800 tokens）
    }
    if contextWindow <= 32768 {
        return 8000 // 中等模型: 8K 字符
    }
    return 32000
}
```

#### 3. AIOps Enhancer (`aiops/ai/enhancer.go`)

Enhancer 已有 `buildPromptWithTruncation`（上限 16K chars）。改造为感知模型上下文窗口：

```go
// 方案: Enhancer 的 MaxPromptChars 从写死常量改为根据 context_window 动态计算
// context_window 通过 llmFactory 解析后传给 Enhancer

func maxPromptCharsForContext(contextWindow int) int {
    if contextWindow <= 0 {
        return 16000 // 云端大模型: 保持现有默认值
    }
    // 上下文窗口的 50% 给 prompt（另 50% 给输出 + overhead）
    // 1 token ≈ 2.5 chars（中文保守估计）
    chars := contextWindow / 2 * 25 / 10
    if chars < 2000 {
        return 2000
    }
    return chars
}
// Ollama 8K → maxPromptChars = 10000 chars (~4000 tokens)
// Gemini 1M → maxPromptChars = 16000 chars (保持默认)
```

---

## 角色路由解析

### `loadAIConfigForRole` 函数

```go
// ai/role.go

// RoleConfig 角色路由解析结果
type RoleConfig struct {
    llm.Config
    ContextWindow int // Provider 的上下文窗口
}

// loadAIConfigForRole 按角色加载 AI 配置
// 解析优先级:
//   1. 查找持有该角色的 Provider → 检查预算 → 返回
//   2. 预算耗尽 → 使用 fallback Provider
//   3. 无角色分配 → 退回 ai_active_config.provider_id（向后兼容）
func (s *aiServiceImpl) loadAIConfigForRole(ctx context.Context, role string) (*RoleConfig, error) {
    // 1. 检查 AI 总开关
    active, err := s.activeRepo.Get(ctx)
    if err != nil || active == nil || !active.Enabled {
        return nil, fmt.Errorf("AI 功能未启用")
    }

    // 2. 查找持有该角色的 Provider
    providers, _ := s.providerRepo.List(ctx)
    for _, p := range providers {
        if !containsRole(p.Roles, role) {
            continue
        }

        // 找到了持有该角色的 Provider
        // 检查预算
        if budget, _ := s.budgetRepo.Get(ctx, role); budget != nil {
            if !checkBudget(budget) {
                // 预算耗尽 → 尝试 fallback
                if budget.FallbackProviderID != nil {
                    fallback, err := s.providerRepo.GetByID(ctx, *budget.FallbackProviderID)
                    if err == nil && fallback != nil {
                        log.Warn("角色预算耗尽，使用降级 Provider",
                            "role", role, "fallback", fallback.Name)
                        return providerToRoleConfig(fallback), nil
                    }
                }
                return nil, fmt.Errorf("角色 %s 每日预算已用尽", role)
            }
        }

        return providerToRoleConfig(p), nil
    }

    // 3. 无角色分配 → 退回全局 (向后兼容)
    return s.loadAIConfigFallback(ctx, active)
}
```

---

## 切换行为（无需特殊设计）

```
时间线:
  T1: 用户发送 Chat 消息 → chatLoop 启动 → loadAIConfigForRole("chat") → 获取 Gemini
  T2: 管理员在 Web UI 将 "chat" 从 Gemini 移到 Claude
  T3: chatLoop 仍在用 Gemini 完成多轮 Tool Calling（进程中已持有 Gemini Client）
  T4: chatLoop 结束
  T5: 用户发送新消息 → loadAIConfigForRole("chat") → 获取 Claude
       → 历史消息发送给 Claude（Claude 看到完整对话历史，包括 Gemini 生成的内容）
```

**关键**: 每次 `chatLoop` 启动时创建 LLM Client，运行期间不会切换。
新消息触发新的 `chatLoop` → 自然使用新配置。历史消息（从 DB 加载）包含之前的对话内容，新 Provider 可以理解上下文。

---

## API 设计

### Provider 角色管理（扩展现有 Provider API）

| 方法 | 路径 | 说明 |
|------|------|------|
| PUT | `/api/admin/ai/providers/:id/roles` | 设置 Provider 的角色列表 |
| DELETE | `/api/admin/ai/providers/:id/roles/:role` | 从 Provider 移除指定角色 |

### 角色预算管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/admin/ai/roles` | 获取所有角色状态（哪个 Provider、预算、用量） |
| PUT | `/api/admin/ai/roles/:role/budget` | 设置角色预算 |
| POST | `/api/admin/ai/roles/:role/reset-usage` | 手动重置用量 |

### 请求/响应示例

**PUT `/api/admin/ai/providers/1/roles`** — 给 Ollama 分配角色
```json
{
    "roles": ["background"]
}
```
→ 校验: "background" 未被其他 Provider 持有 → 成功
→ 校验失败: `{"error": "角色 background 已被 [Gemini Flash] 持有，请先移除"}`

**PUT `/api/admin/ai/roles/analysis/budget`** — 设置深度分析预算
```json
{
    "dailyTokenLimit": 500000,
    "dailyCallLimit": 50,
    "fallbackProviderId": 1
}
```

**GET `/api/admin/ai/roles`** — 角色总览
```json
{
    "message": "获取成功",
    "data": [
        {
            "role": "background",
            "roleName": "后台分析",
            "provider": {
                "id": 1,
                "name": "Ollama (本地)",
                "model": "qwen2.5:14b",
                "contextWindow": 8192
            },
            "budget": null,
            "usage": { "dailyTokensUsed": 12500, "dailyCallsUsed": 8 }
        },
        {
            "role": "chat",
            "roleName": "交互对话",
            "provider": {
                "id": 2,
                "name": "Gemini Flash",
                "model": "gemini-2.5-flash",
                "contextWindow": 1000000
            },
            "budget": null,
            "usage": { "dailyTokensUsed": 45000, "dailyCallsUsed": 15 }
        },
        {
            "role": "analysis",
            "roleName": "深度分析",
            "provider": null,
            "budget": {
                "dailyTokenLimit": 500000,
                "dailyCallLimit": 50,
                "fallbackProviderId": 1,
                "fallbackProviderName": "Ollama (本地)"
            },
            "usage": null
        }
    ]
}
```

---

## 前端 UI 设计

### Provider 卡片改造

现有:
```
┌─────────────────────────────┐
│ Ollama (本地)          [编辑] │
│ Provider: ollama             │
│ Model: qwen2.5:14b           │
│ Status: healthy              │
└─────────────────────────────┘
```

改造后:
```
┌─────────────────────────────────────────────┐
│ Ollama (本地)                     [编辑]     │
│ Provider: ollama | Model: qwen2.5:14b       │
│ Context: 8,192 tokens                       │
│                                             │
│ 工作角色:  [后台分析 x]    [+ 分配角色]       │
│                                             │
│ Status: healthy                             │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ Gemini Flash                      [编辑]     │
│ Provider: gemini | Model: gemini-2.5-flash  │
│ Context: 1,000,000 tokens                   │
│                                             │
│ 工作角色:  [交互对话 x]  [深度分析 x]  [+]   │
│                                             │
│  深度分析 预算:                               │
│  Token: ████████░░ 120K / 500K (24%)        │
│  调用:   ████░░░░░░ 12 / 50 (24%)           │
│  降级: → Ollama (本地)         [重置用量]     │
│                                             │
│ Status: healthy                             │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ Claude Sonnet                     [编辑]     │
│ Provider: anthropic | Model: claude-sonnet-4 │
│ Context: 200,000 tokens                     │
│                                             │
│ 工作角色:  (待机)            [+ 分配角色]     │
│                                             │
│ Status: unknown                             │
└─────────────────────────────────────────────┘
```

**UI 交互**:
- `[+ 分配角色]` 点击弹出下拉，仅显示未被分配的角色
- `analysis` 角色显示灰色 + "即将推出" 标签，暂不可分配
- 角色标签上的 `x` 移除角色（确认弹窗）
- 有预算配置的角色直接在卡片内展示进度条
- 预算配置通过点击角色标签弹出编辑弹窗
- Context 行: 显示模型默认值; 若有用户覆盖则显示 "8,192 → 32,768 tokens (已覆盖)"
- Provider 编辑弹窗中可设置 context_window 覆盖值（输入框 + "使用模型默认值" 复选框）

### Settings → AI 页面结构（改造后）

```
Settings → AI
├── GlobalSettingsCard   (AI 总开关 + 默认 Provider + Tool 超时)
│   └── 提示: "未分配角色时，所有 AI 功能使用默认 Provider"
├── ProviderCard[]       (每个 Provider 卡片，含角色分配 + 预算展示)
└── 角色总览 (可选)       (简洁表格: 角色 → Provider → 今日用量)
```

---

## Repository 层

### `AIProviderRepository` 扩展

```go
// 现有接口新增方法
type AIProviderRepository interface {
    // ... 已有 CRUD ...

    // UpdateRoles 更新 Provider 的角色列表
    UpdateRoles(ctx context.Context, id int64, roles []string) error

    // FindByRole 查找持有指定角色的 Provider
    FindByRole(ctx context.Context, role string) (*AIProvider, error)
}
```

### 新增 `AIRoleBudgetRepository`

```go
type AIRoleBudgetRepository interface {
    Get(ctx context.Context, role string) (*AIRoleBudget, error)
    Upsert(ctx context.Context, budget *AIRoleBudget) error
    Delete(ctx context.Context, role string) error

    // 用量操作
    IncrementUsage(ctx context.Context, role string, tokens int) error
    ResetDailyUsage(ctx context.Context, role string) error
}
```

---

## `llm.Config` 扩展

```go
// ai/llm/interfaces.go — Config 不变，context_window 不属于 LLM 协议层
// context_window 在 RoleConfig 中传递，由 ai/role.go 负责解析

// ai/role.go — RoleConfig 包装 llm.Config + 上下文信息
type RoleConfig struct {
    llm.Config
    ContextWindow int // 有效上下文窗口（模型默认值 or 用户覆盖值）
}
```

---

## 文件变更清单

> 路径均相对于 `atlhyper_master_v2/`（前端路径标注 `atlhyper_web/`），遵循项目分层：
> types.go(模型) → interfaces.go(接口+Dialect) → sqlite/(Dialect 实现) → repo/(Repository 实现) → 业务层 → Gateway → 前端

### 新增文件

| # | 文件 | 层级 | 内容 |
|---|------|------|------|
| 1 | `ai/context.go` | 业务层 | ContextManager: FitMessages + estimateTokens |
| 2 | `ai/role.go` | 业务层 | 角色常量 + loadAIConfigForRole + checkBudget |
| 3 | `database/sqlite/ai_role_budget.go` | Dialect 实现 | `AIRoleBudgetDialect` SQLite 实现（SQL 生成 + ScanRow） |
| 4 | `database/repo/ai_role_budget.go` | Repository 实现 | `AIRoleBudgetRepository` 实现（调用 Dialect 执行 SQL） |
| 5 | `gateway/handler/admin/ai_role.go` | Gateway | 角色预算 API Handler |
| 6 | `atlhyper_web/src/api/ai-role.ts` | 前端 | 角色/预算 API 调用 |

### 修改文件

| # | 文件 | 层级 | 变更 |
|---|------|------|------|
| 7 | `database/types.go` | 数据模型 | AIProvider 新增 Roles; AIProviderModel 新增 ContextWindow; 新增 AIRoleBudget struct |
| 8 | `database/interfaces.go` | 接口定义 | 新增 `AIRoleBudgetRepository` + `AIRoleBudgetDialect` 接口; `AIProviderRepository` 扩展 UpdateRoles/FindByRole; `AIProviderDialect` 扩展; `DB.AIRoleBudget` 字段; `Dialect.AIRoleBudget()` 方法 |
| 9 | `database/sqlite/migrations.go` | 迁移 | ai_providers 新增 roles + context_window_override 列; ai_provider_models 新增 context_window 列; 新增 ai_role_budget 表 |
| 10 | `database/sqlite/ai.go` | Dialect 实现 | AIProviderDialect 支持 roles/context_window_override 读写; AIProviderModelDialect 支持 context_window |
| 11 | `database/repo/ai_provider.go` | Repository 实现 | AIProvider CRUD 适配新字段（roles JSON 编解码） |
| 12 | `database/repo/init.go` | 注入 | `Init()` 新增 `db.AIRoleBudget = newAIRoleBudgetRepo(...)` |
| 13 | `config/defaults.go` | 配置 | initDefaultAIModels 填充各模型 context_window 默认值 |
| 14 | `ai/service.go` | 业务层 | aiServiceImpl 新增 budgetRepo; NewService 参数扩展 |
| 15 | `ai/chat.go` | 业务层 | chatLoop 使用 loadAIConfigForRole + ContextManager |
| 16 | `aiops/ai/enhancer.go` | 业务层 | Prompt 截断感知 context_window（maxPromptCharsForContext） |
| 17 | `master.go` | 启动入口 | llmFactory 适配 role routing; 传入 budgetRepo |
| 18 | `gateway/routes.go` | Gateway | 注册角色 API 路由 |
| 19 | `gateway/handler/admin/ai_provider.go` | Gateway | Provider API 支持 roles/context_window 读写 |
| 20 | `atlhyper_web/src/app/settings/ai/page.tsx` | 前端 | Provider 卡片加角色标签 + 预算展示 + context_window 显示 |
| 21 | `atlhyper_web/src/types/i18n.ts` | 前端 | 新增 aiRole 翻译类型 |
| 22 | `atlhyper_web/src/i18n/locales/zh.ts` | 前端 | 新增翻译 |
| 23 | `atlhyper_web/src/i18n/locales/ja.ts` | 前端 | 新增翻译 |
| | **合计** | | **6 新增 + 17 修改** |

---

## 数据流

### Chat 请求流（含上下文管理）

```
用户发送消息
  → chatLoop 启动
    → loadAIConfigForRole(ctx, "chat")
      → 找到持有 "chat" 角色的 Provider (Gemini)
      → 检查预算（如有配置）
      → 返回 RoleConfig { Provider, Model, ContextWindow }
    → NewContextManager(contextWindow)
    → 每轮 LLM 调用前:
      → ctxMgr.FitMessages(systemPrompt, messages)
        → contextWindow > 0 → 从最新消息往回填充，超出预算则裁剪
        → contextWindow = 0 → 原样返回（Gemini/Claude 等大窗口）
      → 裁剪时通知前端 "[历史消息已裁剪]"
    → chatLoop 完成
    → budgetRepo.IncrementUsage("chat", totalTokens)
```

### AIOps 后台分析流

```
AIOps Engine 触发事件摘要
  → Enhancer.Summarize()
    → llmFactory(ctx)
      → 找到持有 "background" 角色的 Provider (Ollama)
    → buildPromptWithTruncation()
      → 已有截断逻辑，但现在感知 context_window
      → Ollama 8K → MaxPromptChars = 4000 (而非默认 16000)
    → LLM 单轮调用
    → budgetRepo.IncrementUsage("background", tokens)
```

### 上下文管理决策树

```
发送消息前:
  ├── contextWindow = 0 (Gemini/Claude/OpenAI)
  │   → 不裁剪，原样发送
  │   → Tool 结果截断: 32K 字符
  │
  └── contextWindow > 0 (Ollama)
      → estimateTokens(system + messages)
      ├── 未超限 → 原样发送
      └── 超限 → FitMessages 从最新往回填充
          → 被丢弃的消息不可恢复（OK，DB 中有完整记录）
          → 前端提示用户
          → Tool 结果截断: 2K 字符 (contextWindow <= 8K)
```

---

## 实施阶段

### Phase 1: 数据模型 + Repository (后端基础)
- ai_providers 新增 roles + context_window_override 列
- ai_provider_models 新增 context_window 列 + 默认值填充
- ai_role_budget 表 + 迁移
- AIProvider struct 扩展 + AIProviderModel 扩展 + AIRoleBudget struct
- AIRoleBudgetRepository 接口/实现
- AIProviderRepository 扩展 (UpdateRoles, FindByRole)
- effectiveContextWindow 解析函数

### Phase 2: 上下文管理器 (核心)
- `ai/context.go`: ContextManager + FitMessages + estimateTokens
- `ai/chat.go`: chatLoop 集成 ContextManager
- Tool 结果截断按 context_window 调整
- 单元测试: 验证裁剪策略

### Phase 3: 角色路由逻辑 (核心)
- `ai/role.go`: loadAIConfigForRole + checkBudget
- `ai/service.go`: 注入 budgetRepo
- `master.go`: llmFactory 适配
- AIOps Enhancer 适配 context_window
- 集成测试

### Phase 4: API + Gateway (后端 API)
- Provider roles API (分配/移除 + 互斥校验)
- 角色预算 API (CRUD + 重置)
- 路由注册

### Phase 5: 前端 (Web UI)
- Provider 卡片改造（角色标签 + 预算展示 + context_window 显示）
- 角色分配交互（下拉 + 互斥提示）
- 预算编辑弹窗
- i18n

---

## 验证方法

1. **向后兼容**: 不分配任何角色，AI Chat 和 AIOps Enhancer 使用全局 Provider（行为不变）
2. **角色路由**: Ollama=[background], Gemini=[chat]，验证 AIOps 用 Ollama、Chat 用 Gemini
3. **互斥约束**: 尝试将已被 Gemini 持有的 "chat" 分配给 Ollama → 应返回错误
4. **上下文裁剪**: Ollama 做 Chat，3 轮 Tool Calling 后验证消息被裁剪 + 前端提示
5. **预算控制**: 设置 analysis 的 daily_call_limit=3，验证第 4 次调用被拒或降级
6. **跨日重置**: 修改 daily_reset_at 为昨天，验证下次调用自动重置计数
7. **热更新**: Web UI 移除/分配角色后，下次调用使用新 Provider
8. **构建**: `go build ./cmd/master/` 通过
