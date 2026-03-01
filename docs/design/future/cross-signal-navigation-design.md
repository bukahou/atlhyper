# 跨信号穿透式导航 — 全链路连查 + 层层下挖

> 状态：未来规划
> 创建：2026-02-27
> 更新：2026-02-27
> 前置：[K8s 元数据展示增强（Phase 0）](../active/k8s-metadata-display-design.md)

---

## 1. 目标

AtlHyper 五层可观测性架构（L1 SLO → L2 Mesh → L3 APM → L4 Logs → L5 Infra）当前各层独立运作。
本设计统一解决两个问题：

1. **数据关联建模**：五层信号之间的桥接键是什么？如何映射？
2. **UI 穿透体验**：用户如何从任意一层异常出发，层层下钻到问题根因？

理想排障路径：

```
L1 SLO 域名异常
  → L2 服务网格（哪条调用链路有问题）
  → L3 APM Trace（具体哪个 Span 耗时/报错）
  → L4 日志（报错 Stacktrace、业务日志）
  → L5 节点指标（CPU/Memory/Disk 是否瓶颈）
```

当前体验 vs 期望体验：

```
当前：用户手动在 4 个独立页面跳转，脑内关联信息
期望：统一概览 → 点击异常服务 → 多信号聚合详情 → 点击 Trace → Span 日志 → 节点指标
```

---

## 2. 当前状态

### 2.1 基础设施

OTel Operator + k8sattributes processor 已部署（2026-02-27），ClickHouse 中 Traces 和 Logs 的 `ResourceAttributes` 携带完整的 K8s 元数据。

**关键结论**：OTel Operator 自动从 Deployment name 派生 `service.name`（即 `OTEL_SERVICE_NAME=geass-gateway` 等于 K8s Deployment name `geass-gateway`）。因此 **SLO 的 `deployment` 标签直接等于 APM 的 `ServiceName`**，无需额外映射。

### 2.2 已实现的关联

| 方向 | 桥接键 | 实现方式 |
|------|--------|----------|
| Trace → Logs | TraceId + SpanId | SpanDrawer 内嵌日志，Logs Tab |
| Logs → Trace | TraceId | LogDetail 点击「查看 Trace」跳转 APM |
| Overview → SLO | 页面链接 | SloOverviewCard「查看详情」按钮 |
| **K8s 元数据展示** | ResourceAttributes | **Phase 0 处理（独立设计文档）** |

### 2.3 未实现的关联（本文档范围）

| 方向 | 困难 | 优先级 |
|------|------|--------|
| SLO → APM | 域名→服务名映射（已自动解决） | 高 |
| APM → SLO | 同上 | 中 |
| SLO → Logs | 需按服务名过滤 | 中 |
| Metrics → Logs | 需经 Node → Pod → Service | 低 |
| Cluster Pod/Node → Observe | 需 Pod → Service 映射 | 中 |
| Observe Landing Page | 不存在 | 高 |

---

## 3. 数据关联建模

### 3.1 SLO（L1/L2） ↔ APM（L3）

**问题**：SLO 基于 Linkerd 服务网格的 L7 指标（域名/路由级别），APM 基于 OTel Trace 的 Span。

| 维度 | SLO（网格层） | APM（追踪层） |
|------|--------------|---------------|
| 数据来源 | Linkerd Proxy（sidecar） | OTel SDK / auto-instrumentation |
| 粒度 | 域名 + 路由路径 | 服务名 + 操作名（Span） |
| 指标 | 可用性、P95 延迟、Error Rate | Span 耗时、状态码、异常 |
| 实体标识 | Ingress Host（域名） | ServiceName（OTel resource） |

**桥接路径**：域名 → K8s Ingress → K8s Service → OTel ServiceName → APM

**已确认**：`OTel service.name == K8s Deployment name`。
Linkerd 网格指标中的 `deployment` 标签也是 K8s Deployment name。

### 3.2 Metrics（L5） ↔ Logs/APM（L4/L3）

K8s 元数据同时存在于 Traces 和 Logs 的 `ResourceAttributes` 中：

```
Node CPU 异常
  → 查 otel_traces/otel_logs WHERE ResourceAttributes['k8s.node.name'] = 'desk-one'
  → 直接获取该节点上的 Traces / Logs
```

### 3.3 统一实体标识

所有信号最终映射到统一的实体层级：

```
Domain (SLO Ingress Host)
  └─ Service (K8s Deployment = OTel ServiceName = Linkerd deployment)
       └─ Pod (k8s.pod.name in ResourceAttributes)
            └─ Node (k8s.node.name in ResourceAttributes)
```

桥接方式：

