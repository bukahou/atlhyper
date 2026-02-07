# 任务追踪

> 当前待办和进行中的任务

---

## SLO OTel 改造

> 设计文档: [Agent](../../design/active/slo-otel-agent-design.md) | [Master](../../design/active/slo-otel-master-design.md)

### 依赖关系

```
Agent P1 ─────→ Agent P2 ─────→ Agent P3 ─────→ Agent P4 ─────→ Agent P5
(数据模型)       (SDK)           (Repository)     (集成)          (E2E)
    │                                                               │
    │ 共享 model_v2/slo.go                                          │
    ▼                                                               ▼
Master P1 ────→ Master P2 ────→ Master P3 ────→ Master P4 ────→ 全链路 E2E
(数据库)         (Processor)     (Aggregator)     (Service+API)
```

- Agent P1 和 Master P1 共享 `model_v2/slo.go`，Agent P1 先行
- Agent P2~P4 和 Master P1~P3 可并行
- 全链路 E2E 需要 Agent P5 + Master P4 都完成

---

### Agent 侧 — ✅ 全部完成

- **P1 数据模型**: model_v2/slo.go 重写完成
- **P2 SDK 层**: OTelClient 接口 + 实现 + Prometheus 解析器完成
- **P3 Repository 层**: SLO 采集编排（filter→delta→aggregate→routes）完成
- **P4 集成**: Config + agent.go 依赖注入 + interfaces 适配完成

#### P5: 端到端验证（待办）

| 状态 | 任务 |
|:---:|------|
| [ ] | 对接真实 OTel Collector (已部署)，验证 Linkerd/Traefik 指标采集 |
| [ ] | 验证 ClusterSnapshot.SLOData 上报 Master |
| [ ] | 验证 delta 计算正确性（重启后重置） |

---

### Master 侧 — ✅ P1~P4 全部完成

- **P1 数据库层**: 4 新表 + 2 重建表 + 2 DROP snapshot + 新 Dialect + interfaces + repo
- **P2 Processor + Sync**: 三层数据写入（service/edge/ingress raw）+ slo_persist 简化
- **P3 Aggregator + Cleaner**: 三层聚合（raw→hourly）+ ParseJSONBuckets + 6 表清理
- **P4 Service 层 + API**: Query 接口扩展 + mesh 拓扑/详情 API + aggregateRawMetrics bucket 精确计算

---

### 全链路 E2E（待办）

| 状态 | 任务 | 验证内容 |
|:---:|------|----------|
| [ ] | Agent → Master 数据写入 | Agent 上报 SLOSnapshot → Master processor 正确写入 3 张 raw 表 |
| [ ] | Aggregator 聚合 | raw 数据正确聚合为 hourly; 刚部署时 raw 回退可用 |
| [ ] | 服务网格 API | /mesh/topology 返回正确的节点+边+黄金指标 |
| [ ] | 域名 SLO API | /domains/v2 返回延迟分布+请求分布+关联服务 |
| [ ] | 前端对接 | style-preview 两层展示数据正确渲染 |

---

### 进度统计

| 侧 | Phase | 状态 |
|-----|-------|------|
| Agent | P1~P4 | ✅ 完成 |
| Agent | P5 E2E | 待办 |
| Master | P1~P4 | ✅ 完成 |
| 全链路 | E2E | 待办 |
