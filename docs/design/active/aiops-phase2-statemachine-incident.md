# AIOps Phase 2b — 状态机引擎 + 事件存储

## 概要

实现 **状态机引擎 (State Machine)** 管理实体和集群的事件生命周期（Healthy → Warning → Incident → Recovery → Stable），以及 **事件存储 (Incident Store)** 结构化持久化事件数据，支持历史查询、模式匹配和统计分析。

**前置依赖**: Phase 2a（风险评分引擎）— 需要 `R_final` 和 `ClusterRisk`

**中心文档**: [`aiops-engine-design.md`](../future/aiops-engine-design.md) §4.4 (M4) + §4.5 (M5)

**关联设计**: [`aiops-phase2-risk-scorer.md`](./aiops-phase2-risk-scorer.md)

---

## 1. 文件夹结构

```
atlhyper_master_v2/
├── master.go                                (现有)  不动: Phase 1 已完成 AIOps 初始化
│
├── aiops/
│   ├── interfaces.go                        (Phase 1) <- 修改: +事件查询方法
│   ├── engine.go                            (Phase 2a) <- 修改: +状态机评估 + 定时检查
│   ├── types.go                             (Phase 2a) <- 修改: +事件相关类型
│   │
│   ├── statemachine/                                  <- NEW (整个目录)
│   │   ├── machine.go                                 <- NEW  状态定义 + 转换规则
│   │   ├── trigger.go                                 <- NEW  持续时间评估 + 转换触发
│   │   └── suppressor.go                              <- NEW  告警抑制逻辑
│   │
│   └── incident/                                      <- NEW (整个目录)
│       ├── store.go                                   <- NEW  Incident CRUD
│       ├── timeline.go                                <- NEW  时间线事件管理
│       ├── stats.go                                   <- NEW  统计查询
│       └── patterns.go                                <- NEW  历史模式匹配
│
├── database/
│   ├── interfaces.go                        (现有)  <- 修改: +AIOpsIncidentRepository
│   ├── sqlite/
│   │   ├── migrations.go                    (现有)  <- 修改: +3 张表 + 4 个索引
│   │   └── aiops_incident.go                          <- NEW  事件 SQL Dialect
│   └── repo/
│       └── aiops_incident.go                          <- NEW  事件 Repository 实现
│
├── service/
│   ├── interfaces.go                        (现有)  <- 修改: Query 接口 +4 方法
│   └── query/
│       └── aiops.go                         (Phase 1) <- 修改: +事件查询实现
│
└── gateway/
    ├── routes.go                            (现有)  <- 修改: +4 路由
    └── handler/
        └── aiops_incident.go                          <- NEW  事件 API Handler
```

### 变更统计

| 操作 | 文件数 | 文件 |
|------|--------|------|
| **新建** | 9 | `statemachine/` 下 3 个 + `incident/` 下 4 个 + `sqlite/aiops_incident.go` + `repo/aiops_incident.go` |
| **修改** | 7 | `aiops/interfaces.go`, `engine.go`, `types.go`, `database/interfaces.go`, `migrations.go`, `service/interfaces.go`, `query/aiops.go`, `routes.go` |

---

## 2. 调用链路

### 2.1 状态机评估路径（OnSnapshot 触发）

```
aiopsEngine.OnSnapshot(ctx, clusterID)
    │
    ├── 1. correlator.Update(snapshot)        ← Phase 1
    ├── 2. baseline.Update(points)            ← Phase 1
    ├── 3. scorer.Calculate(...)              ← Phase 2a
    │       → entityRisks, clusterRisk
    │
    └── 4. ★ stateMachine.Evaluate(clusterID, entityRisks, clusterRisk)  ← NEW
            │
            ├── 对每个实体评估状态转换
            │   ├── Healthy → Warning?  (R_final > 0.5 持续 > 2min)
            │   ├── Warning → Incident? (R_final > 0.8 持续 > 5min 或 SLO burn > 2x)
            │   ├── Incident → Recovery? (R_final < 0.3 持续 > 10min)
            │   ├── Recovery → Stable?  (由定时任务处理，48h 无复发)
            │   └── Recovery → Warning? (R_final 再次 > 0.5，复发)
            │
            ├── 状态转换时触发动作
            │   ├── Healthy → Warning:  创建 Incident + 记录时间线
            │   ├── Warning → Incident: 更新 Incident severity + 记录时间线
            │   ├── Incident → Recovery: 更新 Incident state + 记录时间线
            │   └── Recovery → Warning: 标记复发 + 记录时间线
            │
            └── 告警抑制
                └── 同一实体在 Incident/Recovery 期间不重复创建新 Incident
```

### 2.2 定时检查路径（后台任务）

```
aiopsEngine 后台 goroutine (每 1 分钟)
    │
    └── stateMachine.CheckRecoveryToStable()
        │
        ├── 遍历所有 Recovery 状态的实体
        ├── 检查是否 48h 内 R_final 未再 > 0.5
        └── 满足条件: Recovery → Stable
            ├── 更新 Incident (state=stable, resolved_at 写入)
            └── 记录时间线 (event_type=state_change)
```

### 2.3 事件查询路径（API）

```
GET /api/v2/aiops/incidents?cluster={id}&state=incident&from=...&to=...
    → handler/aiops_incident.go → service/query/aiops.go
    → incidentStore.List(opts) → database

GET /api/v2/aiops/incidents/{id}
    → handler → service → incidentStore.GetByID(id) → database
    → 附带: incident_entities + incident_timeline

GET /api/v2/aiops/incidents/stats?cluster={id}&period=7d
    → handler → service → incidentStore.GetStats(clusterID, period) → database

GET /api/v2/aiops/incidents/patterns?entity={key}&period=30d
    → handler → service → incidentStore.GetPatterns(entityKey, period) → database
```

