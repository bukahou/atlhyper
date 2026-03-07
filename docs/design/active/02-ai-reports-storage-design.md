# AI 分析报告持久化设计

> 状态: active | 创建: 2026-03-07
> 关联文档:
> - [00-ai-role-definition.md](./00-ai-role-definition.md) — 三个角色定义
> - [04-ai-background-analysis-design.md](./04-ai-background-analysis-design.md) — background + analysis 功能实现
> - [01-aiops-data-tiered-design.md](./01-aiops-data-tiered-design.md) — AIOps 分层架构（数据源）

## 1. 背景

### 1.1 问题

三个 AI 角色的产出缺乏统一的持久化方案：

| 角色 | 输出 | 当前持久化 | 问题 |
|------|------|-----------|------|
| **chat** | 对话流 | `ai_conversations` + `ai_messages` | 已完善，无需改动 |
| **background** | 事件摘要 | `aiops_incidents.summary`（单字段） | 只存一个 string，丢失结构化数据（根因/建议/相似事件） |
| **analysis** | 深度报告 | 无 | 完全没有存储 |

**background 的具体问题**：

Enhancer 的 `SummarizeResponse` 包含丰富的结构化输出：

```go
type SummarizeResponse struct {
    Summary           string           // 1-2 句话摘要
    RootCauseAnalysis string           // 根因分析
    Recommendations   []Recommendation // 处置建议（优先级/动作/原因/影响）
    SimilarIncidents  []SimilarMatch   // 相似历史事件
}
```

但最终只有 `Summary` 被写入 `aiops_incidents.summary`，其余字段仅在内存缓存中存活，重启即丢失。

### 1.2 目标

设计统一的 AI 分析报告存储，满足：

1. **background + analysis 产出统一持久化**（chat 已有独立存储，不在本设计范围）
2. **版本历史**：同一事件多次分析（状态变化后重新分析），每次保留独立记录
3. **成本追踪**：每条报告记录使用的 Provider/Model/Token 消耗
4. **巡检报告**：支持不关联事件的独立报告（定时巡检）
5. **不破坏现有表结构**：`aiops_incidents` 和 `ai_conversations` 保持不变

---

## 2. 设计

### 2.1 数据关系

```
AIOps Incident (已有，不变)
  │
  ├── 1:N ──> ai_reports (新增)
  │            ├── background 摘要 (事件创建时自动生成)
  │            ├── background 更新 (状态变化时重新分析)
  │            ├── analysis 报告 (用户手动 / critical 自动升级)
  │            └── ...每次分析都是一条独立记录
  │
  ├── 1:N ──> aiops_incident_timeline (已有，事件时间线)
  │
  └── 1:N ──> aiops_incident_entities (已有，受影响实体)

AI Conversations (已有，不变)
  └── chat 角色独立存储，与 ai_reports 无关
```

**关键决策**：background 和 analysis 共用 `ai_reports` 表，chat 保持独立的 `ai_conversations` + `ai_messages`。原因：

- chat 是多轮对话（N 条消息），需要消息级存储
- background/analysis 是单次产出（一份报告），适合文档级存储
- 两者的查询模式完全不同（对话历史 vs 报告列表）

### 2.2 数据模型

```go
// database/types.go

// AIReport AI 分析报告（background/analysis 的持久化产出）
type AIReport struct {
    ID         int64
    IncidentID string    // 关联事件 ID（可为空：巡检报告无事件）
    ClusterID  string    // 集群 ID
    Role       string    // "background" / "analysis"
    Trigger    string    // 触发方式（见下表）

    // 报告内容
    Summary           string // 1-2 句话摘要
    RootCauseAnalysis string // 根因分析
    Recommendations   string // JSON: []Recommendation
    SimilarIncidents  string // JSON: []SimilarMatch

    // analysis 专属（深度报告扩展章节）
    InvestigationSteps string // JSON: 调查步骤（Tool Call 记录）
    EvidenceChain      string // JSON: 证据链（指标/日志/事件引用）

    // 生成元数据
    ProviderName string // 使用的 Provider 名称
    Model        string // 使用的模型
    InputTokens  int    // 输入 token 数
    OutputTokens int    // 输出 token 数
    DurationMs   int64  // 生成耗时 (ms)

    CreatedAt time.Time
}
```

### 2.3 Trigger 类型

