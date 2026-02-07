# SLO OTel Agent 设计书

## 概要

本文档描述 AtlHyper Agent 的 SLO 数据采集设计方案。**设计遵循现有 Agent 架构模式：SDK → Repository → Service**，将 SLO 采集无缝集成到现有的 SnapshotService 中，与其他 K8s 资源一起并发采集和上报。

### 旧架构重构说明

旧的 Traefik 直连采集存在以下架构问题，已在本次重构中修复：

| 问题 | 旧架构 | 新架构 |
|------|--------|--------|
| SDK 接口分散 | 单独的 `interfaces_metrics.go` | 统一到 `interfaces.go` |
| SDK 实现无隔离 | `scraper.go` 混在 `impl/` 下 | 独立 `impl/ingress/` 子目录 |
| 跳过 Repository | Service 直接调用 SDK | Service → Repository → SDK |
| Service 接口未注册 | 接口定义在实现文件中 | 统一到 `service/interfaces.go` |
| 独立调度循环 | `runSLOLoop()` 独立推送 | 合入 `runSnapshotLoop()` 统一推送 |

### 数据源规划

| 阶段 | 数据源 | 说明 |
|------|--------|------|
| 当前 | Traefik Pod 直连 (`:9100/metrics`) | IngressClient → SLORepository |
| 下一步 | OTel Collector (`:8889/metrics`) | 扩展 IngressClient 或新增 OTelClient |

两种数据源共享相同的 Repository → Service → Scheduler 链路，仅 SDK 层实现不同。

---

## 1. 设计目标

### 1.1 功能目标

| 目标 | 说明 |
|------|------|
| OTel 数据源 | 从 OTel Collector 8889 端口采集 Prometheus 格式指标 |
| 服务发现 | 自动发现服务和服务间调用关系 |
| 黄金指标 | 采集 RPS、成功率、延迟分布 |
| 架构一致 | 遵循现有 SDK → Repository → Service 架构 |
| 统一上报 | SLO 数据随 ClusterSnapshot 一起上报 Master |

### 1.2 数据来源

| 数据源 | 端点 | 指标类型 |
|--------|------|----------|
| OTel Collector | `otel-collector.otel:8889/metrics` | Prometheus 格式 |
| 原始来源 | Linkerd Prometheus (federate) | 服务网格指标 |
| 原始来源 | Traefik metrics | 入口网关指标 |

---

## 2. 架构设计

### 2.1 架构位置

遵循现有 Agent 架构模式：

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              AtlHyper Agent 架构                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌───────────────────────────────────────────────────────────────────┐    │
│   │                        Service Layer                              │    │
│   │                                                                   │    │
│   │   snapshot_service.go                                             │    │
│   │   ├── podRepo.List()                                              │    │
│   │   ├── nodeRepo.List()                                             │    │
│   │   ├── deploymentRepo.List()                                       │    │
│   │   ├── ...（20 个 K8s Repository）                                 │    │
│   │   └── sloRepo.Collect()        ← 新增：SLO 数据采集               │    │
│   │                                                                   │    │
│   └───────────────────────────────────────────────────────────────────┘    │
│                               ↓ 调用                                        │
│   ┌───────────────────────────────────────────────────────────────────┐    │
│   │                       Repository Layer                            │    │
│   │                                                                   │    │
│   │   pod_repository.go       → 使用 K8sClient                        │    │
│   │   node_repository.go      → 使用 K8sClient                        │    │
│   │   ...                                                             │    │
│   │   slo_repository.go       → 使用 OTelClient   ← 新增              │    │
│   │                                                                   │    │
│   └───────────────────────────────────────────────────────────────────┘    │
│                               ↓ 调用                                        │
│   ┌───────────────────────────────────────────────────────────────────┐    │
│   │                          SDK Layer                                │    │
│   │                                                                   │    │
│   │   interfaces.go                                                   │    │
│   │   ├── K8sClient interface   → client-go → K8s API Server         │    │
│   │   └── OTelClient interface  → HTTP → OTel Collector   ← 新增     │    │
│   │                                                                   │    │
│   │   impl/                                                           │    │
│   │   ├── k8s_client.go         → K8sClient 实现                      │    │
│   │   └── otel_client.go        → OTelClient 实现        ← 新增       │    │
│   │                                                                   │    │
│   └───────────────────────────────────────────────────────────────────┘    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 文件夹结构

