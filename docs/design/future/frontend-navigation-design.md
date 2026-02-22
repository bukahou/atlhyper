# 前端导航栏重设计

> 设计目标：以「数据关联」为核心，观测 + K8s 资源 + AIOps 三位一体
> 数据关联参考：`docs/static/reference/data-correlation-reference.md`

---

## 现有问题

1. **数据孤岛** — K8s 资源、指标、Trace、Log 各自独立页面，无关联
2. **缺少服务视角** — 没有以「服务」为中心的 360° 视图
3. **导航分组不合理** — Node/Event 被拆出集群，SLO 藏在工作台
4. **AIOps 已实现但未纳入设计** — 风险仪表盘、事件管理、依赖拓扑已有完整实现

---

## 核心设计原则

### 原则 1：汇总 + 详情，每层都有

每个导航项既有**汇总列表**，也有**详情页面**。汇总看全貌，详情看一个。

### 原则 2：详情组件全局共享

同一个 Pod/Node/Service/Trace 详情，无论从哪个入口打开，内容一致。

### 原则 3：跨页面穿透导航

任何页面看到可点击的实体，都能直接打开对应详情或跳转到所属页面。

```
用户看到异常 → 点击实体 → 两种选择:
  ├── 抽屉详情 (Drawer): 不离开当前页面，侧边展开详情
  └── 跳转页面 (Navigate): 去到所属分类页面查看完整上下文
```

---

## 导航结构

```
┌──────────────────────────────────────────────────────┐
│  AtlHyper                                            │
├──────────────────────────────────────────────────────┤
│                                                      │
│  概览                            /overview           │
│                                                      │
│  ─── 观测 ─────────────────────────────────          │
│                                                      │
│  APM                             /observe/apm        │
│  日志                            /observe/logs       │
│  指标                            /observe/metrics    │
│  SLO                             /observe/slo        │
│                                                      │
│  ─── AIOps ────────────────────────────────          │
│                                                      │
│  风险仪表盘                      /aiops/risk         │
│  事件管理                        /aiops/incidents    │
│  依赖拓扑                        /aiops/topology     │
│  AI 对话                         /aiops/chat         │
│                                                      │
│  ─── 集群资源 ─────────────────────────────          │
│                                                      │
│     核心                                             │
│  Pod                             /cluster/pod        │
│  Node                            /cluster/node       │
│  Deployment                      /cluster/deployment │
│  Service                         /cluster/service    │
│  Namespace                       /cluster/namespace  │
│  Ingress                         /cluster/ingress    │
│  Event                           /cluster/event      │
│     工作负载                                         │
│  DaemonSet                       /cluster/daemonset  │
│  StatefulSet                     /cluster/statefulset│
│  Job                             /cluster/job        │
│  CronJob                         /cluster/cronjob    │
│     存储                                             │
│  PV                              /cluster/pv         │
│  PVC                             /cluster/pvc        │
│     策略                                             │
│  NetworkPolicy                   /cluster/netpol     │
│  ResourceQuota                   /cluster/quota      │
│  LimitRange                      /cluster/limit      │
│  ServiceAccount                  /cluster/sa         │
│                                                      │
│  ─── 设置 ─────────────────────────────────          │
│                                                      │
│  AI 配置                         /settings/ai        │
│  通知配置                        /settings/notif     │
│                                                      │
│  ─── 管理 ─────────────────────────────────          │
│                                                      │
│  用户管理                        /admin/users        │
│  角色权限                        /admin/roles        │
│  审计日志                        /admin/audit        │
│  命令历史                        /admin/commands     │
│                                                      │
└──────────────────────────────────────────────────────┘
```

### 与现有导航的对照

| 现有位置 | 新位置 | 变更原因 |
|---------|--------|---------|
| `/workbench/slo` | `/observe/slo` | SLO 是观测能力，不是工具 |
| `/monitoring/metrics` | `/observe/metrics` | 统一到观测分组 |
| `/monitoring/apm` | `/observe/apm` | 统一到观测分组 |
| `/monitoring/logs` | `/observe/logs` | 统一到观测分组 |
| `/monitoring/risk` | `/aiops/risk` | AIOps 独立分组 |
| `/monitoring/incidents` | `/aiops/incidents` | AIOps 独立分组 |
| `/monitoring/topology` | `/aiops/topology` | AIOps 独立分组 |
| `/workbench/ai` | `/aiops/chat` | AI 对话归入 AIOps，后续可调用聚合数据工具 |
| `/workbench/commands` | `/admin/commands` | 命令历史归入管理 |
| `/cluster/alert` | `/cluster/event` | 重命名为 Event（K8s 原生概念） |
| Node/Event 在集群中 | 保持在集群中 | Node/Event 本身就是 K8s 集群资源 |

---

## 页面层级：汇总 → 详情

### 概览 `/overview`

概览是**两层 AIOps 的聚合入口**，不是独立 widget 的拼盘。

#### 核心理念

用户最关心「我的业务是否正常」，而非单个指标数值。
概览以 **Namespace 级服务健康汇总**为第一视觉重心，集群级为辅。