| 关联方向 | 桥接键 | 数据源 |
|----------|--------|--------|
| Domain → Service | Ingress Rules | ClusterSnapshot |
| Service → Traces/Logs | `ServiceName` 列 | ClickHouse |
| Pod → Traces/Logs | `ResourceAttributes['k8s.pod.name']` | ClickHouse |
| Node → Traces/Logs | `ResourceAttributes['k8s.node.name']` | ClickHouse |
| Trace ↔ Log | `TraceId` + `SpanId` | ClickHouse |

---

## 4. 跨页面跳转设计

### 4.1 标准 URL 参数

所有 Observe 页面统一支持以下 URL 参数：

```typescript
interface ObserveNavParams {
  service?: string;      // 按服务过滤（= OTel ServiceName = K8s Deployment name）
  node?: string;         // 按节点过滤（k8s.node.name）
  traceId?: string;      // 按 TraceId 定位（已实现）
  spanId?: string;       // 按 SpanId 定位（已实现）
  startTime?: string;    // 时间窗口开始（ISO 8601）
  endTime?: string;      // 时间窗口结束
}
```

### 4.2 跳转路由表

| 来源 | 目标 | URL | 状态 |
|------|------|-----|------|
| SLO 域名 → APM | `/observe/apm?service={deployment}` | deployment 标签即 ServiceName | 待实现 |
| SLO 域名 → Logs | `/observe/logs?service={deployment}` | 同上 | 待实现 |
| APM Span → Logs | SpanDrawer 内嵌 + `/observe/logs?traceId=` | — | ✅ 已实现 |
| Logs → Trace | `/observe/apm?trace={traceId}` | — | ✅ 已实现 |
| APM Span → Pod | `/cluster?section=pod&name={podName}` | Phase 0 提供 podName | 待实现 |
| APM Span → Node | `/cluster?section=node&name={nodeName}` | Phase 0 提供 nodeName | 待实现 |
| Metrics Node → Logs | `/observe/logs?node={nodeName}` | 后端按 ResourceAttributes 过滤 | 待实现 |
| Pod 详情 → Logs | `/observe/logs?service={deploymentName}` | deployment = service | 待实现 |
| Pod 详情 → APM | `/observe/apm?service={deploymentName}` | 同上 | 待实现 |

### 4.3 时间窗口同步

跨页面跳转时，时间窗口必须一致：

- URL 参数携带 `startTime` 和 `endTime`
- 目标页面优先使用 URL 参数的时间范围
- 如无 URL 时间参数，使用页面默认时间范围（15min）

---

## 5. UI 设计

### 5.1 Observe Landing Page（统一入口）

新增 `/observe` 页面（`atlhyper_web/src/app/observe/page.tsx`）：