```
atlhyper_agent_v2/
├── sdk/
│   ├── interfaces.go             # 统一接口 (K8sClient + IngressClient)
│   ├── types.go                  # 统一类型 (含 IngressMetrics 等)
│   └── impl/
│       ├── client.go             # K8sClient 实现入口
│       ├── core.go               # corev1 操作
│       ├── apps.go               # appsv1 操作
│       ├── batch.go              # batchv1 操作
│       ├── networking.go         # networkingv1 操作
│       ├── metrics.go            # metrics-server 操作
│       ├── generic.go            # 通用操作
│       └── ingress/              # IngressClient 实现（独立子目录）
│           ├── client.go         # 实现入口 + ScrapeMetrics
│           ├── discover.go       # DiscoverURL 自动发现
│           ├── parser.go         # Prometheus 文本解析
│           └── route_collector.go # IngressRoute CRD 采集
│
├── repository/
│   ├── interfaces.go             # 统一接口（含 SLORepository）
│   ├── pod_repository.go         # 现有
│   ├── ...                       # 其他 K8s Repository
│   ├── slo_repository.go         # SLO 数据仓库（调用 IngressClient）
│   └── slo_snapshot.go           # Counter 快照管理（增量计算）
│
├── service/
│   ├── interfaces.go             # 统一接口（含 SLOService）
│   ├── snapshot_service.go       # 快照服务（含 sloRepo 并发采集）
│   └── slo_service.go            # SLO 服务（依赖 SLORepository）
│
├── scheduler/
│   └── scheduler.go              # 调度器（SLO 随 Snapshot 统一推送）
│
├── model/
│   └── slo.go                    # SLO 类型别名
│
└── agent.go                      # 初始化和依赖注入
```

### 2.3 数据流

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              数据采集流程                                     │
└─────────────────────────────────────────────────────────────────────────────┘

                              OTel Collector
                                   │
                                   │ Prometheus format (:8889/metrics)
                                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  SDK Layer: OTelClient                                                      │
│                                                                             │
│  otel_client.go                                                             │
│  ├── ScrapeMetrics(ctx) ([]RawMetric, error)                               │
│  │   └── HTTP GET → 解析 Prometheus 文本 → 返回原始指标                     │
│  │                                                                          │
│  └── 返回原始 Prometheus 指标列表                                            │
└──────────────────────────────────┬──────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  Repository Layer: SLORepository                                            │
│                                                                             │
│  slo_repository.go                                                          │
│  ├── Collect(ctx) (*SLOData, error)                                        │
│  │   ├── 调用 otelClient.ScrapeMetrics()                                   │
│  │   ├── 过滤 SLO 相关指标 (response_total, traefik_*)                     │
│  │   ├── 调用 snapshotManager.CalculateDelta() 计算增量                    │
│  │   ├── 服务发现：从标签提取服务和拓扑                                      │
│  │   ├── 黄金指标：计算 RPS、成功率、延迟分布                                │
│  │   └── 返回 SLOData                                                       │
│  │                                                                          │
│  slo_snapshot.go                                                            │
│  └── snapshotManager                                                        │
│      ├── prev map[string]float64   # 上一次快照                             │
│      └── CalculateDelta(cur) []MetricDelta                                 │
└──────────────────────────────────┬──────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  Service Layer: SnapshotService                                             │
│                                                                             │
│  snapshot_service.go                                                        │
│  ├── Collect(ctx) (*ClusterSnapshot, error)                                │
│  │   ├── 并发采集 20 种 K8s 资源                                            │
│  │   ├── 并发采集 SLO 数据 (sloRepo.Collect)                               │
│  │   └── 组装 ClusterSnapshot（含 SLOData）                                │
│  │                                                                          │
│  └── ClusterSnapshot 通过 DataHub 上报 Master                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. SDK 层设计

### 3.1 OTelClient 接口

**文件**: `sdk/interfaces_otel.go`