---

## 3. 数据模型

### 3.1 状态机类型

```go
// aiops/types.go — 新增

// EntityState 实体当前状态
type EntityState string

const (
    StateHealthy  EntityState = "healthy"
    StateWarning  EntityState = "warning"
    StateIncident EntityState = "incident"
    StateRecovery EntityState = "recovery"
    StateStable   EntityState = "stable"
)

// StateMachineEntry 状态机条目（每个实体一个）
type StateMachineEntry struct {
    EntityKey       string      `json:"entityKey"`
    CurrentState    EntityState `json:"currentState"`
    IncidentID      string      `json:"incidentId"`      // 当前关联的 Incident ID
    ConditionMetSince int64     `json:"conditionMetSince"` // 条件持续开始时间 (Unix)
    LastRFinal      float64     `json:"lastRFinal"`       // 最近一次 R_final
    LastEvaluatedAt int64       `json:"lastEvaluatedAt"`  // 最近评估时间
}

// TransitionCondition 状态转换条件
type TransitionCondition struct {
    FromState    EntityState
    ToState      EntityState
    RiskCheck    func(rFinal float64) bool // 风险分检查
    MinDuration  time.Duration              // 最小持续时间
    Description  string
}
```

### 3.2 事件类型

```go
// aiops/types.go — 新增

// Incident 事件
type Incident struct {
    ID         string      `json:"id"`         // UUID
    ClusterID  string      `json:"clusterId"`
    State      EntityState `json:"state"`      // "warning" | "incident" | "recovery" | "stable"
    Severity   string      `json:"severity"`   // "low" | "medium" | "high" | "critical"
    RootCause  string      `json:"rootCause"`  // 根因实体 key
    PeakRisk   float64     `json:"peakRisk"`   // 峰值 ClusterRisk
    StartedAt  time.Time   `json:"startedAt"`
    ResolvedAt *time.Time  `json:"resolvedAt"` // null = 未解决
    DurationS  int64       `json:"durationS"`  // 持续秒数
    Recurrence int         `json:"recurrence"` // 复发次数
    Summary    string      `json:"summary"`    // JSON 结构化摘要
    CreatedAt  time.Time   `json:"createdAt"`
}

// IncidentEntity 受影响实体
type IncidentEntity struct {
    IncidentID string  `json:"incidentId"`
    EntityKey  string  `json:"entityKey"`
    EntityType string  `json:"entityType"` // "service" | "pod" | "node" | "ingress"
    RLocal     float64 `json:"rLocal"`
    RFinal     float64 `json:"rFinal"`
    Role       string  `json:"role"`       // "root_cause" | "affected" | "symptom"
}

// IncidentTimeline 事件时间线条目
type IncidentTimeline struct {
    ID         int64     `json:"id"`
    IncidentID string    `json:"incidentId"`
    Timestamp  time.Time `json:"timestamp"`
    EventType  string    `json:"eventType"`  // 见下方枚举
    EntityKey  string    `json:"entityKey"`
    Detail     string    `json:"detail"`     // JSON
}

// 时间线事件类型
const (
    TimelineAnomalyDetected   = "anomaly_detected"
    TimelineStateChange       = "state_change"
    TimelineMetricSpike       = "metric_spike"
    TimelineRootCauseIdentified = "root_cause_identified"
    TimelineRecoveryStarted   = "recovery_started"
    TimelineRecurrence        = "recurrence"
)

// IncidentDetail 事件详情（API 响应）
type IncidentDetail struct {
    Incident                              // 嵌入基本信息
    Entities []*IncidentEntity   `json:"entities"` // 受影响实体
    Timeline []*IncidentTimeline `json:"timeline"` // 时间线
}

// IncidentStats 事件统计
type IncidentStats struct {
    TotalIncidents    int            `json:"totalIncidents"`
    ActiveIncidents   int            `json:"activeIncidents"`   // 非 stable
    MTTR              float64        `json:"mttr"`              // 平均恢复时间 (分钟)
    RecurrenceRate    float64        `json:"recurrenceRate"`    // 复发率 %
    BySeverity        map[string]int `json:"bySeverity"`        // 按 severity 分布
    ByState           map[string]int `json:"byState"`           // 按 state 分布
    TopRootCauses     []RootCauseCount `json:"topRootCauses"`   // 最常见根因
}

// RootCauseCount 根因统计
type RootCauseCount struct {
    EntityKey string `json:"entityKey"`
    Count     int    `json:"count"`
}

// IncidentPattern 历史事件模式
type IncidentPattern struct {
    EntityKey     string      `json:"entityKey"`
    PatternCount  int         `json:"patternCount"`  // 相似事件次数
    AvgDuration   float64     `json:"avgDuration"`   // 平均持续时间 (分钟)
    LastOccurrence time.Time  `json:"lastOccurrence"`
    CommonMetrics  []string   `json:"commonMetrics"` // 常见异常指标
    Incidents      []*Incident `json:"incidents"`     // 相关事件列表
}

// IncidentQueryOpts 事件查询选项
type IncidentQueryOpts struct {
    ClusterID string
    State     string // 过滤状态
    Severity  string // 过滤严重度
    From      time.Time
    To        time.Time
    Limit     int
    Offset    int
}
```

---

## 4. 详细设计

### 4.1 状态转换规则 (statemachine/machine.go)