#### 布局

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  ┌─ ① AIOps 健康评分 ───────────────────────────────────────┐  │
│  │                                                           │  │
│  │  服务 AIOps: 2 Namespace 异常 ⚠   集群 AIOps: 正常 ✓    │  │
│  │  (ClickHouse 多表聚合)             (Agent + ClickHouse)   │  │
│  │                                                           │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ ② Namespace 健康卡片 (主区域) ──────────────────────────┐  │
│  │                                                           │  │
│  │  ┌─ geass ──────────────────────────── 3/5 异常 ⚠ ──┐   │  │
│  │  │                                                    │   │  │
│  │  │  ✗ auth       ✗ favorites   ⚠ gateway             │   │  │
│  │  │  ⚠ history    ✓ media                              │   │  │
│  │  │                                                    │   │  │
│  │  │  聚合: RPS 37/s  成功率 93.4%  P99 3200ms         │   │  │
│  │  │  [展开详情]                                        │   │  │
│  │  └────────────────────────────────────────────────────┘   │  │
│  │                                                           │  │
│  │  ┌─ monitoring ─────────────────────── 全部正常 ✓ ───┐   │  │
│  │  │  ✓ prometheus   ✓ grafana   ✓ node-exporter       │   │  │
│  │  │  聚合: RPS 2/s  成功率 100%  P99 12ms             │   │  │
│  │  └────────────────────────────────────────────────────┘   │  │
│  │                                                           │  │
│  │  ┌─ kube-system ───────────────────── 全部正常 ✓ ───┐   │  │
│  │  │  ✓ coredns  ✓ traefik  ✓ linkerd  ... (8 服务)   │   │  │
│  │  └────────────────────────────────────────────────────┘   │  │
│  │                                                           │  │
│  │  每个服务用色块表示: ✓绿 ⚠黄 ✗红                       │  │
│  │  点击 Namespace 卡片 → 展开该 NS 下的服务列表           │  │
│  │  点击服务色块 → 服务详情 Drawer                          │  │
│  │                                                           │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ ③ 活跃问题 (异常时显示) ────────────────────────────────┐  │
│  │                                                           │  │
│  │  ⚠ geass-auth CrashLoopBackOff                           │  │
│  │    Pod 重启 5 次 → gateway 成功率降至 96.8% → 入口 502   │  │
│  │    [查看因果链] [查看关联链路]                            │  │
│  │                                                           │  │
│  │  ⚠ geass-history 延迟异常                                │  │
│  │    P99 30ms → 3200ms → 12 条慢查询 Trace                 │  │
│  │    [查看因果链] [查看关联链路]                            │  │
│  │                                                           │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ ④ 集群 AIOps 摘要 ──────────────────────────────────────┐  │
│  │                                                           │  │
│  │  3 节点 | 42 Pod | 17 Deployment                         │  │
│  │  raspi-nfs: Mem 92% ⚠    (其余正常)                      │  │
│  │  [查看详情 →]                                             │  │
│  │                                                           │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 展开 Namespace 卡片后

点击 geass 卡片的 `[展开详情]`，在卡片内展开服务列表：

```
┌─ geass ──────────────────────────────────── 3/5 异常 ⚠ ──┐
│                                                            │
│  服务名        健康度   Pod     成功率   P99     异常信号  │
│  ────────────────────────────────────────────────────      │
│  auth          0.15 ✗   0/1     --       --      CrashLoop│
│  favorites     0.21 ✗   0/1     --       --      OOM      │
│  gateway       0.42 ⚠   2/2     96.8%    87ms    502↑     │
│  history       0.38 ⚠   1/1     100%     3200ms  延迟↑    │
│  media         0.92 ✓   1/1     100%     26ms    --       │
│                                                            │
│  点击服务行 → 服务详情 Drawer                              │
│  点击 Pod 数 → /cluster/pod?namespace=geass&owner=xxx     │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

#### 区域说明

| # | 区域 | 说明 | 数据源 |
|---|------|------|--------|
| ① | AIOps 健康评分 | 服务 AIOps + 集群 AIOps 各一个评分 | AIOps 引擎聚合 |
| ② | Namespace 健康卡片 | 按 Namespace 分组，每个服务用色块表示健康度，可展开看服务列表 | Linkerd + Traces + Logs + Pod + Events |
| ③ | 活跃问题 | 异常时显示，每个问题是跨数据源的因果链 | AIOps 因果树 + 证据 |
| ④ | 集群 AIOps 摘要 | 紧凑的集群级汇总，仅显示异常项 | Agent + ClickHouse node_* |

#### 服务健康度计算（核心）

每个服务的健康度不再是单一数据源，而是 **ClickHouse 多表聚合**：

| 信号 | 来源 | 权重说明 |
|------|------|---------|
| 成功率 | Linkerd `otel_metrics_sum` (response_total, classification=success) | 最直接的业务指标 |
| 延迟 P99 | Linkerd `otel_metrics_histogram` (response_latency_ms) | 用户体验指标 |
| Error Trace 比例 | `otel_traces` WHERE StatusCode = 'ERROR' | 应用层错误 |
| Error 日志数 | `otel_logs` WHERE SeverityText IN ('ERROR', 'WARN') | 应用内部异常 |
| Pod 状态 | Agent 快照 (Pod.Status, Restarts, Conditions) | 运行时健康 |
| K8s Events | Agent 快照 (Warning/Critical Events for this workload) | 集群层异常信号 |
| 入口错误率 | Traefik `otel_metrics_sum` (非 2xx 占比) | 用户可见错误 |

**对比现有 AIOps**：

| | 现有 | 新架构 |
|--|------|--------|
| **监控粒度** | 集群整体 | 集群 + 每个服务 |
| **数据源** | Agent 采集 (K8s API) | Agent + ClickHouse (7 种数据源) |
| **能检测** | Node NotReady, Pod CrashLoop | + 成功率下降, 延迟飙升, 错误日志激增, Trace 异常 |
| **典型盲区** | geass 80% 瘫痪但集群评分正常 | 能检测到每个服务的异常并关联根因 |

---

### 观测 — APM `/observe/apm`

**已有实现**：`monitoring/apm/`（13 个组件），迁移路径即可。

APM 是观测的核心入口，包含服务拓扑、链路追踪、入口流量的聚合视图。

#### 汇总层：服务列表 + 拓扑图

```
┌─ 服务拓扑 (G6 力导向布局，已实现) ───────────────────────────┐
│                                                               │
│  [Traefik] → [geass-gateway] → [geass-auth]                 │
│                    │                                          │
│                    ├──→ [geass-history] → [MySQL]            │
│                    └──→ [geass-media]   → [MySQL]            │
│                    └──→ [geass-favorites]                    │
│                                                               │
│  节点上直接展示: RPS / 延迟 / 成功率颜色编码                  │
│  节点 icon/badge、选中态、展开/收起（已实现）                  │
└───────────────────────────────────────────────────────────────┘

