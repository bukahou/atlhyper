# AtlHyper 通知系统设计方案

> 版本: v1.0
> 日期: 2025-01-25
> 状态: 待实施

---

## 一、设计目标

### 1.1 核心功能

- **告警聚合**: 将多条告警合并为单条消息发送，避免消息轰炸
- **多渠道支持**: Slack、Email（已有基础实现）
- **告警风暴防护**: 去重 + 聚合 + 限流三层防护
- **可配置**: 通过 Web UI 配置渠道参数

### 1.2 设计原则

- **只聚合，不单发**: 所有告警进入缓冲区，批量发送
- **Critical 优先**: 有 Critical 告警时立即触发发送
- **简单优先**: AlertManager 无接口，直接使用具体类型
- **最小依赖**: 仅依赖 NotifyChannelRepository 读取配置

### 1.3 不做

- ~~单条告警消息~~ — 级联告警场景下无意义
- ~~AlertManager 接口~~ — 单实现，无需抽象
- ~~Webhook/DingTalk~~ — 首期不实现，预留扩展

---

## 二、整体架构

### 2.1 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                     触发源 (Event Sources)                   │
├─────────────────────────────────────────────────────────────┤
│ 1. Agent 心跳超时 (service/agent.go)                         │
│ 2. K8s Warning Events (预留)                                 │
│ 3. 手动测试发送 (handler/notify.go)                          │
└────────────────────┬────────────────────────────────────────┘
                     │ alertManager.Send()
                     ↓
┌─────────────────────────────────────────────────────────────┐
│                   notifier.AlertManager                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Layer 1: 去重过滤 (dedupCache)                          │ │
│  │ - Key = ClusterID + Resource + Reason + Severity        │ │
│  │ - TTL = 10 分钟                                         │ │
│  └────────────────────┬───────────────────────────────────┘ │
│                       ↓ 通过                                 │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Layer 2: 聚合缓冲 (aggregateBuffer)                     │ │
│  │ - 窗口时间: 30 秒                                       │ │
│  │ - 最大容量: 100 条                                      │ │
│  │ - Critical 立即 flush                                   │ │
│  └────────────────────┬───────────────────────────────────┘ │
│                       ↓ flush                                │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Layer 3: 限流 (rateLimiter)                             │ │
│  │ - 5 条/分钟                                             │ │
│  │ - 超限时延迟到下一窗口                                   │ │
│  └────────────────────┬───────────────────────────────────┘ │
│                       ↓                                      │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Dispatcher: 分发到各渠道                                 │ │
│  │ - 读取 NotifyChannelRepository 获取启用的渠道            │ │
│  │ - 构建渠道专属消息格式                                   │ │
│  │ - 调用 Notifier.Send()                                  │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
└────────────────────┬────────────────────────────────────────┘
                     │
          ┌──────────┴──────────┐
          ↓                     ↓
   ┌─────────────┐       ┌─────────────┐
   │SlackNotifier│       │EmailNotifier│
   │ (BlockKit)  │       │   (HTML)    │
   └─────────────┘       └─────────────┘
```

### 2.2 依赖关系

```
notifier.AlertManager
    │
    ├── 依赖 (注入)
    │   └── database.NotifyChannelRepository  // 读取渠道配置
    │
    ├── 被调用方
    │   ├── service.AgentService              // 心跳检测
    │   ├── gateway.NotifyHandler             // 测试发送
    │   └── (future) EventHandler             // K8s Event
    │
    └── 内部组件 (不暴露)
        ├── dedupCache
        ├── aggregateBuffer
        └── Notifier (Slack/Email)
```

---

## 三、数据结构

### 3.1 Alert 告警结构

```go
// notifier/alert.go

type Alert struct {
    ID        string            // 唯一标识 (UUID)
    Title     string            // 告警标题
    Message   string            // 详细消息
    Severity  string            // critical / warning / info
    Source    string            // agent_heartbeat / k8s_event / manual
    ClusterID string            // 集群 ID
    Resource  string            // 资源标识 (Pod/default/nginx-xxx)
    Reason    string            // 原因代码 (CrashLoopBackOff)
    Fields    map[string]string // 扩展字段
    Timestamp time.Time         // 发生时间
}

// 去重 Key 生成
func (a *Alert) DedupKey() string {
    return fmt.Sprintf("%s|%s|%s|%s",
        a.ClusterID, a.Resource, a.Reason, a.Severity)
}
```

### 3.2 聚合结果结构

```go
// notifier/aggregate.go

type AlertSummary struct {
    Total       int                    // 告警总数
    BySeverity  map[string]int         // 按级别统计
    Clusters    []string               // 涉及集群
    Namespaces  []string               // 涉及命名空间
    Alerts      []*Alert               // 告警列表 (最多 15 条)
    HasMore     bool                   // 是否有更多
    MoreCount   int                    // 省略条数
    GeneratedAt time.Time              // 生成时间
}
```

---

## 四、核心组件

### 4.1 AlertManager

```go
// notifier/manager.go

