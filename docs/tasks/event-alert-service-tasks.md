# Event Alert Service - Implementation Tasks

> Reference: [event-alert-service-design.md](./event-alert-service-design.md)
> Estimated Tasks: 10
> Priority: P1

---

## Task Overview

| # | Task | Files | Status |
|---|------|-------|--------|
| 1 | Add Query interface methods | `service/interfaces.go`, `service/query/impl.go` | ✅ |
| 2 | Add configuration | `config/types.go`, `config/loader.go`, `config/defaults.go` | ✅ |
| 3 | Create alert rules | `service/operations/event_alert_rules.go` | ✅ |
| 4 | Create resource enricher | `service/operations/event_alert_enrich.go` | ✅ |
| 5 | Create main service | `service/operations/event_alert.go` | ✅ |
| 6 | Update Master initialization | `master.go` | ✅ |
| 7 | Add EventRepo query method | `database/repo/event.go` | ✅ |
| 8 | Update environment config | `~/.env_profile` | ✅ |
| 9 | Unit tests | `service/operations/event_alert_test.go` | ☐ |
| 10 | Integration test | Manual testing | ☐ |

---

## Task Details

### Task 1: Add Query Interface Methods

**Files:**
- `service/query/query.go`
- `service/query/query_impl.go`

**Changes:**

```go
// service/query/query.go - Add to Query interface
type Query interface {
    // ... existing methods ...

    // Resource getters for event enrichment
    GetPod(clusterID, namespace, name string) (*types.Pod, error)
    GetNode(clusterID, name string) (*types.Node, error)
    GetDeployment(clusterID, namespace, name string) (*types.Deployment, error)
    GetDeploymentByReplicaSet(clusterID, namespace, rsName string) (*types.Deployment, error)
}
```

```go
// service/query/query_impl.go - Implement methods
func (q *queryImpl) GetPod(clusterID, namespace, name string) (*types.Pod, error) {
    snapshot, err := q.store.GetSnapshot(clusterID)
    if err != nil {
        return nil, err
    }
    for _, pod := range snapshot.Pods {
        if pod.Namespace == namespace && pod.Name == name {
            return &pod, nil
        }
    }
    return nil, nil // Not found
}

// Similar for GetNode, GetDeployment, GetDeploymentByReplicaSet
```

---

### Task 2: Add Configuration

**Files:**
- `config/types.go`
- `config/loader.go`
- `config/defaults.go`

**Changes in types.go:**

```go
type Config struct {
    // ... existing fields ...
    EventAlert EventAlertConfig
}

type EventAlertConfig struct {
    Enabled       bool
    CheckInterval time.Duration
}
```

**Changes in loader.go:**

```go
func LoadConfig() error {
    // ... existing code ...

    // Event Alert
    cfg.EventAlert = EventAlertConfig{
        Enabled:       getBool("MASTER_EVENT_ALERT_ENABLED"),
        CheckInterval: getDuration("MASTER_EVENT_ALERT_INTERVAL"),
    }

    return nil
}
```

**Changes in defaults.go:**

```go
var defaultBools = map[string]bool{
    // ... existing ...
    "MASTER_EVENT_ALERT_ENABLED": true,
}

var defaultDurations = map[string]string{
    // ... existing ...
    "MASTER_EVENT_ALERT_INTERVAL": "30s",
}
```

---

### Task 3: Create Alert Rules

**File:** `service/operations/event_alert_rules.go`