┌─ 服务列表 ────────────────────────────────────────────────────┐
│  服务名        Namespace   RPS     成功率   P50    P99       │
│  geass-gateway geass       12.3/s  99.2%    5ms    87ms      │
│  geass-media   geass       8.1/s   100%     26ms   120ms     │
│  ...                                                          │
└───────────────────────────────────────────────────────────────┘
```

已有组件：ServiceList, ServiceOverview, ServiceTopology, ThroughputChart, LatencyChart, ErrorRateChart, LatencyDistribution, DependenciesTable, SpanTypeChart, MiniSparkline, ImpactBar

#### 详情层：点击服务 → 服务详情（已实现 ServiceOverview）

| 区域 | 已有组件 | 数据源 |
|------|---------|--------|
| 黄金指标 (RPS/延迟/成功率) | ThroughputChart, LatencyChart, ErrorRateChart | Linkerd |
| 延迟分布 | LatencyDistribution | Linkerd Histogram |
| Span 类型分布 | SpanTypeChart | Traces |
| 下游依赖 | DependenciesTable | Traces (ParentSpanId) |
| 最近链路 | TransactionsTable | Traces |
| 链路瀑布图 | TraceWaterfall | Traces (点击某条 Trace) |

**穿透扩展**（新增，原实现无）：
- 服务详情面板中增加: `[Pod 列表]` `[日志]` `[事件]` 区域
- 点击 Pod → Pod 详情 Drawer
- 点击 TraceId → Trace 详情 Drawer
- `[在日志中查看]` → `/observe/logs?service=xxx`
- `[在集群中查看]` → `/cluster/deployment?name=xxx`

---

### 观测 — 日志 `/observe/logs`

**现状**：占位页，待实现。数据源为 ClickHouse `otel_logs` 表。

#### 汇总层：日志搜索

| 筛选条件 | 字段 |
|---------|------|
| 服务 | ServiceName (ResourceAttributes) |
| 级别 | SeverityText (INFO/WARN/ERROR/DEBUG) |
| 来源类 | ScopeName |
| 关键词 | Body LIKE |
| TraceId | TraceId (从其他页面跳入时预填) |
| 时间范围 | Timestamp |

#### 详情层：展开日志行

点击日志行展开详情：完整 Body、所有 Attributes、关联 TraceId/SpanId。

**穿透链接**：
- TraceId → Trace 详情 Drawer 或 `/observe/apm` (展开该 Trace)
- ServiceName → 服务详情 Drawer
- host.name (Pod) → Pod 详情 Drawer

**URL 参数预筛选**（供其他页面跳入）：
- `/observe/logs?service=xxx` — 从 APM 服务详情跳入
- `/observe/logs?traceId=xxx` — 从 Trace 详情跳入
- `/observe/logs?pod=xxx` — 从 Pod 详情跳入

---

### 观测 — 指标 `/observe/metrics`

**已有实现**：`monitoring/metrics/`，完整的节点级指标页面。迁移路径即可。

#### 汇总层：集群指标总览

已有组件：ClusterOverviewChart（集群全局视图）

| 区域 | 已有组件 | 数据源 |
|------|---------|--------|
| CPU 使用率 | CPUCard | node_cpu |
| 内存使用率 | MemoryCard | node_memory |
| 磁盘使用率 | DiskCard | node_filesystem / node_disk |
| 网络流量 | NetworkCard | node_network |
| 温度 | TemperatureCard | node_hwmon |
| GPU | GPUCard | nvidia_smi |
| PSI 压力 | PSICard | node_pressure |
| TCP 连接 | TCPCard | node_netstat |
| 系统资源 | SystemResourcesCard | node_filefd / node_entropy |
| VMStat | VMStatCard | node_vmstat |
| 进程列表 | ProcessTable | container_* |
| 资源趋势图 | ResourceChart | 历史指标 |

#### 详情层：点击节点 → 节点指标详情

展开该节点的完整指标面板（已实现，节点选择后展示）。

**穿透扩展**：
- 节点名 → Node 详情 Drawer 或 `/cluster/node/:name`
- 进程/容器 → Pod 详情 Drawer

---

### 观测 — SLO `/observe/slo`

**已有实现**：`workbench/slo/`，迁移路径即可。

#### 汇总层：SLO 仪表盘

已有组件：SummaryCard（4 个统计卡片）, DomainCard

| 区域 | 已有组件 | 数据源 |
|------|---------|--------|
| 统计摘要 | SummaryCard | SLO 聚合 (服务数/达标率/总RPS) |
| 域名级 SLO 卡片 | DomainCard | Traefik Ingress 指标 |

#### 详情层：展开域名卡片 → Tab 切换

已有组件：OverviewTab, LatencyTab, MeshTab, CompareTab, SLOTargetModal

| Tab | 已有组件 | 内容 |
|-----|---------|------|
| 概览 | OverviewTab | 成功率、RPS、延迟趋势 |
| 延迟 | LatencyTab | P50/P90/P99 分位数 |
| 服务网格 | MeshTab | Linkerd mTLS 覆盖率/服务间调用 |
| 对比 | CompareTab | 多时间窗口对比 |

**穿透扩展**：
- 服务名 → 服务详情 Drawer 或 `/observe/apm` (选中该服务)
- 域名 → `/observe/apm` (入口流量视角)

---

### AIOps 架构升级：两层 AIOps

#### 现有 AIOps 的问题

现有 AIOps 只有集群级监控，数据源仅来自 Agent（K8s API Server）：

```
现有: 集群级 AIOps (单一监控体)

  数据源: K8s Pod 状态 + Events + Node 状态
  评估: 集群整体健康度
  盲区: geass namespace 80% 服务瘫痪 → 集群评分仍然 0.85 "健康"
        因为节点 Ready、大部分 Pod Running（系统 Pod 正常）