type AlertManager struct {
    channelRepo database.NotifyChannelRepository

    // 内部组件
    dedup    *dedupCache
    buffer   *aggregateBuffer
    limiter  *rateLimiter

    // 状态
    running  bool
    stopCh   chan struct{}
    mu       sync.Mutex
}

// 构造函数
func NewAlertManager(repo database.NotifyChannelRepository) *AlertManager

// 公开方法
func (m *AlertManager) Start()                                        // 启动
func (m *AlertManager) Stop()                                         // 停止
func (m *AlertManager) Send(ctx context.Context, alert *Alert) error  // 发送告警
func (m *AlertManager) Test(ctx context.Context, chType string) error // 测试发送
```

### 4.2 去重缓存

```go
// notifier/dedup.go

type dedupCache struct {
    cache map[string]time.Time
    ttl   time.Duration  // 10 分钟
    mu    sync.Mutex
}

func newDedupCache(ttl time.Duration) *dedupCache
func (d *dedupCache) IsDuplicate(key string) bool  // 检查并记录
func (d *dedupCache) cleanup()                     // 清理过期
```

### 4.3 聚合缓冲

```go
// notifier/buffer.go

type aggregateBuffer struct {
    alerts   []*Alert
    window   time.Duration  // 30 秒
    maxSize  int            // 100 条
    timer    *time.Timer
    flushFn  func([]*Alert) // flush 回调
    mu       sync.Mutex
}

func newAggregateBuffer(window time.Duration, max int, flush func([]*Alert)) *aggregateBuffer
func (b *aggregateBuffer) Add(alert *Alert)   // 添加告警
func (b *aggregateBuffer) FlushNow()          // 立即 flush
func (b *aggregateBuffer) Stop()              // 停止
```

### 4.4 限流器

```go
// notifier/limiter.go

type rateLimiter struct {
    maxPerMinute int           // 5
    sent         []time.Time   // 发送记录
    mu           sync.Mutex
}

func newRateLimiter(maxPerMinute int) *rateLimiter
func (r *rateLimiter) Allow() bool  // 是否允许发送
```

---

## 五、消息模板

### 5.1 Slack BlockKit 格式

```
⚠️ 集群告警汇总（共 12 条）

📊 级别分布
🔴 Critical: 2  🟠 Warning: 8  🔵 Info: 2

🏷️ 集群: prod-cluster-01, prod-cluster-02
📁 命名空间: default, monitoring, app

━━━━━━━━━━━━━━━━━━━━━━

🔴 Pod default/api-server-xyz
   CrashLoopBackOff | 容器反复重启，已重启 5 次

🟠 Endpoint default/api-service
   NotReady | 无可用后端 Pod

🟠 Deployment default/api-server
   Unavailable | 期望 3 副本，当前 2 副本

🟠 Node node-03
   MemoryPressure | 内存使用率 92%

... 还有 8 条告警

━━━━━━━━━━━━━━━━━━━━━━
⏰ 2025-01-25 14:30:00 CST | AtlHyper
```

### 5.2 Slack BlockKit 结构

```go
// notifier/slack.go

func buildSlackBlocks(summary *AlertSummary) map[string]interface{} {
    return map[string]interface{}{
        "blocks": []interface{}{
            // Header
            headerBlock(summary.Total),
            dividerBlock(),
            // 统计
            statsBlock(summary.BySeverity, summary.Clusters, summary.Namespaces),
            dividerBlock(),
            // 告警列表
            alertListBlocks(summary.Alerts),
            // 省略提示
            moreBlock(summary.HasMore, summary.MoreCount),
            dividerBlock(),
            // Footer
            footerBlock(summary.GeneratedAt),
        },
    }
}
```

### 5.3 Email HTML 格式

```go
// notifier/email.go

func buildEmailHTML(summary *AlertSummary) string {
    // HTML 模板，包含:
    // - 标题 + 统计
    // - 告警表格
    // - Footer
}
```

---

## 六、触发点集成

### 6.1 Agent 心跳超时

```go
// service/agent.go

type AgentService struct {
    repo         database.AgentRepository
    alertManager *notifier.AlertManager
}