```go
package operations

import "AtlHyper/atlhyper_master_v2/notifier"

type AlertRule struct {
    Reason      string
    Severity    notifier.Level
    Description string
}

var AlertRules = map[string]AlertRule{
    // Pod lifecycle events
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
    "CrashLoopBackOff": {
        Reason:      "CrashLoopBackOff",
        Severity:    notifier.LevelCritical,
        Description: "Container is in crash loop",
    },

    // Health check events
    "Unhealthy": {
        Reason:      "Unhealthy",
        Severity:    notifier.LevelCritical,
        Description: "Container health check failed",
    },
    "ProbeWarning": {
        Reason:      "ProbeWarning",
        Severity:    notifier.LevelWarning,
        Description: "Container probe returned warning",
    },

    // Scheduling events
    "FailedScheduling": {
        Reason:      "FailedScheduling",
        Severity:    notifier.LevelCritical,
        Description: "Pod cannot be scheduled",
    },
    "FailedMount": {
        Reason:      "FailedMount",
        Severity:    notifier.LevelCritical,
        Description: "Volume mount failed",
    },
    "FailedAttachVolume": {
        Reason:      "FailedAttachVolume",
        Severity:    notifier.LevelCritical,
        Description: "Volume attachment failed",
    },

    // Resource events
    "OOMKilled": {
        Reason:      "OOMKilled",
        Severity:    notifier.LevelCritical,
        Description: "Container killed due to out of memory",
    },
    "Evicted": {
        Reason:      "Evicted",
        Severity:    notifier.LevelCritical,
        Description: "Pod evicted from node",
    },

    // Image events
    "Failed": {
        Reason:      "Failed",
        Severity:    notifier.LevelWarning,
        Description: "Container operation failed",
    },
    "ImagePullBackOff": {
        Reason:      "ImagePullBackOff",
        Severity:    notifier.LevelCritical,
        Description: "Image pull failed, backing off",
    },
    "ErrImagePull": {
        Reason:      "ErrImagePull",
        Severity:    notifier.LevelCritical,
        Description: "Image pull error",
    },

    // Node events
    "NodeNotReady": {
        Reason:      "NodeNotReady",
        Severity:    notifier.LevelCritical,
        Description: "Node is not ready",
    },
    "NodeNotSchedulable": {
        Reason:      "NodeNotSchedulable",
        Severity:    notifier.LevelWarning,
        Description: "Node is not schedulable",
    },
    "MemoryPressure": {
        Reason:      "MemoryPressure",
        Severity:    notifier.LevelCritical,
        Description: "Node has memory pressure",
    },
    "DiskPressure": {
        Reason:      "DiskPressure",
        Severity:    notifier.LevelCritical,
        Description: "Node has disk pressure",
    },
}

// MatchRule finds matching rule for event reason
func MatchRule(reason string) *AlertRule {
    if rule, ok := AlertRules[reason]; ok {
        return &rule
    }
    return nil
}

// DefaultRule returns a default rule for unknown events
func DefaultRule() *AlertRule {
    return &AlertRule{
        Reason:      "Unknown",
        Severity:    notifier.LevelWarning,
        Description: "Unknown warning event",
    }
}
```

---

### Task 4: Create Resource Enricher

**File:** `service/operations/event_alert_enrich.go`

