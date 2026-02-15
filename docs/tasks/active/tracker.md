# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

---

## AIOps 引擎 — 🔄 进行中

> 中心文档: [aiops-engine-design.md](../../design/future/aiops-engine-design.md)
> 设计文档: [docs/design/active/](../../design/active/)

---

### Phase 1: 依赖图引擎 + 基线引擎 — 待办

> 设计文档: [aiops-phase1-graph-baseline.md](../../design/active/aiops-phase1-graph-baseline.md)
> 前置依赖: 无（SLO OTel 改造已完成）

**P1: 数据模型 + 数据库**

- [ ] `aiops/types.go` — GraphNode, GraphEdge, DependencyGraph, BaselineState, AnomalyResult 等类型定义
- [ ] `aiops/helpers.go` — EntityKey 生成函数
- [ ] `database/interfaces.go` — +AIOpsBaselineRepository, +AIOpsGraphRepository, +Dialect 接口
- [ ] `database/sqlite/migrations.go` — +aiops_baseline_states 表, +aiops_dependency_graph_snapshots 表
- [ ] `database/sqlite/aiops_baseline.go` — 基线 SQL Dialect 实现
- [ ] `database/sqlite/aiops_graph.go` — 依赖图 SQL Dialect 实现
- [ ] `database/repo/aiops_baseline.go` — 基线 Repository 实现
- [ ] `database/repo/aiops_graph.go` — 依赖图 Repository 实现
- [ ] 单元测试: Repository CRUD + BatchUpsert 验证

**P2: 依赖图引擎**

- [ ] `aiops/correlator/builder.go` — 从 ClusterSnapshot 构建 DAG（4 种边关系）
- [ ] `aiops/correlator/updater.go` — diff 增量更新（新增/删除节点和边）
- [ ] `aiops/correlator/query.go` — BFS 上下游遍历 + GetGraph
- [ ] `aiops/correlator/serializer.go` — gzip/JSON 序列化/反序列化
- [ ] 单元测试: 图构建正确性、增量更新、BFS 遍历（含环路）、序列化往返

**P3: 基线引擎**

- [ ] `aiops/baseline/detector.go` — EMA + 3σ 异常检测 + sigmoid 归一化
- [ ] `aiops/baseline/extractor.go` — 从 Store/SLO/NodeMetrics 提取指标数据点
- [ ] `aiops/baseline/state.go` — StateManager（内存缓存 + 定期 flush SQLite + 启动恢复）
- [ ] 单元测试: 冷启动（前 100 点无告警）、正常值不触发、3σ 异常触发、FlushToDB/LoadFromDB 一致性

**P4: 引擎编排 + 集成**

- [ ] `aiops/interfaces.go` — AIOpsEngine 对外接口定义
- [ ] `aiops/factory.go` — NewAIOpsEngine() 工厂函数
- [ ] `aiops/engine.go` — Engine 核心（OnSnapshot 编排 + Start/Stop 生命周期）
- [ ] `master.go` — AIOps 初始化 + OnSnapshotReceived 回调追加 + Start/Stop
- [ ] `config/types.go` — +AIOpsConfig（FlushInterval, BaselineAlpha, AnomalyThreshold）
- [ ] 集成测试: 快照到达 → 图自动更新 + 基线更新 + 重启恢复

**P5: API 层**

- [ ] `service/interfaces.go` — Query 接口 +GetAIOpsGraph, +GetAIOpsGraphTrace, +GetAIOpsBaseline
- [ ] `service/query/aiops.go` — 3 个查询方法实现（构造函数注入 aiopsEngine）
- [ ] `gateway/handler/aiops_graph.go` — GetGraph + Trace Handler
- [ ] `gateway/handler/aiops_baseline.go` — GetBaseline Handler
- [ ] `gateway/routes.go` — +3 路由（/api/v2/aiops/graph, /graph/trace, /baseline）
- [ ] API 端到端测试

**阶段完成后**

- [ ] `go build ./atlhyper_master_v2/...` 构建通过
- [ ] `go test ./atlhyper_master_v2/aiops/... -v` 全部通过
- [ ] 评审后续设计文档: Phase 2a, 2b, 3, 4（见设计文档 §15）

---

### Phase 2a: 风险评分引擎 — 待办

> 设计文档: [aiops-phase2-risk-scorer.md](../../design/active/aiops-phase2-risk-scorer.md)
> 前置依赖: Phase 1 ✅ 后开始

**P1: 三阶段流水线核心**

- [ ] `aiops/risk/scorer.go` — 三阶段流水线主逻辑（Calculate 入口）
- [ ] `aiops/risk/local.go` — Stage 1: 加权异常分数聚合 → R_local
- [ ] `aiops/risk/temporal.go` — Stage 2: 因果排序时序权重 → W_time
- [ ] `aiops/risk/propagation.go` — Stage 3: 依赖图反向拓扑传播 → R_final
- [ ] 单元测试: 各 Stage 独立测试 + 流水线端到端

