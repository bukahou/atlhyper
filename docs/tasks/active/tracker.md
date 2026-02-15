# 任务追踪

> 当前待办和进行中的任务。已完成的任务归档到 `docs/tasks/archive/`。
>
> **本文件是任务状态的唯一权威源。** Claude Code MEMORY.md 禁止复制此处的状态信息。

---

（当前无待办或进行中的任务）

---

## 已完成（待归档）

### SLO OTel 改造 — ✅ 完成

> 设计文档: [Agent](../../design/archive/slo-otel-agent-design.md) | [Master](../../design/archive/slo-otel-master-design.md)

- Agent P1 数据模型 — ✅ 完成
- Agent P2 SDK (OTel Client + Parser) — ✅ 完成
- Agent P3 Repository (filter/snapshot/aggregate) — ✅ 完成
- Agent P4 集成 — ✅ 完成
- Agent P5 E2E — ✅ 完成
- Master P1 数据库 (service_raw/hourly + edge_raw/hourly) — ✅ 完成
- Master P2 Processor — ✅ 完成
- Master P3 Aggregator — ✅ 完成
- Master P4 API (slo_mesh + slo 增强) — ✅ 完成

---

### 大后端小前端重构 — ✅ 完成

> 设计文档: [big-backend-small-frontend.md](../../design/archive/big-backend-small-frontend.md)

- Phase 1: NodeMetrics camelCase — ✅ 完成
- Phase 2: Overview camelCase — ✅ 完成
- Phase 3: K8s 资源扁平化（9/9） — ✅ 完成
- Phase 4: Command/SLOTarget camelCase — ✅ 完成
- Phase 5: 废弃文件清理 — ✅ 完成

---

### 节点指标 OTel 迁移 — ✅ 完成

> 设计文档: [Phase 1](../../design/archive/node-metrics-phase1-infra.md) | [Phase 2](../../design/archive/node-metrics-phase2-agent.md) | [Phase 3](../../design/archive/node-metrics-phase3-master.md)

- Phase 1: 基础设施部署 — ✅ 完成
- Phase 2: Agent 改造 — ✅ 完成
- Phase 3: Master 适配 + 前端完善 — ✅ 完成

---

### 8 个 K8s 资源 API + 详情扩展 — ✅ 完成

> 设计文档: [cluster-resources-api-design.md](../../design/archive/cluster-resources-api-design.md)

- model_v2 数据模型 — ✅ 完成
- Agent converter — ✅ 完成
- Master model/convert/handler — ✅ 完成
- 前端 DetailModal + i18n — ✅ 完成

---

### 前端路由重构 + 导航栏优化 — ✅ 完成

- /system 拆分为 /monitoring + /settings + /admin — ✅ 完成
- 4 个 K8s 导航组合并为 1 组 + section 分隔线 — ✅ 完成
- 导航组从 10 个精简到 6 个 — ✅ 完成