```go
package operations

import (
    "context"
    "fmt"

    "AtlHyper/atlhyper_master_v2/database"
    "AtlHyper/atlhyper_master_v2/service/query"
)

type ResourceEnricher struct {
    query query.Query
}

type EnrichedEvent struct {
    Event      *database.ClusterEvent
    Pod        *PodInfo
    Node       *NodeInfo
    Deployment *DeploymentInfo
}

type PodInfo struct {
    Phase    string
    Restarts int32
    Ready    string // "1/2"
    NodeName string
}

type NodeInfo struct {
    Ready      bool
    Conditions string // Abbreviated
}

type DeploymentInfo struct {
    Name      string
    Namespace string
    Replicas  string // "2/3"
}

func NewResourceEnricher(q query.Query) *ResourceEnricher {
    return &ResourceEnricher{query: q}
}

func (e *ResourceEnricher) Enrich(ctx context.Context, event *database.ClusterEvent) *EnrichedEvent {
    enriched := &EnrichedEvent{Event: event}

    switch event.InvolvedKind {
    case "Pod":
        e.enrichPod(event.ClusterID, event.InvolvedNamespace, event.InvolvedName, enriched)
    case "Node":
        e.enrichNode(event.ClusterID, event.InvolvedName, enriched)
    case "Deployment":
        e.enrichDeployment(event.ClusterID, event.InvolvedNamespace, event.InvolvedName, enriched)
    case "ReplicaSet":
        e.enrichFromReplicaSet(event.ClusterID, event.InvolvedNamespace, event.InvolvedName, enriched)
    }

    return enriched
}

func (e *ResourceEnricher) enrichPod(clusterID, ns, name string, enriched *EnrichedEvent) {
    pod, err := e.query.GetPod(clusterID, ns, name)
    if err != nil || pod == nil {
        return
    }

    var restarts int32
    var readyCount, totalCount int
    for _, cs := range pod.Status.ContainerStatuses {
        restarts += cs.RestartCount
        totalCount++
        if cs.Ready {
            readyCount++
        }
    }

    enriched.Pod = &PodInfo{
        Phase:    string(pod.Status.Phase),
        Restarts: restarts,
        Ready:    fmt.Sprintf("%d/%d", readyCount, totalCount),
        NodeName: pod.Spec.NodeName,
    }

    // Try to find owner Deployment
    for _, ref := range pod.OwnerReferences {
        if ref.Kind == "ReplicaSet" {
            e.enrichFromReplicaSet(clusterID, ns, ref.Name, enriched)
            break
        }
    }
}

func (e *ResourceEnricher) enrichNode(clusterID, name string, enriched *EnrichedEvent) {
    node, err := e.query.GetNode(clusterID, name)
    if err != nil || node == nil {
        return
    }

    var ready bool
    var conditions []string
    for _, cond := range node.Status.Conditions {
        if cond.Type == "Ready" {
            ready = cond.Status == "True"
        }
        if cond.Status != "False" && cond.Type != "Ready" {
            conditions = append(conditions, string(cond.Type))
        }
    }

    condStr := "None"
    if len(conditions) > 0 {
        condStr = fmt.Sprintf("%v", conditions)
    }

    enriched.Node = &NodeInfo{
        Ready:      ready,
        Conditions: condStr,
    }
}

func (e *ResourceEnricher) enrichDeployment(clusterID, ns, name string, enriched *EnrichedEvent) {
    dep, err := e.query.GetDeployment(clusterID, ns, name)
    if err != nil || dep == nil {
        return
    }

    enriched.Deployment = &DeploymentInfo{
        Name:      dep.Name,
        Namespace: dep.Namespace,
        Replicas:  fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, dep.Status.Replicas),
    }
}

func (e *ResourceEnricher) enrichFromReplicaSet(clusterID, ns, rsName string, enriched *EnrichedEvent) {
    dep, err := e.query.GetDeploymentByReplicaSet(clusterID, ns, rsName)
    if err != nil || dep == nil {
        return
    }

    enriched.Deployment = &DeploymentInfo{
        Name:      dep.Name,
        Namespace: dep.Namespace,
        Replicas:  fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, dep.Status.Replicas),
    }
}

// FormatEnrichedInfo formats enriched info for alert message
func FormatEnrichedInfo(e *EnrichedEvent) string {
    var info string

    if e.Pod != nil {
        info += fmt.Sprintf("\nPod Status:\n  Phase: %s\n  Restarts: %d\n  Ready: %s\n  Node: %s",
            e.Pod.Phase, e.Pod.Restarts, e.Pod.Ready, e.Pod.NodeName)
    }

    if e.Node != nil {
        readyStr := "Yes"
        if !e.Node.Ready {
            readyStr = "No"
        }
        info += fmt.Sprintf("\nNode Status:\n  Ready: %s\n  Conditions: %s",
            readyStr, e.Node.Conditions)
    }

    if e.Deployment != nil {
        info += fmt.Sprintf("\nRelated Deployment:\n  Name: %s\n  Namespace: %s\n  Replicas: %s",
            e.Deployment.Name, e.Deployment.Namespace, e.Deployment.Replicas)
    }

    return info
}
```

---

### Task 5: Create Main Service

**File:** `service/operations/event_alert.go`

