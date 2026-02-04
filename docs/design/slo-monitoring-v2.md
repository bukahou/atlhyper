# SLO Monitoring V2 设计文档

> Version: 2.0
> Date: 2026-02-01
> Author: AtlHyper Team

---

## 1. 概述

### 1.1 背景

当前 SLO 监控存在以下问题：

| 问题 | 示例 | 影响 |
|------|------|------|
| Service 名称不直观 | `atlantis-atlantis-web-3000@kubernetes` | 用户无法快速识别服务 |
| Entrypoint 数据无意义 | `web` / `websecure` | 是所有服务的聚合，不应单独展示 |
| 缺少域名维度 | - | 用户关心的是"我的域名是否正常" |
| 缺少路径维度 | - | 无法区分 `/api` 和 `/static` 的性能差异 |

### 1.2 目标

以**域名 + 路径**作为 SLO 监控的核心维度：

```
域名: atlantis.example.com
├── /           → 可用性 99.9%, P95 145ms
├── /api/*      → 可用性 99.5%, P95 320ms
└── /static/*   → 可用性 100%, P95 50ms

域名: atlhyper.example.com
├── /
├── /api/*
└── /ws/*
```

### 1.3 核心变更

| 当前 | 目标 |
|------|------|
| Traefik service 名称 | 实际域名 (Host) |
| 无路径维度 | IngressRoute 定义的路径 |
| 显示 entrypoint | 过滤掉 entrypoint 级别数据 |

---

## 2. 架构设计

### 2.1 数据流

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              数据采集与映射                                   │
└─────────────────────────────────────────────────────────────────────────────┘

                    Kubernetes Cluster
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│  IngressRoute │  │    Traefik    │  │   Traefik     │
│  (CRD 配置)    │  │   /metrics    │  │   /metrics    │
│               │  │  (service)    │  │ (entrypoint)  │
│ • host        │  │               │  │               │
│ • path        │  │ 采集 ✓        │  │ 过滤 ✗        │
│ • service     │  │               │  │               │
└───────┬───────┘  └───────┬───────┘  └───────────────┘
        │                  │
        │                  │
        ▼                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Agent                                           │
│                                                                              │
│  1. 采集 IngressRoute，构建映射：                                             │
│     service_name → { domain, path, namespace, tls }                         │
│                                                                              │
│  2. 采集 Traefik service metrics（过滤 entrypoint）                          │
│                                                                              │
│  3. 关联数据：将 service metrics 与 IngressRoute 映射关联                     │
│                                                                              │
└───────────────────────────────────────┬─────────────────────────────────────┘
                                        │
                                        │ 上报 Master
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Master                                          │
│                                                                              │
│  存储结构：                                                                   │
│  • domain (域名)                                                             │
│  • path (路径前缀)                                                            │
│  • metrics (可用性、延迟、错误率)                                              │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 IngressRoute 解析

Traefik IngressRoute CRD 示例：

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: atlantis-web
  namespace: atlantis
spec:
  entryPoints:
    - websecure
  routes:
    - match: Host(`atlantis.example.com`) && PathPrefix(`/api`)
      kind: Rule
      services:
        - name: atlantis-api
          port: 8080
    - match: Host(`atlantis.example.com`) && PathPrefix(`/`)
      kind: Rule
      services:
        - name: atlantis-web
          port: 3000
  tls:
    secretName: atlantis-tls
```

解析后的映射关系：

| Service (Traefik 内部名) | Domain | Path | Namespace | TLS |
|-------------------------|--------|------|-----------|-----|
| `atlantis-atlantis-api-8080@kubernetes` | `atlantis.example.com` | `/api` | `atlantis` | `true` |
| `atlantis-atlantis-web-3000@kubernetes` | `atlantis.example.com` | `/` | `atlantis` | `true` |

### 2.3 数据过滤规则

**采集**：
- `traefik_service_requests_total` - Service 级别指标 ✓
- `traefik_service_request_duration_seconds` - Service 级别延迟 ✓

**过滤**：
- `traefik_entrypoint_*` - Entrypoint 级别指标 ✗
- 无法映射到 IngressRoute 的 service ✗

---

## 3. 数据模型

### 3.1 新增表：slo_route_mapping

存储 IngressRoute 配置与 Traefik service 的映射关系。

```sql
CREATE TABLE slo_route_mapping (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id      TEXT NOT NULL,
    -- IngressRoute 信息
    domain          TEXT NOT NULL,              -- 域名 (从 Host() 解析)
    path_prefix     TEXT NOT NULL DEFAULT '/',  -- 路径前缀 (从 PathPrefix() 解析)
    ingress_name    TEXT NOT NULL,              -- IngressRoute 名称
    namespace       TEXT NOT NULL,              -- 命名空间
    tls             INTEGER NOT NULL DEFAULT 1, -- 是否 TLS
    -- Traefik 内部标识
    service_key     TEXT NOT NULL,              -- Traefik service 名称 (如 atlantis-atlantis-web-3000@kubernetes)
    -- 元信息
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,

    UNIQUE(cluster_id, service_key)
);

