# Event Alert Service Design Document

> Version: 1.0
> Date: 2026-01-25
> Author: AtlHyper Team

---

## 1. Overview

### 1.1 Purpose

EventAlertService monitors cluster events stored in the database and sends alerts through configured notification channels (Slack, Email). It enriches alert messages with related Kubernetes resource information to provide actionable context.

### 1.2 Design Principles

- **Low Coupling**: Read from database and Service layer, not directly from DataHub
- **Database as Source of Truth**: DB already stores Warning events with deduplication
- **Two-Level Enrichment**: Event → Related Resources (Pod/Node/Deployment)
- **Configurable Rules**: Alert rules are defined per event type

---

## 2. Architecture

### 2.1 Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Event Alert Flow                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────┐     ┌──────────────────┐     ┌───────────────────────┐   │
│   │  Database   │────▶│ EventAlertService│────▶│   AlertManager       │   │
│   │ (EventRepo) │     │                  │     │ (dedup + dispatch)   │   │
│   └─────────────┘     └────────┬─────────┘     └───────────────────────┘   │
│                                │                           │               │
│                                ▼                           ▼               │
│                       ┌────────────────┐          ┌────────────────┐       │
│                       │ Service.Query  │          │   Channels     │       │
│                       │ (GetPod, etc.) │          │ (Slack, Email) │       │
│                       └────────────────┘          └────────────────┘       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Component Responsibilities

| Component | Responsibility |
|-----------|----------------|
| EventRepo | Stores Warning events with deduplication (DedupKey = MD5 hash) |
| EventAlertService | Polls events, applies rules, enriches with resources, sends alerts |
| Service.Query | Provides resource snapshots (Pod, Node, Deployment) |
| AlertManager | Deduplicates alerts (10min TTL), dispatches to channels |
| Channels | Slack/Email adapters |

### 2.3 Why Database-First Approach

The database-first approach was chosen for several reasons:

1. **Events are already deduplicated**: Each event has a `DedupKey` (MD5 of cluster+kind+namespace+name+reason)
2. **Only Warning events are stored**: No need to filter by event type
3. **Count aggregation**: Multiple occurrences are tracked via `Count` field
4. **Persistence**: Events survive Master restarts

---

## 3. Directory Structure

```
service/operations/
├── event_alert.go         # Main service: polling, coordination
├── event_alert_rules.go   # Alert rules: severity, message templates
└── event_alert_enrich.go  # Resource enrichment: Pod/Node/Deployment
```

---

## 4. Detailed Design

### 4.1 Configuration

```go
// config/types.go
type EventAlertConfig struct {
    Enabled       bool          // Enable event alerting
    CheckInterval time.Duration // Polling interval (default: 30s)
}
```

Environment variables:
- `MASTER_EVENT_ALERT_ENABLED` (bool, default: true)
- `MASTER_EVENT_ALERT_INTERVAL` (duration, default: 30s)

### 4.2 Event Alert Service

```go
// service/operations/event_alert.go
type EventAlertService struct {
    eventRepo  database.EventRepository
    query      query.Query
    alertMgr   notifier.AlertManager
    config     EventAlertConfig
    lastSeenID int64          // Track last processed event
    stopCh     chan struct{}
    wg         sync.WaitGroup
}

// Core methods
func NewEventAlertService(eventRepo, query, alertMgr, config) *EventAlertService
func (s *EventAlertService) Start() error
func (s *EventAlertService) Stop() error
func (s *EventAlertService) processEvents()
func (s *EventAlertService) processEvent(event *database.ClusterEvent) error
```

**Polling Logic:**
1. Query events where `ID > lastSeenID`
2. For each event, apply alert rules to determine severity
3. Enrich event with related resource information
4. Send to AlertManager
5. Update `lastSeenID`

### 4.3 Alert Rules

```go
// service/operations/event_alert_rules.go
type AlertRule struct {
    Reason      string          // Event reason to match
    Severity    notifier.Level  // Alert severity
    Description string          // Human-readable description
}

// Pre-defined rules
var AlertRules = map[string]AlertRule{
    "Killing": {
        Reason:      "Killing",
        Severity:    notifier.LevelWarning,
        Description: "Pod is being terminated",
    },
    "BackOff": {
        Reason:      "BackOff",
        Severity:    notifier.LevelWarning,
        Description: "Container restart backoff",
    },
    "Unhealthy": {
        Reason:      "Unhealthy",
        Severity:    notifier.LevelCritical,
        Description: "Container health check failed",
    },
    "FailedScheduling": {
        Reason:      "FailedScheduling",
        Severity:    notifier.LevelCritical,
        Description: "Pod cannot be scheduled",
    },
    "OOMKilled": {
        Reason:      "OOMKilled",
        Severity:    notifier.LevelCritical,
        Description: "Container killed due to out of memory",
    },
    // ... more rules
}

func MatchRule(event *database.ClusterEvent) *AlertRule
func GetAlertTitle(event *database.ClusterEvent, rule *AlertRule) string
```