```go
package operations

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"

    "AtlHyper/atlhyper_master_v2/database"
    "AtlHyper/atlhyper_master_v2/notifier"
    "AtlHyper/atlhyper_master_v2/service/query"
)

type EventAlertConfig struct {
    CheckInterval time.Duration
}

type EventAlertService struct {
    eventRepo  database.EventRepository
    enricher   *ResourceEnricher
    alertMgr   notifier.AlertManager
    config     EventAlertConfig
    lastSeenID int64
    stopCh     chan struct{}
    wg         sync.WaitGroup
    mu         sync.Mutex
}

func NewEventAlertService(
    eventRepo database.EventRepository,
    q query.Query,
    alertMgr notifier.AlertManager,
    config EventAlertConfig,
) *EventAlertService {
    return &EventAlertService{
        eventRepo: eventRepo,
        enricher:  NewResourceEnricher(q),
        alertMgr:  alertMgr,
        config:    config,
        stopCh:    make(chan struct{}),
    }
}

func (s *EventAlertService) Start() error {
    // Get latest event ID to start from
    latestID, err := s.eventRepo.GetLatestEventID(context.Background())
    if err != nil {
        log.Printf("[EventAlert] Failed to get latest event ID, starting from 0: %v", err)
        latestID = 0
    }
    s.lastSeenID = latestID
    log.Printf("[EventAlert] Starting from event ID: %d", s.lastSeenID)

    s.wg.Add(1)
    go s.pollLoop()

    log.Printf("[EventAlert] Started with interval: %v", s.config.CheckInterval)
    return nil
}

func (s *EventAlertService) Stop() error {
    close(s.stopCh)
    s.wg.Wait()
    log.Println("[EventAlert] Stopped")
    return nil
}

func (s *EventAlertService) pollLoop() {
    defer s.wg.Done()

    ticker := time.NewTicker(s.config.CheckInterval)
    defer ticker.Stop()

    for {
        select {
        case <-s.stopCh:
            return
        case <-ticker.C:
            s.processEvents()
        }
    }
}

func (s *EventAlertService) processEvents() {
    ctx := context.Background()

    // Query new events since lastSeenID
    events, err := s.eventRepo.GetEventsSince(ctx, s.lastSeenID)
    if err != nil {
        log.Printf("[EventAlert] Failed to query events: %v", err)
        return
    }

    if len(events) == 0 {
        return
    }

    var alertsSent, enrichFailed int

    for _, event := range events {
        if err := s.processEvent(ctx, event); err != nil {
            log.Printf("[EventAlert] Failed to process event %d: %v", event.ID, err)
            if err.Error() == "enrichment failed" {
                enrichFailed++
            }
        } else {
            alertsSent++
        }

        // Update lastSeenID
        s.mu.Lock()
        if event.ID > s.lastSeenID {
            s.lastSeenID = event.ID
        }
        s.mu.Unlock()
    }

    log.Printf("[EventAlert] Processed %d events, sent %d alerts, %d enrichment failures",
        len(events), alertsSent, enrichFailed)
}

func (s *EventAlertService) processEvent(ctx context.Context, event *database.ClusterEvent) error {
    // Match rule
    rule := MatchRule(event.Reason)
    if rule == nil {
        rule = DefaultRule()
    }

    // Enrich with resource info
    enriched := s.enricher.Enrich(ctx, event)

    // Build alert
    alert := &notifier.Alert{
        Level:    rule.Severity,
        Title:    s.buildTitle(event, rule),
        Message:  s.buildMessage(event, rule, enriched),
        DedupKey: event.DedupKey,
    }

    // Send to AlertManager
    if err := s.alertMgr.Send(ctx, alert); err != nil {
        return fmt.Errorf("failed to send alert: %w", err)
    }

    log.Printf("[EventAlert] Alert sent: cluster=%s, reason=%s, kind=%s, name=%s",
        event.ClusterID, event.Reason, event.InvolvedKind, event.InvolvedName)

    return nil
}

func (s *EventAlertService) buildTitle(event *database.ClusterEvent, rule *AlertRule) string {
    emoji := "🟡"
    if rule.Severity == notifier.LevelCritical {
        emoji = "🔴"
    }
    return fmt.Sprintf("%s [%s] %s: %s", emoji, event.ClusterID, event.Reason, rule.Description)
}

func (s *EventAlertService) buildMessage(event *database.ClusterEvent, rule *AlertRule, enriched *EnrichedEvent) string {
    msg := fmt.Sprintf("Resource: %s/%s/%s\n", event.InvolvedKind, event.InvolvedNamespace, event.InvolvedName)
    msg += fmt.Sprintf("Message: %s\n", event.Message)
    msg += fmt.Sprintf("Occurrences: %d\n", event.Count)

    // Add enriched info
    msg += FormatEnrichedInfo(enriched)

    msg += fmt.Sprintf("\n\nTime: %s", event.LastSeen.Format("2006-01-02 15:04:05 MST"))

    return msg
}
```