```

#### 新架构：服务 AIOps + 集群 AIOps

```
┌─────────────────────────────────────────────────────────┐
│  服务 AIOps（业务层）                    ← 用户最关心   │
│                                                         │
│  粒度: 每个服务 (Deployment/StatefulSet)                │
│  数据源: ClickHouse 数据湖                              │
│    Linkerd  → 成功率, RPS, 延迟 P99                    │
│    Traces   → Error Trace 比例, 慢查询                  │
│    Logs     → ERROR/WARN 日志数量, 异常模式             │
│    Traefik  → 入口错误率, 入口延迟                      │
│    Agent    → Pod 状态 (Restarts, OOM, CrashLoop)      │
│    Agent    → K8s Events (Warning for this workload)    │
│                                                         │
│  输出: 每个服务的健康度评分 + 异常原因                  │
├─────────────────────────────────────────────────────────┤
│  集群 AIOps（集群层）                  ← 现有能力扩展   │
│                                                         │
│  粒度: 每个节点                                         │
│  数据源:                                                │
│    Agent     → Node 状态, Pod 分布                      │
│    ClickHouse → node_* 指标 (CPU/Mem/Disk/Temp/PSI)    │
│    Agent     → K8s Events (Node 相关)                   │
│                                                         │
│  输出: 每个节点的健康度评分 + 资源瓶颈                  │
├─────────────────────────────────────────────────────────┤
│  关联层（跨层因果分析）                                 │
│                                                         │
│  节点内存不足 → Pod OOMKilled → 服务成功率下降          │
│  → 依赖服务延迟上升 → 入口 502 错误                    │
│                                                         │
│  现有因果树/图传播算法在此层工作，但数据源从            │
│  「仅 K8s 状态」扩展到「全量 ClickHouse 数据」         │
└─────────────────────────────────────────────────────────┘
```

---

### AIOps — 风险仪表盘 `/aiops/risk`

**已有实现**：`monitoring/risk/`（RiskGauge + TopEntities），需要**升级为双层视图**。

#### 布局

```
┌─ 服务风险 (主视图) ──────────────────────────────────────────┐
│                                                               │
│  服务名          健康度   成功率   P99     Pod    信号       │
│  ─────────────────────────────────────────────────────        │
│  geass-auth      0.15 ✗   --      --      Crash  5 异常信号 │
│  geass-favorites 0.21 ✗   --      --      OOM    3 异常信号 │
│  geass-gateway   0.42 ⚠   96.8%   87ms    2/2    2 异常信号 │
│  geass-history   0.38 ⚠   100%    3200ms  1/1    1 异常信号 │
│  geass-media     0.92 ✓   100%    26ms    1/1    0          │
│  linkerd-*       0.95 ✓   --      --      ok     0          │
│                                                               │
│  点击服务 → 展开该服务的异常信号明细:                        │
│                                                               │
│  ▼ geass-auth (0.15)                                         │
│    ⚠ Pod CrashLoopBackOff (Agent)                            │
│    ⚠ 5 次重启 in 10min (Events)                              │
│    ⚠ 32 条 ERROR 日志 (ClickHouse otel_logs)                │
│    ⚠ gateway 依赖此服务，成功率受影响 (Linkerd)             │
│    → [查看因果链] [打开 APM] [打开 Pod 详情]                │
│                                                               │
└───────────────────────────────────────────────────────────────┘

┌─ 集群风险 ────────────────────────────────────────────────────┐
│                                                               │
│  节点名        健康度   CPU    Mem     Disk    信号          │
│  ─────────────────────────────────────────────────            │
│  raspi-nfs     0.65 ⚠   23%    92%⚠   67%     1 异常信号   │
│  raspi-master  0.92 ✓   15%    45%    34%     0             │
│  raspi-worker  0.95 ✓   8%     38%    28%     0             │
│                                                               │
│  ▼ raspi-nfs (0.65)                                          │
│    ⚠ 内存使用率 92% 持续 2h (ClickHouse node_memory)        │
│    → 影响 Pod: geass-history-xxx (OOM 风险)                  │
│    → [查看节点详情] [查看指标趋势]                           │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

已有组件可复用：RiskGauge（改为服务级+节点级双仪表盘）, TopEntities（改为服务列表+节点列表）

---

### AIOps — 事件管理 `/aiops/incidents`

**已有实现**：`monitoring/incidents/`（5 个组件）。

升级点：事件不再只是 K8s 级别，还包含**服务级事件**（基于 ClickHouse 检测）。

#### 事件来源扩展

| 事件类型 | 来源 | 示例 |
|---------|------|------|
| Pod 异常（已有） | Agent K8s Events | CrashLoopBackOff, OOMKilled |
| 节点异常（已有） | Agent K8s Events | NodeNotReady, DiskPressure |
| **服务成功率下降（新）** | ClickHouse Linkerd | success_rate < 99% 持续 5min |
| **服务延迟飙升（新）** | ClickHouse Linkerd | P99 > 基线 3x 持续 5min |
| **Error Trace 激增（新）** | ClickHouse Traces | ERROR 比例 > 5% |
| **Error 日志激增（新）** | ClickHouse Logs | ERROR 数量 > 基线 5x |
| **入口错误率上升（新）** | ClickHouse Traefik | 非 2xx > 1% |

#### 详情层（在已有基础上扩展）

| 区域 | 已有组件 | 升级内容 |
|------|---------|---------|
| 事件详情 | IncidentDetailModal | 增加 ClickHouse 数据源的证据 |
| 时间线 | TimelineView | 同一时间线上展示多数据源的信号 |
| 根因分析 | RootCauseCard | 因果树节点附带具体数值证据 |

已有共享组件保留：CausalTreeNodeView, RiskBadge, EntityLink

**穿透链接**：
- EntityLink → 对应 K8s 资源详情 Drawer
- 根因实体 → `/aiops/topology?entity=xxx`
- 证据链路 → `/observe/apm?traceId=xxx`
- 证据日志 → `/observe/logs?service=xxx`

---

### AIOps — 依赖拓扑 `/aiops/topology`

**已有实现**：`monitoring/topology/`（TopologyGraph + NodeDetail）。