| trigger 值 | 角色 | 触发时机 | 说明 |
|-----------|------|---------|------|
| `incident_created` | background | 事件创建 | Engine 创建 Incident 后自动触发 |
| `state_changed` | background | 事件状态变化 | Warning→Incident 等，重新分析 |
| `manual` | analysis | 用户点击"深度分析" | 前端事件详情页触发 |
| `auto_escalation` | analysis | severity=critical | 系统自动升级触发 |
| `patrol` | background | 定时巡检 | 无 incident_id，独立集群报告 |

### 2.4 表结构

```sql
CREATE TABLE IF NOT EXISTS ai_reports (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    incident_id           TEXT,             -- 关联事件 ID (可为空)
    cluster_id            TEXT NOT NULL,
    role                  TEXT NOT NULL,    -- 'background' / 'analysis'
    trigger               TEXT NOT NULL,    -- 触发方式

    -- 报告内容
    summary               TEXT DEFAULT '',
    root_cause_analysis   TEXT DEFAULT '',
    recommendations       TEXT DEFAULT '[]',  -- JSON
    similar_incidents     TEXT DEFAULT '[]',  -- JSON

    -- analysis 扩展
    investigation_steps   TEXT DEFAULT '',    -- JSON
    evidence_chain        TEXT DEFAULT '',    -- JSON

    -- 生成元数据
    provider_name         TEXT DEFAULT '',
    model                 TEXT DEFAULT '',
    input_tokens          INTEGER DEFAULT 0,
    output_tokens         INTEGER DEFAULT 0,
    duration_ms           INTEGER DEFAULT 0,

    created_at            TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_reports_incident ON ai_reports(incident_id);
CREATE INDEX IF NOT EXISTS idx_ai_reports_cluster_role ON ai_reports(cluster_id, role, created_at);
```

### 2.5 与现有存储的关系

| 存储 | 用途 | 变更 |
|------|------|------|
| `aiops_incidents.summary` | 事件快速摘要（列表页展示） | **保留不变**，background 写报告时同步更新此字段 |
| `ai_reports` | 完整分析报告（详情页展示） | **新增** |
| `ai_conversations` + `ai_messages` | chat 对话记录 | **不变** |
| `aiops_incident_timeline` | 事件时间线 | **不变**，可新增 `ai_report_generated` 事件类型 |

**双写策略**：background 生成报告时：
1. 完整报告写入 `ai_reports`
2. `summary` 字段同步写入 `aiops_incidents.summary`（保持列表页快速展示）

---

## 3. Repository 接口

```go
// database/interfaces.go

type AIReportRepository interface {
    // 写入
    Create(ctx context.Context, report *AIReport) error

    // 查询
    GetByID(ctx context.Context, id int64) (*AIReport, error)
    ListByIncident(ctx context.Context, incidentID string) ([]*AIReport, error)
    ListByCluster(ctx context.Context, clusterID, role string, limit int) ([]*AIReport, error)

    // 统计
    CountByClusterAndRole(ctx context.Context, clusterID, role string, since time.Time) (int, error)

    // 清理
    DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}
```

---

## 4. 数据流

### 4.1 background 自动摘要

```
AIOps Engine: OnSnapshot → StateMachine → 新事件创建
  │
  └──> Enhancer.Summarize(incident)
         │
         ├── 构建 Prompt（实体/时间线/历史）
         ├── LLM 单轮调用 → SummarizeResponse
         │
         ├── 写入 ai_reports（完整报告）
         │     role="background", trigger="incident_created"
         │     summary + root_cause + recommendations + similar
         │     provider_name + model + tokens + duration
         │
         └── 同步更新 aiops_incidents.summary（快速展示）
```

### 4.2 background 状态变化更新

```
AIOps Engine: StateMachine → Warning → Incident
  │
  └──> Enhancer.Summarize(incident)  // 状态变化触发重新分析
         │
         └── 写入 ai_reports（新记录，不覆盖旧的）
               role="background", trigger="state_changed"
```

### 4.3 analysis 深度分析

```
用户点击"深度分析" / severity=critical 自动升级
  │
  └──> AnalysisService.Analyze(incidentID)
         │
         ├── 加载事件数据 + 现有 background 报告
         ├── 多轮 Tool Calling（query_cluster）
         │     记录每一步 investigation_steps
         ├── 汇总证据链 evidence_chain
         ├── LLM 输出最终分析报告
         │
         └── 写入 ai_reports
               role="analysis", trigger="manual" / "auto_escalation"
               summary + root_cause + recommendations + similar
               + investigation_steps + evidence_chain
```

### 4.4 定时巡检

