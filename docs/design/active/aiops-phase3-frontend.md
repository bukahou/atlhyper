# AIOps Phase 3 — 前端可视化

## 概要

实现 AIOps 的三个核心前端页面：**风险仪表盘**（ClusterRisk 概览 + 趋势图 + Top 实体列表）、**事件管理**（事件列表 + 详情/时间线 + 统计）、**拓扑图**（依赖关系力导向图 + 节点风险着色）。遵循「大后端小前端」原则，前端只做数据绑定和 UI 渲染。

**前置依赖**: Phase 2a + 2b（所有后端 API 已就绪）

**中心文档**: [`aiops-engine-design.md`](../future/aiops-engine-design.md) §6

---

## 1. 文件夹结构

```
atlhyper_web/src/
├── api/
│   └── aiops.ts                                       <- NEW  AIOps API 封装
│
├── app/monitoring/
│   ├── risk/                                          <- NEW  风险仪表盘
│   │   ├── page.tsx                                   <- NEW  页面入口（布局 + 状态编排）
│   │   └── components/
│   │       ├── RiskGauge.tsx                           <- NEW  风险仪表盘大数字
│   │       ├── TopEntities.tsx                         <- NEW  风险 Top N 实体列表
│   │       └── RiskTrendChart.tsx                      <- NEW  24h 风险趋势图
│   │
│   ├── incidents/                                     <- NEW  事件管理
│   │   ├── page.tsx                                   <- NEW  页面入口
│   │   └── components/
│   │       ├── IncidentList.tsx                        <- NEW  事件列表（过滤/排序）
│   │       ├── IncidentDetailModal.tsx                 <- NEW  事件详情弹窗
│   │       ├── TimelineView.tsx                        <- NEW  事件时间线
│   │       ├── RootCauseCard.tsx                       <- NEW  根因卡片
│   │       └── IncidentStats.tsx                       <- NEW  统计仪表盘
│   │
│   └── topology/                                      <- NEW  拓扑图
│       ├── page.tsx                                   <- NEW  页面入口
│       └── components/
│           ├── TopologyGraph.tsx                       <- NEW  力导向图
│           └── NodeDetail.tsx                          <- NEW  节点详情面板
│
├── components/aiops/                                  <- NEW  通用 AIOps 组件
│   ├── RiskBadge.tsx                                  <- NEW  风险等级徽章
│   └── EntityLink.tsx                                 <- NEW  实体跳转链接
│
├── i18n/
│   ├── locales/
│   │   ├── zh.ts                              (现有)  <- 修改: +aiops 翻译 (~60 个键)
│   │   └── ja.ts                              (现有)  <- 修改: +aiops 翻译 (~60 个键)
│   └── types/
│       └── i18n.ts                            (现有)  <- 修改: +AIOpsTranslations 接口
│
└── components/common/
    └── Sidebar.tsx                             (现有)  <- 修改: monitoring 分组新增 3 个菜单项
```

### 变更统计

| 操作 | 文件数 | 文件 |
|------|--------|------|
| **新建** | 17 | `api/aiops.ts` + `risk/` 下 4 个 + `incidents/` 下 6 个 + `topology/` 下 3 个 + `components/aiops/` 下 2 个 |
| **修改** | 4 | `zh.ts`, `ja.ts`, `types/i18n.ts`, `Sidebar.tsx` |

---

## 2. API 封装

### 2.1 api/aiops.ts

