# SLO Monitoring V2 任务清单

> 关联设计文档: `docs/design/slo-monitoring-v2.md`
> 创建日期: 2026-02-01
> 目标: 以域名+路径作为 SLO 监控的核心维度

---

## 任务总览

| Phase | 任务数 | 优先级 | 状态 |
|-------|--------|--------|------|
| Phase 1: Agent IngressRoute 采集 | 4 | P0 | 待开始 |
| Phase 2: Agent Metrics 过滤 | 2 | P0 | 待开始 |
| Phase 3: Master 映射存储 | 4 | P0 | 待开始 |
| Phase 4: Master 数据关联 | 3 | P0 | 待开始 |
| Phase 5: API 改造 | 3 | P1 | 待开始 |
| Phase 6: 前端适配 | 4 | P1 | 待开始 |
| Phase 7: 清理与优化 | 3 | P2 | 待开始 |

---

## Phase 1: Agent IngressRoute 采集

### 1.1 定义 IngressRoute 数据结构

- **文件**: `atlhyper_agent_v2/model/ingress_route.go`
- **状态**: [ ] 待开始
- **描述**: 定义 IngressRoute 相关的数据结构

```go
// IngressRouteInfo IngressRoute 解析后的信息
type IngressRouteInfo struct {
    Name        string `json:"name"`         // IngressRoute 名称
    Namespace   string `json:"namespace"`    // 命名空间
    Domain      string `json:"domain"`       // 域名 (从 Host() 解析)
    PathPrefix  string `json:"path_prefix"`  // 路径前缀 (从 PathPrefix() 解析)
    ServiceKey  string `json:"service_key"`  // Traefik service 标识
    ServiceName string `json:"service_name"` // K8s Service 名称
    ServicePort int    `json:"service_port"` // K8s Service 端口
    TLS         bool   `json:"tls"`          // 是否启用 TLS
}
```

### 1.2 实现 IngressRoute 采集接口

- **文件**: `atlhyper_agent_v2/sdk/interfaces.go`
- **状态**: [ ] 待开始
- **描述**: 在 SDK 中定义 IngressRoute 采集接口

```go
// IngressRouteCollector IngressRoute 采集器接口
type IngressRouteCollector interface {
    // CollectRoutes 采集所有 IngressRoute 配置
    CollectRoutes(ctx context.Context) ([]model.IngressRouteInfo, error)
}
```

### 1.3 实现 IngressRoute 采集器

- **文件**: `atlhyper_agent_v2/sdk/impl/ingress_route_collector.go`
- **状态**: [ ] 待开始
- **描述**: 实现 Traefik IngressRoute CRD 的采集和解析
- **要点**:
  - 使用 K8s dynamic client 获取 IngressRoute CRD
  - 解析 `spec.routes[].match` 规则，提取 Host() 和 PathPrefix()
  - 构建 Traefik service key: `{namespace}-{serviceName}-{port}@kubernetes`
  - 处理多路由情况（一个 IngressRoute 可能有多个 routes）

```go
// 解析 match 规则示例
// match: Host(`example.com`) && PathPrefix(`/api`)
// → domain: example.com, pathPrefix: /api

func parseMatchRule(match string) (domain, pathPrefix string) {
    // 正则提取 Host(`xxx`) 和 PathPrefix(`xxx`)
}
```

### 1.4 集成到 SnapshotService

- **文件**: `atlhyper_agent_v2/service/snapshot_service.go`
- **状态**: [ ] 待开始
- **描述**: 将 IngressRoute 采集集成到现有的 Snapshot 采集流程
- **修改**:
  - 在 `Collect()` 方法中并发采集 IngressRoute
  - 将结果添加到 `ClusterSnapshot.IngressRoutes`

---

## Phase 2: Agent Metrics 过滤

### 2.1 修改 Traefik Metrics 解析

- **文件**: `atlhyper_agent_v2/sdk/impl/scraper.go`
- **状态**: [ ] 待开始
- **描述**: 只采集 service 级别指标，过滤 entrypoint 级别指标
- **修改函数**: `parseTraefikMetric()`

