# AI 配置系统实现计划

> 继承文档：供下一个对话窗口继续实现

## 项目背景

AtlHyper 是一个 Kubernetes 集群管理平台，包含：
- `atlhyper_master_v2` - Go 后端（端口 8080 Gateway, 8081 AgentSDK）
- `atlhyper_agent_v2` - Go Agent（部署在 K8s 集群中）
- `atlhyper_web` - Next.js 前端

### 已完成的工作

1. **AI Chat 功能** - SSE 流式对话，Tool 调用（query_cluster）
2. **告警分析功能** - `/cluster/alert` 页面多选告警，跳转 AI 分析
3. **通知渠道配置** - Slack/Email 在 Web 端配置，联动启用判断

### 通知渠道的联动判断（已实现，供参考）

```go
// gateway/handler/notify.go
func validateChannelConfig(channelType, config string) []string {
    var errors []string
    switch channelType {
    case "slack":
        if cfg.WebhookURL == "" {
            errors = append(errors, "webhook_url 未配置")
        }
    case "email":
        if cfg.SMTPHost == "" { errors = append(errors, "smtp_host 未配置") }
        if cfg.SMTPPort == 0 { errors = append(errors, "smtp_port 未配置") }
        if cfg.SMTPUser == "" { errors = append(errors, "smtp_user 未配置") }
        if cfg.SMTPPassword == "" { errors = append(errors, "smtp_password 未配置") }
        if cfg.FromAddress == "" { errors = append(errors, "from_address 未配置") }
    }
    return errors
}

// API 响应包含
type ChannelResponse struct {
    Enabled          bool     `json:"enabled"`           // 用户开关
    EffectiveEnabled bool     `json:"effective_enabled"` // 实际可用
    ValidationErrors []string `json:"validation_errors"` // 缺少什么
    Config           json.RawMessage `json:"config"`
}
```

---

## 用户需求

让 Master 的 AI 配置：
1. **启动时**：从环境变量读取，同步到数据库
2. **运行时**：可在 Web 界面修改或添加
3. **联动判断**：类似通知渠道，需要 provider + api_key + model 都配置才算有效启用
4. **多 Provider 支持**：Gemini、OpenAI、Anthropic

---

## 设计方案

### 1. 数据存储（复用 Settings 表）

| Key | Type | 说明 |
|-----|------|------|
| `ai.enabled` | bool | 用户开关 |
| `ai.provider` | string | `gemini` / `openai` / `anthropic` |
| `ai.api_key` | string | 加密存储 |
| `ai.model` | string | 模型名称 |
| `ai.tool_timeout` | int | Tool 调用超时（秒） |

### 2. 启用条件

```
effective_enabled = enabled
                    AND provider != ""
                    AND api_key != ""
                    AND model != ""
```

### 3. 配置同步策略

**策略 B：数据库优先**
- 首次部署：环境变量 → 数据库
- 后续启动：以数据库为准，环境变量忽略
- Web 修改后重启不丢失

```go
func SyncAIConfig(db *database.DB) {
    ctx := context.Background()

    // 检查是否已有配置
    existing, _ := db.Settings.Get(ctx, "ai.api_key")
    if existing != "" {
        log.Println("[AI] 使用数据库中的 AI 配置")
        return
    }

    // 首次同步
    cfg := config.GlobalConfig.AI
    if cfg.APIKey != "" {
        db.Settings.Set(ctx, "ai.enabled", "true")
        db.Settings.Set(ctx, "ai.provider", cfg.Provider)
        db.Settings.Set(ctx, "ai.api_key", cfg.APIKey)
        db.Settings.Set(ctx, "ai.model", cfg.Model)
        db.Settings.Set(ctx, "ai.tool_timeout", strconv.Itoa(cfg.ToolTimeout))
        log.Println("[AI] 从环境变量同步 AI 配置到数据库")
    }
}
```

### 4. API 设计

```
GET  /api/v2/settings/ai           # 获取配置（API Key 脱敏）
PUT  /api/v2/settings/ai           # 更新配置
POST /api/v2/settings/ai/test      # 测试连接
```

**GET 响应**
```json
{
  "enabled": true,
  "effective_enabled": false,
  "validation_errors": ["api_key 未配置"],
  "provider": "gemini",
  "api_key_masked": "AIza****xxxx",
  "api_key_set": true,
  "model": "gemini-2.0-flash",
  "tool_timeout": 30,
  "available_providers": [
    {
      "id": "gemini",
      "name": "Google Gemini",
      "models": ["gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash"]
    },
    {
      "id": "openai",
      "name": "OpenAI",
      "models": ["gpt-4o", "gpt-4o-mini", "gpt-4-turbo"]
    },
    {
      "id": "anthropic",
      "name": "Anthropic Claude",
      "models": ["claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022"]
    }
  ]
}
```