```typescript
// api/aiops.ts — AIOps API 封装
import request from './request'

// ==================== 类型定义 ====================

// 风险相关
export interface ClusterRisk {
  clusterId: string
  risk: number        // [0, 100]
  level: string       // "healthy" | "low" | "warning" | "critical"
  topEntities: EntityRisk[]
  totalEntities: number
  anomalyCount: number
  updatedAt: number
}

export interface EntityRisk {
  entityKey: string
  entityType: string  // "service" | "pod" | "node" | "ingress"
  namespace: string
  name: string
  rLocal: number
  wTime: number
  rWeighted: number
  rFinal: number
  riskLevel: string
  firstAnomaly: number
}

export interface EntityRiskDetail extends EntityRisk {
  metrics: AnomalyResult[]
  propagation: PropagationPath[]
  causalChain: CausalEntry[]
}

export interface AnomalyResult {
  entityKey: string
  metricName: string
  currentValue: number
  baseline: number
  deviation: number
  score: number
  isAnomaly: boolean
  detectedAt: number
}

export interface PropagationPath {
  from: string
  to: string
  edgeType: string
  contribution: number
}

export interface CausalEntry {
  entityKey: string
  metricName: string
  deviation: number
  detectedAt: number
}

// 依赖图相关
export interface DependencyGraph {
  clusterId: string
  nodes: Record<string, GraphNode>
  edges: GraphEdge[]
  updatedAt: string
}

export interface GraphNode {
  key: string
  type: string
  namespace: string
  name: string
  metadata: Record<string, string>
}

export interface GraphEdge {
  from: string
  to: string
  type: string
  weight: number
}

// 事件相关
export interface Incident {
  id: string
  clusterId: string
  state: string
  severity: string
  rootCause: string
  peakRisk: number
  startedAt: string
  resolvedAt: string | null
  durationS: number
  recurrence: number
  createdAt: string
}

export interface IncidentDetail extends Incident {
  entities: IncidentEntity[]
  timeline: IncidentTimeline[]
}

export interface IncidentEntity {
  incidentId: string
  entityKey: string
  entityType: string
  rLocal: number
  rFinal: number
  role: string
}

export interface IncidentTimeline {
  id: number
  incidentId: string
  timestamp: string
  eventType: string
  entityKey: string
  detail: string
}

export interface IncidentStats {
  totalIncidents: number
  activeIncidents: number
  mttr: number
  recurrenceRate: number
  bySeverity: Record<string, number>
  byState: Record<string, number>
  topRootCauses: { entityKey: string; count: number }[]
}

export interface IncidentPattern {
  entityKey: string
  patternCount: number
  avgDuration: number
  lastOccurrence: string
  commonMetrics: string[]
  incidents: Incident[]
}

// 查询参数
export interface IncidentListParams {
  cluster: string
  state?: string
  from?: string
  to?: string
  limit?: number
  offset?: number
}

// ==================== API 方法 ====================

// 风险
export function getClusterRisk(cluster: string) {
  return request.get<ClusterRisk>(`/api/v2/aiops/risk/cluster`, { params: { cluster } })
}

export function getClusterRiskTrend(cluster: string, period = '24h') {
  return request.get<{ timestamp: number; risk: number; level: string }[]>(
    `/api/v2/aiops/risk/cluster/trend`, { params: { cluster, period } }
  )
}

export function getEntityRisks(cluster: string, sort = 'r_final', limit = 20) {
  return request.get<EntityRisk[]>(`/api/v2/aiops/risk/entities`, {
    params: { cluster, sort, limit }
  })
}

export function getEntityRisk(cluster: string, entityKey: string) {
  return request.get<EntityRiskDetail>(
    `/api/v2/aiops/risk/entity/${encodeURIComponent(entityKey)}`,
    { params: { cluster } }
  )
}

// 依赖图
export function getGraph(cluster: string) {
  return request.get<DependencyGraph>(`/api/v2/aiops/graph`, { params: { cluster } })
}

export function getGraphTrace(cluster: string, from: string, direction = 'upstream', maxDepth = 10) {
  return request.get(`/api/v2/aiops/graph/trace`, {
    params: { cluster, from: encodeURIComponent(from), direction, max_depth: maxDepth }
  })
}

// 事件
export function getIncidents(params: IncidentListParams) {
  return request.get<Incident[]>(`/api/v2/aiops/incidents`, { params })
}

export function getIncidentDetail(id: string) {
  return request.get<IncidentDetail>(`/api/v2/aiops/incidents/${encodeURIComponent(id)}`)
}

export function getIncidentStats(cluster: string, period = '7d') {
  return request.get<IncidentStats>(`/api/v2/aiops/incidents/stats`, {
    params: { cluster, period }
  })
}

export function getIncidentPatterns(entity: string, period = '30d') {
  return request.get<IncidentPattern>(`/api/v2/aiops/incidents/patterns`, {
    params: { entity: encodeURIComponent(entity), period }
  })
}

// 基线
export function getBaseline(cluster: string, entity: string) {
  return request.get(`/api/v2/aiops/baseline`, {
    params: { cluster, entity: encodeURIComponent(entity) }
  })
}
```

