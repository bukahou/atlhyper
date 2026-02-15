# AtlHyper AIOps 引擎 — 中心设计文档

> **状态**: 规划中（中心文档，后续需拆分为多个子设计文档）
> **前置依赖**: SLO OTel 改造（已设计完成，实现中）
> **目标定位**: 开源轻量 AIOps 平台 —— 算法驱动根因分析 + AI 辅助建议

---

## 目录

1. [愿景与目标](#1-愿景与目标)
2. [核心理念](#2-核心理念)
3. [架构总览](#3-架构总览)
4. [五大模块设计](#4-五大模块设计)
   - 4.1 依赖图引擎 (Correlator)
   - 4.2 基线引擎 (Baseline Engine)
   - 4.3 风险评分引擎 (Risk Scorer)
   - 4.4 状态机引擎 (State Machine)
   - 4.5 事件存储 (Incident Store)
5. [数据资产与映射](#5-数据资产与映射)
6. [前端能力规划](#6-前端能力规划)
7. [实现路线图](#7-实现路线图)
8. [架构契合度分析](#8-架构契合度分析)
9. [差异化竞争分析](#9-差异化竞争分析)
10. [风险与约束](#10-风险与约束)
11. [子设计文档索引](#11-子设计文档索引)

---

## 1. 愿景与目标

### 1.1 一句话定位

> AtlHyper 从「K8s 监控面板」进化为「算法驱动的 AIOps 平台」——能预测告警趋势，能自动定位根因，能给出处置建议。

### 1.2 三大核心能力

| 能力 | 说明 | 技术手段 |
|------|------|---------|
| **预测告警趋势** | 在问题升级为事件之前发出早期预警 | 基线学习 + 趋势检测 + 连续风险评分 |
| **算法根因分析** | 自动定位问题根因，不依赖 LLM | 依赖图 + 风险传播 + 时序因果排序 |
| **SLO 全链路溯源** | 从域名 SLO 异常一路钻取到 Pod/Node/日志 | 依赖图查询 + 多维指标关联 |

### 1.3 目标用户场景

```
场景 1: 主动发现
  运维人员打开 AtlHyper → 看到 ClusterRisk = 72 (Warning)
  → 点击查看风险最高的实体: Service "api-gateway" (R_final=0.85)
  → 展开根因链: Node memory pressure → Pod OOMKill → Service 错误率上升
  → 查看建议: "Node worker-3 内存使用率 94%，建议扩容或驱逐低优先级 Pod"

场景 2: SLO 钻取
  域名 app.example.com SLO 达成率从 99.9% 降到 98.2%
  → 点击钻取: Ingress → Service "frontend" → Service "api-server" → Service "db-proxy"
  → 发现 db-proxy P99 延迟从 50ms 飙升到 2000ms
  → 关联 Pod 日志: "connection pool exhausted"
  → 关联 Node: 该 Pod 所在 Node 磁盘 I/O 饱和

场景 3: 事件回顾
  运维查看过去 7 天的事件列表
  → 发现 Service "payment" 本周出现 3 次类似的 P99 飙升
  → 每次根因都是 Pod 重启导致冷启动
  → AI 建议: "考虑增加 Pod 副本数或配置 preStop hook"
```

---

## 2. 核心理念

### 2.1 算法为主，AI 参考

```
算法层（确定性、可解释）         AI 层（灵活、自然语言）
┌─────────────────────┐         ┌──────────────────────┐
│ 风险评分 R=0.85     │         │ "Node worker-3 内存  │
│ 根因: node/worker-3 │ ──────→ │  不足导致 Pod 驱逐， │
│ 因果链: 3 跳        │         │  建议扩容至 16GB"    │
│ 置信度: 92%         │         │                      │
└─────────────────────┘         └──────────────────────┘
  ↑ 提供结构化数据                 ↑ 消费数据，生成建议
```

**设计原则**：
- 算法层提供**确定性和可解释性**——每个评分都能追溯到具体公式和输入指标
- AI 层提供**自然语言总结和对策建议**——消费算法层的结构化输出
- 即使没有 AI 模块，算法层也能独立工作（降级为纯数字展示）
- AI 不参与异常检测和根因定位的核心计算

### 2.2 不引入新依赖

保持 AtlHyper 的「单二进制」部署优势：
- 存储：继续用 SQLite（事件量不大，结构化查询够用）
- 图计算：内存中计算（集群规模 < 1000 节点时无需图数据库）
- 时序基线：滚动存储在 SQLite（保留 7 天原始 + 30 天聚合）
- 不引入 Kafka / Elasticsearch / 专用时序库 / 图数据库

### 2.3 数据驱动，不造数据

只使用 AtlHyper 已经采集到的数据：
- K8s 快照（30s 周期）：Pod/Service/Node/Ingress 等 21 种资源
- Node 指标（OTel）：CPU/Memory/Disk/Network/PSI/TCP
- SLO 指标（OTel，改造中）：Service/Edge/Ingress 三层的错误率/延迟/请求量

不额外部署 agent、sidecar 或采集器。

---

## 3. 架构总览

```
┌─────────────────────────────────────────────────────────────────┐
│                        前端展示层                                │
│                                                                 │
│  ClusterRisk 仪表盘 ─ 趋势图 ─ 事件时间线 ─ 根因卡片           │
│  SLO 钻取链路: Ingress → Service → Pod → Node → 日志           │
│  拓扑图可视化（力导向图 / 层级图）                               │
│                                                                 │
└──────────────────────────┬──────────────────────────────────────┘
                           │ API
┌──────────────────────────▼──────────────────────────────────────┐
│                     Master AIOps 引擎                           │
│                                                                 │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │  M1: 依赖图  │  │ M3: 风险评分  │  │ M4: 状态机            │ │
│  │  引擎       │  │    引擎       │  │    引擎               │ │
│  │ (Correlator)│  │ (Risk Scorer)│  │ (State Machine)       │ │
│  │             │  │              │  │                        │ │
│  │ K8s 拓扑    │  │ 局部风险     │  │ Healthy → Warning →   │ │
│  │ SLO 边关联  │  │ 时序权重     │  │ Incident → Recovery → │ │
│  │ Linkerd 边  │  │ 图传播       │  │ Stable                │ │
│  │             │  │ ClusterRisk  │  │                        │ │
│  └──────┬──────┘  └──────┬───────┘  └───────────┬────────────┘ │
│         │                │                      │               │
│  ┌──────▼────────────────▼──────────────────────▼────────────┐ │
│  │              M2: 基线引擎 (Baseline Engine)                │ │
│  │                                                            │ │
│  │  EMA + 3σ │ 滑动窗口分位数 │ CUSUM │ Holt-Winters         │ │
│  └──────────────────────┬────────────────────────────────────┘ │
│                         │                                       │
│  ┌──────────────────────▼────────────────────────────────────┐ │
│  │          M5: 事件存储 (Incident Store)                     │ │
│  │                                                            │ │
│  │  事件表 │ 受影响实体表 │ 时间线表 │ 基线数据表 │ 依赖图表  │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │          AI 增强层 (消费结构化数据，生成建议)               │ │
│  │  自然语言摘要 │ 处置建议 │ 历史模式匹配                    │ │
│  └───────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                           ▲
                           │ Snapshot 上报 (30s)
                    ┌──────┴──────┐
                    │    Agent    │
                    │ K8s 快照    │
                    │ OTel 指标   │
                    │ Linkerd SLO │
                    └─────────────┘
```

**数据流**：
```
Agent 上报 Snapshot
  → Processor 写入 DataHub (现有流程)
  → AIOps 引擎消费 DataHub 数据:
      1. Correlator 更新依赖图
      2. Baseline 更新基线 + 检测异常
      3. Risk Scorer 计算各实体风险分
      4. State Machine 驱动状态转换
      5. Incident Store 持久化事件
  → Gateway 暴露 AIOps API
  → 前端展示
```

---

## 4. 五大模块设计

### 4.1 M1: 依赖图引擎 (Correlator)

#### 职责

构建并维护集群内的服务依赖关系图 (DAG)，为风险传播和根因分析提供拓扑基础。

#### 数据来源

| 数据 | 构建的边 | AtlHyper 现状 |
|------|---------|--------------|
| K8s Pod spec | Pod → Node（nodeName） | ✅ 已有 |
| K8s Service spec | Service → Pod（selector） | ✅ 已有 |
| K8s Ingress spec | Ingress → Service（rules） | ✅ 已有 |
| Linkerd outbound (SLO Edge) | Service A → Service B（调用关系） | 🔄 SLO OTel 改造中 |
| Ingress SLO 指标 | 外部域名 → 内部 Service | 🔄 SLO OTel 改造中 |

#### 数据模型

```go
// 依赖图
type DependencyGraph struct {
    Nodes map[string]*GraphNode   // key = "namespace/type/name"
    Edges []*GraphEdge            // 有向边列表
    // 内部维护邻接表和反向邻接表，用于正向/反向遍历
}

// 图节点
type GraphNode struct {
    Key       string            // "default/service/api-server"
    Type      string            // "ingress" | "service" | "pod" | "node"
    Namespace string
    Name      string
    Metadata  map[string]string // 附加信息（如 Pod 的 image、Node 的 capacity）
}

// 图边
type GraphEdge struct {
    From   string  // source node key
    To     string  // target node key
    Type   string  // "routes_to" | "calls" | "runs_on" | "selects"
    Weight float64 // 边权重（可基于调用频率/错误率动态调整）
}
```

#### DAG 层级

AtlHyper 的 K8s 拓扑天然形成 DAG（无环有向图）：

```
Layer 0 (外部入口):  Ingress / 域名
      │ routes_to
Layer 1 (服务层):    Service
      │ selects        │ calls (Linkerd outbound)
Layer 2 (实例层):    Pod ←──── Service (其他服务调用)
      │ runs_on
Layer 3 (基础设施):  Node
```

- 无需额外处理环路（K8s 拓扑不存在环路）
- Service 间的 Linkerd 调用关系可能有环（A↔B 互调），但风险传播使用反向拓扑排序，环路节点取迭代收敛值

#### 刷新策略

- 图结构随 Agent 快照更新（30s），但拓扑变更不频繁
- 使用 diff 更新：只处理新增/删除的节点和边，避免全量重建
- 边权重（调用频率/错误率）可实时更新

#### API

```
GET /api/v2/aiops/graph?cluster={id}
  → 返回完整依赖图（节点 + 边）

GET /api/v2/aiops/graph/trace?from={entity_key}&direction=upstream|downstream
  → 从指定实体出发，返回上游或下游链路
```

---

### 4.2 M2: 基线引擎 (Baseline Engine)

#### 职责

为每个实体的每个指标建立"正常"基线，检测偏离，输出归一化的异常分数。

#### 算法清单

| 优先级 | 算法 | 用途 | 适用场景 |
|--------|------|------|---------|
| **P0** | EMA + 3σ | 简单异常检测 | 错误率突增、延迟飙升 |
| **P0** | 滑动窗口分位数 | P99/P95 基线 | SLO 延迟分布 |
| **P1** | CUSUM / Page-Hinkley | 趋势变化点检测 | 渐进式性能退化 |
| **P1** | Holt-Winters 三次指数平滑 | 周期性模式识别 | 日/周流量规律 |
| **P2** | 线性回归 | 趋势外推预测 | 资源耗尽预测（磁盘/内存） |

#### EMA + 3σ 详细说明（P0 首选算法）

```
指数移动平均 (EMA):
  EMA_t = α × x_t + (1-α) × EMA_{t-1}
  α = 2 / (N+1), N = 窗口大小（如 60 个采样点 = 30 分钟）

指数移动标准差:
  σ_t = sqrt(α × (x_t - EMA_t)² + (1-α) × σ²_{t-1})

异常判定:
  deviation = |x_t - EMA_t| / σ_t
  if deviation > 3: 异常（3σ 规则）

归一化到 [0, 1]:
  anomaly_score = sigmoid(deviation - threshold)
  = 1 / (1 + exp(-k × (deviation - 3)))
  k = 2 (控制 sigmoid 斜率)
```

#### 基线数据持久化

```go
// 每个实体-指标对的基线状态
type BaselineState struct {
    EntityKey  string  // "default/service/api-server"
    MetricName string  // "error_rate" | "p99_latency" | "request_rate"
    EMA        float64 // 当前 EMA 值
    Variance   float64 // 当前方差
    Count      int64   // 已处理的数据点数
    UpdatedAt  int64   // 最后更新时间戳
}
```

- 存入 SQLite，Agent 快照到达时更新
- 冷启动：前 100 个数据点（约 50 分钟）只学习不告警
- 滚动保留：7 天明细 + 30 天聚合

#### 输出

基线引擎的输出是风险评分引擎 (M3) 的输入：

```go
// 异常检测结果
type AnomalyResult struct {
    EntityKey    string
    MetricName   string
    CurrentValue float64
    Baseline     float64 // EMA
    Deviation    float64 // 偏离度（σ 倍数）
    Score        float64 // 归一化异常分数 [0, 1]
    IsAnomaly    bool    // deviation > 3
    DetectedAt   int64
}
```

---

### 4.3 M3: 风险评分引擎 (Risk Scorer)

#### 职责

综合各实体的异常分数，通过依赖图传播，计算最终风险评分和集群总体风险。

#### 三阶段计算流程

```
Stage 1: 局部风险 ──→ Stage 2: 时序权重 ──→ Stage 3: 图传播
(每个实体独立)        (因果排序)              (依赖关系传播)
```

##### Stage 1: 局部风险 (Local Risk Score)

每个实体独立计算，综合其所有指标的异常分数：

```
R_local(entity) = Σ(w_i × score_i)

其中 score_i 来自基线引擎的 AnomalyResult.Score
```

**权重分配**（按实体类型不同）：

| 实体类型 | 指标 | 权重 w_i |
|---------|------|---------|
| **Service** | 错误率偏离 | 0.40 |
| | P99 延迟偏离 | 0.30 |
| | 请求量异常 | 0.20 |
| | 关联 Pod 健康度 | 0.10 |
| **Pod** | 重启次数异常 | 0.35 |
| | 状态非 Running | 0.35 |
| | CPU/Memory 使用率 | 0.20 |
| | Ready 条件 | 0.10 |
| **Node** | Memory 压力 | 0.30 |
| | CPU 使用率 | 0.25 |
| | Disk 压力 | 0.25 |
| | Network 异常 | 0.10 |
| | PSI 指标 | 0.10 |
| **Ingress** | 错误率偏离 | 0.45 |
| | P99 延迟偏离 | 0.35 |
| | 请求量异常 | 0.20 |

##### Stage 2: 时序权重 (Temporal Weight)

用于因果排序——"哪个实体先出问题，更可能是根因"：

```
W_time(entity) = exp(-Δt / τ)

Δt = 当前时间 - 该实体首次检测到异常的时间
τ = 300s (5 分钟半衰期)

加权后的局部风险:
R_weighted(entity) = R_local(entity) × W_time(entity)
```

**效果**：
- 5 分钟前出异常的实体，时序权重 = 1.0（最高）
- 10 分钟前出异常的实体，时序权重 ≈ 0.37
- 20 分钟前出异常的实体，时序权重 ≈ 0.02（几乎忽略）

##### Stage 3: 图传播 (Graph Propagation)

按依赖图的反向拓扑排序遍历（从基础设施层向上传播）：

```
R_final(v) = α × R_weighted(v) + (1-α) × Σ(w_edge × R_final(u))

α = 0.6 (自身权重 60%，上游/下游传播 40%)
u ∈ dependencies(v) (v 所依赖的实体)
w_edge = 边权重（默认 1.0 / 出度，可按调用频率调整）
```

**传播方向**（反向拓扑排序）：
```
Node (先算) → Pod → Service → Ingress (后算)
```

**效果示例**：
```
Node worker-3 内存 95% → R_local = 0.9
  → Pod api-server-xyz (runs_on worker-3) → R_final 被传播提升
    → Service api-server (selects pod) → R_final 进一步提升
      → Ingress app.example.com (routes_to service) → R_final 最终传播
```

##### 聚合输出: ClusterRisk

```
ClusterRisk ∈ [0, 100]

ClusterRisk = w1 × max(R_final) × 100
            + w2 × SLO_burn_rate_factor
            + w3 × error_growth_rate_factor

w1 = 0.5 (最高风险实体的权重)
w2 = 0.3 (SLO 烧尽速率)
w3 = 0.2 (错误增长趋势)

SLO_burn_rate_factor:
  = 0    if burn_rate < 1x
  = 0.5  if 1x ≤ burn_rate < 2x
  = 1.0  if burn_rate ≥ 2x

error_growth_rate_factor:
  = sigmoid(error_growth - threshold)
```

#### API

```
GET /api/v2/aiops/risk/cluster?cluster={id}
  → { "risk": 72, "level": "warning", "topEntities": [...] }

GET /api/v2/aiops/risk/entities?cluster={id}&sort=r_final&limit=20
  → [{ "key": "default/service/api", "rLocal": 0.7, "rFinal": 0.85, ... }]

GET /api/v2/aiops/risk/entity/{key}?cluster={id}
  → { "key": "...", "metrics": [...], "propagation": [...], "causalChain": [...] }
```

---

### 4.4 M4: 状态机引擎 (State Machine)

#### 职责

管理集群和实体的事件生命周期，防止告警风暴，提供结构化的事件记录。

#### 状态转换图

```
              R_final > 0.5, 持续 > 2min
  Healthy ──────────────────────────────→ Warning
     ↑                                      │
     │                                      │ R_final > 0.8, 持续 > 5min
     │                                      │ 或 SLO burn_rate > 2x
     │            48h 无复发                 ▼
  Stable ←───────────────────── Recovery ← Incident
                                    ↑         │
                                    │         │ R_final < 0.3
                                    │         │ 持续 > 10min
                                    └─────────┘

  Recovery → Warning: R_final 再次 > 0.5 (复发)
```

#### 转换条件详表

| 转换 | 触发条件 | 动作 |
|------|---------|------|
| Healthy → Warning | R_final > 0.5 持续 > 2 分钟 | 创建 Incident (state=warning) |
| Warning → Incident | R_final > 0.8 持续 > 5 分钟，或 SLO burn_rate > 2x | 更新 Incident (state=incident, severity 升级) |
| Incident → Recovery | R_final < 0.3 持续 > 10 分钟 | 更新 Incident (state=recovery) |
| Recovery → Stable | 48 小时内 R_final 未再 > 0.5 | 更新 Incident (state=stable, resolved_at 写入) |
| Recovery → Warning | R_final 再次 > 0.5 | 更新 Incident (state=warning, 标记复发) |

#### 与现有告警的关系

```
现有告警系统（阈值触发）
     │
     │ 告警事件作为输入信号之一
     ▼
状态机引擎（管理事件生命周期）
     │
     │ Warning: 发送通知
     │ Incident: 升级通知 + 触发根因分析
     │ Recovery: 发送恢复通知
     ▼
事件存储（结构化记录）
```

- 现有告警 = Warning 阶段的触发器之一（但不是唯一来源，风险评分也可触发）
- 状态机增加了 Incident（确认升级）和 Recovery（恢复跟踪）两个现有系统缺少的阶段
- 告警抑制：同一实体在 Incident 状态期间，不重复发送 Warning 通知

#### 粒度

- 每个实体独立维护状态机实例
- 集群级别有一个聚合状态机（基于 ClusterRisk）

---

### 4.5 M5: 事件存储 (Incident Store)

#### 职责

结构化存储事件数据，支持历史查询、模式匹配、趋势统计。

#### 为什么用结构化存储而非 RAG

| 维度 | 结构化存储 (SQL) | RAG (向量检索) |
|------|-----------------|---------------|
| 查询精度 | 精确匹配（时间/实体/状态） | 语义相似（可能返回不相关结果） |
| 模式统计 | SQL 聚合（"过去 30 天 Service X 出现几次 P99 飙升"） | 不适合统计 |
| 存储效率 | 结构化数据压缩率高 | 向量嵌入占用空间大 |
| 依赖性 | SQLite（已有） | 需要向量数据库（新依赖） |
| 适用场景 | 已知结构的事件数据 | 非结构化文本搜索 |

**结论**：事件数据是结构化的（时间、实体、分数、状态），SQL 查询远比向量检索精确。AI 只在展示层消费结构化数据。

#### 数据库表设计

```sql
-- 事件主表
CREATE TABLE incidents (
    id            TEXT PRIMARY KEY,        -- UUID
    cluster_id    TEXT NOT NULL,
    state         TEXT NOT NULL,           -- "warning" | "incident" | "recovery" | "stable"
    severity      TEXT NOT NULL,           -- "low" | "medium" | "high" | "critical"
    root_cause    TEXT,                    -- 根因实体 key
    peak_risk     REAL,                   -- 峰值 ClusterRisk
    started_at    DATETIME NOT NULL,
    resolved_at   DATETIME,
    duration_s    INTEGER,
    recurrence    INTEGER DEFAULT 0,       -- 复发次数
    summary       TEXT,                    -- 算法生成的结构化摘要 (JSON)
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 受影响实体（多对多）
CREATE TABLE incident_entities (
    incident_id TEXT NOT NULL,
    entity_key  TEXT NOT NULL,             -- "default/pod/api-xxx"
    entity_type TEXT NOT NULL,             -- "service" | "pod" | "node" | "ingress"
    r_local     REAL,                      -- 局部风险分
    r_final     REAL,                      -- 传播后风险分
    role        TEXT NOT NULL,             -- "root_cause" | "affected" | "symptom"
    PRIMARY KEY (incident_id, entity_key),
    FOREIGN KEY (incident_id) REFERENCES incidents(id)
);

-- 事件时间线（事件内的关键时刻）
CREATE TABLE incident_timeline (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    incident_id TEXT NOT NULL,
    timestamp   DATETIME NOT NULL,
    event_type  TEXT NOT NULL,             -- "anomaly_detected" | "state_change" |
                                          -- "metric_spike" | "root_cause_identified" |
                                          -- "recovery_started"
    entity_key  TEXT,
    detail      TEXT,                      -- JSON (包含具体指标值、阈值等)
    FOREIGN KEY (incident_id) REFERENCES incidents(id)
);

-- 基线数据（持久化 EMA 状态）
CREATE TABLE baseline_states (
    entity_key  TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    ema         REAL NOT NULL,
    variance    REAL NOT NULL,
    count       INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    PRIMARY KEY (entity_key, metric_name)
);

-- 依赖图快照（定期持久化，用于重启恢复）
CREATE TABLE dependency_graph_snapshots (
    cluster_id TEXT NOT NULL,
    snapshot   BLOB NOT NULL,             -- gzip(JSON) 序列化的图
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (cluster_id)
);

-- 索引
CREATE INDEX idx_incidents_cluster_state ON incidents(cluster_id, state);
CREATE INDEX idx_incidents_started_at ON incidents(started_at);
CREATE INDEX idx_incident_entities_entity ON incident_entities(entity_key);
CREATE INDEX idx_incident_timeline_incident ON incident_timeline(incident_id, timestamp);
```

#### API

```
GET /api/v2/aiops/incidents?cluster={id}&state={state}&from={time}&to={time}
  → 事件列表（分页）

GET /api/v2/aiops/incidents/{id}
  → 事件详情 + 受影响实体 + 时间线

GET /api/v2/aiops/incidents/stats?cluster={id}&period=7d
  → 统计数据（事件数、MTTR、按 severity 分布、复发率）

GET /api/v2/aiops/incidents/patterns?entity={key}&period=30d
  → 指定实体的历史事件模式（用于 AI 层生成"类似事件"建议）
```

---

## 5. 数据资产与映射

### 5.1 现有数据 → AIOps 输入

| 数据源 | 采集方式 | AIOps 用途 | 当前状态 |
|--------|---------|-----------|---------|
| K8s Pod | Agent 快照 (30s) | 依赖图节点 + Pod 健康度 | ✅ 已有 |
| K8s Service | Agent 快照 | 依赖图节点 + selector 关联 | ✅ 已有 |
| K8s Node | Agent 快照 | 依赖图节点 + 基础设施层 | ✅ 已有 |
| K8s Ingress | Agent 快照 | 依赖图入口层 | ✅ 已有 |
| K8s Event | Agent 快照 | 时间线事件补充 | ✅ 已有 |
| Node CPU/Memory/Disk | OTel → Agent | Node 基线指标 | ✅ 已有 |
| Node Network/PSI/TCP | OTel → Agent | Node 基线指标 | ✅ 已有 |
| Service 错误率/延迟 | OTel Linkerd inbound | Service 基线 + SLO 烧尽率 | 🔄 SLO OTel 改造中 |
| Edge 调用关系 | OTel Linkerd outbound | 依赖图 Service 间边 | 🔄 SLO OTel 改造中 |
| Ingress 指标 | OTel Traefik/Nginx | Ingress 基线 + 入口映射 | 🔄 SLO OTel 改造中 |

### 5.2 数据缺口

| 缺口 | 影响 | 解决方案 |
|------|------|---------|
| SLO 指标未就绪 | 风险评分中 Service/Ingress 层无数据 | 等 SLO OTel 改造完成 |
| 无 APM/Trace 数据 | 无法做请求级别的溯源 | 当前不做请求级溯源，用指标级关联替代 |
| 无日志结构化索引 | 前端钻取到 Pod 后无法直接看日志 | Phase 3 可考虑关联 Pod 日志 API |

---

## 6. 前端能力规划

### 6.1 ClusterRisk 仪表盘

```
┌──────────────────────────────────────────┐
│  ClusterRisk                             │
│  ┌────┐  72 / 100  ██████████░░  Warning │
│  │ 🟡 │                                  │
│  └────┘  趋势: ↗ 上升中 (过去 30 分钟)   │
│                                          │
│  ─── 24h 趋势图 ────────────────────     │
│  |      _____                            │
│  |     /     \    /\                     │
│  |____/       \__/  \_______             │
│                                          │
│  风险最高实体 Top 5:                      │
│  1. default/service/api-server   0.85    │
│  2. default/pod/api-xxx-123      0.78    │
│  3. node/worker-3                0.72    │
│  4. default/service/db-proxy     0.65    │
│  5. ingress/app.example.com      0.61    │
└──────────────────────────────────────────┘
```

### 6.2 SLO 钻取链路

```
Ingress (app.example.com)
  SLO 达成率: 98.2% (目标 99.9%)
  错误率: 1.8%  |  P99: 850ms
      │
      ├─→ Service: frontend (正常)
      │     错误率: 0.1%  |  P99: 120ms
      │
      └─→ Service: api-server (异常 ⚠️)
            错误率: 3.2%  |  P99: 2100ms
                │
                ├─→ Pod: api-server-abc (异常 ⚠️)
                │     Restarts: 3  |  Memory: 95%
                │     Node: worker-3 (异常 ⚠️)
                │       Memory: 94%  |  Disk I/O: 87%
                │
                └─→ Pod: api-server-def (正常)
                      Node: worker-1 (正常)
```

### 6.3 事件时间线

```
事件 #INC-2025-0042
状态: Incident  |  严重度: High  |  持续: 23 分钟

根因: node/worker-3 内存压力 (94%)

时间线:
  14:02:15  [异常检测] Node worker-3 内存使用率超过基线 3.2σ
  14:03:45  [状态变更] Node worker-3: Healthy → Warning
  14:04:10  [指标飙升] Pod api-server-abc 内存达到 limit 的 95%
  14:05:22  [异常检测] Service api-server 错误率 3.2% (基线 0.3%)
  14:06:00  [根因识别] 根因链: Node(memory) → Pod(OOM) → Service(errors)
  14:08:15  [状态变更] 集群: Warning → Incident
  14:25:30  [恢复开始] Node worker-3 内存降至 72% (Pod 被驱逐后)

受影响实体: 3 个
  node/worker-3          (root_cause)  R=0.90
  default/pod/api-abc    (affected)    R=0.78
  default/service/api    (symptom)     R=0.85
```

### 6.4 拓扑图

- 力导向图或层级图，展示依赖关系
- 节点颜色映射 R_final（绿→黄→红）
- 边粗细映射调用频率
- 点击节点展开详情面板

---

## 7. 实现路线图

### 前置依赖

```
Phase 0: SLO OTel 改造 ← 当前进行中
  ├── Agent: 数据模型 → SDK → Repository → 集成 → E2E
  └── Master: 数据库 → Processor → Aggregator → API → E2E
  状态: 已设计完成，58 个任务待实现
  重要性: 这是所有 AIOps 功能的数据基础
```

### AIOps 分阶段实施

```
Phase 1: 依赖图 + 基线（纯后端，无前端）
  ├── M1: Correlator
  │   ├── 数据模型 (GraphNode/GraphEdge/DependencyGraph)
  │   ├── 图构建器 (从 K8s 快照提取拓扑)
  │   ├── 图更新器 (diff-based 增量更新)
  │   ├── 图查询 API
  │   └── 单元测试
  ├── M2: Baseline Engine
  │   ├── EMA + 3σ 异常检测器
  │   ├── 滑动窗口分位数计算器
  │   ├── 基线状态持久化 (SQLite)
  │   ├── 冷启动逻辑
  │   └── 单元测试
  └── 产出:
      ├── /api/v2/aiops/graph
      └── /api/v2/aiops/baseline/{entity}

Phase 2: 风险评分 + 状态机 + 事件存储（纯后端）
  ├── M3: Risk Scorer
  │   ├── 局部风险计算 (加权求和)
  │   ├── 时序权重 (指数衰减)
  │   ├── 图传播 (反向拓扑排序)
  │   ├── ClusterRisk 聚合
  │   └── 单元测试
  ├── M4: State Machine
  │   ├── 状态定义与转换规则
  │   ├── 事件触发器
  │   ├── 告警抑制逻辑
  │   └── 单元测试
  ├── M5: Incident Store
  │   ├── 数据库表创建 (SQLite migration)
  │   ├── Repository 实现
  │   ├── 事件查询/统计 API
  │   └── 单元测试
  └── 产出:
      ├── /api/v2/aiops/risk/cluster
      ├── /api/v2/aiops/risk/entities
      ├── /api/v2/aiops/incidents
      └── /api/v2/aiops/incidents/stats

Phase 3: 前端可视化
  ├── ClusterRisk 仪表盘页面
  │   ├── 大数字 + 趋势图
  │   ├── 风险实体 Top N 列表
  │   └── 风险级别指示器
  ├── SLO 钻取页面
  │   ├── 从 Ingress 到 Node 的级联展开
  │   ├── 每层展示关键指标
  │   └── 关联 Pod 详情/日志
  ├── 事件页面
  │   ├── 事件列表（过滤/排序）
  │   ├── 事件详情 + 时间线
  │   ├── 根因卡片
  │   └── 统计仪表盘（MTTR/复发率）
  ├── 拓扑图页面
  │   ├── 力导向图渲染
  │   ├── 节点风险着色
  │   └── 交互（点击展开/缩放/拖拽）
  └── i18n (zh + ja)

Phase 4: AI 增强层
  ├── 结构化事件 → 自然语言摘要
  ├── 根因分析 → 处置建议生成
  ├── 历史模式匹配 → "类似事件"推荐
  └── 集成到现有 AI Chat 模块
```

### 依赖关系

```
Phase 0 (SLO OTel)
  └─→ Phase 1 (依赖图 + 基线)  ← 需要 SLO 指标数据
       └─→ Phase 2 (风险评分 + 状态机 + 事件存储)  ← 需要基线输出
            ├─→ Phase 3 (前端)  ← 需要所有后端 API
            └─→ Phase 4 (AI)   ← 需要事件数据积累
```

注意：Phase 1 可以在 SLO OTel 改造完成后立即开始。在 SLO 数据未就绪期间，可以先用 K8s 资源状态和 Node 指标进行基线学习和依赖图构建（覆盖 Pod/Node 层），SLO 就绪后再补充 Service/Ingress 层。

---

## 8. 架构契合度分析

### 可直接复用的现有组件

| 现有组件 | AIOps 复用方式 |
|---------|---------------|
| Master-Agent 架构 | Agent 已采集 K8s + 指标，无需额外采集器 |
| 30s 快照周期 + Processor | AIOps 引擎挂载到 Processor 之后消费数据 |
| DataHub (内存快照存储) | 依赖图引擎读取最新 K8s/SLO 快照 |
| SQLite + Database 层 | 事件表、基线表直接扩展 |
| model_v2 共享模型 | 图节点类型对应已有 K8s 资源模型 |
| AI 模块 (Gemini) | Phase 4 复用现有 AI Chat 基础设施 |
| Gateway + Service 分层 | AIOps API 遵循现有 Handler → Service 模式 |
| 前端组件库 | 仪表盘、表格、Badge、Modal 等复用 |
| OTel Collector | 已部署，采集 Linkerd + Traefik 数据 |

### 需要新建的代码

| 新组件 | 位置 | 符合架构规范 |
|--------|------|------------|
| AIOps 引擎模块 | `atlhyper_master_v2/aiops/` | ✅ 新模块规则: interfaces.go + 子包实现 |
| AIOps 数据库扩展 | `database/sqlite/aiops_*.go` | ✅ 扩展现有 Database 层 |
| AIOps API Handler | `gateway/handler/aiops_*.go` | ✅ 遵循 Handler → Service 模式 |
| 前端 Risk 页面 | `app/monitoring/risk/` | ✅ 遵循路由规范 |
| 前端 Incidents 页面 | `app/monitoring/incidents/` | ✅ 遵循路由规范 |
| 前端 Topology 页面 | `app/monitoring/topology/` | ✅ 遵循路由规范 |

### AIOps 模块内部结构（预览）

```
atlhyper_master_v2/aiops/
├── interfaces.go           # 对外接口: AIOpsEngine
├── factory.go              # NewAIOpsEngine(...)
├── correlator/             # M1: 依赖图引擎
│   ├── graph.go            #   图数据结构 + 构建
│   ├── updater.go          #   增量更新
│   └── query.go            #   图查询（上下游遍历）
├── baseline/               # M2: 基线引擎
│   ├── detector.go         #   EMA + 3σ 异常检测
│   ├── quantile.go         #   滑动窗口分位数
│   └── state.go            #   基线状态管理
├── risk/                   # M3: 风险评分引擎
│   ├── scorer.go           #   三阶段评分流水线
│   ├── propagation.go      #   图传播算法
│   └── cluster_risk.go     #   ClusterRisk 聚合
├── statemachine/           # M4: 状态机引擎
│   ├── machine.go          #   状态定义 + 转换
│   └── trigger.go          #   事件触发器
└── incident/               # M5: 事件存储
    ├── store.go            #   事件 CRUD
    ├── timeline.go         #   时间线管理
    └── stats.go            #   统计查询
```

---

## 9. 差异化竞争分析

| 竞品 | 异常检测 | 根因分析 | 部署成本 | AtlHyper 差异 |
|------|---------|---------|---------|--------------|
| **Prometheus + Alertmanager** | 静态阈值 | 无 | 低 | 动态基线 + 风险传播 + 事件管理 |
| **Grafana ML** | ML 插件 | 无 | 中 | 检测→根因→事件全链路集成 |
| **K8s Dashboard / Lens** | 无 | 无 | 低 | 从监控升级为 AIOps |
| **Datadog** | ML 异常检测 | APM 溯源 | 高 (SaaS) | 开源 + 轻量 + 算法透明 |
| **Dynatrace Davis** | AI 全自动 | AI RCA | 高 (SaaS) | 算法可解释 + 开源 + 无厂商锁定 |
| **PagerDuty AIOps** | 事件聚合 | 有限 | 中 (SaaS) | 深度 K8s 原生 + SLO 钻取 |

### AtlHyper 的核心差异化

1. **算法可解释**——每个风险评分都能追溯到具体公式和输入指标，不是 ML 黑盒
2. **开源单二进制**——无 Kafka/ES/时序库依赖，Master+Agent 各一个二进制部署
3. **K8s 原生拓扑**——依赖图直接从 K8s API + Linkerd + OTel 构建，零额外配置
4. **SLO 驱动溯源**——从用户可感知的 SLO（域名错误率/延迟）出发，向下钻取到基础设施
5. **CJK 友好**——原生中文/日文 i18n，面向亚洲市场
6. **AI 增强而非 AI 依赖**——算法层独立工作，AI 层锦上添花

---

## 10. 风险与约束

### 技术风险

| 风险 | 影响 | 缓解方案 |
|------|------|---------|
| 参数调优 (α, τ, w_i) | 风险评分不准确 | 提供配置 API + 运行时可调；初期硬编码合理默认值 |
| 基线冷启动 | 初期 50 分钟内无法检测异常 | 降级为阈值检测，冷启动期间使用保守阈值 |
| 图传播环路 | Linkerd 服务互调形成环 | 迭代收敛（最多 10 轮）或打断弱边 |
| SQLite 写入瓶颈 | 大量事件并发写入 | 批量写入 + WAL 模式；事件量不大（分钟级），预期无瓶颈 |
| 内存图大小 | 超大集群（>1000 节点） | 按 namespace 分区；当前目标场景 < 500 节点 |

### 约束条件

| 约束 | 说明 |
|------|------|
| 无 APM/Trace | 不做请求级溯源，用指标级关联替代 |
| 无日志索引 | Phase 3 展示 Pod 日志需要额外的日志 API |
| 数据分辨率 30s | 异常检测最快响应 ~1 分钟（2 个采样点确认） |
| 单集群独立分析 | 多集群间不做跨集群关联（可未来扩展） |

---

## 11. 子设计文档索引

> 本文档是中心设计文档。后续实施时，每个 Phase 需要拆分为独立的子设计文档。

| Phase | 子设计文档 | 状态 |
|-------|-----------|------|
| Phase 0 | SLO OTel Agent 设计 (`archive/slo-otel-agent-design.md`) | ✅ 已完成 |
| Phase 0 | SLO OTel Master 设计 (`archive/slo-otel-master-design.md`) | ✅ 已完成 |
| Phase 1 | 依赖图引擎设计 (待创建) | 📋 待规划 |
| Phase 1 | 基线引擎设计 (待创建) | 📋 待规划 |
| Phase 2 | 风险评分引擎设计 (待创建) | 📋 待规划 |
| Phase 2 | 状态机 + 事件存储设计 (待创建) | 📋 待规划 |
| Phase 3 | 前端可视化设计 (待创建) | 📋 待规划 |
| Phase 4 | AI 增强层设计 (待创建) | 📋 待规划 |