```
Scheduler: 每 N 小时触发
  │
  └──> Enhancer.Patrol(clusterID)
         │
         ├── 汇总当前集群状态（AIOps 风险概况 + 活跃事件）
         ├── LLM 生成巡检报告
         │
         └── 写入 ai_reports
               incident_id=NULL, role="background", trigger="patrol"
```

---

## 5. 前端展示

### 5.1 事件详情页

```
事件详情页 (incident/:id)
├── 事件概况（已有：状态/实体/风险分数）
├── AI 分析记录 (新增)
│   ├── [最新] background: 事件摘要         2026-03-07 10:30
│   │         根因: Node desk-one MemoryPressure 导致...
│   │         建议: 1. 释放节点内存 2. 调整资源限制
│   │         Ollama qwen2.5-14b-32k | 1.2K tokens | 12s
│   │
│   ├── [深度] analysis: 完整分析报告       2026-03-07 11:00
│   │         调查步骤: describe Pod → get_logs → get_events → ...
│   │         证据链: OOM Exit Code 137 + MemoryPressure + ...
│   │         Gemini Pro | 15K tokens | 45s
│   │
│   └── [历史] background: 初始摘要         2026-03-07 09:00
│             (事件创建时的第一次分析)
│
├── 时间线（已有）
└── 受影响实体（已有）
```

### 5.2 巡检报告页（后续）

```
AIOps → 巡检报告
├── 2026-03-07 06:00 — 集群 zgmf-x10a 巡检
│   摘要: 集群整体健康，2 个低风险实体需关注...
├── 2026-03-06 06:00 — 集群 zgmf-x10a 巡检
│   ...
```

---

## 6. 清理策略

| 数据 | 保留时长 | 说明 |
|------|---------|------|
| background 报告 | 30 天 | 同事件保留策略 |
| analysis 报告 | 90 天 | 深度分析价值更高 |
| patrol 巡检报告 | 14 天 | 日常巡检，保留周期短 |

清理由现有的定时任务执行，调用 `DeleteBefore()`。

---

## 7. 文件变更清单

| # | 文件 | 操作 | 说明 |
|---|------|------|------|
| 1 | `database/types.go` | 修改 | 新增 `AIReport` struct |
| 2 | `database/interfaces.go` | 修改 | 新增 `AIReportRepository` 接口 |
| 3 | `database/sqlite/migrations.go` | 修改 | 新增 `ai_reports` 表 + 索引 |
| 4 | `database/sqlite/ai_report.go` | 新增 | Repository SQLite 实现 |
| 5 | `aiops/ai/enhancer.go` | 修改 | Summarize 结果写入 `ai_reports` + 同步 incidents.summary |
| 6 | `database/repo/init.go` | 修改 | 初始化 AIReportRepository |
| 7 | `master.go` | 修改 | 注入 AIReportRepository |
| **合计** | | **1 新增 + 6 修改** | |

> analysis 角色的执行逻辑（AnalysisService）属于 [04-ai-background-analysis-design.md](./04-ai-background-analysis-design.md) 的范畴，本文档只定义存储层。

---

## 8. 实施优先级

| 优先级 | 内容 | 依赖 |
|--------|------|------|
| **P1** | `ai_reports` 表 + Repository + 迁移 | 无 |
| **P1** | Enhancer 改造：结果写入 `ai_reports` | P1 表 |
| **P2** | 事件详情页展示 AI 分析记录 | P1 + 前端 API |
| **P3** | 定时巡检报告写入 | P1 + 巡检调度器 |
| **P3** | analysis 报告写入 | P1 + analysis 实现 |
| **P4** | 清理策略 | P1 |

---

## 9. 验证

```bash
# 编译
go build ./atlhyper_master_v2/...

# 迁移验证
go test ./atlhyper_master_v2/database/sqlite/ -v -run TestMigration

# Repository 测试
go test ./atlhyper_master_v2/database/sqlite/ -v -run TestAIReport

# Enhancer 集成测试
go test ./atlhyper_master_v2/aiops/ai/ -v -run TestSummarize_PersistReport
```

### 验证场景

| 场景 | 预期 |
|------|------|
| 事件创建 → background 自动摘要 | ai_reports 写入 1 条 + incidents.summary 同步更新 |
| 事件状态变化 → 重新分析 | ai_reports 写入新记录（不覆盖旧的），incidents.summary 更新 |
| 同一事件查看分析记录 | ListByIncident 返回按时间排序的所有报告 |
| 巡检报告 | incident_id=NULL，ListByCluster 可查 |
| 30 天后清理 | DeleteBefore 删除过期 background 报告，保留 analysis |