```go
// Package sdk 封装外部客户端
//
// interfaces_otel.go - OTelClient 接口定义
//
// OTelClient 封装与 OTel Collector Prometheus 端点的交互。
// 与 K8sClient 类似，Repository 层只依赖此接口。
//
// 架构位置:
//
//	SLORepository
//	    ↓ 调用
//	SDK (OTelClient) ← OTel 客户端封装
//	    ↓ 使用
//	net/http         ← Go 标准库
//	    ↓
//	OTel Collector (:8889/metrics)
package sdk

import (
	"context"
)

// OTelClient OTel Collector 客户端接口
//
// 封装从 OTel Collector Prometheus 端点采集指标的操作。
// Repository 层只依赖此接口，不直接使用 HTTP。
type OTelClient interface {
	// ScrapeMetrics 从 OTel Collector 采集原始指标
	//
	// 返回 Prometheus 格式的原始指标列表。
	// 调用方（Repository）负责过滤和处理。
	ScrapeMetrics(ctx context.Context) ([]RawMetric, error)

	// IsHealthy 检查 OTel Collector 是否健康
	IsHealthy(ctx context.Context) bool
}
```

### 3.2 OTel 类型定义

**文件**: `sdk/types_otel.go`

```go
package sdk

// RawMetric Prometheus 原始指标
//
// 从 Prometheus 文本格式解析出的单个指标。
// 包含指标名、标签和值，不做业务处理。
type RawMetric struct {
	// Name 指标名称
	// 例如: response_total, response_latency_ms_bucket
	Name string

	// Labels 标签 map
	// 例如: {"namespace": "default", "deployment": "nginx", "status_code": "200"}
	Labels map[string]string

	// Value 指标值
	Value float64
}

// OTelConfig OTel 客户端配置
type OTelConfig struct {
	// Endpoint OTel Collector Prometheus 端点
	// 例如: http://otel-collector.otel.svc:8889/metrics
	Endpoint string

	// Timeout HTTP 请求超时
	Timeout string // "5s"

	// Enabled 是否启用
	Enabled bool
}
```

### 3.3 OTelClient 实现

**文件**: `sdk/impl/otel_client.go`

```go
package impl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_agent_v2/sdk"
)

// otelClient OTelClient 实现
type otelClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewOTelClient 创建 OTel 客户端
func NewOTelClient(cfg sdk.OTelConfig) (sdk.OTelClient, error) {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		timeout = 5 * time.Second
	}

	return &otelClient{
		endpoint: cfg.Endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// ScrapeMetrics 采集原始指标
func (c *otelClient) ScrapeMetrics(ctx context.Context) ([]sdk.RawMetric, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 OTel Collector 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OTel Collector 返回 %d", resp.StatusCode)
	}

	return c.parsePrometheus(resp.Body)
}

// IsHealthy 检查健康状态
func (c *otelClient) IsHealthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint, nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// parsePrometheus 解析 Prometheus 文本格式
//
// 格式示例:
// response_total{namespace="default",deployment="nginx",status_code="200"} 1234
func (c *otelClient) parsePrometheus(r io.Reader) ([]sdk.RawMetric, error) {
	var metrics []sdk.RawMetric

	// 正则: 指标名{标签} 值
	metricRegex := regexp.MustCompile(`^([a-zA-Z_:][a-zA-Z0-9_:]*)\{([^}]*)\}\s+(.+)$`)
	// 无标签: 指标名 值
	simpleRegex := regexp.MustCompile(`^([a-zA-Z_:][a-zA-Z0-9_:]*)\s+(.+)$`)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过注释和空行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var metric sdk.RawMetric

		if matches := metricRegex.FindStringSubmatch(line); matches != nil {
			metric.Name = matches[1]
			metric.Labels = parseLabels(matches[2])
			if v, err := strconv.ParseFloat(matches[3], 64); err == nil {
				metric.Value = v
			}
			metrics = append(metrics, metric)
		} else if matches := simpleRegex.FindStringSubmatch(line); matches != nil {
			metric.Name = matches[1]
			metric.Labels = make(map[string]string)
			if v, err := strconv.ParseFloat(matches[2], 64); err == nil {
				metric.Value = v
			}
			metrics = append(metrics, metric)
		}
	}

	return metrics, scanner.Err()
}

// parseLabels 解析标签字符串
// 输入: namespace="default",deployment="nginx"
// 输出: map[namespace:default deployment:nginx]
func parseLabels(s string) map[string]string {
	labels := make(map[string]string)
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), `"`)
			labels[key] = val
		}
	}
	return labels
}
```

---

## 4. Repository 层设计

### 4.1 SLO 数据模型

**文件**: `model/slo.go`

```go
package model

import "time"