CREATE INDEX idx_route_mapping_domain ON slo_route_mapping(cluster_id, domain);
```

### 3.2 修改表：slo_metrics_raw

增加 `domain` 和 `path_prefix` 字段，替代原来的 `host`（Traefik service 名称）。

```sql
-- 新增列
ALTER TABLE slo_metrics_raw ADD COLUMN domain TEXT;
ALTER TABLE slo_metrics_raw ADD COLUMN path_prefix TEXT DEFAULT '/';

-- 迁移数据：通过 slo_route_mapping 关联
UPDATE slo_metrics_raw SET
    domain = (SELECT domain FROM slo_route_mapping WHERE service_key = slo_metrics_raw.host),
    path_prefix = (SELECT path_prefix FROM slo_route_mapping WHERE service_key = slo_metrics_raw.host)
WHERE EXISTS (SELECT 1 FROM slo_route_mapping WHERE service_key = slo_metrics_raw.host);
```

### 3.3 修改表：slo_metrics_hourly

同样增加 `domain` 和 `path_prefix` 字段。

---

## 4. Agent 改动

### 4.1 新增：IngressRoute 采集器

```go
// atlhyper_agent_v2/sdk/interfaces.go

type IngressRouteCollector interface {
    // CollectRoutes 采集所有 IngressRoute 配置
    CollectRoutes(ctx context.Context) ([]IngressRouteInfo, error)
}

type IngressRouteInfo struct {
    Name        string   // IngressRoute 名称
    Namespace   string   // 命名空间
    Domain      string   // 域名 (从 match 规则解析)
    PathPrefix  string   // 路径前缀
    ServiceKey  string   // Traefik service 标识
    TLS         bool     // 是否启用 TLS
}
```

### 4.2 修改：MetricsScraper

过滤 entrypoint 级别的数据：

```go
// atlhyper_agent_v2/sdk/impl/scraper.go

func (s *metricsScraper) parseTraefikMetric(line string) *cleanedMetric {
    // 只采集 service 级别指标
    if strings.HasPrefix(line, "traefik_service_requests_total{") {
        return s.parseTraefikCounterLine(line)
    }
    if strings.HasPrefix(line, "traefik_service_request_duration_seconds_bucket{") {
        return s.parseHistogramBucketLine(line)
    }

    // 过滤 entrypoint 级别指标
    // traefik_entrypoint_* → 忽略

    return nil
}
```

### 4.3 修改：ClusterSnapshot

新增 IngressRoute 信息：

```go
// model/snapshot.go

type ClusterSnapshot struct {
    // ... 现有字段 ...

    IngressMetrics  *IngressMetrics   `json:"ingress_metrics,omitempty"`
    IngressRoutes   []IngressRouteInfo `json:"ingress_routes,omitempty"` // 新增
}
```

---

## 5. Master 改动

### 5.1 新增：RouteMapping 处理

```go
// atlhyper_master_v2/slo/route_mapper.go

type RouteMapper struct {
    repo database.SLORepository
}

// UpdateMappings 更新 IngressRoute 映射
func (m *RouteMapper) UpdateMappings(ctx context.Context, clusterID string, routes []IngressRouteInfo) error {
    for _, route := range routes {
        mapping := &database.SLORouteMapping{
            ClusterID:   clusterID,
            Domain:      route.Domain,
            PathPrefix:  route.PathPrefix,
            IngressName: route.Name,
            Namespace:   route.Namespace,
            ServiceKey:  route.ServiceKey,
            TLS:         route.TLS,
        }
        if err := m.repo.UpsertRouteMapping(ctx, mapping); err != nil {
            return err
        }
    }
    return nil
}

// ResolveServiceKey 将 Traefik service 名称解析为 domain + path
func (m *RouteMapper) ResolveServiceKey(ctx context.Context, clusterID, serviceKey string) (*ResolvedRoute, error) {
    mapping, err := m.repo.GetRouteMappingByServiceKey(ctx, clusterID, serviceKey)
    if err != nil {
        return nil, err
    }
    return &ResolvedRoute{
        Domain:     mapping.Domain,
        PathPrefix: mapping.PathPrefix,
        TLS:        mapping.TLS,
    }, nil
}
```

### 5.2 修改：SLO Processor

写入 raw 数据时关联域名和路径：

```go
// atlhyper_master_v2/slo/processor.go