### 4.4 Resource Enrichment

```go
// service/operations/event_alert_enrich.go
type ResourceEnricher struct {
    query query.Query
}

type EnrichedEvent struct {
    Event       *database.ClusterEvent
    PodStatus   *PodStatusInfo   // If kind=Pod
    NodeStatus  *NodeStatusInfo  // If kind=Node
    Deployment  *DeploymentInfo  // If kind=Pod → find owner Deployment
}

type PodStatusInfo struct {
    Phase        string
    Restarts     int32
    Ready        string   // "1/2"
    NodeName     string
}

type NodeStatusInfo struct {
    Ready        bool
    Allocatable  string   // CPU/Memory summary
    Conditions   []string // Abbreviated condition list
}

type DeploymentInfo struct {
    Name         string
    Replicas     string   // "2/3"
    Namespace    string
}

func (e *ResourceEnricher) Enrich(ctx context.Context, event *database.ClusterEvent) *EnrichedEvent
```

**Enrichment Logic by Kind:**

| InvolvedKind | Enrichment |
|--------------|------------|
| Pod | GetPod → PodStatus + GetDeploymentByReplicaSet → DeploymentInfo |
| Node | GetNode → NodeStatus |
| Deployment | GetDeployment → DeploymentInfo |
| ReplicaSet | GetDeploymentByReplicaSet → DeploymentInfo |
| Other | No enrichment, use event message only |

### 4.5 Query Interface Extensions

```go
// service/query/query.go
type Query interface {
    // Existing methods...

    // New methods for enrichment
    GetPod(clusterID, namespace, name string) (*types.Pod, error)
    GetNode(clusterID, name string) (*types.Node, error)
    GetDeployment(clusterID, namespace, name string) (*types.Deployment, error)
    GetDeploymentByReplicaSet(clusterID, namespace, rsName string) (*types.Deployment, error)
}
```

---

## 5. Alert Message Format

### 5.1 Slack Message Example

```
🔴 [ZGMF-X10A] OOMKilled: Container killed due to out of memory

Resource: Pod/atlhyper/atlhyper-agent-7d8f9b6c5d-x4k2m
Message: Container exceeded memory limit and was killed
Occurrences: 3

Status:
  Phase: Running
  Restarts: 5
  Ready: 0/1
  Node: node-1

Related Deployment:
  Name: atlhyper-agent
  Replicas: 2/3

Time: 2026-01-25 10:30:45 JST
```

### 5.2 Alert Deduplication

AlertManager deduplicates by `DedupKey` with 10-minute TTL:
- Same event within 10 minutes → skip
- After 10 minutes → alert again (with updated count)

---

## 6. Initialization Flow

```go
// master.go New()

// ... existing initialization ...

// 12. Initialize EventAlertService (optional)
var eventAlertService *operations.EventAlertService
if cfg.EventAlert.Enabled {
    eventAlertService = operations.NewEventAlertService(
        db.Event,
        q,
        alertMgr,
        operations.EventAlertConfig{
            CheckInterval: cfg.EventAlert.CheckInterval,
        },
    )
    log.Println("[Master] Event alert service initialized")
}

// master.go Run()
if m.eventAlertService != nil {
    if err := m.eventAlertService.Start(); err != nil {
        return fmt.Errorf("failed to start event alert service: %w", err)
    }
}

// master.go Stop()
if m.eventAlertService != nil {
    if err := m.eventAlertService.Stop(); err != nil {
        log.Printf("[Master] Failed to stop event alert service: %v", err)
    }
}
```

---

## 7. Error Handling

| Scenario | Handling |
|----------|----------|
| Database query fails | Log error, retry next interval |
| Resource enrichment fails | Send alert without enrichment |
| AlertManager.Send fails | Log error, continue to next event |
| Query.GetPod returns nil | Skip enrichment for that resource |

---

## 8. Monitoring

Metrics to track:
- Events processed per interval
- Alerts sent per channel
- Enrichment failures
- Processing latency

Log format:
```
[EventAlert] Processed 5 events, sent 3 alerts, 0 enrichment failures
[EventAlert] Alert sent: cluster=ZGMF-X10A, reason=OOMKilled, kind=Pod, name=xxx
```

---

## 9. Security Considerations

- Event data comes from internal database, not external sources
- No sensitive data (Secrets) are included in alerts
- AlertManager already handles channel authentication (Slack webhook, SMTP auth)

---

## 10. Future Enhancements

1. **Customizable Rules**: Allow users to define alert rules via API
2. **Alert Silencing**: Temporarily mute specific alerts
3. **Alert Routing**: Route different severity levels to different channels
4. **Webhook Channel**: Support generic webhook notifications
