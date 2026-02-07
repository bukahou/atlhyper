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

### Agent 侧

#### P1: 数据模型

| 状态 | 任务 | 文件 | 设计文档 |
|:---:|------|------|----------|
| [ ] | 重写 SLOSnapshot + ServiceMetrics + ServiceEdge + IngressMetrics + IngressRouteInfo | `model_v2/slo.go` | Agent §3 |
| [ ] | 删除旧类型 (IngressMetrics/IngressCounterMetric/IngressHistogramMetric) | `model_v2/slo.go` | Agent §3 |
| [ ] | 验证 JSON 序列化/反序列化 | 单元测试 | — |

#### P2: SDK 层

| 状态 | 任务 | 文件 | 设计文档 |
|:---:|------|------|----------|
| [ ] | 新增 OTelClient 接口 | `sdk/interfaces.go` | Agent §4.1 |
| [ ] | 新增 OTel 内部类型 (OTelRawMetrics + LinkerdResponseMetric + LinkerdLatencyMetric + TraefikRequestMetric + TraefikLatencyMetric) | `sdk/types.go` | Agent §4.2 |
| [ ] | 实现 OTelClient (NewOTelClient + HTTP 采集 + 健康检查) | `sdk/impl/otel/client.go` | Agent §4.3 |
| [ ] | 实现 Prometheus 文本解析器 | `sdk/impl/otel/parser.go` | Agent §4.4 |
| [ ] | 单元测试: mock OTel Prometheus 输出 → 验证解析结果 | 测试文件 | — |

#### P3: Repository 层

| 状态 | 任务 | 文件 | 设计文档 |
|:---:|------|------|----------|
| [ ] | 重写 SLORepository 主入口 (Collect 编排 filter→delta→aggregate→routes) | `repository/slo/slo.go` | Agent §5.1 |
| [ ] | 新建内部增量类型定义 (linkerdResponseDelta, ingressRequestDelta 等) | `repository/slo/types.go` | Agent §5 |
| [ ] | 新建 Filter (过滤 probe/admin/系统 namespace) | `repository/slo/filter.go` | Agent §5.2 |
| [ ] | 新建 snapshotManager (per-pod delta 计算 + prev 维护) | `repository/slo/snapshot.go` | Agent §5.3 |
| [ ] | 新建 Aggregate (Pod→Service 聚合 + Edge 提取 + Ingress 聚合 + mTLS) | `repository/slo/aggregate.go` | Agent §5.4~§5.6 |
| [ ] | 删除旧 Ingress 直连代码 | `sdk/impl/ingress/client.go`, `discover.go`, `parser.go` | Agent §11 删除 |
| [ ] | 单元测试: mock OTelRawMetrics → 验证 filter/delta/aggregate 各阶段 | 测试文件 | — |

#### P4: 集成

| 状态 | 任务 | 文件 | 设计文档 |
|:---:|------|------|----------|
| [ ] | Config 新增 OTelMetricsURL/OTelHealthURL/ExcludeNamespaces | `config/types.go`, `config/defaults.go` | Agent §6 |
| [ ] | agent.go 依赖注入 OTelClient + SLORepository | `agent.go` | Agent §8 |
| [ ] | SLORepository 接口去掉 CollectRoutes (合并到 Collect) | `repository/interfaces.go` | Agent §7 |
| [ ] | Service 层适配 (删除 CollectRoutes 调用) | `service/snapshot/snapshot.go` | Agent §7 |

#### P5: 端到端验证

| 状态 | 任务 | 文件 | 设计文档 |
|:---:|------|------|----------|
| [ ] | 对接真实 OTel Collector (已部署) | — | — |
| [ ] | 验证 Linkerd inbound 指标采集 + delta 正确性 | — | — |
| [ ] | 验证 Linkerd outbound 拓扑 Edge 提取 | — | — |
| [ ] | 验证 Traefik 入口指标采集 (Controller 无关) | — | — |
| [ ] | 验证 ClusterSnapshot.SLOData 上报 Master | — | — |

---

### Master 侧

#### P1: 数据库 + 清理旧代码

| 状态 | 任务 | 文件 | 设计文档 |
|:---:|------|------|----------|
| [ ] | 新增 4 张表 (slo_service_raw/hourly, slo_edge_raw/hourly) | `database/sqlite/migrations.go` | Master §4.1~§4.4 |
| [ ] | DROP 重建 slo_metrics_raw/hourly (12列bucket→JSON) | `database/sqlite/migrations.go` | Master §4.5 |
| [ ] | ALTER slo_targets (+target_type) | `database/sqlite/migrations.go` | Master §4.7 |
| [ ] | DROP snapshot 表 (ingress_counter_snapshot, ingress_histogram_snapshot) | `database/sqlite/migrations.go` | Master §4.8 |
| [ ] | 新建服务网格表 SQL Dialect | `database/sqlite/slo_service.go` | Master §9 新建 |
| [ ] | 新建拓扑边表 SQL Dialect | `database/sqlite/slo_edge.go` | Master §9 新建 |
| [ ] | 重写入口表 SQL (移除 snapshot SQL + bucket JSON 化) | `database/sqlite/slo.go` | Master §9 重写 |
| [ ] | 新增 SLOServiceRepository + SLOEdgeRepository 接口 | `database/interfaces.go` | Master §9 修改 |
| [ ] | 实现新 Repository 接口 (NewSLOServiceRepository / NewSLOEdgeRepository); 移除 snapshot 实现 | `database/repo/slo.go` | Master §9 修改 |
| [ ] | 新建 slo/ 领域处理器接口 (SLOProcessor / SLOAggregator / SLOCleaner) | `slo/interfaces.go` | Master §3.2 |
| [ ] | 删除旧模式 HTTP Handler | `agentsdk/slo.go` | Master §9 删除 |
| [ ] | 移除 /agent/slo 路由注册 | `agentsdk/server.go` | Master §9 修改 |

