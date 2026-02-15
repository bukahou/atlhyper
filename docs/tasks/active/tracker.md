# 任务追踪

> 当前待办和进行中的任务。已完成的任务归档到 `docs/tasks/archive/`。

---

## 大后端小前端重构 — 🔧 进行中

> 原设计文档: [big-backend-small-frontend.md](../../design/archive/big-backend-small-frontend.md)
> 剩余工作设计: [big-backend-phase3-5-remaining.md](../../design/active/big-backend-phase3-5-remaining.md)

- Phase 1: NodeMetrics camelCase — ✅ 完成
- Phase 2: Overview camelCase — ✅ 完成

### Phase 3: K8s 资源扁平化（9 种资源）

| 状态 | 任务 |
|:---:|------|
| [x] | Pod: model + convert + handler + 前端删 transform |
| [x] | Node: model + convert（含单位转换）+ handler + 前端删 transform |
| [x] | Deployment: model + convert + handler + 前端删 transform |
| [ ] | **StatefulSet: model + convert + handler + 前端删 transform** |
| [ ] | **DaemonSet: model + convert + handler + 前端删 transform** |
| [x] | Service: model + convert + handler + 前端删 transform |
| [x] | Namespace: model + convert + handler + 前端删 transform |
| [x] | Ingress: model + convert + handler + 前端删 transform |
| [x] | Event: model + convert + handler + 前端删 transform |

### Phase 4: SLO/Command camelCase

| 状态 | 任务 |
|:---:|------|
| [x] | model/slo.go JSON tags → 已是 camelCase |
| [ ] | 新增 model/command.go camelCase + convert + handler 修改 |
| [ ] | 前端 types/slo.ts SLOTarget → camelCase |
| [ ] | 前端 SLO 组件属性名同步修改 |
| [ ] | (可选) Error budget 后端计算 |
| [ ] | (可选) 拓扑 BFS 过滤后端化 |

### Phase 5: 废弃文件清理

| 状态 | 任务 |
|:---:|------|
| [ ] | 删除 api/metrics.ts（旧指标 API，无引用） |
| [x] | ~~删除 api/config.ts~~ |
| [x] | ~~删除 api/test.ts~~ |
| [x] | ~~审查 utils/safeData.ts~~（已删除） |

---

## 节点指标 OTel 迁移 — 🔧 进行中

> 原设计文档: [Phase 1](../../design/archive/node-metrics-phase1-infra.md) | [Phase 2](../../design/archive/node-metrics-phase2-agent.md) | [Phase 3](../../design/archive/node-metrics-phase3-master.md)
> 剩余工作设计: [node-metrics-phase3-remaining.md](../../design/active/node-metrics-phase3-remaining.md)

- Phase 1: 基础设施部署 — ✅ 完成
- Phase 2: Agent 改造 — ✅ 完成（35 测试全通过，6 节点 E2E 验证通过）

### Phase 3: Master 适配 + 前端完善

| 状态 | 任务 |
|:---:|------|
| [x] | ~~前端 PSI 卡片简化（三窗口 → 单数字）~~ |
| [x] | ~~前端 TCP 卡片调整（移除不存在的状态字段）~~ |
| [x] | ~~style-preview mock 数据对齐真实格式~~ |
| [ ] | **节点指标组件 i18n 国际化**（11 个组件硬编码英文） |
| [ ] | 下线 atlhyper-metrics DaemonSet（删除部署文件） |
| [ ] | 删除 api/metrics.ts（与 Phase 5 合并） |

### 关键设计决策

1. 数据来源：node_exporter → OTel Collector → Agent 拉取
2. 模型扩展：NodeMetricsSnapshot 新增 PSI/TCP/System/VMStat/NTP/Softnet
3. 过滤规则：文件系统只保留 /dev/、网络排除虚拟接口、磁盘排除 dm-*
4. PSI 计算：从累积 counter 做 rate 得近似百分比
5. TDD 驱动：先写测试数据和期望 → 写测试 → 实现代码