升级点：拓扑节点的风险着色从「K8s 状态」升级为「服务健康度」。

| 功能 | 现有 | 升级后 |
|------|------|--------|
| 节点数据 | K8s 实体（Pod/Deployment/Node） | + Service（业务服务） |
| 风险着色 | 基于 K8s Events 的 r_final | 基于多数据源的服务健康度 |
| 边的数据 | 依赖关系（静态） | + Linkerd 实际流量数据（RPS/延迟/成功率） |
| 视图模式 | service / anomaly / full | 保持不变 |

**详情层升级**（NodeDetail）：

```
点击拓扑中的服务节点 →

┌─ geass-gateway ──────────────────────────────────────┐
│                                                       │
│  健康度: 0.42 ⚠                                     │
│                                                       │
│  ┌─ 异常信号 ─────────────────────────────────────┐  │
│  │ ⚠ 成功率 96.8% (正常 99.9%)  来源: Linkerd    │  │
│  │ ⚠ 依赖 geass-auth 不可用     来源: Traces     │  │
│  └─────────────────────────────────────────────────┘  │
│                                                       │
│  ┌─ 黄金指标 ─────────────────────────────────────┐  │
│  │ RPS: 12/s  成功率: 96.8%  P99: 87ms            │  │
│  └─────────────────────────────────────────────────┘  │
│                                                       │
│  ┌─ 上下游 ───────────────────────────────────────┐  │
│  │ ← Traefik (入口)    12/s  99.8%                │  │
│  │ → geass-auth ✗       0/s   --    (不可用)      │  │
│  │ → geass-history ⚠    8/s   100%  P99: 3200ms   │  │
│  │ → geass-media ✓      5/s   100%  P99: 26ms     │  │
│  └─────────────────────────────────────────────────┘  │
│                                                       │
│  [打开 APM] [打开集群资源] [查看日志] [查看链路]     │
└───────────────────────────────────────────────────────┘
```

---

### AIOps — AI 对话 `/aiops/chat`

**已有实现**：`workbench/ai/`，迁移路径即可。

定位：从「通用聊天工具」升级为「智能运维助手」。

**当前能力**：
- 集群状态查询（自然语言 → K8s API）
- 命令执行建议

**后续扩展方向**（本次不实现，仅规划）：
- 调用聚合数据工具：查询 ClickHouse 中的 Traces/Logs/Metrics
- 关联 AIOps 引擎：获取服务健康度、事件根因分析
- 上下文感知：从当前页面携带实体信息进入对话
- 自然语言排障：「geass-gateway 为什么 502？」→ 自动查询因果链

---

### 集群资源 `/cluster/*`

#### 通用特性

所有集群资源页面：**列表页 + 详情页**。

列表页支持 URL 参数预筛选（供其他页面跳转时使用）：

| 参数 | 用途 | 示例 |
|------|------|------|
| `?namespace=geass` | 按 Namespace 筛选 | 从 Namespace 详情跳转 |
| `?node=raspi-nfs` | 按节点筛选 (Pod) | 从节点详情跳转 |
| `?owner=geass-gateway` | 按 Owner 筛选 (Pod) | 从 Deployment/服务详情跳转 |
| `?name=xxx` | 按名称筛选 | 从外部页面跳转 |

#### Pod `/cluster/pod`

**详情** `/cluster/pod/:namespace/:name`

```
┌─ Pod 详情 ─────────────────────────────────────────────────────────┐
│                                                                     │
│  基本信息 | 容器列表 | Spec | Conditions                           │
│  ← 纯 K8s 数据 (集群快照)                                         │
│                                                                     │
│  ┌─ 关联数据 Tab ──────────────────────────────────────────────┐   │
│  │                                                              │   │
│  │  [日志] [链路] [事件] [指标]                                 │   │
│  │                                                              │   │
│  │  日志 Tab:                                                   │   │
│  │    来源: otel_logs WHERE host.name = pod_name               │   │
│  │    穿透: 点击 TraceId → Trace 详情                          │   │
│  │    跳转: [在日志页面中查看] → /observe/logs?pod=xxx         │   │
│  │                                                              │   │
│  │  链路 Tab:                                                   │   │
│  │    来源: otel_traces WHERE host.name = pod_name             │   │
│  │    穿透: 点击 TraceId → Trace 详情 Drawer                   │   │
│  │    跳转: [在链路页面中查看] → /observe/apm?service=xxx      │   │
│  │                                                              │   │
│  │  事件 Tab:                                                   │   │
│  │    来源: Events WHERE involvedObject = this pod             │   │
│  │    跳转: [在事件页面中查看] → /cluster/event?pod=xxx        │   │
│  │                                                              │   │
│  │  指标 Tab:                                                   │   │
│  │    来源: container_* WHERE pod = pod_name                   │   │
│  │    展示: CPU/内存用量趋势图                                  │   │
│  │                                                              │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  所属 Deployment: geass-gateway [打开 Deployment 详情]             │
│  所在节点: raspi-nfs [打开节点详情]                                 │
└─────────────────────────────────────────────────────────────────────┘
```

#### Node `/cluster/node`

**详情** `/cluster/node/:name`

```
┌─ 节点详情: raspi-nfs ─────────────────────────────────────────────┐
│                                                                    │
│  ┌─ 基本信息 ────────────────────────────────────────────────┐    │
│  │ 状态: Ready   角色: worker   OS: Ubuntu 22.04             │    │
│  │ IP: 192.168.0.46   Kubelet: v1.28.2   Runtime: containerd │    │
│  │ CPU: 6 cores   Memory: 16Gi   Pods: 23/110               │    │
│  └────────────────────────────────────────────────────────────┘    │
│                                                                    │
│  ┌─ 资源使用趋势 (时间序列图) ────────────────────────────────┐   │
│  │  CPU 使用率  |  内存使用率  |  磁盘 I/O  |  网络流量       │   │
│  └────────────────────────────────────────────────────────────┘    │
│                                                                    │
│  ┌─ 节点上的 Pod ──────────────── [在 Pod 中筛选此节点] ─────┐   │
│  │ geass-gateway-xxx    geass   Running  CPU: 12%  Mem: 128Mi │   │
│  │ geass-auth-xxx       geass   Running  CPU: 5%   Mem: 96Mi  │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  ┌─ 节点事件 ────────────────────── [在事件中筛选此节点] ────┐   │
│  │ (无 Warning 事件)                                          │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  ┌─ 扩展指标 ─────────────────────────────────────────────────┐   │
│  │  温度 | PSI 压力 | TCP 连接 | 文件描述符 | 系统负载         │   │
│  └─────────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────────┘
```