func (s *AgentService) StartHeartbeatChecker(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    for {
        select {
        case <-ticker.C:
            s.checkHeartbeat(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (s *AgentService) checkHeartbeat(ctx context.Context) {
    agents, _ := s.repo.ListAll(ctx)
    for _, agent := range agents {
        if agent.Status == "online" && time.Since(agent.LastHeartbeat) > 30*time.Second {
            // 更新状态
            s.repo.UpdateStatus(ctx, agent.ID, "offline")

            // 发送告警
            if s.alertManager != nil {
                s.alertManager.Send(ctx, &notifier.Alert{
                    Title:     "Agent 离线",
                    Message:   fmt.Sprintf("Agent %s 已离线超过 30 秒", agent.ClusterID),
                    Severity:  "critical",
                    Source:    "agent_heartbeat",
                    ClusterID: agent.ClusterID,
                    Resource:  "agent/" + agent.ClusterID,
                    Reason:    "HeartbeatTimeout",
                })
            }
        }
    }
}
```

### 6.2 测试发送 API

```go
// gateway/handler/notify.go

func (h *NotifyHandler) TestChannel(w http.ResponseWriter, r *http.Request) {
    channelType := chi.URLParam(r, "type") // slack / email

    err := h.alertManager.Test(r.Context(), channelType)
    if err != nil {
        writeError(w, 500, err.Error())
        return
    }

    writeJSON(w, map[string]string{"status": "ok", "message": "测试消息已发送"})
}
```

---

## 七、配置

### 7.1 数据库配置 (已有)

```sql
-- notify_channels 表
CREATE TABLE notify_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT UNIQUE NOT NULL,      -- slack / email
    name TEXT NOT NULL,
    enabled INTEGER DEFAULT 0,
    config TEXT,                     -- JSON
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

### 7.2 Slack 配置结构

```go
type SlackConfig struct {
    WebhookURL string `json:"webhook_url"`
}
```

### 7.3 Email 配置结构

```go
type EmailConfig struct {
    SMTPHost     string   `json:"smtp_host"`
    SMTPPort     int      `json:"smtp_port"`
    SMTPUser     string   `json:"smtp_user"`
    SMTPPassword string   `json:"smtp_password"`
    SMTPTLS      bool     `json:"smtp_tls"`
    FromAddress  string   `json:"from_address"`
    ToAddresses  []string `json:"to_addresses"`
}
```

### 7.4 AlertManager 配置 (硬编码，可后续改为配置)

```go
const (
    DedupTTL        = 10 * time.Minute  // 去重 TTL
    AggregateWindow = 30 * time.Second  // 聚合窗口
    AggregateMax    = 100               // 最大缓冲
    RateLimitPerMin = 5                 // 每分钟限制
    MaxAlertsInMsg  = 15                // 消息内最多展示
)
```

---

## 八、文件结构

```
atlhyper_master_v2/notifier/
├── interfaces.go       # Notifier 接口定义 (已有)
├── alert.go            # Alert 结构体 (新增)
├── manager.go          # AlertManager 主逻辑 (新增)
├── dedup.go            # 去重缓存 (新增)
├── buffer.go           # 聚合缓冲 (新增)
├── limiter.go          # 限流器 (新增)
├── dispatch.go         # 分发逻辑 (新增)
├── slack.go            # Slack 发送器 (已有，增强 BlockKit)
└── email.go            # Email 发送器 (已有，增强 HTML)
```

---

## 九、API 接口

### 9.1 已有接口

| Method | Path | Auth | 说明 |
|--------|------|------|------|
| GET | `/api/v2/notify/channels` | 无 | 获取所有渠道配置 |
| GET | `/api/v2/notify/channels/{type}` | Admin | 获取单个渠道配置 |
| PUT | `/api/v2/notify/channels/{type}` | Admin | 更新渠道配置 |

### 9.2 需增强

| Method | Path | Auth | 说明 |
|--------|------|------|------|
| POST | `/api/v2/notify/channels/{type}/test` | Admin | **真实发送测试消息** |

---

## 十、错误处理

### 10.1 发送失败

- 记录日志，不阻塞流程
- 不重试（避免消息重复）

### 10.2 配置缺失

- 渠道未配置/未启用时跳过
- 日志记录: `[Notifier] Slack 未配置，跳过发送`

### 10.3 限流触发

- 告警保留在缓冲区
- 下一窗口继续尝试发送
- 日志记录: `[Notifier] 限流触发，延迟发送 N 条告警`

---

## 十一、日志规范

```
[AlertManager] 启动告警管理器
[AlertManager] 收到告警: Agent 离线 (cluster-01)
[AlertManager] 告警已去重，跳过: Agent 离线 (cluster-01)
[AlertManager] 缓冲区 flush: 12 条告警
[AlertManager] 限流触发，延迟发送
[Slack] 发送成功: 12 条告警
[Slack] 发送失败: connection timeout
[Email] 发送成功: 12 条告警
[AlertManager] 停止告警管理器
```

---

## 十二、测试计划

### 12.1 单元测试

- [ ] dedupCache: 去重逻辑、TTL 过期
- [ ] aggregateBuffer: 添加、定时 flush、立即 flush
- [ ] rateLimiter: 限流逻辑
- [ ] AlertManager: Send/Test 流程

### 12.2 集成测试

- [ ] 配置 Slack → 发送测试消息
- [ ] Agent 离线 → 收到 Slack 告警
- [ ] 批量告警 → 聚合为单条消息
- [ ] 重复告警 → 被去重过滤
