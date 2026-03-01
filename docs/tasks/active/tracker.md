# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

---

## 代码库审计优化 — 🔄 进行中

> 原设计文档: [codebase-audit-solutions.md](../../design/active/codebase-audit-solutions.md)

- Phase 0: model_v2 → model_v3 迁移（54 文件） — ✅ 完成
- Phase 1: CLAUDE.md 全面更新（15 章节） — ✅ 完成
- Phase 2: Master 架构违规整改 — 🔄 部分完成
  - 2.2 工厂函数命名统一（15 个 New() → NewXxx()） ✅
  - 2.3 统一 logger（16 文件 74 处迁移至 common/logger） ✅
  - 4.3 interface.go → interfaces.go 重命名（3 个 notifier 文件） ✅
  - 2.1 Gateway 跳层修复（Batch A+B: 6 Handler 迁移至 Service 层） ✅
  - 2.5 其他小修 ✅
    - database/interfaces.go 拆分为 types.go + interfaces.go ✅
    - CreateCommandRequest/Response 移至 model 包（消除 service→operations 反向依赖）✅
    - service/sync/ 评估：独立后台同步服务，保持现状 ✅
  - 2.4 handler/ 目录拆分 — 待办
- Phase 3: 前端规范整改 — 🔄 部分完成
  - cluster-resources.ts 拆分为 8 个按资源 API 文件 ✅
  - 300 行组件拆分 — 🔄 部分完成
    - Sidebar（699→5 文件）✅
    - MeshTab（895→7 文件）✅
    - TraceWaterfall（691→5 文件）✅
    - Chat/page（534→5 文件）✅
    - MessageBubble（468→3 文件）✅
    - ClusterOverviewChart（475→4 文件）✅
    - 剩余 DetailModal 等 >400 行组件 — 待办
  - i18n 硬编码补全（~15 组件）— 待办
  - 组件可见性修复（GPUCard/MiniSparkline/LogHistogram/ProbesDisplay） ✅
- Phase 4: 低优先级清理 — ✅ 完成
  - 4.1 Agent 死代码删除（scanFacets/computeRate/safeDiv/unused types） ✅
  - 4.3 interface.go → interfaces.go ✅
  - 4.2 Agent concentrator 接口化 ✅