---

## 3. 风险仪表盘页面

### 3.1 page.tsx

```
risk/page.tsx
  布局:
  ┌─────────────────────────────────────────────────────┐
  │  ┌──────────────┐  ┌──────────────────────────────┐ │
  │  │  RiskGauge    │  │  RiskTrendChart (24h)        │ │
  │  │  大数字+等级   │  │  折线图                      │ │
  │  └──────────────┘  └──────────────────────────────┘ │
  │                                                     │
  │  ┌──────────────────────────────────────────────────│
  │  │  TopEntities (表格，默认 Top 20)                 │ │
  │  │  列: 实体名 | 类型 | R_local | R_final | 等级   │ │
  │  │  点击行 → 展开 EntityRiskDetail                  │ │
  │  └──────────────────────────────────────────────────│
  └─────────────────────────────────────────────────────┘

  状态: 30s 自动轮询 (useEffect + setInterval)
  数据流: getClusterRisk() + getEntityRisks()
```

### 3.2 RiskGauge.tsx

```
组件: RiskGauge
  Props: { risk: number; level: string; anomalyCount: number; totalEntities: number }
  渲染:
    - 大数字: risk / 100 (加粗，颜色映射 level)
    - 进度条: 底色灰，填充色根据 level 变化
    - 标签: level 文字 + anomalyCount / totalEntities
    - 颜色映射:
      healthy  → 绿色 (#22c55e)
      low      → 蓝色 (#3b82f6)
      warning  → 黄色 (#eab308)
      critical → 红色 (#ef4444)
```

### 3.3 TopEntities.tsx

```
组件: TopEntities
  Props: { entities: EntityRisk[]; onSelect: (key: string) => void }
  渲染: 表格
    列:
      - 实体名 (EntityLink 组件, 点击跳转)
      - 类型 (RiskBadge 组件)
      - R_local (进度条)
      - R_final (进度条 + 数字)
      - 风险等级 (RiskBadge)
      - 首次异常时间 (相对时间: "5分钟前")
    排序: 默认按 R_final 降序
    操作: 点击行展开详情 → getEntityRisk() → 展示 metrics + causalChain
```

### 3.4 RiskTrendChart.tsx

```
组件: RiskTrendChart
  Props: { clusterId: string }
  说明: 展示 ClusterRisk 的 24 小时趋势
  数据流: getClusterRiskTrend(clusterId, '24h')
  实现:
    - 调用后端趋势 API 获取历史数据点
    - 使用 SVG 或 Canvas 绘制简单折线图
    - Y 轴: [0, 100], 背景色分区 (healthy/warning/critical)
    - X 轴: 时间 (最近 24h)
    - 30s 自动轮询刷新（与 page.tsx 共用轮询周期）
  注意: 趋势数据由后端存储和返回，遵循「大后端小前端」原则
```

> **后端支持**: 需要在 Phase 2a 的 `risk/cluster_risk.go` 中增加趋势数据存储
> （每次 OnSnapshot 时将 ClusterRisk 值追加到环形缓冲区或 SQLite，保留 24h 数据点）。
> API: `GET /api/v2/aiops/risk/cluster/trend?cluster={id}&period=24h`
> 响应: `{ "data": [{"timestamp": 1737364200, "risk": 42.5, "level": "low"}, ...] }`

---

## 4. 事件管理页面

### 4.1 page.tsx

```
incidents/page.tsx
  布局:
  ┌─────────────────────────────────────────────────────┐
  │  IncidentStats (统计卡片行)                          │
  │  [ 总数 ] [ 活跃 ] [ MTTR ] [ 复发率 ]              │
  │                                                     │
  │  ┌────── 过滤栏 ────────────────────────────────── │
  │  │ 状态: [全部|warning|incident|recovery|stable]    │ │
  │  │ 时间: [最近7天|30天|自定义]                      │ │
  │  └──────────────────────────────────────────────── │
  │                                                     │
  │  IncidentList (事件列表表格)                         │
  │  点击行 → 打开 IncidentDetailModal                  │
  └─────────────────────────────────────────────────────┘

  数据流: getIncidentStats() + getIncidents()
```

### 4.2 IncidentList.tsx