**P2: ClusterRisk 聚合 + 配置**

- [ ] `aiops/risk/cluster_risk.go` — 集群风险聚合（Top 实体 + 等级映射）
- [ ] `aiops/risk/config.go` — 权重配置（指标权重、传播衰减等）
- [ ] 单元测试: 聚合计算、等级阈值

**P3: 引擎集成 + API**

- [ ] `aiops/interfaces.go` — +GetClusterRisk, +GetEntityRisks, +GetEntityRisk 方法
- [ ] `aiops/engine.go` — OnSnapshot 中调用 scorer.Calculate()
- [ ] `service/interfaces.go` — Query 接口 +3 风险查询方法
- [ ] `service/query/aiops.go` — +风险查询实现
- [ ] `gateway/handler/aiops_risk.go` — ClusterRisk + EntityRisks + EntityRisk Handler
- [ ] `gateway/routes.go` — +3 路由（/api/v2/aiops/risk/cluster, /entities, /entity/）
- [ ] API 测试

**阶段完成后**

- [ ] `go build` + `go test` 全部通过
- [ ] 评审后续设计文档: Phase 2b, 3, 4（见设计文档 §13）

---

### Phase 2b: 状态机引擎 + 事件存储 — 待办

> 设计文档: [aiops-phase2-statemachine-incident.md](../../design/active/aiops-phase2-statemachine-incident.md)
> 前置依赖: Phase 2a ✅ 后开始

**P1: 数据库 + Repository**

- [ ] `database/interfaces.go` — +AIOpsIncidentRepository + 模型（AIOpsIncident, Entity, Timeline, StatsRaw）
- [ ] `database/sqlite/migrations.go` — +aiops_incidents 表 + aiops_incident_entities 表（CASCADE） + aiops_incident_timeline 表（CASCADE） + 4 个索引
- [ ] `database/sqlite/aiops_incident.go` — 事件 SQL Dialect 实现
- [ ] `database/repo/aiops_incident.go` — 事件 Repository 实现（含 GetIncidentStats 聚合查询）
- [ ] 单元测试: CRUD + 统计聚合 + TopRootCauses

**P2: 状态机引擎**

- [ ] `aiops/statemachine/machine.go` — 状态定义 + TransitionCallback 接口 + 转换条件
- [ ] `aiops/statemachine/trigger.go` — Evaluate 评估 + transition 回调触发 + CheckRecoveryToStable
- [ ] `aiops/statemachine/suppressor.go` — ShouldSuppress + GetActiveIncidentID
- [ ] 单元测试: 5 条状态转换路径、持续时间不足不触发、告警抑制、Recovery→Stable（48h）

**P3: 事件存储**

- [ ] `aiops/incident/store.go` — Create / UpdateState / Resolve / UpdateRootCause / IncrementRecurrence
- [ ] `aiops/incident/timeline.go` — AddTimeline
- [ ] `aiops/incident/stats.go` — GetStats（使用 Repository 聚合方法 + 业务层计算复发率）
- [ ] `aiops/incident/patterns.go` — GetPatterns（历史模式匹配）
- [ ] 单元测试: 创建/更新/解决/统计/模式查询

**P4: 引擎集成 + API**

- [ ] `aiops/engine.go` — OnSnapshot +状态机评估 + Engine 实现 TransitionCallback + 启动定时检查 goroutine
- [ ] `aiops/interfaces.go` — +GetIncidents, +GetIncidentDetail 等事件查询方法
- [ ] `service/interfaces.go` — Query 接口 +4 事件查询方法
- [ ] `service/query/aiops.go` — +4 事件查询实现
- [ ] `gateway/handler/aiops_incident.go` — List / Detail / Stats / Patterns Handler
- [ ] `gateway/routes.go` — +4 路由（/api/v2/aiops/incidents, /{id}, /stats, /patterns）
- [ ] 端到端测试: 完整生命周期 Healthy→Warning→Incident→Recovery→Stable

**阶段完成后**

- [ ] `go build` + `go test` 全部通过
- [ ] 评审后续设计文档: Phase 3, 4（见设计文档 §14）

---

### Phase 3: 前端可视化 — 待办

> 设计文档: [aiops-phase3-frontend.md](../../design/active/aiops-phase3-frontend.md)
> 前置依赖: Phase 2a + 2b ✅ 后开始

**P1: API 封装 + 通用组件 + i18n**

- [ ] `api/aiops.ts` — 全部 API 方法 + TypeScript 类型定义（风险/图/事件/基线）
- [ ] `components/aiops/RiskBadge.tsx` — 风险等级徽章（颜色映射）
- [ ] `components/aiops/EntityLink.tsx` — 实体跳转链接（解析 entityKey → 路由）
- [ ] `types/i18n.ts` — +AIOpsTranslations 接口
- [ ] `locales/zh.ts` — +aiops 翻译（~60 个键）
- [ ] `locales/ja.ts` — +aiops 翻译（~60 个键）