func (p *Processor) processCounterMetrics(ctx context.Context, clusterID string, counters []IngressCounterMetric, now time.Time) error {
    for _, c := range counters {
        // 解析 service 名称为 domain + path
        resolved, err := p.routeMapper.ResolveServiceKey(ctx, clusterID, c.Host)
        if err != nil {
            // 无法映射的 service 跳过（可能是 entrypoint 级别的漏网之鱼）
            continue
        }

        // 写入时使用 domain 和 path
        raw := &database.SLOMetricsRaw{
            ClusterID:     clusterID,
            Domain:        resolved.Domain,      // 新字段
            PathPrefix:    resolved.PathPrefix,  // 新字段
            Host:          c.Host,               // 保留原始 service 名称用于关联
            // ... 其他字段 ...
        }
        // ...
    }
}
```

### 5.3 修改：API Handler

按域名和路径返回数据：

```go
// GET /api/v2/slo/domains
// 返回按域名分组的数据，每个域名下有多个路径

type DomainSLOResponse struct {
    Domain string      `json:"domain"`
    TLS    bool        `json:"tls"`
    Routes []RouteSLO  `json:"routes"`
    // 域名级别的聚合指标
    Summary *SLOMetrics `json:"summary"`
}

type RouteSLO struct {
    PathPrefix string      `json:"path_prefix"`
    Current    *SLOMetrics `json:"current"`
    Status     string      `json:"status"`
}
```

---

## 6. API 设计

### 6.1 GET /api/v2/slo/domains

返回按域名分组的 SLO 数据：

```json
{
  "domains": [
    {
      "domain": "atlantis.example.com",
      "tls": true,
      "routes": [
        {
          "path_prefix": "/",
          "current": {
            "availability": 99.9,
            "p95_latency": 145,
            "p99_latency": 241,
            "error_rate": 0.1,
            "requests_per_sec": 1.136,
            "total_requests": 159
          },
          "status": "healthy"
        },
        {
          "path_prefix": "/api",
          "current": {
            "availability": 99.5,
            "p95_latency": 320,
            "p99_latency": 580,
            "error_rate": 0.5,
            "requests_per_sec": 50.2,
            "total_requests": 12500
          },
          "status": "warning"
        }
      ],
      "summary": {
        "availability": 99.7,
        "p95_latency": 280,
        "error_rate": 0.3,
        "total_requests": 12659
      },
      "error_budget_remaining": 85.0,
      "status": "healthy"
    }
  ],
  "summary": {
    "total_domains": 5,
    "healthy_count": 4,
    "warning_count": 1,
    "critical_count": 0
  }
}
```

### 6.2 GET /api/v2/slo/domains/{domain}

获取单个域名的详细信息：

```json
{
  "domain": "atlantis.example.com",
  "tls": true,
  "routes": [
    {
      "path_prefix": "/",
      "ingress_name": "atlantis-web",
      "namespace": "atlantis",
      "current": { ... },
      "previous": { ... },
      "targets": {
        "1d": { "availability": 95, "p95_latency": 300 }
      }
    }
  ]
}
```

---

## 7. 前端改动

### 7.1 域名列表页

```
┌─────────────────────────────────────────────────────────────────┐
│  SLO 监控                                     [日] [周] [月]    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  atlantis.example.com                              [健康]       │
│  ├── /         可用性 99.9%  P95 145ms   1.1 req/s            │
│  └── /api      可用性 99.5%  P95 320ms   50 req/s   [警告]    │
│                                                                 │
│  atlhyper.example.com                              [健康]       │
│  ├── /         可用性 100%   P95 98ms    1.8 req/s            │
│  ├── /api      可用性 99.8%  P95 156ms   120 req/s            │
│  └── /ws       可用性 100%   P95 12ms    5 req/s              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 域名详情页

点击域名展开，显示各路径的详细指标和历史趋势。

---

## 8. 实现计划

### Phase 1: 基础映射 (优先级 P0)

1. Agent: 实现 IngressRoute 采集器
2. Agent: 过滤 entrypoint 级别指标
3. Master: 新增 slo_route_mapping 表
4. Master: 实现 RouteMapper

### Phase 2: 数据关联 (优先级 P0)

1. Master: 修改 SLO Processor，写入时关联 domain/path
2. Master: 修改 Aggregator，按 domain/path 聚合
3. Master: 修改 API Handler，返回新格式

### Phase 3: 前端适配 (优先级 P1)

1. 修改域名列表组件，支持展开路径
2. 修改详情页，按路径展示
3. 更新目标配置，支持路径级别配置

### Phase 4: 数据迁移 (优先级 P2)

1. 迁移脚本：旧数据关联到新的 domain/path
2. 清理：删除 entrypoint 级别的历史数据

---

## 9. 兼容性

### 9.1 向后兼容

- 保留 `host` 字段（Traefik service 名称）用于内部关联
- 新增 `domain` 和 `path_prefix` 字段用于展示
- API 返回新格式，旧字段标记为 deprecated

### 9.2 降级处理

如果 IngressRoute 采集失败：
- 使用 service 名称作为 fallback
- 日志警告提示配置问题