```
组件: IncidentList
  Props: { incidents: Incident[]; onSelect: (id: string) => void }
  列:
    - ID (短格式: inc-xxxx)
    - 状态 (颜色标签)
    - 严重度 (颜色标签)
    - 根因实体 (EntityLink)
    - 峰值风险
    - 开始时间 (相对/绝对可切换)
    - 持续时间 (分钟)
    - 复发次数
  排序: 默认按 startedAt 降序
  分页: limit=20, offset 翻页
```

### 4.3 IncidentDetailModal.tsx

```
组件: IncidentDetailModal
  Props: { incidentId: string; open: boolean; onClose: () => void }
  数据流: getIncidentDetail(incidentId)
  布局:
  ┌─────────────────────────────────────────────────────┐
  │  事件 #INC-xxxx                            [关闭]  │
  │  状态: [Incident] 严重度: [High] 持续: 23分钟       │
  │                                                     │
  │  ┌── RootCauseCard ─────────────────────────────── │
  │  │ 根因: node/worker-3 — 内存使用率 94%             │ │
  │  │ R_final: 0.90                                   │ │
  │  └──────────────────────────────────────────────── │
  │                                                     │
  │  ┌── 受影响实体 ────────────────────────────────── │
  │  │ node/worker-3    root_cause  R=0.90             │ │
  │  │ pod/api-abc      affected    R=0.78             │ │
  │  │ service/api      symptom     R=0.85             │ │
  │  └──────────────────────────────────────────────── │
  │                                                     │
  │  ┌── TimelineView ─────────────────────────────── │
  │  │ 14:02 [异常检测] worker-3 内存 3.2σ             │ │
  │  │ 14:04 [状态变更] Healthy → Warning               │ │
  │  │ 14:05 [指标飙升] api-server 错误率 3.2%          │ │
  │  │ 14:06 [根因识别] 根因链确定                      │ │
  │  │ 14:08 [状态变更] Warning → Incident              │ │
  │  └──────────────────────────────────────────────── │
  └─────────────────────────────────────────────────────┘
```

### 4.4 TimelineView.tsx

```
组件: TimelineView
  Props: { timeline: IncidentTimeline[] }
  渲染: 垂直时间线
    每条:
      - 时间戳 (左侧)
      - 事件图标 (根据 eventType 映射)
      - 事件描述 (从 detail JSON 解析)
      - 相关实体 (EntityLink)
    事件图标映射:
      anomaly_detected → 感叹号 (黄)
      state_change → 箭头 (蓝)
      metric_spike → 上升箭头 (红)
      root_cause_identified → 靶心 (紫)
      recovery_started → 对勾 (绿)
      recurrence → 循环 (橙)
```

### 4.5 IncidentStats.tsx

```
组件: IncidentStats
  Props: { stats: IncidentStats }
  渲染: 4 个统计卡片
    1. 总事件数 (totalIncidents)
    2. 活跃事件数 (activeIncidents, 标红如果 > 0)
    3. 平均恢复时间 (mttr, 格式: "45分钟")
    4. 复发率 (recurrenceRate, 格式: "13.3%")
  可选: 按 severity 的饼图（如果 bySeverity 有数据）
```

---

## 5. 拓扑图页面

### 5.1 page.tsx

```
topology/page.tsx
  布局:
  ┌─────────────────────────────────────────────────────┐
  │  ┌────────────────────────────────┐ ┌────────────┐ │
  │  │                                │ │ NodeDetail  │ │
  │  │  TopologyGraph                 │ │ (选中节点   │ │
  │  │  (力导向图，占满左侧)           │ │  的详情)    │ │
  │  │                                │ │             │ │
  │  │  节点: 风险着色                 │ │ 指标列表    │ │
  │  │  边: 粗细映射调用频率           │ │ 异常状态    │ │
  │  │  点击: 选中节点                 │ │ 上下游链路  │ │
  │  │  缩放/拖拽: 交互               │ │             │ │
  │  │                                │ │             │ │
  │  └────────────────────────────────┘ └────────────┘ │
  └─────────────────────────────────────────────────────┘

  数据流: getGraph() + getEntityRisks() (合并风险着色)
```

### 5.2 TopologyGraph.tsx