// SLOData SLO 采集数据
//
// 包含服务发现、黄金指标、Traefik 指标。
// 随 ClusterSnapshot 一起上报 Master。
type SLOData struct {
	// 采集时间
	Timestamp time.Time

	// 服务发现
	Services []ServiceInfo
	Edges    []ServiceEdge

	// Linkerd 黄金指标（增量）
	GoldenMetrics []GoldenMetric

	// Traefik 入口指标（增量）
	TraefikMetrics []TraefikMetric
}

// ServiceInfo 发现的服务
type ServiceInfo struct {
	Namespace   string
	Name        string
	ServiceType string // gateway / service / database / cache
}

// ServiceEdge 服务调用边
type ServiceEdge struct {
	SourceNs   string
	SourceName string
	TargetNs   string
	TargetName string
	Protocol   string
}

// GoldenMetric Linkerd 黄金指标
type GoldenMetric struct {
	TargetNs   string
	TargetName string
	SourceNs   string // 调用方（可为空）
	SourceName string

	// 请求统计（增量）
	TotalReq   int64
	SuccessReq int64
	ErrorReq   int64

	// 延迟分布（增量）- Histogram buckets
	Buckets      map[string]int64 // "1ms", "2ms", ..., "inf"
	LatencySum   float64
	LatencyCount int64
}

// TraefikMetric Traefik 入口指标
type TraefikMetric struct {
	Service string
	Method  string
	Code    string

	// 请求统计（增量）
	TotalReq int64

	// 延迟分布（增量）
	Buckets map[string]int64
}
```

### 4.2 SLO Repository 接口

**文件**: `repository/interfaces.go` (新增接口)

```go
// SLORepository SLO 数据仓库接口
//
// 负责从 OTel Collector 采集 SLO 数据，处理后返回。
// 与其他 Repository 一样，被 SnapshotService 调用。
type SLORepository interface {
	// Collect 采集 SLO 数据
	//
	// 从 OTel Collector 采集指标，计算增量，返回处理后的 SLO 数据。
	// 如果 OTel Collector 不可用，返回空数据和 error。
	Collect(ctx context.Context) (*model.SLOData, error)
}
```

### 4.3 SLO Repository 实现

**文件**: `repository/slo_repository.go`

```go
package repository

import (
	"context"
	"strings"
	"time"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/sdk"
)

// sloRepository SLO 数据仓库实现
type sloRepository struct {
	otelClient      sdk.OTelClient
	snapshotManager *SLOSnapshotManager
	ignoreNS        map[string]bool // 忽略的 namespace
}

// NewSLORepository 创建 SLO 仓库
func NewSLORepository(otelClient sdk.OTelClient, ignoreNS []string) SLORepository {
	ignore := make(map[string]bool)
	for _, ns := range ignoreNS {
		ignore[ns] = true
	}

	return &sloRepository{
		otelClient:      otelClient,
		snapshotManager: NewSLOSnapshotManager(),
		ignoreNS:        ignore,
	}
}