```go
// TransitionCallback 状态转换回调接口
// 解耦 StateMachine 和 incident.Store，由 Engine 实现此接口
type TransitionCallback interface {
    // OnWarningCreated 创建新 Incident (Healthy → Warning)
    OnWarningCreated(ctx context.Context, clusterID, entityKey string, risk *EntityRisk, now time.Time) (incidentID string)
    // OnStateEscalated 升级 Incident (Warning → Incident)
    OnStateEscalated(ctx context.Context, incidentID string, state EntityState, risk *EntityRisk, now time.Time)
    // OnRecoveryStarted 开始恢复 (Incident → Recovery)
    OnRecoveryStarted(ctx context.Context, incidentID string, risk *EntityRisk, now time.Time)
    // OnRecurrence 复发 (Recovery → Warning)
    OnRecurrence(ctx context.Context, incidentID string, risk *EntityRisk, now time.Time)
    // OnStable 已稳定 (Recovery → Stable)
    OnStable(ctx context.Context, incidentID string, entityKey string, now time.Time)
}

// StateMachine 状态机管理器
type StateMachine struct {
    mu         sync.RWMutex
    entries    map[string]*StateMachineEntry // entityKey -> entry
    callback   TransitionCallback
    conditions []TransitionCondition
}

// NewStateMachine 创建状态机
func NewStateMachine(callback TransitionCallback) *StateMachine {
    sm := &StateMachine{
        entries:  make(map[string]*StateMachineEntry),
        callback: callback,
    }
    sm.conditions = []TransitionCondition{
        {
            FromState:   StateHealthy,
            ToState:     StateWarning,
            RiskCheck:   func(r float64) bool { return r > 0.5 },
            MinDuration: 2 * time.Minute,
            Description: "R_final > 0.5 持续 > 2 分钟",
        },
        {
            FromState:   StateWarning,
            ToState:     StateIncident,
            RiskCheck:   func(r float64) bool { return r > 0.8 },
            MinDuration: 5 * time.Minute,
            Description: "R_final > 0.8 持续 > 5 分钟",
        },
        {
            FromState:   StateIncident,
            ToState:     StateRecovery,
            RiskCheck:   func(r float64) bool { return r < 0.3 },
            MinDuration: 10 * time.Minute,
            Description: "R_final < 0.3 持续 > 10 分钟",
        },
        {
            FromState:   StateRecovery,
            ToState:     StateWarning,
            RiskCheck:   func(r float64) bool { return r > 0.5 },
            MinDuration: 0, // 立即触发（复发）
            Description: "R_final 再次 > 0.5 (复发)",
        },
    }
    return sm
}
```

### 4.2 状态评估 (statemachine/trigger.go)

```go
// Evaluate 评估所有实体的状态转换
func (sm *StateMachine) Evaluate(
    ctx context.Context,
    clusterID string,
    entityRisks map[string]*EntityRisk,
    clusterRisk *ClusterRisk,
) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    now := time.Now()

    for entityKey, risk := range entityRisks {
        entry := sm.getOrCreate(entityKey)
        entry.LastRFinal = risk.RFinal
        entry.LastEvaluatedAt = now.Unix()

        sm.evaluateEntity(ctx, clusterID, entry, risk, clusterRisk, now)
    }
}

// evaluateEntity 评估单个实体的状态转换
func (sm *StateMachine) evaluateEntity(
    ctx context.Context,
    clusterID string,
    entry *StateMachineEntry,
    risk *EntityRisk,
    clusterRisk *ClusterRisk,
    now time.Time,
) {
    for _, cond := range sm.conditions {
        if entry.CurrentState != cond.FromState {
            continue
        }

        conditionMet := cond.RiskCheck(risk.RFinal)

        // 特殊处理: Warning → Incident 也可由 SLO burn rate 触发
        if cond.FromState == StateWarning && cond.ToState == StateIncident {
            if clusterRisk != nil && clusterRisk.Risk > 80 {
                conditionMet = true
            }
        }

        if conditionMet {
            // 记录条件开始时间
            if entry.ConditionMetSince == 0 {
                entry.ConditionMetSince = now.Unix()
            }

            // 检查持续时间
            duration := time.Duration(now.Unix()-entry.ConditionMetSince) * time.Second
            if duration >= cond.MinDuration {
                sm.transition(ctx, clusterID, entry, risk, cond, now)
                entry.ConditionMetSince = 0 // 重置
            }
        } else {
            entry.ConditionMetSince = 0 // 条件不满足，重置计时
        }
    }
}

// transition 执行状态转换
// 通过 TransitionCallback 通知 Engine 层处理副作用（创建/更新 Incident）
func (sm *StateMachine) transition(
    ctx context.Context,
    clusterID string,
    entry *StateMachineEntry,
    risk *EntityRisk,
    cond TransitionCondition,
    now time.Time,
) {
    oldState := entry.CurrentState
    entry.CurrentState = cond.ToState

    switch {
    case oldState == StateHealthy && cond.ToState == StateWarning:
        // 通过回调创建新 Incident
        incidentID := sm.callback.OnWarningCreated(ctx, clusterID, entry.EntityKey, risk, now)
        entry.IncidentID = incidentID

    case oldState == StateWarning && cond.ToState == StateIncident:
        // 通过回调升级 Incident
        sm.callback.OnStateEscalated(ctx, entry.IncidentID, StateIncident, risk, now)

    case oldState == StateIncident && cond.ToState == StateRecovery:
        // 通过回调开始恢复
        sm.callback.OnRecoveryStarted(ctx, entry.IncidentID, risk, now)

    case oldState == StateRecovery && cond.ToState == StateWarning:
        // 通过回调标记复发
        sm.callback.OnRecurrence(ctx, entry.IncidentID, risk, now)
    }

    log.Info("AIOps 状态转换",
        "entity", entry.EntityKey,
        "from", oldState,
        "to", cond.ToState,
        "rFinal", risk.RFinal,
    )
}

// severityFromRisk 从 R_final 映射严重度
func severityFromRisk(rFinal float64) string {
    switch {
    case rFinal >= 0.9:
        return "critical"
    case rFinal >= 0.7:
        return "high"
    case rFinal >= 0.5:
        return "medium"
    default:
        return "low"
    }
}
```