**PUT 请求**
```json
{
  "enabled": true,
  "provider": "openai",
  "api_key": "sk-xxxxx",
  "model": "gpt-4o",
  "tool_timeout": 30
}
```

### 5. 热更新策略

**方案 A：需要重启**（推荐）
- 保存后提示需要重启服务
- 简单稳定，适合生产环境

---

## 实现步骤

### 后端（8 步）

| 步骤 | 内容 | 文件 |
|------|------|------|
| 1 | Settings Repository 增加批量读写方法 | `database/repo/settings.go` |
| 2 | 添加 AI 配置同步逻辑 | `database/sync.go` |
| 3 | 创建 Settings Handler（AI 配置 API） | `gateway/handler/settings.go` |
| 4 | 注册路由 | `gateway/routes.go` |
| 5 | AI Service 从 Settings 加载配置 | `ai/service.go` |
| 6 | Master 启动时调用同步 | `master.go` |
| 7 | （可选）添加 OpenAI Provider | `ai/provider/openai.go` |
| 8 | （可选）添加 Anthropic Provider | `ai/provider/anthropic.go` |

### 前端（4 步）

| 步骤 | 内容 | 文件 |
|------|------|------|
| 1 | Settings API 封装 | `api/settings.ts` |
| 2 | AI 配置页面 | `app/system/settings/ai/page.tsx` |
| 3 | 更新导航 | `components/navigation/Sidebar.tsx` |
| 4 | 添加系统设置入口 | 可选 |

---

## 关键文件路径

### 后端
```
/home/wuxiafeng/AtlHyper/GitHub/atlhyper/atlhyper_master_v2/
├── config/
│   ├── types.go          # AI 配置结构体
│   ├── loader.go         # 环境变量加载
│   └── defaults.go       # 默认值
├── database/
│   ├── interfaces.go     # Repository 接口
│   ├── sync.go           # 配置同步（需修改）
│   ├── repo/
│   │   └── settings.go   # Settings Repository
│   └── sqlite/
│       └── settings.go   # SQLite 实现
├── gateway/
│   ├── routes.go         # 路由注册
│   └── handler/
│       ├── notify.go     # 通知配置（参考）
│       └── settings.go   # 新建：Settings Handler
├── ai/
│   ├── service.go        # AI 服务
│   ├── prompts.go        # 提示词
│   └── gemini.go         # Gemini 客户端
└── master.go             # 启动入口
```

### 前端
```
/home/wuxiafeng/AtlHyper/GitHub/atlhyper/atlhyper_web/
├── src/
│   ├── api/
│   │   ├── notify.ts     # 通知 API（参考）
│   │   └── settings.ts   # 新建：Settings API
│   ├── app/
│   │   └── system/
│   │       ├── notifications/  # 通知配置（参考）
│   │       └── settings/
│   │           └── ai/
│   │               └── page.tsx  # 新建：AI 配置页
│   └── components/
│       └── navigation/
│           └── Sidebar.tsx  # 导航栏
```

---

## 现有 AI 配置（环境变量）

```go
// config/types.go
type AIConfig struct {
    Enabled     bool
    Provider    string        // gemini
    GeminiKey   string        // MASTER_AI_GEMINI_API_KEY
    GeminiModel string        // gemini-2.0-flash
    ToolTimeout time.Duration // 30s
}

// config/defaults.go
defaultStrings["MASTER_AI_PROVIDER"] = "gemini"
defaultStrings["MASTER_AI_GEMINI_MODEL"] = "gemini-2.0-flash"
defaultBools["MASTER_AI_ENABLED"] = true
```

---

## 前端参考：通知配置页面

`/system/notifications/page.tsx` 和 `components/EmailCard.tsx` 已实现：
- 表单编辑所有字段
- 密码字段显示/隐藏
- validation_errors 提示
- effective_enabled 状态显示
- 测试连接功能

AI 配置页面可参考此实现。

---

## 启动命令

```bash
# 后端
cd /home/wuxiafeng/AtlHyper/GitHub/atlhyper/atlhyper_master_v2
go run .

# 前端
cd /home/wuxiafeng/AtlHyper/GitHub/atlhyper/atlhyper_web
npm run dev
```

---

## 下一步

在新对话窗口中：
1. 读取此文件 `/home/wuxiafeng/AtlHyper/GitHub/atlhyper/docs/AI_CONFIG_PLAN.md`
2. 按步骤实现 AI 配置系统
3. 先实现后端，再实现前端