**穿透链接**：
- Pod → Pod 详情 Drawer
- `[在 Pod 中筛选此节点]` → `/cluster/pod?node=raspi-nfs`
- `[在事件中筛选此节点]` → `/cluster/event?node=raspi-nfs`
- `[在指标中查看]` → `/observe/metrics` (选中该节点)

#### Event `/cluster/event`

**列表页**：事件时间线

| 筛选条件 | 字段 |
|---------|------|
| 类型 | Normal / Warning |
| 严重程度 | info / warning / critical (由 IsCritical() 判断) |
| 资源类型 | involvedObject.kind |
| Namespace | involvedObject.namespace |
| 时间范围 | lastTimestamp |

**详情层**：点击事件行内展开，显示完整 Message + 关联资源可点击链接：
- `involvedObject: Pod/geass/geass-auth-xxx` → [打开 Pod 详情]
- `involvedObject: Node/raspi-nfs` → [打开节点详情]

**URL 参数预筛选**：
- `/cluster/event?node=xxx` — 从节点详情跳入
- `/cluster/event?pod=xxx` — 从 Pod 详情跳入
- `/cluster/event?resource=kind/ns/name` — 通用资源筛选

#### Deployment `/cluster/deployment`

**详情** `/cluster/deployment/:namespace/:name`

| 区域 | 数据源 | 穿透 |
|------|--------|------|
| 基本信息/Spec/Status | 集群快照 | — |
| 黄金指标 (RPS/延迟/成功率) | Linkerd | → `/observe/apm?service=xxx` |
| 关联 Pod 列表 | 集群快照 (OwnerName) | → Pod 详情 Drawer |
| 关联 ReplicaSet | 集群快照 | — |
| 关联 Event | Events | → `/cluster/event?resource=xxx` |
| 入口流量 (如有 Ingress 指向) | Traefik | → SLO 或 APM |

#### Service `/cluster/service`

**详情** `/cluster/service/:namespace/:name`

| 区域 | 数据源 | 穿透 |
|------|--------|------|
| 基本信息/Spec/Ports | 集群快照 | — |
| 后端 Pod (Endpoints) | 集群快照 | → Pod 详情 Drawer |
| Linkerd 流量指标 | Linkerd | → `/observe/apm?service=xxx` |
| 关联 Ingress | 集群快照 (backend 匹配) | → Ingress 详情 |

#### 其他资源（标准列表 + 详情，无特殊关联数据）

- **Namespace** `/cluster/namespace` — 详情页展示该 namespace 下的资源统计 + 快捷跳转
- **Ingress** `/cluster/ingress` — 详情页展示路由规则 + 关联 Traefik 流量数据
- **StatefulSet** `/cluster/statefulset` — 详情类似 Deployment
- **DaemonSet** `/cluster/daemonset` — 详情类似 Deployment
- **Job** `/cluster/job` — 详情展示 Pod 列表 + 完成/失败状态
- **CronJob** `/cluster/cronjob` — 详情展示调度历史 + 关联 Job
- **PV/PVC** `/cluster/pv`, `/cluster/pvc` — 详情展示绑定关系 + 容量
- **NetworkPolicy** `/cluster/netpol` — 详情展示规则
- **ResourceQuota** `/cluster/quota` — 详情展示使用率
- **LimitRange** `/cluster/limit` — 详情展示限制项
- **ServiceAccount** `/cluster/sa` — 详情展示关联 Secret

---

## 页面关联拓扑

7 种数据不是线性的 A→B→C，而是蛛网——从任何页面都可以跳到多个关联页面。

### 关联拓扑图

```
                            ┌───────────┐
                    ┌──────→│  概览     │←─────┐
                    │       │ /overview │      │
                    │       └─────┬─────┘      │
                    │             │             │
                    │    NS卡片/活跃问题        │
                    │             │             │
              ┌─────┴─────┐      │      ┌──────┴──────┐
              │  AIOps    │←─────┼─────→│   APM       │
              │  风险     │      │      │ /observe/apm│
              │  事件     │      │      │  服务拓扑   │
              │  拓扑     │      │      │  链路追踪   │
              │  AI对话   │      │      │  入口流量   │
              └──┬──┬──┬──┘      │      └──┬──┬──┬────┘
    因果链/实体  │  │  │         │  服务名 │  │  │ TraceId
                 │  │  │         │         │  │  │
         ┌───────┘  │  └───┐    │    ┌────┘  │  └────┐
         │          │      │    │    │       │       │
         ▼          ▼      ▼    ▼    ▼       ▼       ▼
   ┌──────────┐ ┌──────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
   │  日志    │ │ SLO  │ │ 集群资源 │ │  指标    │ │ AIOps    │
   │ /observe │ │/obse-│ │ /cluster │ │ /observe │ │ /aiops   │
   │ /logs    │ │rve/  │ │ Pod/Node │ │ /metrics │ │ topology │
   │          │ │slo   │ │ Deploy/  │ │          │ │          │
   │          │ │      │ │ Svc/...  │ │          │ │          │
   └──┬──┬────┘ └──┬───┘ └┬──┬──┬──┘ └────┬─────┘ └──┬───────┘
      │  │         │      │  │  │          │          │
      │  │         │      │  │  │          │          │
      └──┴────┬────┴──────┴──┴──┴────┬─────┴──────────┘
              │                      │
              ▼                      ▼
        任意实体均可              任意实体均可
        跳到任意关联页面          打开 Drawer 详情
```