### 4.3 Recovery → Stable 定时检查

```go
// statemachine/trigger.go

// CheckRecoveryToStable 检查 Recovery 状态的实体是否可以转为 Stable
// 条件: 48h 内 R_final 未再 > 0.5
func (sm *StateMachine) CheckRecoveryToStable(ctx context.Context) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    now := time.Now()
    stableThreshold := 48 * time.Hour

    for entityKey, entry := range sm.entries {
        if entry.CurrentState != StateRecovery {
            continue
        }

        // 检查是否已过 48 小时
        if entry.ConditionMetSince == 0 {
            entry.ConditionMetSince = now.Unix()
            continue
        }

        duration := time.Duration(now.Unix()-entry.ConditionMetSince) * time.Second
        if duration < stableThreshold {
            continue
        }

        // 48h 内 R_final 始终 < 0.5 → 转为 Stable
        entry.CurrentState = StateStable
        sm.callback.OnStable(ctx, entry.IncidentID, entityKey, now)

        log.Info("AIOps 事件已关闭", "entity", entityKey, "incident", entry.IncidentID)

        // 清理状态机条目
        delete(sm.entries, entityKey)
    }
}
```

### 4.4 告警抑制 (statemachine/suppressor.go)

```go
// ShouldSuppress 判断是否应该抑制告警
// 同一实体在 Incident/Recovery 状态期间不创建新 Incident
func (sm *StateMachine) ShouldSuppress(entityKey string) bool {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    entry, ok := sm.entries[entityKey]
    if !ok {
        return false
    }

    return entry.CurrentState == StateIncident || entry.CurrentState == StateRecovery
}

// GetActiveIncidentID 获取实体当前关联的 Incident ID
func (sm *StateMachine) GetActiveIncidentID(entityKey string) string {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    entry, ok := sm.entries[entityKey]
    if !ok {
        return ""
    }
    return entry.IncidentID
}
```

### 4.5 事件存储 (incident/store.go)

```go
// Store 事件存储
type Store struct {
    repo database.AIOpsIncidentRepository
}

// NewStore 创建事件存储
func NewStore(repo database.AIOpsIncidentRepository) *Store {
    return &Store{repo: repo}
}

// Create 创建新事件
func (s *Store) Create(
    ctx context.Context,
    clusterID, entityKey string,
    risk *EntityRisk,
    now time.Time,
) *Incident {
    inc := &Incident{
        ID:        uuid.New().String(),
        ClusterID: clusterID,
        State:     StateWarning,
        Severity:  severityFromRisk(risk.RFinal),
        RootCause: entityKey,
        PeakRisk:  risk.RFinal * 100,
        StartedAt: now,
        CreatedAt: now,
    }

    s.repo.CreateIncident(ctx, toDB(inc))

    // 添加初始受影响实体
    s.repo.AddEntity(ctx, &database.AIOpsIncidentEntity{
        IncidentID: inc.ID,
        EntityKey:  entityKey,
        EntityType: risk.EntityType,
        RLocal:     risk.RLocal,
        RFinal:     risk.RFinal,
        Role:       "root_cause",
    })

    return inc
}

// UpdateState 更新事件状态
func (s *Store) UpdateState(ctx context.Context, incidentID string, state EntityState, severity string) {
    s.repo.UpdateState(ctx, incidentID, string(state), severity)
}

// Resolve 关闭事件
func (s *Store) Resolve(ctx context.Context, incidentID string, resolvedAt time.Time) {
    s.repo.Resolve(ctx, incidentID, resolvedAt)
}

// UpdateRootCause 更新根因和受影响实体
func (s *Store) UpdateRootCause(ctx context.Context, incidentID string, risk *EntityRisk) {
    s.repo.UpdateRootCause(ctx, incidentID, risk.EntityKey)
    // 更新峰值 ClusterRisk（如果当前更高）
    s.repo.UpdatePeakRisk(ctx, incidentID, risk.RFinal*100)
}

// IncrementRecurrence 增加复发计数
func (s *Store) IncrementRecurrence(ctx context.Context, incidentID string) {
    s.repo.IncrementRecurrence(ctx, incidentID)
}
```

### 4.6 时间线管理 (incident/timeline.go)

```go
// AddTimeline 添加时间线条目
func (s *Store) AddTimeline(
    ctx context.Context,
    incidentID string,
    timestamp time.Time,
    eventType, entityKey, detail string,
) {
    s.repo.AddTimeline(ctx, &database.AIOpsIncidentTimeline{
        IncidentID: incidentID,
        Timestamp:  timestamp,
        EventType:  eventType,
        EntityKey:  entityKey,
        Detail:     detail,
    })
}
```

### 4.7 统计查询 (incident/stats.go)

```go
// GetStats 获取事件统计
// 使用 Repository 的单一聚合查询 GetIncidentStats，避免多次 DB 调用
func (s *Store) GetStats(ctx context.Context, clusterID string, period time.Duration) (*IncidentStats, error) {
    since := time.Now().Add(-period)

    // 单次 Repository 调用获取所有统计原始数据
    raw, err := s.repo.GetIncidentStats(ctx, clusterID, since)
    if err != nil {
        return nil, err
    }

    // 根因排行（独立查询，因为需要 LIMIT + ORDER BY）
    topRootCauses, err := s.repo.TopRootCauses(ctx, clusterID, since, 5)
    if err != nil {
        return nil, err
    }

    // 业务层计算复发率
    recurrenceRate := 0.0
    if raw.TotalIncidents > 0 {
        recurrenceRate = float64(raw.RecurringCount) / float64(raw.TotalIncidents) * 100
    }

    return &IncidentStats{
        TotalIncidents:  raw.TotalIncidents,
        ActiveIncidents: raw.ActiveIncidents,
        MTTR:            raw.MTTR,
        RecurrenceRate:  recurrenceRate,
        BySeverity:      raw.BySeverity,
        ByState:         raw.ByState,
        TopRootCauses:   topRootCauses,
    }, nil
}
```

