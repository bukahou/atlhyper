# AIOps 状态按集群区分

## 背景

Master 设计上管理多个集群，实体键（`namespace/type/name`）只在一个集群内唯一。
AIOps 引擎里有三处状态只用实体键做键，没有带集群：

| 状态 | 现在的键 | 两个集群同时上报时的后果 |
|------|----------|--------------------------|
| 基线（`baseline.StateManager`） | 实体键 + 指标名 | 同名实体共用一条基线；每个集群的快照都会把「不在本快照里的实体」当作已消失而清掉，另一个集群的基线永远停在冷启动 |
| 状态机条目（`statemachine.StateMachine`） | 实体键 | 同名实体共用一个状态与一个事件 ID |
| 历史事件模式（`incident.GetPatterns`、AI 证据里的历史事件） | 实体键 | 把另一个集群同名实体的事件算进来 |

依赖图、风险评分、异常缓存已经按集群区分，不在本次范围内。

当前每个 Master 只接了一个集群，所以线上还没有出现上述后果。

## 目标

- 上述三处状态都以「集群 + 实体键」为键
- 基线表带集群列；实体消失后库里的基线行一并删除（现在只清内存，库里只增不减）
- 对外接口的参数与返回字段补上集群

## 核心设计

共享一个键类型，三处状态都用它：

```go
// aiops/types.go
// EntityRef 指向某个集群里的某个实体。实体键只在集群内唯一，跨集群保存的状态必须用它做键。
type EntityRef struct {
    ClusterID string
    EntityKey string
}
```

数据流不变，只是每一步都带上集群：

```
OnSnapshot(clusterID)
  ├─ stateManager.CleanupStaleEntities(clusterID, activeKeys)   只清本集群
  ├─ stateManager.Update(clusterID, points)
  └─ sm.Evaluate(ctx, clusterID, entityRisks, clusterRisk)      条目键 = EntityRef
```

## 功能一：基线按集群区分

**接口变化**

```go
// aiops/baseline
func (m *StateManager) Update(clusterID string, points []aiops.MetricDataPoint) []*aiops.AnomalyResult
func (m *StateManager) GetStates(clusterID, entityKey string) *aiops.EntityBaseline
func (m *StateManager) CleanupStaleEntities(clusterID string, activeKeys map[string]bool) int

// aiops.Engine
GetBaseline(clusterID, entityKey string) *EntityBaseline

// database.AIOpsBaselineRepository
BatchUpsert(ctx, states []*AIOpsBaselineState) error
ListAll(ctx) ([]*AIOpsBaselineState, error)
DeleteEntities(ctx, clusterID string, entityKeys []string) error   // 新增，取代未被调用的 ListByEntity / DeleteByEntity
```

**数据对照**

| API JSON（`GET /api/v2/aiops/baseline`） | Go（`aiops.BaselineState`） | 表 `aiops_baseline_states` |
|---|---|---|
| `clusterId` | `ClusterID` | `cluster_id TEXT NOT NULL` |
| `entityKey` | `EntityKey` | `entity_key TEXT NOT NULL` |
| `metricName` | `MetricName` | `metric_name TEXT NOT NULL` |
| `ema` / `variance` / `count` / `updatedAt` | 同名 | 同名 |

主键由 `(entity_key, metric_name)` 改为 `(cluster_id, entity_key, metric_name)`。

**库内清理**：`CleanupStaleEntities` 把清掉的实体记下来，下一次 `FlushToDB` 先删这些实体的行、再写脏状态。
实体在两次 flush 之间消失又出现（StatefulSet 的 Pod 重建）时，`Update` 把它从待删集合里拿掉。

**边界**

- 两个集群有同名实体：基线互不影响
- 集群 A 的快照不会清掉集群 B 的基线
- 待删与待写同时存在：先删后写，重新出现的实体不被删

## 功能二：状态机条目按集群区分

```go
// aiops.StateMachineEntry 增加 ClusterID
func (sm *StateMachine) GetEntry(clusterID, entityKey string) *aiops.StateMachineEntry
func (sm *StateMachine) RestoreEntry(entry *aiops.StateMachineEntry)   // 用 entry.ClusterID
```

`Evaluate` 已有 `clusterID` 参数；条目的键改为 `EntityRef`。回调签名不变。
启动恢复（`reloadActiveIncidents`）用事件上的 `ClusterID`。

**边界**：两个集群的同名实体各自走状态转换、各自创建事件；一个集群的条目关闭不影响另一个。

## 功能三：历史事件模式按集群区分

```
GET /api/v2/aiops/incidents/patterns?cluster_id={id}&entity={key}&period=30d
```

`cluster_id` 改为必填（前端已经在传）。缺失返回 400。

```go
// database.AIOpsIncidentRepository
ListByEntity(ctx, clusterID, entityKey string, since time.Time) ([]*AIOpsIncident, error)
// aiops.Engine
GetIncidentPatterns(ctx, clusterID, entityKey string, since time.Time) []*IncidentPattern
```

AI 证据收集里的「该实体的历史事件」同样带上事件所属集群。

## 上线影响

- **基线表结构变了**（多一列、主键变了）。按 CLAUDE.md §1.9，表结构变更不写增量迁移，上线时删库重建。
  处理方式待用户裁决，见任务追踪。
- 基线需要重新学习：冷启动 100 个数据点，按 10 秒一轮约 17 分钟。期间统计类异常不告警；
  确定性异常（崩溃、NotReady、OOM 等）走直注路径，不受影响。
- 前端无改动：两个接口的 `cluster_id` 参数已经在传；基线响应里多出的 `clusterId` 字段前端不使用。

## 文件变更清单

```
atlhyper_master_v2/
├── aiops/
│   ├── types.go                    [修改] EntityRef；BaselineState / StateMachineEntry 加 ClusterID
│   ├── interfaces.go               [修改] GetBaseline / GetIncidentPatterns 加 clusterID
│   ├── baseline/state.go           [修改] 键改为 EntityRef；清理只限本集群；待删集合
│   ├── baseline/state_test.go      [新增]
│   ├── statemachine/machine.go     [修改] 条目键改为 EntityRef
│   ├── statemachine/trigger.go     [修改]
│   ├── statemachine/machine_test.go[修改]
│   ├── incident/patterns.go        [修改]
│   ├── enricher/evidence.go        [修改]
│   └── core/engine.go              [修改]
├── database/
│   ├── interfaces.go               [修改]
│   ├── types.go                    [修改] AIOpsBaselineState 加 ClusterID
│   ├── sqlite/migrations.go        [修改] 基线表定义
│   ├── sqlite/aiops_baseline.go    [修改]
│   ├── repo/aiops_baseline.go      [修改]
│   ├── repo/aiops_baseline_test.go [新增]
│   └── repo/aiops_incident.go      [修改]
├── service/interfaces.go           [修改]
├── service/query/aiops.go          [修改]
└── gateway/handler/aiops/aiops_incident.go [修改] cluster_id 必填
```
