# 任务追踪

> 当前待办和进行中的任务。已完成的任务归档到 `docs/tasks/archive/`。

---

## 大后端小前端重构 — 🔧 进行中

> 设计文档: [big-backend-small-frontend.md](../../design/archive/big-backend-small-frontend.md)

- Phase 1: NodeMetrics camelCase — ✅ 完成
- Phase 2: Overview camelCase — ✅ 完成

### Phase 3: K8s 资源扁平化（9 种资源，消除 ~963 行）

| 状态 | 任务 |
|:---:|------|
| [ ] | Pod: model + convert + handler + 前端删 transform |
| [ ] | Node: model + convert（含单位转换）+ handler + 前端删 transform |
| [ ] | Deployment: model + convert + handler + 前端删 transform |
| [ ] | StatefulSet: model + convert + handler + 前端删 transform |
| [ ] | DaemonSet: model + convert + handler + 前端删 transform |
| [ ] | Service: model + convert + handler + 前端删 transform |
| [ ] | Namespace: model + convert + handler + 前端删 transform |
| [ ] | Ingress: model + convert + handler + 前端删 transform |
| [ ] | Event: model + convert + handler + 前端删 transform |
| [ ] | Cluster: model + convert + handler |

### Phase 4: SLO/Mesh camelCase + 业务逻辑后端化

| 状态 | 任务 |
|:---:|------|
| [ ] | model/slo.go JSON tags -> camelCase |
| [ ] | model/command.go JSON tags -> camelCase |
| [ ] | 前端 types/slo.ts + types/mesh.ts -> camelCase |
| [ ] | 前端 SLO/Mesh 组件属性名同步修改 |
| [ ] | (可选) Error budget 后端计算 |
| [ ] | (可选) 拓扑 BFS 过滤后端化 |

### Phase 5: 废弃文件清理

| 状态 | 任务 |
|:---:|------|
| [ ] | 删除 api/metrics.ts |
| [ ] | 删除 api/config.ts |
| [ ] | 删除 api/test.ts |
| [ ] | 审查 utils/safeData.ts |

---

## 节点指标 OTel 迁移 — 🔧 进行中

> 设计文档: [Phase 1](../../design/archive/node-metrics-phase1-infra.md) | [Phase 2](../../design/archive/node-metrics-phase2-agent.md) | [Phase 3](../../design/archive/node-metrics-phase3-master.md)
> TDD 规范: [node-metrics-tdd.md](../../design/archive/node-metrics-tdd.md) | Mock 数据: [node-metrics-mock-data.md](../../design/archive/node-metrics-mock-data.md)

- Phase 1: 基础设施部署 — ✅ 完成
- Phase 2: Agent 改造 — ✅ 完成（35 测试全通过，6 节点 E2E 验证通过）

### Phase 3: Master 适配

| 状态 | 任务 |
|:---:|------|
| [ ] | 前端 PSI 卡片简化（三窗口 → 单数字） |
| [ ] | 前端 TCP 卡片调整（移除不存在的状态字段） |
| [ ] | style-preview mock 数据对齐真实格式 |
| [ ] | 下线 atlhyper-metrics DaemonSet |

### 关键设计决策

1. 数据来源：node_exporter → OTel Collector → Agent 拉取
2. 模型扩展：NodeMetricsSnapshot 新增 PSI/TCP/System/VMStat/NTP/Softnet
3. 过滤规则：文件系统只保留 /dev/、网络排除虚拟接口、磁盘排除 dm-*
4. PSI 计算：从累积 counter 做 rate 得近似百分比
5. TDD 驱动：先写测试数据和期望 → 写测试 → 实现代码