### 每个页面的出站链接

任何页面看到的实体都是**可点击**的，点击后根据实体类型跳转到对应页面。

#### APM `/observe/apm`

```
APM 中可见的实体            点击后去向
─────────────────           ───────────────────────────
服务名 (拓扑节点)     ───→  服务详情 Drawer
                      ───→  /cluster/deployment/:ns/:name
                      ───→  /observe/logs?service=xxx
                      ───→  /observe/slo (该服务的 SLO)
                      ───→  /aiops/topology?entity=xxx
Pod 名                ───→  Pod 详情 Drawer
                      ───→  /cluster/pod/:ns/:name
TraceId               ───→  Trace 详情 Drawer
                      ───→  /observe/logs?traceId=xxx
Span 中的 host.name   ───→  Pod 详情 Drawer
Span 中的 db.statement───→  /observe/logs?service=xxx (关联日志)
上下游服务 (边)       ───→  APM 选中对端服务
                      ───→  /aiops/topology (查看依赖链)
```

#### 日志 `/observe/logs`

```
日志中可见的实体            点击后去向
─────────────────           ───────────────────────────
TraceId               ───→  Trace 详情 Drawer (APM 瀑布图)
SpanId                ───→  Trace 详情 Drawer (定位到该 Span)
ServiceName           ───→  /observe/apm?service=xxx
                      ───→  /aiops/risk (查看该服务风险)
host.name (Pod)       ───→  Pod 详情 Drawer
                      ───→  /cluster/pod/:ns/:name
SeverityText=ERROR    ───→  /aiops/incidents (是否已有关联事件)
```

#### 指标 `/observe/metrics`

```
指标中可见的实体            点击后去向
─────────────────           ───────────────────────────
节点名                ───→  Node 详情 Drawer
                      ───→  /cluster/node/:name
                      ───→  /cluster/pod?node=xxx
                      ───→  /cluster/event?node=xxx
容器/进程 (Pod)       ───→  Pod 详情 Drawer
                      ───→  /observe/logs?pod=xxx
                      ───→  /observe/apm?service=xxx (所属服务)
异常指标              ───→  /aiops/risk (是否已有风险评估)
```

#### SLO `/observe/slo`

```
SLO 中可见的实体            点击后去向
─────────────────           ───────────────────────────
域名/入口服务         ───→  /observe/apm?service=xxx
                      ───→  /cluster/ingress/:ns/:name
后端服务名            ───→  /observe/apm?service=xxx
                      ───→  /cluster/service/:ns/:name
成功率/延迟异常       ───→  /aiops/risk (查看该服务风险)
                      ───→  /observe/logs?service=xxx (查看错误日志)
```

#### AIOps (风险/事件/拓扑) `/aiops/*`

```
AIOps 中可见的实体          点击后去向
─────────────────           ───────────────────────────
风险实体 (服务)       ───→  /observe/apm?service=xxx
                      ───→  /aiops/topology?entity=xxx
                      ───→  /observe/logs?service=xxx
                      ───→  /cluster/deployment/:ns/:name
风险实体 (节点)       ───→  /cluster/node/:name
                      ───→  /observe/metrics (选中该节点)
因果链中的证据        ───→  各证据来源页面:
  Linkerd 成功率           /observe/apm?service=xxx
  Error Trace              Trace 详情 Drawer
  ERROR 日志               /observe/logs?service=xxx
  Pod OOMKilled            Pod 详情 Drawer
  Node 内存                /observe/metrics
  Traefik 502              /observe/slo
拓扑中的边 (调用关系) ───→  /observe/apm (查看该调用的链路)
```

#### 集群资源 `/cluster/*`

```
集群资源中可见的实体         点击后去向
─────────────────           ───────────────────────────
Pod
  → 所属 Deployment   ───→  /cluster/deployment/:ns/:name
  → 所在 Node         ───→  /cluster/node/:name
  → 关联日志          ───→  /observe/logs?pod=xxx
  → 关联链路          ───→  /observe/apm?service=xxx
  → 关联事件          ───→  /cluster/event?pod=xxx
  → 容器指标          ───→  /observe/metrics
  → AIOps 风险        ───→  /aiops/risk

Node
  → 节点上的 Pod      ───→  /cluster/pod?node=xxx
  → 节点事件          ───→  /cluster/event?node=xxx
  → 节点指标趋势      ───→  /observe/metrics
  → AIOps 风险        ───→  /aiops/risk

Deployment
  → 关联 Pod 列表     ───→  /cluster/pod?owner=xxx
  → 黄金指标          ───→  /observe/apm?service=xxx
  → 关联事件          ───→  /cluster/event?resource=xxx
  → SLO 状态          ───→  /observe/slo
  → AIOps 风险        ───→  /aiops/risk

Service
  → 后端 Pod          ───→  Pod 详情 Drawer
  → Linkerd 流量      ───→  /observe/apm?service=xxx
  → 关联 Ingress      ───→  /cluster/ingress/:ns/:name

Event
  → involvedObject    ───→  对应资源详情 Drawer (Pod/Node/Deployment/...)
  → 关联 AIOps 事件   ───→  /aiops/incidents
```

#### 概览 `/overview`

```
概览中可见的实体             点击后去向
─────────────────           ───────────────────────────
NS 卡片中的服务色块   ───→  服务详情 Drawer
                      ───→  /observe/apm?service=xxx
NS 卡片展开后的服务行 ───→  服务详情 Drawer
                      ───→  /cluster/pod?owner=xxx (Pod 数列)
活跃问题              ───→  /aiops/incidents (因果链)
                      ───→  问题中任意实体的对应页面
集群 AIOps 中的节点   ───→  Node 详情 Drawer
                      ───→  /observe/metrics
```

### 关联矩阵：页面 × 页面

从行页面可以跳到列页面（✓ = 有直接链接）：