```
组件: TopologyGraph
  Props: {
    graph: DependencyGraph;
    entityRisks: Record<string, EntityRisk>;
    selectedNode: string | null;
    onNodeSelect: (key: string) => void;
  }

  技术选型: @antv/g6 (推荐) 或 react-force-graph

  实现:
    - 节点渲染:
      - 形状: 按类型区分 (圆形=Service, 方形=Pod, 六边形=Node, 菱形=Ingress)
      - 大小: 固定或按重要性调整
      - 颜色: 根据 R_final 映射 (绿→黄→红 渐变)
        healthy  (#22c55e) → low (#3b82f6) → medium (#eab308)
        → high (#f97316) → critical (#ef4444)
      - 标签: name (namespace 作为副标签)

    - 边渲染:
      - 有向箭头
      - 颜色: 灰色 (正常) / 红色 (传播路径上有异常)
      - 粗细: 固定 (Phase 3 简化版)

    - 交互:
      - 点击节点: 选中，触发 onNodeSelect
      - 双击节点: 居中放大
      - 缩放: 鼠标滚轮
      - 拖拽: 调整布局
      - Hover: 显示 tooltip (entityKey + R_final)

    - 布局:
      - 层级布局 (dagre): Ingress 在上，Node 在下
      - 或力导向布局: 自动分组
```

### 5.3 NodeDetail.tsx

```
组件: NodeDetail
  Props: { entityKey: string; clusterId: string }
  数据流: getEntityRisk(clusterId, entityKey) + getBaseline(clusterId, entityKey)
  渲染:
    - 实体信息: key, type, namespace, name
    - 风险分数: R_local / R_final / 等级 (RiskBadge)
    - 指标列表:
      每个指标:
        - 名称 (metricName)
        - 当前值 / 基线值
        - 偏离度 (σ 倍数)
        - 异常状态 (是/否)
    - 因果链: CausalEntry[] (按时间排序)
    - 上下游链路: getGraphTrace() → 链路节点列表
```

---

## 6. 通用组件

### 6.1 RiskBadge.tsx

```
组件: RiskBadge
  Props: { level: string; size?: 'sm' | 'md' }
  渲染: 彩色标签
    healthy  → 绿色背景, 白色文字
    low      → 蓝色背景
    medium   → 黄色背景, 暗色文字
    high     → 橙色背景, 白色文字
    critical → 红色背景, 白色文字
  i18n: t.aiops.riskLevel.{level}
```

### 6.2 EntityLink.tsx

```
组件: EntityLink
  Props: { entityKey: string; showType?: boolean }
  渲染: 可点击的链接
    解析 entityKey: "namespace/type/name"
    显示: [type图标] name
    点击行为:
      - service → /cluster/services?name={name}&namespace={namespace}
      - pod → /cluster/pods?name={name}&namespace={namespace}
      - node → /cluster/nodes?name={name}
      - ingress → /cluster/ingresses?name={name}&namespace={namespace}
```

---

## 7. 侧边栏变更

```
// Sidebar.tsx — monitoring 分组新增 3 个菜单项

monitoring 分组:
  - 指标监控 (已有)
  - 日志查看 (已有)
  - ────────────────  (分隔线)
  - 风险仪表盘    → /monitoring/risk        ← NEW
  - 事件管理      → /monitoring/incidents   ← NEW
  - 拓扑图        → /monitoring/topology    ← NEW
```

---

## 8. 国际化 (i18n)

### 8.1 类型定义