```go
func (s *metricsScraper) parseTraefikMetric(line string) *cleanedMetric {
    // ✓ 采集 service 级别
    if strings.HasPrefix(line, "traefik_service_requests_total{") {
        return s.parseTraefikCounterLine(line)
    }
    if strings.HasPrefix(line, "traefik_service_request_duration_seconds_bucket{") {
        return s.parseHistogramBucketLine(line)
    }
    // ... sum, count ...

    // ✗ 过滤 entrypoint 级别 (不再处理)
    // traefik_entrypoint_requests_total → 忽略
    // traefik_entrypoint_request_duration_seconds → 忽略

    return nil
}
```

### 2.2 更新 ClusterSnapshot 结构

- **文件**: `atlhyper_agent_v2/model/snapshot.go`
- **状态**: [ ] 待开始
- **描述**: 添加 IngressRoutes 字段

```go
type ClusterSnapshot struct {
    // ... 现有字段 ...
    IngressMetrics *IngressMetrics    `json:"ingress_metrics,omitempty"`
    IngressRoutes  []IngressRouteInfo `json:"ingress_routes,omitempty"` // 新增
}
```

---

## Phase 3: Master 映射存储

### 3.1 新增数据库表

- **文件**: `atlhyper_master_v2/database/sqlite/migrations.go`
- **状态**: [ ] 待开始
- **描述**: 添加 `slo_route_mapping` 表

```sql
CREATE TABLE IF NOT EXISTS slo_route_mapping (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id      TEXT NOT NULL,
    domain          TEXT NOT NULL,
    path_prefix     TEXT NOT NULL DEFAULT '/',
    ingress_name    TEXT NOT NULL,
    namespace       TEXT NOT NULL,
    tls             INTEGER NOT NULL DEFAULT 1,
    service_key     TEXT NOT NULL,
    service_name    TEXT NOT NULL,
    service_port    INTEGER NOT NULL,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    UNIQUE(cluster_id, service_key)
);
CREATE INDEX IF NOT EXISTS idx_route_mapping_domain ON slo_route_mapping(cluster_id, domain);
CREATE INDEX IF NOT EXISTS idx_route_mapping_service ON slo_route_mapping(cluster_id, service_key);
```

### 3.2 定义 RouteMapping 数据结构

- **文件**: `atlhyper_master_v2/database/types.go`
- **状态**: [ ] 待开始
- **描述**: 定义 SLORouteMapping 结构体

```go
type SLORouteMapping struct {
    ID          int64     `json:"id"`
    ClusterID   string    `json:"cluster_id"`
    Domain      string    `json:"domain"`
    PathPrefix  string    `json:"path_prefix"`
    IngressName string    `json:"ingress_name"`
    Namespace   string    `json:"namespace"`
    TLS         bool      `json:"tls"`
    ServiceKey  string    `json:"service_key"`
    ServiceName string    `json:"service_name"`
    ServicePort int       `json:"service_port"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### 3.3 实现 RouteMapping Repository

- **文件**: `atlhyper_master_v2/database/repo/slo.go`
- **状态**: [ ] 待开始
- **描述**: 添加 RouteMapping 相关的数据库操作
- **新增方法**:
  - `UpsertRouteMapping(ctx, mapping) error`
  - `GetRouteMappingByServiceKey(ctx, clusterID, serviceKey) (*SLORouteMapping, error)`
  - `GetRouteMappingsByDomain(ctx, clusterID, domain) ([]*SLORouteMapping, error)`
  - `GetAllDomains(ctx, clusterID) ([]string, error)`

### 3.4 实现 RouteMapper 服务

- **文件**: `atlhyper_master_v2/slo/route_mapper.go` (新建)
- **状态**: [ ] 待开始
- **描述**: 管理 IngressRoute 映射关系
- **功能**:
  - `UpdateMappings()`: 接收 Agent 上报的 IngressRoute，更新数据库
  - `ResolveServiceKey()`: 将 Traefik service 名称解析为 domain + path
  - `GetDomainRoutes()`: 获取某个域名下的所有路径

