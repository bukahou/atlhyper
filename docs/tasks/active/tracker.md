# 任务追踪

> 当前待办和进行中的任务。已完成的任务归档到 `docs/tasks/archive/`。

---

## 大后端小前端重构 — ✅ 完成

> 原设计文档: [big-backend-small-frontend.md](../../design/archive/big-backend-small-frontend.md)
> 剩余工作设计: [big-backend-phase3-5-remaining.md](../../design/active/big-backend-phase3-5-remaining.md)

- Phase 1: NodeMetrics camelCase — ✅ 完成
- Phase 2: Overview camelCase — ✅ 完成
- Phase 3: K8s 资源扁平化（9/9） — ✅ 完成
- Phase 4: Command/SLOTarget camelCase — ✅ 完成
- Phase 5: 废弃文件清理 — ✅ 完成

---

## 节点指标 OTel 迁移 — ✅ 完成

> 原设计文档: [Phase 1](../../design/archive/node-metrics-phase1-infra.md) | [Phase 2](../../design/archive/node-metrics-phase2-agent.md) | [Phase 3](../../design/archive/node-metrics-phase3-master.md)
> 剩余工作设计: [node-metrics-phase3-remaining.md](../../design/active/node-metrics-phase3-remaining.md)

- Phase 1: 基础设施部署 — ✅ 完成
- Phase 2: Agent 改造 — ✅ 完成
- Phase 3: Master 适配 + 前端完善 — ✅ 完成
  - PSI/TCP 卡片简化 ✅
  - style-preview mock 对齐 ✅
  - 13 个指标组件 i18n 国际化 ✅
  - 下线 atlhyper-metrics DaemonSet ✅
  - 删除废弃 api/metrics.ts ✅
