# 任务追踪

> 当前待办和进行中的任务

---

## 大后端小前端重构 — 🔧 进行中

> 设计文档: [big-backend-small-frontend.md](../../design/active/big-backend-small-frontend.md)

### Phase 1: NodeMetrics camelCase（消除 ~411 行前端代码）— ✅ 完成

| 状态 | 任务 | 文件 |
|:---:|------|------|
| [x] | 新建 NodeMetrics 响应类型（camelCase） | `model/node_metrics.go` |
| [x] | 新建 NodeMetrics 转换函数 + 单元测试（8 测试通过） | `model/convert/node_metrics.go` |
| [x] | Handler 返回前调用 convert（3 个返回点） | `gateway/handler/node_metrics.go` |
| [x] | 前端删除 16 个 Backend 接口 + 16 个 transform（516→105 行） | `api/node-metrics.ts` |
| [x] | 编译验证（go build + next build 全通过） | |

### Phase 2: Overview camelCase（消除 ~180 行前端代码）— ✅ 完成

| 状态 | 任务 | 文件 |
|:---:|------|------|
| [x] | 新建 Overview 响应类型（17 个 camelCase 类型） | `model/overview.go` |
| [x] | 新建 Overview 转换函数 + 单元测试（3 测试通过） | `model/convert/overview.go` |
| [x] | Handler 返回前调用 convert | `gateway/handler/overview.go` |
| [x] | 前端删除 OverviewApiResponse + transformResponse（218→34 行） | `api/overview.ts` |
| [x] | 编译验证（go build + next build 全通过） | |

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

## SLO OTel 改造 — ✅ 核心完成

> 设计文档: [Agent](../../design/active/slo-otel-agent-design.md) | [Master](../../design/active/slo-otel-master-design.md)

| 侧 | Phase | 状态 |
|-----|-------|------|
| Agent | P1~P5 | ✅ 完成 |
| Master | P1~P4 | ✅ 完成 |
| 全链路 | E2E | ✅ 核心完成（待前端对接） |

---

## 节点指标 OTel 迁移 — 🔧 进行中

> 设计文档: [Phase 1](../../design/active/node-metrics-phase1-infra.md) | [Phase 2](../../design/active/node-metrics-phase2-agent.md) | [Phase 3](../../design/active/node-metrics-phase3-master.md)
> TDD 规范: [node-metrics-tdd.md](../../design/active/node-metrics-tdd.md)（**权威文档**）
> Mock 数据: [node-metrics-mock-data.md](../../design/active/node-metrics-mock-data.md)

### 依赖关系

```
Phase 1 (基础设施) ─→ Phase 2 (Agent) ─→ Phase 3 (Master/前端)
  ✅ 已完成              ✅ 已完成             待开始
```

### Phase 1: 基础设施部署 — ✅ 完成

| 状态 | 任务 |
|:---:|------|
| [x] | node_exporter DaemonSet 部署（6 节点全部 Running） |
| [x] | OTel Collector ConfigMap 更新（node-exporter 抓取 job） |
| [x] | 白名单验证（57 个指标名，1613 条数据） |
| [x] | 白名单修订（补充 crit_celsius、TCP_inuse，移除不存在的 cpu_info、tcp_connection_states） |
| [x] | 真实数据抓取和分析（发现 6 个设计假设差异） |

### Phase 2: Agent 改造 — ✅ 完成

| 状态 | 任务 | 文件 |
|:---:|------|------|
| [x] | TDD 主文档编写 | `node-metrics-tdd.md` |
| [x] | Agent Phase 2 设计修订 | `node-metrics-phase2-agent.md` |
| [x] | Master Phase 3 设计修订 | `node-metrics-phase3-master.md` |
| [x] | 扩展 NodeMetricsSnapshot（新增 PSI/TCP/System/VMStat/NTP/Softnet） | `model_v2/node_metrics.go` |
| [x] | 创建测试数据文件 | `testdata/otel_*.txt` |
| [x] | 新增 OTelNodeRawMetrics 类型 | `sdk/types.go` |
| [x] | OTelClient 接口扩展 | `sdk/interfaces.go` |
| [x] | node_parser 测试 → 实现 (TDD) — 5 测试通过 | `sdk/impl/otel/node_parser*.go` |
| [x] | ScrapeNodeMetrics 实现 | `sdk/impl/otel/client.go` |
| [x] | 过滤规则 + 测试 — 12 测试通过 | `repository/metrics/filter*.go` |
| [x] | rate 计算器测试 → 实现 (TDD) — 7 测试通过 | `repository/metrics/rate*.go` |
| [x] | converter 测试 → 实现 (TDD) — 11 测试通过 | `repository/metrics/converter*.go` |
| [x] | metrics.go 重写 (OTel 拉取 + Receiver 降级) | `repository/metrics/metrics.go` |
| [x] | Scheduler MetricsSync 循环 | `scheduler/scheduler.go` |
| [x] | agent.go 初始化调整 | `agent.go` |
| [x] | go build 编译验证 — 全项目编译通过 | |
| [x] | go test 自动化验证 — 35 测试全部通过 | |
| [x] | 真实数据端到端验证 — 6 节点全部通过（E2E 测试） | `repository/metrics/e2e_test.go` |

### Phase 3: Master 适配 — 待 Phase 2 完成

| 状态 | 任务 |
|:---:|------|
| [ ] | 前端 PSI 卡片简化（三窗口 → 单数字） |
| [ ] | 前端 TCP 卡片调整（移除不存在的状态字段） |
| [ ] | style-preview mock 数据对齐真实格式 |
| [ ] | 下线 atlhyper-metrics DaemonSet |

---

### 关键设计决策（节点指标）

1. **数据来源**：node_exporter → OTel Collector → Agent 拉取（替代 atlhyper_metrics_v2 推送）
2. **模型扩展**：NodeMetricsSnapshot 新增 PSI/TCP/System/VMStat/NTP/Softnet（向后兼容）
3. **过滤规则**：文件系统只保留 /dev/、网络排除虚拟接口、磁盘排除 dm-*
4. **PSI 计算**：从累积 counter 做 rate 得近似百分比（非 10s/60s/300s 窗口）
5. **CPU 型号**：node_exporter 不提供，留空
6. **TDD 驱动**：先写测试数据和期望 → 写测试 → 实现代码