// Collect 采集 SLO 数据
func (r *sloRepository) Collect(ctx context.Context) (*model.SLOData, error) {
	// 1. 从 OTel Collector 采集原始指标
	rawMetrics, err := r.otelClient.ScrapeMetrics(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 过滤 SLO 相关指标
	filtered := r.filterSLOMetrics(rawMetrics)

	// 3. 计算增量
	deltas := r.snapshotManager.CalculateDelta(filtered)

	// 4. 处理数据
	data := &model.SLOData{
		Timestamp: time.Now(),
	}

	// 服务发现
	data.Services = r.discoverServices(filtered)
	data.Edges = r.discoverEdges(filtered)

	// 黄金指标（基于增量）
	data.GoldenMetrics = r.calculateGoldenMetrics(deltas)

	// Traefik 指标（基于增量）
	data.TraefikMetrics = r.calculateTraefikMetrics(deltas)

	return data, nil
}

// filterSLOMetrics 过滤 SLO 相关指标
func (r *sloRepository) filterSLOMetrics(metrics []sdk.RawMetric) []sdk.RawMetric {
	var result []sdk.RawMetric
	for _, m := range metrics {
		// 只保留 response_* 和 traefik_* 指标
		if !strings.HasPrefix(m.Name, "response_") &&
			!strings.HasPrefix(m.Name, "traefik_") {
			continue
		}

		// 过滤忽略的 namespace
		if ns := m.Labels["namespace"]; ns != "" && r.ignoreNS[ns] {
			continue
		}

		result = append(result, m)
	}
	return result
}

// discoverServices 服务发现
func (r *sloRepository) discoverServices(metrics []sdk.RawMetric) []model.ServiceInfo {
	seen := make(map[string]model.ServiceInfo)

	for _, m := range metrics {
		if m.Name != "response_total" {
			continue
		}

		// inbound: 被调用方
		ns := m.Labels["namespace"]
		deploy := m.Labels["deployment"]
		if ns != "" && deploy != "" {
			key := ns + "/" + deploy
			seen[key] = model.ServiceInfo{
				Namespace:   ns,
				Name:        deploy,
				ServiceType: inferServiceType(deploy),
			}
		}

		// outbound: 调用目标
		if m.Labels["direction"] == "outbound" {
			dstNs := m.Labels["dst_namespace"]
			dstSvc := m.Labels["dst_service"]
			if dstNs != "" && dstSvc != "" {
				key := dstNs + "/" + dstSvc
				seen[key] = model.ServiceInfo{
					Namespace:   dstNs,
					Name:        dstSvc,
					ServiceType: inferServiceType(dstSvc),
				}
			}
		}
	}

	result := make([]model.ServiceInfo, 0, len(seen))
	for _, svc := range seen {
		result = append(result, svc)
	}
	return result
}

// discoverEdges 拓扑边发现
func (r *sloRepository) discoverEdges(metrics []sdk.RawMetric) []model.ServiceEdge {
	seen := make(map[string]model.ServiceEdge)

	for _, m := range metrics {
		if m.Name != "response_total" || m.Labels["direction"] != "outbound" {
			continue
		}

		sourceNs := m.Labels["namespace"]
		sourceName := m.Labels["deployment"]
		targetNs := m.Labels["dst_namespace"]
		targetName := m.Labels["dst_service"]

		if sourceNs == "" || sourceName == "" || targetNs == "" || targetName == "" {
			continue
		}

		key := sourceNs + "/" + sourceName + "->" + targetNs + "/" + targetName
		seen[key] = model.ServiceEdge{
			SourceNs:   sourceNs,
			SourceName: sourceName,
			TargetNs:   targetNs,
			TargetName: targetName,
			Protocol:   "http",
		}
	}

	result := make([]model.ServiceEdge, 0, len(seen))
	for _, edge := range seen {
		result = append(result, edge)
	}
	return result
}

// calculateGoldenMetrics 计算黄金指标
func (r *sloRepository) calculateGoldenMetrics(deltas []MetricDelta) []model.GoldenMetric {
	// 按 (target_ns, target_name, source_ns, source_name) 分组
	type key struct {
		TargetNs, TargetName, SourceNs, SourceName string
	}
	groups := make(map[key]*model.GoldenMetric)

	for _, d := range deltas {
		if !strings.HasPrefix(d.Name, "response_") {
			continue
		}

		k := key{
			TargetNs:   d.Labels["namespace"],
			TargetName: d.Labels["deployment"],
		}
		if d.Labels["direction"] == "outbound" {
			k.SourceNs = d.Labels["namespace"]
			k.SourceName = d.Labels["deployment"]
			k.TargetNs = d.Labels["dst_namespace"]
			k.TargetName = d.Labels["dst_service"]
		}

		gm, ok := groups[k]
		if !ok {
			gm = &model.GoldenMetric{
				TargetNs:   k.TargetNs,
				TargetName: k.TargetName,
				SourceNs:   k.SourceNs,
				SourceName: k.SourceName,
				Buckets:    make(map[string]int64),
			}
			groups[k] = gm
		}

		switch {
		case d.Name == "response_total":
			gm.TotalReq += int64(d.Delta)
			if isSuccess(d.Labels["status_code"]) {
				gm.SuccessReq += int64(d.Delta)
			} else {
				gm.ErrorReq += int64(d.Delta)
			}

		case strings.HasPrefix(d.Name, "response_latency_ms_bucket"):
			bucket := leToBucketName(d.Labels["le"])
			gm.Buckets[bucket] += int64(d.Delta)

		case d.Name == "response_latency_ms_sum":
			gm.LatencySum += d.Delta

		case d.Name == "response_latency_ms_count":
			gm.LatencyCount += int64(d.Delta)
		}
	}

	result := make([]model.GoldenMetric, 0, len(groups))
	for _, gm := range groups {
		result = append(result, *gm)
	}
	return result
}

// calculateTraefikMetrics 计算 Traefik 指标
func (r *sloRepository) calculateTraefikMetrics(deltas []MetricDelta) []model.TraefikMetric {
	type key struct {
		Service, Method, Code string
	}
	groups := make(map[key]*model.TraefikMetric)

	for _, d := range deltas {
		if !strings.HasPrefix(d.Name, "traefik_") {
			continue
		}

		k := key{
			Service: d.Labels["service"],
			Method:  d.Labels["method"],
			Code:    d.Labels["code"],
		}

		tm, ok := groups[k]
		if !ok {
			tm = &model.TraefikMetric{
				Service: k.Service,
				Method:  k.Method,
				Code:    k.Code,
				Buckets: make(map[string]int64),
			}
			groups[k] = tm
		}

		switch {
		case d.Name == "traefik_service_requests_total":
			tm.TotalReq += int64(d.Delta)

		case strings.HasPrefix(d.Name, "traefik_service_request_duration_seconds_bucket"):
			bucket := secondsToBucketName(d.Labels["le"])
			tm.Buckets[bucket] += int64(d.Delta)
		}
	}

	result := make([]model.TraefikMetric, 0, len(groups))
	for _, tm := range groups {
		result = append(result, *tm)
	}
	return result
}

// 辅助函数

func inferServiceType(name string) string {
	switch {
	case strings.Contains(name, "traefik"):
		return "gateway"
	case strings.Contains(name, "mysql"), strings.Contains(name, "postgres"), strings.Contains(name, "mongo"):
		return "database"
	case strings.Contains(name, "redis"), strings.Contains(name, "memcache"):
		return "cache"
	default:
		return "service"
	}
}

func isSuccess(code string) bool {
	return strings.HasPrefix(code, "2") || strings.HasPrefix(code, "3")
}

func leToBucketName(le string) string {
	// "0.001" -> "1ms", "0.1" -> "100ms", "+Inf" -> "inf"
	if le == "+Inf" {
		return "inf"
	}
	// 简化处理
	return le + "s"
}

func secondsToBucketName(le string) string {
	if le == "+Inf" {
		return "inf"
	}
	return le + "s"
}
```

### 4.4 快照管理器

**文件**: `repository/slo_snapshot.go`

```go
package repository

import (
	"sync"

	"AtlHyper/atlhyper_agent_v2/sdk"
)

// MetricDelta 指标增量
type MetricDelta struct {
	Name   string
	Labels map[string]string
	Delta  float64 // current - previous
}

// SLOSnapshotManager 快照管理器
//
// 维护上一次采集的 Counter 值，用于计算增量。
// Counter 类型指标（如 response_total）是累计值，需要计算增量才有意义。
type SLOSnapshotManager struct {
	mu   sync.RWMutex
	prev map[string]float64 // key: metric_name + sorted_labels
}

// NewSLOSnapshotManager 创建快照管理器
func NewSLOSnapshotManager() *SLOSnapshotManager {
	return &SLOSnapshotManager{
		prev: make(map[string]float64),
	}
}

// CalculateDelta 计算增量
//
// 对于 Counter 类型指标，计算 current - previous。
// 对于 Gauge 类型指标，直接返回当前值。
// 更新 prev 快照为当前值。
func (m *SLOSnapshotManager) CalculateDelta(current []sdk.RawMetric) []MetricDelta {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deltas []MetricDelta
	newPrev := make(map[string]float64)

	for _, metric := range current {
		key := m.metricKey(metric)
		newPrev[key] = metric.Value

		// 计算增量
		prevVal, hasPrev := m.prev[key]
		delta := metric.Value
		if hasPrev {
			delta = metric.Value - prevVal
			// Counter 重置检测：如果 delta < 0，说明 Counter 被重置
			if delta < 0 {
				delta = metric.Value // 使用当前值作为增量
			}
		}

		deltas = append(deltas, MetricDelta{
			Name:   metric.Name,
			Labels: metric.Labels,
			Delta:  delta,
		})
	}

	m.prev = newPrev
	return deltas
}

// metricKey 生成指标唯一键
func (m *SLOSnapshotManager) metricKey(metric sdk.RawMetric) string {
	// 简单实现：name + 排序后的 labels
	key := metric.Name
	for k, v := range metric.Labels {
		key += "|" + k + "=" + v
	}
	return key
}
```

---

## 5. Service 层修改

### 5.1 修改 ClusterSnapshot 模型

**文件**: `model_v2/cluster_snapshot.go` (修改)

```go
// ClusterSnapshot 集群快照
type ClusterSnapshot struct {
	ClusterID string
	FetchedAt time.Time

	// 现有 K8s 资源
	Pods        []Pod
	Nodes       []Node
	Deployments []Deployment
	// ... 其他资源

	// 新增：SLO 数据
	SLOData *model.SLOData
}
```

### 5.2 修改 SnapshotService

**文件**: `service/snapshot_service.go` (修改)

```go
// snapshotService 快照采集服务实现
type snapshotService struct {
	clusterID string

	// 现有 20 个 K8s Repository
	podRepo        repository.PodRepository
	nodeRepo       repository.NodeRepository
	deploymentRepo repository.DeploymentRepository
	// ...

	// 新增：SLO Repository（可选）
	sloRepo repository.SLORepository
}

// NewSnapshotService 创建快照服务
func NewSnapshotService(
	clusterID string,
	podRepo repository.PodRepository,
	// ... 现有参数
	sloRepo repository.SLORepository, // 新增，可为 nil
) SnapshotService {
	return &snapshotService{
		clusterID: clusterID,
		podRepo:   podRepo,
		// ...
		sloRepo: sloRepo,
	}
}

// Collect 采集集群快照
func (s *snapshotService) Collect(ctx context.Context) (*model_v2.ClusterSnapshot, error) {
	snapshot := &model_v2.ClusterSnapshot{
		ClusterID: s.clusterID,
		FetchedAt: time.Now(),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	recordErr := func(err error) {
		if err != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			mu.Unlock()
		}
	}

	// 并发采集 K8s 资源（现有逻辑）
	wg.Add(20)
	// ... 现有的 20 个 goroutine

	// 新增：并发采集 SLO 数据
	if s.sloRepo != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sloData, err := s.sloRepo.Collect(ctx)
			recordErr(err)
			if err == nil {
				mu.Lock()
				snapshot.SLOData = sloData
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// 生成摘要
	snapshot.Summary = s.generateSummary(snapshot)

	return snapshot, firstErr
}
```

---

## 6. 初始化和依赖注入

### 6.1 配置扩展

**文件**: `config/config.go` (修改)

```go
type Config struct {
	// 现有配置
	ClusterID string
	K8s       K8sConfig

	// 新增：SLO 配置
	SLO SLOConfig
}

type SLOConfig struct {
	// 是否启用
	Enabled bool `yaml:"enabled"`

	// OTel Collector 端点
	Endpoint string `yaml:"endpoint"`

	// 采集超时
	Timeout string `yaml:"timeout"`

	// 忽略的 namespace
	IgnoreNamespaces []string `yaml:"ignore_namespaces"`
}
```

**配置文件示例**: `config.yaml`

```yaml
cluster_id: zgmf-x10a

k8s:
  kubeconfig: ""  # 使用 in-cluster

slo:
  enabled: true
  endpoint: "http://otel-collector.otel.svc:8889/metrics"
  timeout: "5s"
  ignore_namespaces:
    - kube-system
    - linkerd
    - linkerd-viz
    - otel
```

### 6.2 依赖注入

**文件**: `main.go` 或 `wire.go`

```go
func initSnapshotService(cfg *config.Config, k8sClient sdk.K8sClient) service.SnapshotService {
	// 创建 K8s Repository（现有）
	podRepo := repository.NewPodRepository(k8sClient)
	nodeRepo := repository.NewNodeRepository(k8sClient)
	// ...

	// 创建 SLO Repository（新增）
	var sloRepo repository.SLORepository
	if cfg.SLO.Enabled {
		otelClient, err := impl.NewOTelClient(sdk.OTelConfig{
			Endpoint: cfg.SLO.Endpoint,
			Timeout:  cfg.SLO.Timeout,
			Enabled:  true,
		})
		if err != nil {
			log.Printf("[SLO] OTel 客户端初始化失败: %v", err)
		} else {
			sloRepo = repository.NewSLORepository(otelClient, cfg.SLO.IgnoreNamespaces)
			log.Printf("[SLO] 已启用，端点: %s", cfg.SLO.Endpoint)
		}
	}

	return service.NewSnapshotService(
		cfg.ClusterID,
		podRepo,
		nodeRepo,
		// ...
		sloRepo, // 可为 nil
	)
}
```

---

## 7. 目标指标参考

### 7.1 Linkerd 指标

| 指标名 | 类型 | 用途 |
|--------|------|------|
| `response_total` | Counter | 请求总数、成功/失败计数 |
| `response_latency_ms_bucket` | Histogram | 延迟分布 |
| `response_latency_ms_sum` | Counter | 延迟总和 |
| `response_latency_ms_count` | Counter | 请求计数 |

**关键标签:**
```
response_total{
  namespace="atlantis",
  deployment="atlantis",
  direction="inbound",           # inbound=被调用，outbound=调用他人
  authority="atlantis.atlantis", # 请求的 Host
  status_code="200",
  tls="true",
  dst_namespace="...",           # outbound 时的目标
  dst_service="...",
}
```

### 7.2 Traefik 指标

| 指标名 | 类型 | 用途 |
|--------|------|------|
| `traefik_service_requests_total` | Counter | 服务请求总数 |
| `traefik_service_request_duration_seconds_bucket` | Histogram | 延迟分布 |

**关键标签:**
```
traefik_service_requests_total{
  service="atlantis-atlantis-http@kubernetes",
  method="GET",
  protocol="http",
  code="200",
}
```

### 7.3 Histogram Buckets

**Linkerd (ms):**
```
le="0.001" → 1ms
le="0.002" → 2ms
le="0.005" → 5ms
le="0.01"  → 10ms
le="0.02"  → 20ms
le="0.05"  → 50ms
le="0.1"   → 100ms
le="0.2"   → 200ms
le="0.5"   → 500ms
le="1"     → 1s
le="2"     → 2s
le="5"     → 5s
le="10"    → 10s
le="+Inf"  → inf
```

**Traefik (seconds):**
```
le="0.005" → 5ms
le="0.01"  → 10ms
le="0.025" → 25ms
le="0.05"  → 50ms
le="0.1"   → 100ms
le="0.25"  → 250ms
le="0.5"   → 500ms
le="1"     → 1s
le="2.5"   → 2.5s
le="5"     → 5s
le="10"    → 10s
le="+Inf"  → inf
```

---

## 8. 实现计划

### 阶段一：SDK 层

- [ ] 创建 `sdk/interfaces_otel.go` - OTelClient 接口
- [ ] 创建 `sdk/types_otel.go` - OTel 类型定义
- [ ] 创建 `sdk/impl/otel_client.go` - OTelClient 实现
- [ ] 编写单元测试

### 阶段二：Repository 层

- [ ] 创建 `model/slo.go` - SLO 数据模型
- [ ] 创建 `repository/slo_snapshot.go` - 快照管理器
- [ ] 创建 `repository/slo_repository.go` - SLO 仓库
- [ ] 编写单元测试

### 阶段三：Service 层集成

- [ ] 修改 `model_v2/cluster_snapshot.go` - 添加 SLOData 字段
- [ ] 修改 `service/snapshot_service.go` - 添加 sloRepo 依赖
- [ ] 修改 `config/config.go` - 添加 SLO 配置
- [ ] 修改 `main.go` - 依赖注入

### 阶段四：测试验证

- [ ] 部署 OTel Collector 到测试集群
- [ ] 验证指标采集和增量计算
- [ ] 验证数据随 ClusterSnapshot 上报
- [ ] Master 端接收和存储验证

---

## 9. 附录

### 9.1 与 K8sClient 架构对比

| 层次 | K8s 数据采集 | SLO 数据采集 |
|------|-------------|-------------|
| SDK | K8sClient 接口 | OTelClient 接口 |
| SDK 实现 | client-go → K8s API Server | net/http → OTel Prometheus |
| Repository | PodRepository, NodeRepository... | SLORepository |
| Service | snapshotService.Collect() | 同上，并发采集 |
| 模型 | Pod, Node, Deployment... | SLOData |
| 上报 | ClusterSnapshot | ClusterSnapshot.SLOData |

### 9.2 可选扩展

1. **Trace 采集**: 扩展 OTelClient 支持 OTLP Trace 接收
2. **多 Collector 支持**: 支持配置多个 OTel Collector 端点
3. **指标聚合**: 在 Agent 本地做更多预聚合，减少上报数据量
