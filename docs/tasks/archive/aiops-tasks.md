# AIOps 引擎 — 已完成任务归档

> 从 `docs/tasks/active/tracker.md` 归档
> 中心文档: `docs/design/archive/aiops-engine-design.md`

---

## Phase 1: 依赖图引擎 + 基线引擎 — ✅ 完成

> commit: `8ff6fb2`, `927bc6e`

- 数据模型 + DB (baseline_states + dependency_graph_snapshots 表)
- 依赖图引擎 (4 种边关系: routes_to, calls, selects, runs_on)
- 基线引擎 (EMA + 3σ 异常检测 + sigmoid 归一化)
- Engine 编排 + master.go 集成
- 3 个 API 端点 (graph, graph/trace, baseline)
- 15 个测试通过

---

## Phase 2a: 风险评分引擎 — ✅ 完成

> commit: `927bc6e`

- 三阶段流水线 (R_local → R_weighted → R_final)
- ClusterRisk 聚合 [0,100]
- Engine 集成 + 结果缓存
- 3 个 API 端点 (risk/cluster, risk/entities, risk/entity)
- 19 个测试通过

---

## Phase 2b: 状态机引擎 + 事件存储 — ✅ 完成

> commit: `1366f60`

- 3 张 DB 表 (incidents, incident_entities, incident_timeline)
- 状态机引擎 (Healthy→Warning→Incident→Recovery→Stable + 复发检测)
- 事件存储 (CRUD + 统计聚合)
- Engine 集成 + TransitionCallback
- 4 个 API 端点 (incidents, incidents/{id}, incidents/stats, incidents/patterns)
- 11 个测试通过

---

## Phase 3: 前端可视化 — ✅ 完成

> commit: `c223180` (Mock 数据原型), `ee9e431` (@antv/g6 拓扑图)

- API 封装 + TypeScript 类型 (api/aiops.ts)
- 通用组件 (RiskBadge, EntityLink)
- 风险仪表盘 (RiskGauge + TopEntities + RiskTrendChart)
- 事件管理 (IncidentList + IncidentDetailModal + TimelineView + RootCauseCard + IncidentStats)
- 拓扑图 (@antv/g6 v5 力导向图)
- i18n (zh + ja, ~60 键)
- 3 个页面可正常访问和渲染

---

## Phase 4: AI 增强层 — ✅ 完成

> commit: `5b8fee1`

- Enhancer 服务 (LLMClientFactory + Summarize + JSON 响应解析 + 降级)
- Context Builder (结构化数据 → LLM 文本上下文)
- AIOps 专用 Prompt 模板
- 2 个 API 端点 (ai/summarize, ai/recommend, Operator 权限)
- RegisterTool 机制 (开闭原则, customTools map)
- 3 个 AIOps Tool (analyze_incident, get_cluster_risk, get_recent_incidents)
- 前端 AI 分析面板 (按钮 + 结果 + i18n 15 键)
- 12 个测试通过

---

## Phase 3 收尾: Mock → 真实 API — ✅ 完成

> commit: `42b69f7`

- api/aiops.ts 全部 8 个函数从 mock 切换为真实后端 API 调用

---

## 总计

- **后端**: 5 个 AIOps 子包 (correlator, baseline, risk, statemachine, ai)
- **DB**: 5 张新表 (baseline_states, dependency_graph_snapshots, incidents, incident_entities, incident_timeline)
- **API**: 12 个端点
- **前端**: 3 个页面 + AI 面板
- **测试**: 57 个 AIOps 测试全部通过