---

## Phase 4: Master 数据关联

### 4.1 修改 SLO Processor

- **文件**: `atlhyper_master_v2/slo/processor.go`
- **状态**: [ ] 待开始
- **描述**: 处理 IngressRoute 数据，并在写入 metrics 时关联 domain/path
- **修改**:
  - `Process()` 方法中调用 `RouteMapper.UpdateMappings()`
  - `processCounterMetrics()` 中通过 `RouteMapper.ResolveServiceKey()` 获取 domain/path
  - 跳过无法映射的 service（可能是 entrypoint 漏网之鱼）

### 4.2 修改 slo_metrics_raw 表

- **文件**: `atlhyper_master_v2/database/sqlite/migrations.go`
- **状态**: [ ] 待开始
- **描述**: 添加 domain 和 path_prefix 字段

```sql
ALTER TABLE slo_metrics_raw ADD COLUMN domain TEXT;
ALTER TABLE slo_metrics_raw ADD COLUMN path_prefix TEXT DEFAULT '/';
CREATE INDEX IF NOT EXISTS idx_slo_raw_domain ON slo_metrics_raw(cluster_id, domain, path_prefix, timestamp);
```

### 4.3 修改 Aggregator

- **文件**: `atlhyper_master_v2/slo/aggregator.go`
- **状态**: [ ] 待开始
- **描述**: 按 domain + path_prefix 维度聚合数据
- **修改**:
  - 聚合时 GROUP BY `cluster_id, domain, path_prefix`
  - 同时计算域名级别的汇总数据

---

## Phase 5: API 改造

### 5.1 定义新的 API Response 结构

- **文件**: `atlhyper_master_v2/model/slo.go`
- **状态**: [ ] 待开始
- **描述**: 定义按域名分组的响应结构

```go
// DomainSLOResponse 域名级别的 SLO 响应
type DomainSLOResponse struct {
    Domain   string      `json:"domain"`
    TLS      bool        `json:"tls"`
    Routes   []RouteSLO  `json:"routes"`
    Summary  *SLOMetrics `json:"summary"`
    Status   string      `json:"status"`
    ErrorBudgetRemaining float64 `json:"error_budget_remaining"`
}

// RouteSLO 路径级别的 SLO 数据
type RouteSLO struct {
    PathPrefix  string      `json:"path_prefix"`
    IngressName string      `json:"ingress_name"`
    Namespace   string      `json:"namespace"`
    Current     *SLOMetrics `json:"current"`
    Previous    *SLOMetrics `json:"previous,omitempty"`
    Targets     map[string]*SLOTargetSpec `json:"targets,omitempty"`
    Status      string      `json:"status"`
}
```

### 5.2 修改 Domains API Handler

- **文件**: `atlhyper_master_v2/gateway/handler/slo.go`
- **状态**: [ ] 待开始
- **描述**: 修改 `GET /api/v2/slo/domains` 返回新格式
- **逻辑**:
  1. 获取所有域名列表 (`GetAllDomains`)
  2. 对每个域名获取其下的所有路径 (`GetDomainRoutes`)
  3. 查询每个路径的 metrics 数据
  4. 计算域名级别的汇总数据
  5. 返回按域名分组的结构

### 5.3 新增 Domain Detail API

- **文件**: `atlhyper_master_v2/gateway/handler/slo.go`
- **状态**: [ ] 待开始
- **描述**: 修改 `GET /api/v2/slo/domains/{domain}` 返回单个域名详情
- **参数**:
  - `domain`: 域名（URL 编码）
  - `time_range`: 时间范围

---

## Phase 6: 前端适配

### 6.1 修改 SLO 列表组件

- **文件**: `atlhyper_web/src/app/[locale]/workbench/slo/components/DomainList.tsx`
- **状态**: [ ] 待开始
- **描述**:
  - 以域名为主维度展示
  - 支持展开/折叠显示路径
  - 每个路径显示独立的状态指标