### 4.8 历史模式匹配 (incident/patterns.go)

```go
// GetPatterns 获取指定实体的历史事件模式
func (s *Store) GetPatterns(ctx context.Context, entityKey string, period time.Duration) (*IncidentPattern, error) {
    since := time.Now().Add(-period)

    // 查找该实体作为受影响实体或根因的所有 Incident
    incidents, err := s.repo.ListByEntity(ctx, entityKey, since)
    if err != nil {
        return nil, err
    }

    if len(incidents) == 0 {
        return &IncidentPattern{EntityKey: entityKey}, nil
    }

    // 统计
    var totalDuration float64
    var lastOccurrence time.Time
    metricFreq := map[string]int{}

    for _, inc := range incidents {
        if inc.DurationS > 0 {
            totalDuration += float64(inc.DurationS) / 60.0
        }
        if inc.StartedAt.After(lastOccurrence) {
            lastOccurrence = inc.StartedAt
        }
        // 从 timeline 提取常见异常指标
        timelines, _ := s.repo.GetTimeline(ctx, inc.ID)
        for _, tl := range timelines {
            if tl.EventType == TimelineAnomalyDetected || tl.EventType == TimelineMetricSpike {
                metricFreq[tl.EntityKey+":"+extractMetricFromDetail(tl.Detail)]++
            }
        }
    }

    // 提取最常见指标
    var commonMetrics []string
    for metric := range metricFreq {
        commonMetrics = append(commonMetrics, metric)
    }
    sort.Slice(commonMetrics, func(i, j int) bool {
        return metricFreq[commonMetrics[i]] > metricFreq[commonMetrics[j]]
    })
    if len(commonMetrics) > 5 {
        commonMetrics = commonMetrics[:5]
    }

    return &IncidentPattern{
        EntityKey:      entityKey,
        PatternCount:   len(incidents),
        AvgDuration:    totalDuration / float64(len(incidents)),
        LastOccurrence: lastOccurrence,
        CommonMetrics:  commonMetrics,
        Incidents:      incidents,
    }, nil
}
```

### 4.9 引擎集成

```go
// aiops/engine.go — OnSnapshot 扩展

func (e *Engine) OnSnapshot(ctx context.Context, clusterID string) {
    // Phase 1 + 2a (不变)
    // ...

    // ★ Phase 2b: 状态机评估
    entityRisks := e.scorer.GetEntityRiskMap(clusterID)
    clusterRisk := e.scorer.GetClusterRisk(clusterID)
    if entityRisks != nil {
        e.stateMachine.Evaluate(ctx, clusterID, entityRisks, clusterRisk)
    }

    // 持久化 (不变)
    // ...
}

// Start 中追加定时检查
func (e *Engine) Start() {
    // Phase 1 恢复 (不变)
    // ...

    // ★ Phase 2b: 启动 Recovery→Stable 定时检查
    go func() {
        ticker := time.NewTicker(1 * time.Minute)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                e.stateMachine.CheckRecoveryToStable(context.Background())
            case <-e.stopCh:
                return
            }
        }
    }()
}
```

---

## 5. 数据库表结构

### 5.1 事件主表

```sql
CREATE TABLE IF NOT EXISTS aiops_incidents (
    id            TEXT PRIMARY KEY,
    cluster_id    TEXT NOT NULL,
    state         TEXT NOT NULL,           -- "warning" | "incident" | "recovery" | "stable"
    severity      TEXT NOT NULL,           -- "low" | "medium" | "high" | "critical"
    root_cause    TEXT,                    -- 根因实体 key
    peak_risk     REAL,                   -- 峰值 ClusterRisk
    started_at    TEXT NOT NULL,
    resolved_at   TEXT,
    duration_s    INTEGER,
    recurrence    INTEGER DEFAULT 0,
    summary       TEXT,                    -- JSON 结构化摘要
    created_at    TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_aiops_incidents_cluster_state
    ON aiops_incidents(cluster_id, state);
CREATE INDEX IF NOT EXISTS idx_aiops_incidents_started_at
    ON aiops_incidents(started_at);
```

### 5.2 受影响实体表

```sql
CREATE TABLE IF NOT EXISTS aiops_incident_entities (
    incident_id TEXT NOT NULL,
    entity_key  TEXT NOT NULL,
    entity_type TEXT NOT NULL,             -- "service" | "pod" | "node" | "ingress"
    r_local     REAL,
    r_final     REAL,
    role        TEXT NOT NULL,             -- "root_cause" | "affected" | "symptom"
    PRIMARY KEY (incident_id, entity_key),
    FOREIGN KEY (incident_id) REFERENCES aiops_incidents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_aiops_incident_entities_entity
    ON aiops_incident_entities(entity_key);
```

### 5.3 事件时间线表

```sql
CREATE TABLE IF NOT EXISTS aiops_incident_timeline (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    incident_id TEXT NOT NULL,
    timestamp   TEXT NOT NULL,
    event_type  TEXT NOT NULL,
    entity_key  TEXT,
    detail      TEXT,                      -- JSON
    FOREIGN KEY (incident_id) REFERENCES aiops_incidents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_aiops_incident_timeline_incident
    ON aiops_incident_timeline(incident_id, timestamp);
```