```typescript
// types/i18n.ts — 新增

export interface AIOpsTranslations {
  // 页面标题
  riskDashboard: string
  incidents: string
  topology: string

  // 风险等级
  riskLevel: {
    healthy: string
    low: string
    medium: string
    high: string
    critical: string
  }

  // 风险仪表盘
  clusterRisk: string
  riskScore: string
  riskTrend: string
  topRiskEntities: string
  anomalyCount: string
  totalEntities: string

  // 实体风险
  entityKey: string
  entityType: string
  rLocal: string
  rFinal: string
  riskLevel_label: string
  firstAnomaly: string
  noAnomaly: string

  // 事件
  incidentId: string
  incidentState: string
  severity: string
  rootCause: string
  peakRisk: string
  startedAt: string
  resolvedAt: string
  duration: string
  recurrence: string
  affectedEntities: string
  timeline: string

  // 事件状态
  state: {
    warning: string
    incident: string
    recovery: string
    stable: string
  }

  // 严重度
  severityLevel: {
    low: string
    medium: string
    high: string
    critical: string
  }

  // 统计
  stats: {
    total: string
    active: string
    mttr: string
    recurrenceRate: string
    bySeverity: string
    topRootCauses: string
  }

  // 时间线事件类型
  timelineEvent: {
    anomaly_detected: string
    state_change: string
    metric_spike: string
    root_cause_identified: string
    recovery_started: string
    recurrence: string
  }

  // 拓扑图
  dependencyGraph: string
  nodeDetail: string
  upstream: string
  downstream: string
  causalChain: string

  // 通用
  noData: string
  loading: string
  autoRefresh: string
}

// Translations 接口中添加
export interface Translations {
  // ... 现有字段 ...
  aiops: AIOpsTranslations
}
```

### 8.2 中文翻译 (示例)

```typescript
// locales/zh.ts — aiops 部分
aiops: {
  riskDashboard: '风险仪表盘',
  incidents: '事件管理',
  topology: '拓扑图',
  riskLevel: {
    healthy: '健康',
    low: '低风险',
    medium: '中风险',
    high: '高风险',
    critical: '严重',
  },
  clusterRisk: '集群风险',
  riskScore: '风险分数',
  riskTrend: '风险趋势 (24h)',
  topRiskEntities: '高风险实体',
  anomalyCount: '异常实体数',
  totalEntities: '总实体数',
  // ... 其余约 60 个键
  state: {
    warning: '告警',
    incident: '事件',
    recovery: '恢复中',
    stable: '已稳定',
  },
  timelineEvent: {
    anomaly_detected: '异常检测',
    state_change: '状态变更',
    metric_spike: '指标飙升',
    root_cause_identified: '根因识别',
    recovery_started: '开始恢复',
    recurrence: '复发',
  },
  noData: '暂无数据',
  loading: '加载中...',
  autoRefresh: '自动刷新',
}
```

### 8.3 日文翻译 (示例)

```typescript
// locales/ja.ts — aiops 部分
aiops: {
  riskDashboard: 'リスクダッシュボード',
  incidents: 'インシデント管理',
  topology: 'トポロジー',
  riskLevel: {
    healthy: '正常',
    low: '低リスク',
    medium: '中リスク',
    high: '高リスク',
    critical: '重大',
  },
  // ... 对应键
}
```

---

## 9. 实现阶段

```
P1: API 封装 + 通用组件
  ├── api/aiops.ts — 全部 API 方法 + 类型定义
  ├── components/aiops/RiskBadge.tsx
  ├── components/aiops/EntityLink.tsx
  └── i18n 类型定义 + 中文/日文翻译

P2: 风险仪表盘
  ├── risk/page.tsx — 页面布局 + 轮询
  ├── RiskGauge.tsx — 大数字展示
  ├── TopEntities.tsx — 实体列表 + 详情展开
  └── RiskTrendChart.tsx — 趋势折线图

P3: 事件管理
  ├── incidents/page.tsx — 页面布局 + 过滤
  ├── IncidentList.tsx — 事件列表表格
  ├── IncidentDetailModal.tsx — 详情弹窗
  ├── TimelineView.tsx — 时间线
  ├── RootCauseCard.tsx — 根因卡片
  └── IncidentStats.tsx — 统计卡片

P4: 拓扑图
  ├── topology/page.tsx — 页面布局
  ├── TopologyGraph.tsx — 力导向图 (需引入 @antv/g6)
  ├── NodeDetail.tsx — 节点详情面板
  └── 安装依赖: npm install @antv/g6

P5: 集成
  ├── Sidebar.tsx — 新增 3 个菜单项
  └── 构建验证: next build
```

---

## 10. 文件变更清单

### 新建