```
┌─────────────────────────────────────────────────────┐
│                  Observe 概览                        │
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │ 服务健康  │  │ SLO 合规  │  │ 错误率   │  汇总卡片│
│  │   12/15  │  │  95.2%   │  │  2.3%    │          │
│  └──────────┘  └──────────┘  └──────────┘          │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │ 服务列表（按风险排序）                        │   │
│  │ ┌─────────────────────────────────────────┐ │   │
│  │ │ geass-gateway   ⚠ P99↑ 320ms  ERR 2.1% │ │   │
│  │ │ [SLO] [APM] [Logs] [Metrics]            │ │   │
│  │ ├─────────────────────────────────────────┤ │   │
│  │ │ media-server    ✅ P99 45ms   ERR 0.1%  │ │   │
│  │ │ [SLO] [APM] [Logs] [Metrics]            │ │   │
│  │ └─────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │ 最近异常信号                                  │   │
│  │ • 14:23 geass-gateway ERROR 日志突增 (+340%) │   │
│  │ • 14:20 api.example.com SLO 可用性下降 98.1% │   │
│  │ • 14:18 desk-one 节点 CPU 使用率 92%         │   │
│  └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

数据来源：
- 服务列表：OTelSnapshot.APMServices（或 SLOServices）
- SLO 合规：OTelSnapshot.SLOWindows
- 最近异常：AIOps Engine 的高风险实体
- 节点指标：ClusterSnapshot.NodeMetrics

### 5.2 服务详情聚合视图

在 APM 服务详情级别增加其他信号面板（扩展现有页面，改动最小）：

```
服务详情视图（APM page, level="service-detail"）
├── 已有：Trace 列表 + 统计图表
├── 新增：SLO 迷你图表（该服务对应域名的可用性趋势）
├── 新增：最近 ERROR 日志列表（Top 10）
└── 新增：节点资源概览（该服务 Pod 所在节点的 CPU/Memory）
```

### 5.3 面包屑导航

在 Observe 子页面顶部增加导航面包屑，保持穿透链路可见：

```
Observe > SLO > api.example.com > APM > geass-gateway > Trace abc123
```

### 5.4 侧边栏更新

```
observe/
├── (landing)   ← 新增：点击 "observe" 直达概览
├── apm         ← 已有
├── logs        ← 已有
├── metrics     ← 已有
└── slo         ← 已有
```

---

## 6. 前置修复

### SLO 页面集群选择不一致

当前 SLO 页面自行调用 `getClusterList()` 获取第一个集群 ID，
不使用 `clusterStore` 的 `currentClusterId`。
这导致侧边栏切换集群时 SLO 页面不响应。

**修复**：SLO 页面改为使用 `useClusterStore()` 的 `currentClusterId`，
与 APM/Logs/Metrics 保持一致。

---

## 7. 实施路线

### Phase 1：跨页面跳转基础设施 + SLO → APM 连通

1. 修复 SLO 页面 clusterStore 一致性
2. 定义标准 URL 参数接口 `ObserveNavParams`
3. SLO 页面增加「查看 APM」「查看日志」跳转（Linkerd `deployment` 标签 = ServiceName，无需额外映射）
4. APM 页面支持 `?service=xxx` URL 参数过滤
5. Logs 页面支持 `?service=xxx` URL 参数过滤
6. Metrics 页面增加「查看日志」跳转

**改动**：4-6 个前端文件 + 1-2 个后端文件

### Phase 2：Observe Landing Page

1. 创建 `/observe/page.tsx`
2. 实现服务健康汇总卡片
3. 实现异常信号时间线
4. 侧边栏 observe 组增加 href
5. i18n 新增翻译

**改动**：3-4 个前端新建文件 + i18n

### Phase 3：服务详情聚合视图

1. APM 服务详情增加 SLO 迷你图表
2. APM 服务详情增加 ERROR 日志摘要
3. APM 服务详情增加节点资源概览

**改动**：2-3 个前端文件 + 可能需要后端聚合 API

### Phase 4：Cluster → Observe 连通

1. Pod 详情增加「查看日志」「查看 Trace」按钮（利用 Phase 0 的 K8s 元数据）
2. Node 详情增加「查看指标」「查看日志」按钮
3. 后端日志查询增加按 Node 过滤（`WHERE ResourceAttributes['k8s.node.name'] = ?`）

**改动**：2-4 个前端文件 + 1 个后端文件

### Phase 5：统一关联 API + 面包屑

1. 提供 `/api/v2/observe/correlate` 接口，输入任意实体，返回关联信号摘要
2. 实现面包屑导航组件

**改动**：2-3 个后端文件 + 1-2 个前端文件

---

## 8. 与其他设计的关系

| 设计文档 | 关系 |
|----------|------|
| [K8s 元数据展示（Phase 0）](../active/k8s-metadata-display-design.md) | 前置依赖，提供 K8s 字段给跳转使用 |
| 算法范围扩展 | Landing Page 的异常信号依赖 AIOps 引擎输出 |
| AI 工具增强 | AI 分析结果中的"查看详情"链接需要跳转到穿透式页面；AI 工具可调用 correlate API |
| OTel Operator 部署 | 已完成，提供 K8s 元数据注入能力 |

---

## 9. 开放问题

1. **SLO 是域名级别，APM 是服务级别，一个域名可能对应多个服务，如何展示？**
   - 方案：跳转后展示该域名关联的所有服务列表

2. **Metrics 按 Node 关联 Logs/APM 的粒度太粗，实用性如何？**
   - 现在 Traces/Logs 都有 `k8s.node.name`，可以直接按 Node 精确过滤，实用性大幅提升

3. **Landing Page 的服务列表数据来源：SLOServices 还是 APMServices？**
   - SLOServices 基于网格，APMServices 基于 OTel，覆盖范围可能不同
   - 方案：优先 APMServices，SLOServices 作为补充

---

## 10. 文件变更预估

| 模块 | 变更项 | 文件数 | Phase |
|------|--------|--------|-------|
| SLO 页面 | clusterStore 修复 + 跳转按钮 | 1 | 1 |
| APM / Logs 页面 | `?service=` 参数支持 | 2 | 1 |
| Metrics 页面 | 跳转按钮 | 1 | 1 |
| `app/observe/page.tsx` | **新建** Landing Page | 1 | 2 |
| `components/observe/` | **新建** ServiceHealthCard + AnomalyTimeline | 2 | 2 |
| Sidebar | observe 组增加 href | 1 | 2 |
| APM 服务详情 | 聚合面板（SLO + Logs + Metrics） | 1 | 3 |
| Master Handler | 聚合 API | 1-2 | 3 |
| Cluster Pod/Node 页面 | Observe 跳转按钮 | 2 | 4 |
| Agent Repository | Logs 按 node 过滤 | 1 | 4 |
| `Breadcrumb.tsx` | **新建** 面包屑组件 | 1 | 5 |
| Master Handler | correlate API | 1-2 | 5 |
| **合计** | | **~16-18** | |