---

## 6. API 端点

### 6.1 事件列表

```
GET /api/v2/aiops/incidents?cluster={id}&state={state}&from={time}&to={time}&limit=20&offset=0

权限: Public (只读)

响应:
{
    "message": "获取成功",
    "data": [
        {
            "id": "inc-2026-0042",
            "clusterId": "cluster-1",
            "state": "incident",
            "severity": "high",
            "rootCause": "_cluster/node/worker-3",
            "peakRisk": 85.0,
            "startedAt": "2026-01-15T14:02:15Z",
            "resolvedAt": null,
            "durationS": 1380,
            "recurrence": 0,
            "createdAt": "2026-01-15T14:02:15Z"
        }
    ],
    "total": 12
}
```

### 6.2 事件详情

```
GET /api/v2/aiops/incidents/{id}

权限: Public (只读)

响应:
{
    "message": "获取成功",
    "data": {
        "id": "inc-2026-0042",
        "clusterId": "cluster-1",
        "state": "incident",
        "severity": "high",
        "rootCause": "_cluster/node/worker-3",
        "peakRisk": 85.0,
        "startedAt": "2026-01-15T14:02:15Z",
        "entities": [
            {
                "entityKey": "_cluster/node/worker-3",
                "entityType": "node",
                "rLocal": 0.90,
                "rFinal": 0.90,
                "role": "root_cause"
            },
            {
                "entityKey": "default/pod/api-server-abc",
                "entityType": "pod",
                "rLocal": 0.45,
                "rFinal": 0.78,
                "role": "affected"
            }
        ],
        "timeline": [
            {
                "timestamp": "2026-01-15T14:02:15Z",
                "eventType": "anomaly_detected",
                "entityKey": "_cluster/node/worker-3",
                "detail": "{\"metric\":\"memory_usage\",\"value\":94,\"deviation\":3.2}"
            },
            {
                "timestamp": "2026-01-15T14:04:15Z",
                "eventType": "state_change",
                "entityKey": "_cluster/node/worker-3",
                "detail": "{\"from\":\"healthy\",\"to\":\"warning\",\"rFinal\":0.72}"
            }
        ]
    }
}
```

### 6.3 事件统计

```
GET /api/v2/aiops/incidents/stats?cluster={id}&period=7d

权限: Public (只读)

响应:
{
    "message": "获取成功",
    "data": {
        "totalIncidents": 15,
        "activeIncidents": 2,
        "mttr": 45.3,
        "recurrenceRate": 13.3,
        "bySeverity": {"low": 5, "medium": 6, "high": 3, "critical": 1},
        "byState": {"warning": 1, "incident": 1, "recovery": 0, "stable": 13},
        "topRootCauses": [
            {"entityKey": "_cluster/node/worker-3", "count": 4},
            {"entityKey": "default/service/db-proxy", "count": 3}
        ]
    }
}
```

### 6.4 历史模式

```
GET /api/v2/aiops/incidents/patterns?entity={key}&period=30d

权限: Public (只读)

响应:
{
    "message": "获取成功",
    "data": {
        "entityKey": "default/service/payment",
        "patternCount": 3,
        "avgDuration": 23.5,
        "lastOccurrence": "2026-01-15T14:02:15Z",
        "commonMetrics": ["error_rate", "avg_latency"],
        "incidents": [...]
    }
}
```

---

## 7. Service 层接口变更

```go
// service/interfaces.go — Query 接口新增

type Query interface {
    // ... 现有方法 + Phase 1 + Phase 2a ...

    // ==================== AIOps 事件查询 ====================

    GetAIOpsIncidents(ctx context.Context, opts aiops.IncidentQueryOpts) ([]*aiops.Incident, int, error)
    GetAIOpsIncidentDetail(ctx context.Context, incidentID string) (*aiops.IncidentDetail, error)
    GetAIOpsIncidentStats(ctx context.Context, clusterID, period string) (*aiops.IncidentStats, error)
    GetAIOpsIncidentPatterns(ctx context.Context, entityKey, period string) (*aiops.IncidentPattern, error)
}
```

---

## 8. Gateway Handler + 路由注册

### 8.1 Handler

```go
// handler/aiops_incident.go
type AIOpsIncidentHandler struct {
    query service.Query
}

func NewAIOpsIncidentHandler(query service.Query) *AIOpsIncidentHandler {
    return &AIOpsIncidentHandler{query: query}
}

func (h *AIOpsIncidentHandler) List(w http.ResponseWriter, r *http.Request) {
    opts := aiops.IncidentQueryOpts{
        ClusterID: r.URL.Query().Get("cluster"),
        State:     r.URL.Query().Get("state"),
        Limit:     parseIntDefault(r, "limit", 20),
        Offset:    parseIntDefault(r, "offset", 0),
    }
    if from := r.URL.Query().Get("from"); from != "" {
        opts.From, _ = time.Parse(time.RFC3339, from)
    }
    if to := r.URL.Query().Get("to"); to != "" {
        opts.To, _ = time.Parse(time.RFC3339, to)
    }
    incidents, total, err := h.query.GetAIOpsIncidents(r.Context(), opts)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSONWithTotal(w, http.StatusOK, incidents, total)
}

func (h *AIOpsIncidentHandler) Detail(w http.ResponseWriter, r *http.Request) {
    incidentID := extractPathParam(r, "/api/v2/aiops/incidents/")
    if incidentID == "" || incidentID == "stats" || incidentID == "patterns" {
        writeError(w, http.StatusBadRequest, "invalid incident id")
        return
    }
    detail, err := h.query.GetAIOpsIncidentDetail(r.Context(), incidentID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSON(w, http.StatusOK, detail)
}

func (h *AIOpsIncidentHandler) Stats(w http.ResponseWriter, r *http.Request) {
    clusterID := r.URL.Query().Get("cluster")
    period := r.URL.Query().Get("period")
    if period == "" {
        period = "7d"
    }
    stats, err := h.query.GetAIOpsIncidentStats(r.Context(), clusterID, period)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSON(w, http.StatusOK, stats)
}

func (h *AIOpsIncidentHandler) Patterns(w http.ResponseWriter, r *http.Request) {
    entityKey := r.URL.Query().Get("entity")
    period := r.URL.Query().Get("period")
    if period == "" {
        period = "30d"
    }
    patterns, err := h.query.GetAIOpsIncidentPatterns(r.Context(), entityKey, period)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSON(w, http.StatusOK, patterns)
}
```

