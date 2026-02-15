# SLO OTel 改造 — 已完成

> 设计文档: [Agent](../../design/archive/slo-otel-agent-design.md) | [Master](../../design/archive/slo-otel-master-design.md)

| 侧 | Phase | 状态 |
|-----|-------|------|
| Agent | P1~P5 | 完成 |
| Master | P1~P4 | 完成 |
| 全链路 | E2E | 核心完成（待前端对接） |

## 关键设计决策

1. 不支持旧模式 — 旧 Ingress 直连方式完全删除，只走 OTel Collector
2. Agent 算增量 — per-pod delta 在 Agent 侧计算，Master 直接存储
3. Bucket 统一为 JSON — `map[string]int64`（key=毫秒字符串）
4. 入口层 Controller 无关 — IngressMetrics 由 Agent Parser 归一化
5. ServiceKey 标准化 — `"namespace-service-port"` 格式
6. 两层查询策略 — API 查 hourly 优先，无数据回退 raw 实时聚合
7. 三层数据维度 — service (Linkerd inbound) + edge (Linkerd outbound) + ingress (Controller)