#### P2: Processor + Sync

| 状态 | 任务 | 文件 | 设计文档 |
|:---:|------|------|----------|
| [ ] | 重写 ProcessSLOSnapshot 入口 | `slo/processor.go` | Master §5 |
| [ ] | 实现 processServiceMetrics (→ slo_service_raw) | `slo/processor.go` | Master §5.1 |
| [ ] | 实现 processEdge (→ slo_edge_raw) | `slo/processor.go` | Master §5.2 |
| [ ] | 实现 processIngressMetrics (→ slo_metrics_raw, JSON bucket) | `slo/processor.go` | Master §5.3 |
| [ ] | 重写 slo_persist.go (只保留 OTel 路径) | `service/sync/slo_persist.go` | Master §9 重写 |
| [ ] | 修改 calculator.go (删除旧 delta/bucket 函数, 统一 CalculateQuantile 入参) | `slo/calculator.go` | Master §9 修改 |
| [ ] | 单元测试 | 测试文件 | — |

#### P3: Aggregator + Cleaner

| 状态 | 任务 | 文件 | 设计文档 |
|:---:|------|------|----------|
| [ ] | 实现 aggregateServiceHour (raw→hourly, JSON bucket, calcPercentile) | `slo/aggregator.go` | Master §2.2 |
| [ ] | 实现 aggregateEdgeHour (raw→hourly, SUM 聚合) | `slo/aggregator.go` | Master §2.2 |
| [ ] | 实现 aggregateIngressHour (重写, JSON bucket + method 分布) | `slo/aggregator.go` | Master §2.2 |
| [ ] | Cleaner 新增 slo_service_raw + slo_edge_raw 清理 | `slo/cleaner.go` | Master §8 |
| [ ] | StatusChecker 扩展 service 维度 | `slo/status_checker.go` | Master §9 修改 |
| [ ] | 单元测试 | 测试文件 | — |

#### P4: Service 层 + API + 集成

| 状态 | 任务 | 文件 | 设计文档 |
|:---:|------|------|----------|
| [ ] | service.Query 接口新增 SLO 查询方法 (GetMeshTopology / GetServiceDetail / GetDomainsV2) | `service/interfaces.go` | Master §7.0 |
| [ ] | 新建 SLO 查询实现 (含 hourly→raw 回退策略) | `service/query/slo.go` | Master §7.0 |
| [ ] | service/factory.go 注入新 Repository | `service/factory.go` | Master §7.0 |
| [ ] | 新建服务网格 Handler: MeshTopology + ServiceDetail (依赖 service.Query) | `gateway/handler/slo_mesh.go` | Master §7.1 |
| [ ] | 增强 DomainsV2，改为依赖 service.Query (+分布数据+关联服务) | `gateway/handler/slo.go` | Master §7.3 |
| [ ] | 新增 API 响应类型 | `model/slo.go` | Master §7.1~§7.2 |
| [ ] | 注册新路由 /api/v2/slo/mesh/* | `gateway/routes.go` | Master §7.3 |
| [ ] | 依赖注入新 Repository + QueryService + Handler | `master.go` | Master §2.5 |

---

### 全链路 E2E

| 状态 | 任务 | 验证内容 |
|:---:|------|----------|
| [ ] | Agent → Master 数据写入 | Agent 上报 SLOSnapshot → Master processor 正确写入 3 张 raw 表 |
| [ ] | Aggregator 聚合 | raw 数据正确聚合为 hourly; 刚部署时 raw 回退可用 |
| [ ] | 服务网格 API | /mesh/topology 返回正确的节点+边+黄金指标 |
| [ ] | 域名 SLO API | /domains/v2 返回延迟分布+请求分布+关联服务 |
| [ ] | 前端对接 | style-preview 两层展示数据正确渲染 |

---

### 统计

| 侧 | Phase | 任务数 |
|-----|-------|--------|
| Agent | P1 数据模型 | 3 |
| Agent | P2 SDK | 5 |
| Agent | P3 Repository | 7 |
| Agent | P4 集成 | 4 |
| Agent | P5 E2E | 5 |
| Master | P1 数据库 | 12 |
| Master | P2 Processor | 7 |
| Master | P3 Aggregator | 6 |
| Master | P4 Service+API | 8 |
| 全链路 | E2E | 5 |
| **合计** | | **62** |