---

### Task 6: Update Master Initialization

**File:** `master.go`

**Add to Master struct:**
```go
type Master struct {
    // ... existing fields ...
    eventAlertService *operations.EventAlertService
}
```

**Add to New():**
```go
// 14. Initialize EventAlertService (optional)
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

return &Master{
    // ... existing fields ...
    eventAlertService: eventAlertService,
}, nil
```

**Add to Run():**
```go
// Start EventAlertService
if m.eventAlertService != nil {
    if err := m.eventAlertService.Start(); err != nil {
        return fmt.Errorf("failed to start event alert service: %w", err)
    }
}
```

**Add to Stop():**
```go
// Stop EventAlertService
if m.eventAlertService != nil {
    if err := m.eventAlertService.Stop(); err != nil {
        log.Printf("[Master] Failed to stop event alert service: %v", err)
    }
}
```

---

### Task 7: Add EventRepo Query Method

**File:** `database/repo/event_repo.go`

**Add methods:**
```go
// GetLatestEventID returns the highest event ID (for startup sync)
func (r *EventRepo) GetLatestEventID(ctx context.Context) (int64, error) {
    var id int64
    err := r.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM cluster_events").Scan(&id)
    return id, err
}

// GetEventsSince returns events with ID greater than sinceID
func (r *EventRepo) GetEventsSince(ctx context.Context, sinceID int64) ([]*ClusterEvent, error) {
    query := `SELECT id, cluster_id, type, reason, message, involved_kind, involved_namespace,
              involved_name, dedup_key, count, first_seen, last_seen
              FROM cluster_events WHERE id > ? ORDER BY id ASC`

    rows, err := r.db.QueryContext(ctx, query, sinceID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var events []*ClusterEvent
    for rows.Next() {
        var e ClusterEvent
        if err := rows.Scan(&e.ID, &e.ClusterID, &e.Type, &e.Reason, &e.Message,
            &e.InvolvedKind, &e.InvolvedNamespace, &e.InvolvedName,
            &e.DedupKey, &e.Count, &e.FirstSeen, &e.LastSeen); err != nil {
            return nil, err
        }
        events = append(events, &e)
    }
    return events, rows.Err()
}
```

**Add to interface:**
```go
// database/interfaces.go
type EventRepository interface {
    // ... existing methods ...
    GetLatestEventID(ctx context.Context) (int64, error)
    GetEventsSince(ctx context.Context, sinceID int64) ([]*ClusterEvent, error)
}
```

---

### Task 8: Update Environment Config

**File:** `~/.env_profile`

```bash
# ----- Event Alert -----
export MASTER_EVENT_ALERT_ENABLED=true
export MASTER_EVENT_ALERT_INTERVAL="30s"
```

---

### Task 9: Unit Tests

**File:** `service/operations/event_alert_test.go`

Test cases:
1. `TestMatchRule` - verify rule matching
2. `TestEnrichPod` - verify pod enrichment
3. `TestEnrichNode` - verify node enrichment
4. `TestBuildMessage` - verify message formatting
5. `TestProcessEvent` - verify end-to-end processing

---

### Task 10: Integration Test

**Manual testing steps:**
1. Start Master with event alert enabled
2. Deploy a pod with invalid image (ErrImagePull)
3. Verify alert is sent to Slack/Email
4. Check alert message contains:
   - Correct severity emoji
   - Cluster ID
   - Event reason and description
   - Resource kind/namespace/name
   - Pod status (if applicable)
   - Related deployment (if applicable)
5. Verify deduplication (same event within 10min should not re-alert)

---

## Dependencies

```
Task 1 ─┬─▶ Task 4 (enricher needs Query methods)
        │
        └─▶ Task 5 (service needs Query)

Task 2 ─▶ Task 6 (master needs config)

Task 3 ─▶ Task 5 (service uses rules)

Task 7 ─▶ Task 5 (service needs repo methods)

Task 4 ─┬
Task 5 ─┴─▶ Task 6 (master assembles all)

Task 6 ─▶ Task 9, 10 (testing)
```

Suggested execution order: 1 → 2 → 3 → 7 → 4 → 5 → 6 → 8 → 9 → 10
