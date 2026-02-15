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

### Phase 1: 依赖图引擎 + 基线引擎 — ✅ 完成

> commit: `8ff6fb2`, `927bc6e` (types 扩展)

---

### Phase 2a: 风险评分引擎 — ✅ 完成

> commit: `927bc6e`
> 19 个测试通过，3 个 API 端点就绪

---

### Phase 2b: 状态机引擎 + 事件存储 — ✅ 完成

> commit: `1366f60`
> 11 个状态机测试通过，4 个 API 端点就绪
> 全部 45 个 AIOps 测试通过（Phase 1: 15 + Phase 2a: 19 + Phase 2b: 11）

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