### 6.2 修改 SLO 详情组件

- **文件**: `atlhyper_web/src/app/[locale]/workbench/slo/components/DomainDetail.tsx`
- **状态**: [ ] 待开始
- **描述**:
  - 显示域名下所有路径的详细指标
  - 路径级别的历史趋势图
  - 路径级别的目标配置

### 6.3 更新 API 调用

- **文件**: `atlhyper_web/src/app/[locale]/workbench/slo/api.ts`
- **状态**: [ ] 待开始
- **描述**: 适配新的 API 响应格式

### 6.4 更新类型定义

- **文件**: `atlhyper_web/src/types/slo.ts`
- **状态**: [ ] 待开始
- **描述**: 更新 TypeScript 类型定义

```typescript
interface DomainSLO {
  domain: string;
  tls: boolean;
  routes: RouteSLO[];
  summary: SLOMetrics;
  status: 'healthy' | 'warning' | 'critical';
  error_budget_remaining: number;
}

interface RouteSLO {
  path_prefix: string;
  ingress_name: string;
  namespace: string;
  current: SLOMetrics;
  previous?: SLOMetrics;
  targets?: Record<string, SLOTarget>;
  status: 'healthy' | 'warning' | 'critical';
}
```

---

## Phase 7: 清理与优化

### 7.1 清理 entrypoint 历史数据

- **状态**: [ ] 待开始
- **描述**: 删除 `web` / `websecure` 等 entrypoint 级别的历史数据

```sql
DELETE FROM slo_metrics_raw WHERE host IN ('web', 'websecure');
DELETE FROM slo_metrics_hourly WHERE host IN ('web', 'websecure');
DELETE FROM ingress_counter_snapshot WHERE host IN ('web', 'websecure');
DELETE FROM ingress_histogram_snapshot WHERE host IN ('web', 'websecure');
```

### 7.2 数据迁移脚本

- **状态**: [ ] 待开始
- **描述**: 将旧的 `host` (service 名称) 数据关联到 `domain` / `path_prefix`
- **逻辑**:
  1. 遍历 `slo_route_mapping` 表
  2. 更新 `slo_metrics_raw` 和 `slo_metrics_hourly` 中对应记录的 `domain` 和 `path_prefix`

### 7.3 移除调试日志

- **状态**: [ ] 待开始
- **描述**: 移除开发阶段添加的调试日志
- **文件**:
  - `atlhyper_master_v2/slo/processor.go`
  - `atlhyper_master_v2/slo/calculator.go`
  - `atlhyper_agent_v2/sdk/impl/scraper.go`

---

## 验收标准

### 功能验收

- [ ] Agent 能正确采集 IngressRoute 配置
- [ ] Agent 只上报 service 级别指标，不上报 entrypoint 级别
- [ ] Master 能正确存储 service → domain/path 映射
- [ ] API 返回按域名分组的数据
- [ ] 前端正确显示域名和路径层级
- [ ] 无 `web` / `websecure` / `xxx@kubernetes` 这类不直观的名称

### 数据验收

```
期望的前端展示:

atlantis.example.com                    [健康]
├── /         可用性 99.9%  P95 145ms
└── /api      可用性 99.5%  P95 320ms

atlhyper.example.com                    [健康]
├── /         可用性 100%   P95 98ms
├── /api      可用性 99.8%  P95 156ms
└── /ws       可用性 100%   P95 12ms
```

### 性能验收

- [ ] IngressRoute 采集不影响现有 Snapshot 采集性能
- [ ] API 响应时间 < 500ms
- [ ] 数据库查询有正确的索引支持

---

## 风险与依赖

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| IngressRoute CRD 版本差异 | 不同 Traefik 版本 CRD 可能不同 | 支持 v1alpha1 和 v1 两个版本 |
| match 规则解析复杂 | 可能有复杂的 match 表达式 | 只支持 Host() 和 PathPrefix()，其他忽略 |
| 集群无 IngressRoute | 部分集群可能直接用 Ingress | 降级到 service 名称显示 |