| 文件 | 说明 |
|------|------|
| `api/aiops.ts` | AIOps 全部 API 封装 + 类型定义 |
| `app/monitoring/risk/page.tsx` | 风险仪表盘页面 |
| `app/monitoring/risk/components/RiskGauge.tsx` | 风险大数字组件 |
| `app/monitoring/risk/components/TopEntities.tsx` | 风险 Top N 表格 |
| `app/monitoring/risk/components/RiskTrendChart.tsx` | 24h 趋势图 |
| `app/monitoring/incidents/page.tsx` | 事件管理页面 |
| `app/monitoring/incidents/components/IncidentList.tsx` | 事件列表表格 |
| `app/monitoring/incidents/components/IncidentDetailModal.tsx` | 事件详情弹窗 |
| `app/monitoring/incidents/components/TimelineView.tsx` | 时间线组件 |
| `app/monitoring/incidents/components/RootCauseCard.tsx` | 根因卡片 |
| `app/monitoring/incidents/components/IncidentStats.tsx` | 统计卡片 |
| `app/monitoring/topology/page.tsx` | 拓扑图页面 |
| `app/monitoring/topology/components/TopologyGraph.tsx` | 力导向图 |
| `app/monitoring/topology/components/NodeDetail.tsx` | 节点详情面板 |
| `components/aiops/RiskBadge.tsx` | 风险等级徽章 |
| `components/aiops/EntityLink.tsx` | 实体跳转链接 |

### 修改

| 文件 | 变更 |
|------|------|
| `components/common/Sidebar.tsx` | monitoring 分组新增 3 个菜单项 |
| `i18n/types/i18n.ts` | +AIOpsTranslations 接口 |
| `i18n/locales/zh.ts` | +aiops 翻译 (~60 个键) |
| `i18n/locales/ja.ts` | +aiops 翻译 (~60 个键) |

### 依赖安装

| 包 | 用途 |
|----|------|
| `@antv/g6` | 拓扑图力导向图渲染 |

---

## 11. 测试计划

| 组件 | 测试类型 | 验证点 |
|------|---------|--------|
| `api/aiops.ts` | API 调用测试 | 请求路径正确、参数编码、响应类型匹配 |
| `RiskBadge` | 快照测试 | 各 level 渲染正确的颜色和文字 |
| `RiskGauge` | 快照测试 | 数字/进度条/颜色正确 |
| `TopEntities` | 交互测试 | 点击行触发 onSelect、排序切换 |
| `IncidentList` | 交互测试 | 过滤/排序/分页、点击打开详情 |
| `TimelineView` | 快照测试 | 时间线条目渲染、图标映射 |
| `TopologyGraph` | 集成测试 | 节点/边渲染、点击选中、缩放 |
| i18n | 完整性检查 | zh.ts 和 ja.ts 所有键一致 |

---

## 12. 验证命令

```bash
# 安装依赖
cd atlhyper_web && npm install @antv/g6

# 构建验证
npm run build

# 开发模式
npm run dev
# 访问:
#   /monitoring/risk      — 风险仪表盘
#   /monitoring/incidents — 事件管理
#   /monitoring/topology  — 拓扑图

# i18n 键一致性检查
# 验证 zh.ts 和 ja.ts 的 aiops 部分键完全一致
```

---

## 13. 阶段实施后评审规范

> **本阶段实施完成后，必须对后续阶段的设计文档进行重新评审。**

### 原因

每个阶段的实施可能导致代码结构、接口签名、数据模型与设计文档中的预期产生偏差。提前编写的设计文档基于「假设的代码状态」，而实际实施后的代码才是唯一真实状态。不经过评审就直接实施下一阶段，可能导致：

- 前端组件 Props 与实际后端 API 响应不匹配
- i18n 键结构或命名在实施中调整
- 路由路径或组件目录结构变更
- 后端 API 的实际响应格式与前端类型定义产生偏差

### 本阶段实施后需评审的文档

| 文档 | 重点评审内容 |
|------|-------------|
| `aiops-phase4-ai-enhancement.md` | `IncidentDetailModal.tsx` 实际组件结构（AI 分析按钮的集成位置）、`api/aiops.ts` 实际类型定义、`AIOpsTranslations` 实际 i18n 键结构 |

### 评审检查清单

- [ ] 设计文档中引用的组件 Props 与实际实现一致
- [ ] 设计文档中的文件路径与实际目录结构一致
- [ ] 设计文档中的 API 类型定义与实际 `api/aiops.ts` 一致
- [ ] 设计文档中的 i18n 键结构与实际 `types/i18n.ts` 一致
- [ ] 如有偏差，更新设计文档后再开始下一阶段实施