### 8.2 路由注册

```go
// gateway/routes.go — registerRoutes() 中新增

aiopsIncidentHandler := handler.NewAIOpsIncidentHandler(r.service)

register("/api/v2/aiops/incidents", aiopsIncidentHandler.List)
register("/api/v2/aiops/incidents/stats", aiopsIncidentHandler.Stats)
register("/api/v2/aiops/incidents/patterns", aiopsIncidentHandler.Patterns)
register("/api/v2/aiops/incidents/", aiopsIncidentHandler.Detail)
```

---

## 9. Database 接口变更

```go
// database/interfaces.go — 新增

// DB 结构体新增字段
type DB struct {
    // ... 现有字段 ...
    AIOpsIncident AIOpsIncidentRepository
}

// AIOpsIncidentRepository 事件数据访问接口
type AIOpsIncidentRepository interface {
    // Incident CRUD
    CreateIncident(ctx context.Context, inc *AIOpsIncident) error
    GetByID(ctx context.Context, id string) (*AIOpsIncident, error)
    UpdateState(ctx context.Context, id, state, severity string) error
    Resolve(ctx context.Context, id string, resolvedAt time.Time) error
    UpdateRootCause(ctx context.Context, id, rootCause string) error
    UpdatePeakRisk(ctx context.Context, id string, peakRisk float64) error
    IncrementRecurrence(ctx context.Context, id string) error

    // 查询
    List(ctx context.Context, opts AIOpsIncidentQueryOpts) ([]*AIOpsIncident, error)
    Count(ctx context.Context, opts AIOpsIncidentQueryOpts) (int, error)

    // Entity
    AddEntity(ctx context.Context, entity *AIOpsIncidentEntity) error
    GetEntities(ctx context.Context, incidentID string) ([]*AIOpsIncidentEntity, error)

    // Timeline
    AddTimeline(ctx context.Context, entry *AIOpsIncidentTimeline) error
    GetTimeline(ctx context.Context, incidentID string) ([]*AIOpsIncidentTimeline, error)

    // 统计（单一聚合查询，避免 Repository 接口膨胀）
    GetIncidentStats(ctx context.Context, clusterID string, since time.Time) (*AIOpsIncidentStatsRaw, error)
    TopRootCauses(ctx context.Context, clusterID string, since time.Time, limit int) ([]AIOpsRootCauseCount, error)

    // 模式
    ListByEntity(ctx context.Context, entityKey string, since time.Time) ([]*AIOpsIncident, error)
}

// AIOpsIncident 数据库模型
type AIOpsIncident struct {
    ID         string
    ClusterID  string
    State      string
    Severity   string
    RootCause  string
    PeakRisk   float64
    StartedAt  time.Time
    ResolvedAt *time.Time
    DurationS  int64
    Recurrence int
    Summary    string
    CreatedAt  time.Time
}

// AIOpsIncidentEntity 数据库模型
type AIOpsIncidentEntity struct {
    IncidentID string
    EntityKey  string
    EntityType string
    RLocal     float64
    RFinal     float64
    Role       string
}

// AIOpsIncidentTimeline 数据库模型
type AIOpsIncidentTimeline struct {
    ID         int64
    IncidentID string
    Timestamp  time.Time
    EventType  string
    EntityKey  string
    Detail     string
}

// AIOpsIncidentQueryOpts 查询选项
type AIOpsIncidentQueryOpts struct {
    ClusterID string
    State     string
    Severity  string
    From      time.Time
    To        time.Time
    Limit     int
    Offset    int
}

// AIOpsIncidentStatsRaw Repository 层返回的统计原始数据（单次查询）
type AIOpsIncidentStatsRaw struct {
    TotalIncidents  int
    ActiveIncidents int
    MTTR            float64        // 平均恢复时间 (分钟)
    RecurringCount  int
    BySeverity      map[string]int
    ByState         map[string]int
}

// AIOpsRootCauseCount 根因统计
type AIOpsRootCauseCount struct {
    EntityKey string
    Count     int
}
```

---

## 10. 实现阶段（TDD）

```
P1: 数据库 + Repository
  ├── database/interfaces.go 新增模型 + Repository 接口
  ├── migrations.go +3 张表 + 4 个索引
  ├── sqlite/aiops_incident.go Dialect 实现
  ├── repo/aiops_incident.go Repository 实现
  └── 单元测试: CRUD + 统计查询

P2: 状态机引擎
  ├── statemachine/machine.go 状态定义 + 转换规则
  ├── statemachine/trigger.go 评估 + 转换执行
  ├── statemachine/suppressor.go 告警抑制
  └── 单元测试:
      ├── 各状态转换路径（5 条）
      ├── 持续时间检查（不满足不触发）
      ├── 告警抑制（Incident 期间不重复）
      └── Recovery → Stable（48h 定时检查）

P3: 事件存储
  ├── incident/store.go CRUD
  ├── incident/timeline.go 时间线
  ├── incident/stats.go 统计
  ├── incident/patterns.go 模式匹配
  └── 单元测试: 创建/更新/解决/统计/模式查询

P4: 引擎集成 + API
  ├── aiops/engine.go OnSnapshot 调用状态机 + 定时检查启动
  ├── aiops/interfaces.go +事件查询方法
  ├── service/interfaces.go +4 方法
  ├── service/query/aiops.go +事件查询实现
  ├── handler/aiops_incident.go API Handler
  ├── gateway/routes.go +4 路由
  └── 端到端测试
```