```
从 ＼ 到    概览  APM   日志  指标  SLO   风险  事件  拓扑  Pod   Node  Deploy Svc  Event
─────────  ────  ────  ────  ────  ────  ────  ────  ────  ────  ────  ─────  ───  ─────
概览        -     ✓     -     -     -     ✓     ✓     -     ✓     ✓     -      -    -
APM         -     -     ✓     -     ✓     ✓     -     ✓     ✓     -     ✓      -    -
日志        -     ✓     -     -     -     ✓     ✓     -     ✓     -     -      -    -
指标        -     ✓     ✓     -     -     ✓     -     -     ✓     ✓     -      -    ✓
SLO         -     ✓     ✓     -     -     ✓     -     -     -     -     -      ✓    -
風险        -     ✓     ✓     ✓     -     -     ✓     ✓     ✓     ✓     ✓      -    -
事件        -     ✓     ✓     -     ✓     -     -     ✓     ✓     ✓     ✓      -    -
拓扑        -     ✓     ✓     -     -     -     -     -     -     -     ✓      -    -
Pod         -     ✓     ✓     ✓     -     ✓     -     -     -     ✓     ✓      -    ✓
Node        -     -     -     ✓     -     ✓     -     -     ✓     -     -      -    ✓
Deploy      -     ✓     -     -     ✓     ✓     -     -     ✓     -     -      -    ✓
Svc         -     ✓     -     -     -     -     -     -     ✓     -     -      -    -
Event       -     -     -     -     -     -     ✓     -     ✓     ✓     ✓      -    -
```

**关键洞察**：APM 和 Pod 是**连接度最高**的两个页面 — 几乎所有页面都能跳到它们。
这符合实际：APM 是业务视角的中心，Pod 是 K8s 视角的中心。

---

## 详情组件共享机制

以下详情组件在全站共享，从任何页面均可调用：

| 详情组件 | 触发方式 | 展示方式 | 完整页面路由 |
|---------|---------|---------|-------------|
| Pod 详情 | 点击 Pod 名 | Drawer | `/cluster/pod/:ns/:name` |
| Node 详情 | 点击 Node 名 | Drawer | `/cluster/node/:name` |
| Deployment 详情 | 点击 Deployment 名 | Drawer | `/cluster/deployment/:ns/:name` |
| Service 详情 | 点击 Service 名 | Drawer | `/cluster/service/:ns/:name` |
| 服务观测详情 | 点击服务（含指标） | Drawer | `/observe/apm?service=ns/name` |
| Trace 详情 | 点击 TraceId | Drawer | `/observe/apm` (展开该 Trace) |
| Ingress 详情 | 点击 Ingress 名 | Drawer | `/cluster/ingress/:ns/:name` |
| Event 详情 | 点击事件行 | 行内展开 | — |
| AIOps 实体详情 | 点击风险实体 | Drawer | `/aiops/topology?entity=xxx` |

**Drawer vs 页面**：
- Drawer = 侧边抽屉，快速查看，不离开当前上下文
- 页面 = 完整详情，有更多 Tab 和深度内容
- Drawer 右上角始终有 `[在完整页面中打开]` 链接

---

## 路由汇总

```
/overview                              概览仪表盘

── 观测 ──
/observe/apm                           APM (服务拓扑 + 链路追踪 + 入口流量)
/observe/apm?service=xxx               APM (预选某服务)
/observe/apm?traceId=xxx               APM (展开某条链路)
/observe/logs                          日志搜索
/observe/logs?service=xxx              日志 (预筛选某服务)
/observe/logs?traceId=xxx              日志 (预筛选某 Trace)
/observe/logs?pod=xxx                  日志 (预筛选某 Pod)
/observe/metrics                       指标 (节点级)
/observe/slo                           SLO 仪表盘

── AIOps ──
/aiops/risk                            风险仪表盘
/aiops/incidents                       事件管理
/aiops/topology                        依赖拓扑
/aiops/topology?entity=ns/name         依赖拓扑 (定位某实体)
/aiops/chat                            AI 对话 (智能运维助手)

── 集群资源 ──
/cluster/pod                           Pod 列表
/cluster/pod?namespace=xxx             Pod (预筛选 Namespace)
/cluster/pod?node=xxx                  Pod (预筛选节点)
/cluster/pod?owner=xxx                 Pod (预筛选 Owner)
/cluster/pod/:namespace/:name          Pod 详情
/cluster/node                          Node 列表
/cluster/node/:name                    Node 详情
/cluster/deployment                    Deployment 列表
/cluster/deployment/:namespace/:name   Deployment 详情
/cluster/service                       Service 列表
/cluster/service/:namespace/:name      Service 详情
/cluster/namespace                     Namespace 列表
/cluster/namespace/:name               Namespace 详情
/cluster/ingress                       Ingress 列表
/cluster/ingress/:namespace/:name      Ingress 详情
/cluster/event                         Event 列表
/cluster/event?node=xxx                Event (预筛选节点)
/cluster/event?pod=xxx                 Event (预筛选 Pod)
/cluster/event?resource=kind/ns/name   Event (预筛选资源)
/cluster/daemonset                     DaemonSet 列表
/cluster/daemonset/:namespace/:name    DaemonSet 详情
/cluster/statefulset                   StatefulSet 列表
/cluster/statefulset/:namespace/:name  StatefulSet 详情
/cluster/job                           Job 列表
/cluster/job/:namespace/:name          Job 详情
/cluster/cronjob                       CronJob 列表
/cluster/cronjob/:namespace/:name      CronJob 详情
/cluster/pv                            PV 列表
/cluster/pv/:name                      PV 详情
/cluster/pvc                           PVC 列表
/cluster/pvc/:namespace/:name          PVC 详情
/cluster/netpol                        NetworkPolicy 列表
/cluster/quota                         ResourceQuota 列表
/cluster/limit                         LimitRange 列表
/cluster/sa                            ServiceAccount 列表

── 设置 ──
/settings/ai                           AI 配置
/settings/notifications                通知配置

── 管理 ──
/admin/users                           用户管理
/admin/roles                           角色权限
/admin/audit                           审计日志
/admin/commands                        命令历史
```