**P2: 风险仪表盘**

- [ ] `app/monitoring/risk/page.tsx` — 页面布局 + 30s 轮询
- [ ] `risk/components/RiskGauge.tsx` — 大数字 + 进度条 + 颜色映射
- [ ] `risk/components/TopEntities.tsx` — Top N 表格 + 行展开详情
- [ ] `risk/components/RiskTrendChart.tsx` — 24h 趋势图（调用后端趋势 API）

**P3: 事件管理**

- [ ] `app/monitoring/incidents/page.tsx` — 页面布局 + 过滤栏
- [ ] `incidents/components/IncidentList.tsx` — 事件列表表格（过滤/排序/分页）
- [ ] `incidents/components/IncidentDetailModal.tsx` — 事件详情弹窗
- [ ] `incidents/components/TimelineView.tsx` — 垂直时间线 + 事件图标映射
- [ ] `incidents/components/RootCauseCard.tsx` — 根因卡片
- [ ] `incidents/components/IncidentStats.tsx` — 4 个统计卡片

**P4: 拓扑图**

- [ ] 安装依赖: `npm install @antv/g6`
- [ ] `app/monitoring/topology/page.tsx` — 页面布局（左图右详情）
- [ ] `topology/components/TopologyGraph.tsx` — 力导向图（节点风险着色 + 交互）
- [ ] `topology/components/NodeDetail.tsx` — 节点详情面板（指标/异常/上下游）

**P5: 集成**

- [ ] `components/common/Sidebar.tsx` — monitoring 分组 +风险仪表盘, +事件管理, +拓扑图
- [ ] `next build` 构建验证通过

**阶段完成后**

- [ ] `npm run build` 构建通过
- [ ] 3 个页面可正常访问和渲染
- [ ] i18n 中文/日文键一致性验证
- [ ] 评审后续设计文档: Phase 4（见设计文档 §13）

---

### Phase 4: AI 增强层 — 待办

> 设计文档: [aiops-phase4-ai-enhancement.md](../../design/active/aiops-phase4-ai-enhancement.md)
> 前置依赖: Phase 2b + Phase 3 ✅ 后开始

**P1: AI 增强核心**

- [ ] `aiops/ai/enhancer.go` — Enhancer 服务 + LLMProvider 接口 + Summarize 主流程 + 响应解析
- [ ] `aiops/ai/context_builder.go` — BuildIncidentContext（结构化数据 → LLM 文本描述）
- [ ] `aiops/ai/prompts.go` — SystemPrompt + UserPromptTemplate + SummarizePrompt 组装
- [ ] 单元测试: context 构建正确性 + Mock LLM 响应解析 + JSON 提取 + 降级处理

**P2: API 端点**

- [ ] `gateway/handler/aiops_ai.go` — Summarize + Recommend Handler（Operator 权限）
- [ ] `gateway/routes.go` — +2 路由（/api/v2/aiops/ai/summarize, /recommend）
- [ ] `service/interfaces.go` — Query 接口 +SummarizeIncident（通过 Enhancer，非 AIOpsEngine）
- [ ] `service/query/aiops.go` — +SummarizeIncident 实现（调用 aiopsAI.Summarize）
- [ ] API 集成测试

**P3: AI Chat Tool 集成**

- [ ] `ai/tool.go` — +ToolHandler 类型 + customTools map + RegisterTool 方法 + Execute 优先查自定义 Tool
- [ ] `ai/prompts.go` — toolsJSON +3 个 AIOps Tool 定义 + rolePrompt +AIOps 工具说明
- [ ] `master.go` — 初始化时 RegisterTool（analyze_incident, get_cluster_risk, get_recent_incidents）
- [ ] 单元测试: 3 个 Tool 执行 + 参数解析

**P4: 前端集成**

- [ ] `api/aiops.ts` — +SummarizeResponse 等类型 + summarizeIncident() + recommendActions()
- [ ] `IncidentDetailModal.tsx` — +AI 分析按钮 + 分析结果面板 + loading/error 状态 + Operator 权限检查
- [ ] `types/i18n.ts` — AIOpsTranslations +ai 子接口（~15 个键）
- [ ] `locales/zh.ts` — +aiops.ai 翻译
- [ ] `locales/ja.ts` — +aiops.ai 翻译
- [ ] `next build` 构建验证通过

**全部完成后**

- [ ] `go build` + `go test` + `npm run build` 全部通过
- [ ] 中心文档 `aiops-engine-design.md` §11 索引全部更新为 ✅
- [ ] 5 份设计文档 `docs/design/active/` → `docs/design/archive/`
- [ ] 本 tracker 中 AIOps 任务全部归档到 `docs/tasks/archive/aiops-tasks.md`