---

## 11. 文件变更清单

### 新建

| 文件 | 说明 |
|------|------|
| `aiops/statemachine/machine.go` | 状态定义 + 转换规则 |
| `aiops/statemachine/trigger.go` | 持续时间评估 + 转换触发 |
| `aiops/statemachine/suppressor.go` | 告警抑制逻辑 |
| `aiops/incident/store.go` | Incident CRUD |
| `aiops/incident/timeline.go` | 时间线事件管理 |
| `aiops/incident/stats.go` | 统计查询 |
| `aiops/incident/patterns.go` | 历史模式匹配 |
| `database/sqlite/aiops_incident.go` | 事件 SQL Dialect |
| `database/repo/aiops_incident.go` | 事件 Repository 实现 |
| `gateway/handler/aiops_incident.go` | 事件 API Handler |

### 修改

| 文件 | 变更 |
|------|------|
| `aiops/interfaces.go` | +事件查询方法 (GetIncidents, GetIncidentDetail 等) |
| `aiops/engine.go` | +状态机评估 + Recovery→Stable 定时检查 |
| `aiops/types.go` | +Incident, IncidentEntity, IncidentTimeline, IncidentStats 等 |
| `database/interfaces.go` | +AIOpsIncidentRepository + 模型 + Dialect |
| `database/sqlite/migrations.go` | +3 张表 + 4 个索引 |
| `service/interfaces.go` | Query 接口 +4 方法 |
| `service/query/aiops.go` | +4 事件查询实现 |
| `gateway/routes.go` | +4 路由 |

---

## 12. 测试计划

### 单元测试

| 模块 | 测试文件 | 测试内容 |
|------|---------|---------|
| 状态机 | `statemachine/machine_test.go` | 状态转换规则、转换条件验证 |
| 触发器 | `statemachine/trigger_test.go` | 持续时间不足不触发、满足后触发、复发检测 |
| 抑制 | `statemachine/suppressor_test.go` | Incident 期间抑制、Recovery 后解除 |
| 事件 CRUD | `incident/store_test.go` | 创建/更新状态/解决/复发计数 |
| 时间线 | `incident/timeline_test.go` | 时间线添加和查询 |
| 统计 | `incident/stats_test.go` | MTTR/复发率/按 severity 分布 |
| 模式 | `incident/patterns_test.go` | 历史模式匹配准确性 |

### 集成测试

| 场景 | 验证点 |
|------|--------|
| 完整生命周期 | Healthy → Warning → Incident → Recovery → Stable |
| 复发 | Recovery → Warning（标记复发，recurrence++） |
| 告警抑制 | Incident 期间同一实体不创建新 Incident |
| 多实体联动 | Node 异常 → 相关 Pod/Service 进入 Warning |
| 定时检查 | Recovery 超 48h 自动转 Stable |

---

## 13. 验证命令

```bash
# 构建验证
go build ./atlhyper_master_v2/...

# 单元测试
go test ./atlhyper_master_v2/aiops/statemachine/... -v
go test ./atlhyper_master_v2/aiops/incident/... -v
go test ./atlhyper_master_v2/database/repo/ -run AIOpsIncident -v

# API 测试
curl "http://localhost:8080/api/v2/aiops/incidents?cluster=test-cluster"
curl "http://localhost:8080/api/v2/aiops/incidents/stats?cluster=test-cluster&period=7d"
```

---

## 14. 阶段实施后评审规范

> **本阶段实施完成后，必须对后续所有阶段的设计文档进行重新评审。**

### 原因

每个阶段的实施可能导致代码结构、接口签名、数据模型与设计文档中的预期产生偏差。提前编写的设计文档基于「假设的代码状态」，而实际实施后的代码才是唯一真实状态。不经过评审就直接实施下一阶段，可能导致：

- 接口签名不匹配（设计文档引用的方法名/参数与实际实现不一致）
- 文件路径变更（实施中因重构调整了目录结构）
- 数据模型演变（字段增删或类型变更）
- 新增的约束或依赖未在后续设计中体现

### 本阶段实施后需评审的文档

| 文档 | 重点评审内容 |
|------|-------------|
| `aiops-phase3-frontend.md` | Incident API 实际响应格式（`IncidentDetail` / `IncidentStats` / `IncidentPattern` 的实际 JSON 结构）、事件状态枚举值、`TransitionCallback` 实际实现方式 |
| `aiops-phase4-ai-enhancement.md` | `incident.Store` 实际 API、`AIOpsIncidentRepository` 实际接口、事件数据的实际获取方式、`IncidentTimeline` 实际字段 |

### 评审检查清单

- [ ] 设计文档中引用的接口签名与实际代码一致
- [ ] 设计文档中的文件路径与实际目录结构一致
- [ ] 设计文档中的数据模型与实际 struct 定义一致
- [ ] 设计文档中的数据库表结构与实际 migration 一致
- [ ] 设计文档中的初始化链路与 `master.go` 实际代码一致
- [ ] 如有偏差，更新设计文档后再开始下一阶段实施
