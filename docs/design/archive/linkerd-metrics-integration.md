# Linkerd 指標採集統合設計書

## 概要

本文档描述 AtlHyper Agent 与 Linkerd Service Mesh 的指标采集集成设计。通过 Linkerd 统一采集入口层和服务间的 SLO 数据。

---

## 1. 設計目標

### 1.1 機能目標

| 目標 | 説明 |
|------|------|
| 統一データソース | Linkerd metrics-api から全データ取得 |
| 入口 SLO | Traefik 経由のドメイン別 SLO |
| サービス間 SLO | サービス間通信の SLO |
| 低侵入性 | 既存コードへの影響を最小化 |

### 1.2 対象 Namespace

| Namespace | 用途 |
|-----------|------|
| `kube-system` | Traefik (単独注入) |
| `atlhyper` | AtlHyper コンポーネント |
| `atlantis` | Atlantis サービス |
| `nginx` | Nginx サービス |
| `geass` | Geass サービス |
| `elastic` | Elasticsearch |
| `redis` | Redis |

---

## 2. アーキテクチャ

### 2.1 全体構成

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Kubernetes Cluster                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    linkerd-viz namespace                            │   │
│  │                                                                     │   │
│  │   ┌─────────────────────────────────────────────────────────────┐  │   │
│  │   │                     metrics-api                              │  │   │
│  │   │                                                              │  │   │
│  │   │  全 Sidecar のメトリクスを集約                               │  │   │
│  │   │  :9995/metrics                                               │  │   │
│  │   │                                                              │  │   │
│  │   └──────────────────────────┬──────────────────────────────────┘  │   │
│  │                              │                                      │   │
│  └──────────────────────────────┼──────────────────────────────────────┘   │
│                                 │                                           │
│                                 │ HTTP GET /metrics                         │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                       AtlHyper Agent                                │   │
│  │                                                                     │   │
│  │   ┌─────────────────────────────────────────────────────────────┐  │   │
│  │   │                   LinkerdCollector                          │  │   │
│  │   │                                                             │  │   │
│  │   │  ┌─────────────────┐    ┌─────────────────┐                │  │   │
│  │   │  │ 入口 SLO 抽出   │    │ サービス間 SLO  │                │  │   │
│  │   │  │                 │    │ 抽出            │                │  │   │
│  │   │  │ direction=inbound    │ direction=outbound               │  │   │
│  │   │  │ authority=ドメイン   │ dst_service=xxx │                │  │   │
│  │   │  └────────┬────────┘    └────────┬────────┘                │  │   │
│  │   │           │                      │                          │  │   │
│  │   │           ▼                      ▼                          │  │   │
│  │   │  ┌───────────────────────────────────────────────────────┐ │  │   │
│  │   │  │              SLO Processor                            │ │  │   │
│  │   │  │                                                       │ │  │   │
│  │   │  │  - 成功率計算                                         │ │  │   │
│  │   │  │  - レイテンシ P50/P99 計算                           │ │  │   │
│  │   │  │  - RPS 計算                                          │ │  │   │
│  │   │  └───────────────────────────────────────────────────────┘ │  │   │
│  │   └─────────────────────────────────────────────────────────────┘  │   │
│  │                              │                                      │   │
│  │                              ▼                                      │   │
│  │                         Push to Master                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                          Master                                     │   │
│  │                                                                     │   │
│  │   ┌─────────────────┐    ┌─────────────────┐                       │   │
│  │   │    SLO DB       │    │   SLO API       │                       │   │
│  │   │                 │    │                 │                       │   │
│  │   │  入口 SLO       │    │  /api/v2/slo    │                       │   │
│  │   │  サービス間 SLO │    │                 │                       │   │
│  │   └─────────────────┘    └─────────────────┘                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 データフロー

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              データ採集フロー                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Step 1: 各 Sidecar がメトリクス生成                                        │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                                                             │
│    kube-system          atlhyper           atlantis          nginx          │
│    ┌─────────┐         ┌─────────┐        ┌─────────┐      ┌─────────┐     │
│    │ Traefik │         │ Master  │        │ Service │      │ Nginx   │     │
│    │ Sidecar │         │ Sidecar │        │ Sidecar │      │ Sidecar │     │
│    │ :4191   │         │ :4191   │        │ :4191   │      │ :4191   │     │
│    └────┬────┘         └────┬────┘        └────┬────┘      └────┬────┘     │
│         │                   │                  │                 │          │
│         └───────────────────┴──────────────────┴─────────────────┘          │
│                                    │                                        │
│  Step 2: metrics-api が集約        │                                        │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                    ▼                                        │
│                          ┌─────────────────┐                                │
│                          │   metrics-api   │                                │
│                          │                 │                                │
│                          │ 全 namespace の │                                │
│                          │ 全 Sidecar を   │                                │
│                          │ 集約            │                                │
│                          │                 │                                │
│                          │ :9995/metrics   │                                │
│                          └────────┬────────┘                                │
│                                   │                                         │
│  Step 3: Agent が採集             │                                         │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                   ▼                                         │
│                          ┌─────────────────┐                                │
│                          │ AtlHyper Agent  │                                │
│                          │                 │                                │
│                          │ HTTP GET        │                                │
│                          │ 30秒間隔        │                                │
│                          └────────┬────────┘                                │
│                                   │                                         │
│  Step 4: Master に push           │                                         │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                   ▼                                         │
│                          ┌─────────────────┐                                │
│                          │     Master      │                                │
│                          │                 │                                │
│                          │ SLO DB 保存     │                                │
│                          │ 履歴分析        │                                │
│                          │ アラート        │                                │
│                          └─────────────────┘                                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. 指標詳細

### 3.1 Linkerd が提供する指標

| 指標名 | タイプ | 説明 |
|--------|--------|------|
| `request_total` | Counter | リクエスト総数 |
| `response_total` | Counter | レスポンス総数 (classification 付き) |
| `response_latency_ms_bucket` | Histogram | レスポンス遅延分布 |

### 3.2 主要ラベル

| ラベル | 説明 | 例 |
|--------|------|-----|
| `direction` | 通信方向 | `inbound` / `outbound` |
| `authority` | HTTP Host ヘッダ | `api.example.com` |
| `namespace` | 送信元 namespace | `atlhyper` |
| `deployment` | 送信元 deployment | `web` |
| `dst_namespace` | 宛先 namespace | `atlhyper` |
| `dst_service` | 宛先 service | `master` |
| `classification` | 結果分類 | `success` / `failure` |
| `le` | histogram bucket 上限 | `10`, `50`, `100` |

### 3.3 入口 SLO 抽出ロジック

```
Traefik Sidecar の inbound トラフィック:

request_total{
  direction="inbound",           ← inbound を対象
  authority="api.example.com",   ← ドメイン
  namespace="kube-system",       ← Traefik の namespace
  ...
}

抽出条件:
- direction = "inbound"
- namespace = "kube-system" (Traefik)
- authority でグルーピング
```

### 3.4 サービス間 SLO 抽出ロジック

```
各サービスの outbound トラフィック:

request_total{
  direction="outbound",          ← outbound を対象
  namespace="atlhyper",          ← 送信元
  deployment="web",              ← 送信元 deployment
  dst_namespace="atlhyper",      ← 宛先
  dst_service="master",          ← 宛先 service
  ...
}

抽出条件:
- direction = "outbound"
- dst_namespace が対象 namespace リストに含まれる
- namespace + deployment → dst_service でグルーピング
```

---

## 4. Agent 実装

### 4.1 ディレクトリ構成

```
atlhyper_agent_v2/
├── source/
│   └── slo/
│       ├── collector.go       # SLO 採集メイン
│       ├── linkerd.go         # Linkerd 採集器
│       ├── parser.go          # Prometheus フォーマットパーサー
│       └── types.go           # データ型定義
├── config/
│   └── slo.go                 # SLO 設定
└── pusher/
    └── slo_pusher.go          # Master への push
```

### 4.2 データ型定義

```go
// source/slo/types.go

package slo

import "time"

// SLOType SLO データ種別
type SLOType string

const (
    SLOTypeIngress SLOType = "ingress"  // 入口 (Traefik inbound)
    SLOTypeService SLOType = "service"  // サービス間 (outbound)
)

// ServiceSLO サービス SLO 指標
type ServiceSLO struct {
    // 識別
    Type       SLOType `json:"type"`
    Source     string  `json:"source"`      // 送信元 (ドメイン or サービス名)
    Target     string  `json:"target"`      // 宛先サービス
    Namespace  string  `json:"namespace"`   // 宛先 namespace

    // 指標
    SuccessRate  float64 `json:"success_rate"`   // 成功率 (0-100)
    LatencyP50   float64 `json:"latency_p50"`    // P50 遅延 (ms)
    LatencyP99   float64 `json:"latency_p99"`    // P99 遅延 (ms)
    RequestRate  float64 `json:"request_rate"`   // RPS

    // 生カウント
    TotalRequests   int64 `json:"total_requests"`
    SuccessRequests int64 `json:"success_requests"`

    // 時刻
    Timestamp time.Time `json:"timestamp"`
}

// SLOSnapshot 採集スナップショット
type SLOSnapshot struct {
    ClusterID string       `json:"cluster_id"`
    Ingress   []ServiceSLO `json:"ingress"`   // 入口 SLO
    Services  []ServiceSLO `json:"services"`  // サービス間 SLO
    Timestamp time.Time    `json:"timestamp"`
}
```

### 4.3 設定

```go
// config/slo.go

package config

import "time"

// SLOConfig SLO 採集設定
type SLOConfig struct {
    Enabled         bool          `yaml:"enabled"`
    CollectInterval time.Duration `yaml:"collect_interval"`

    Linkerd LinkerdConfig `yaml:"linkerd"`
}

// LinkerdConfig Linkerd 設定
type LinkerdConfig struct {
    Enabled    bool   `yaml:"enabled"`
    MetricsURL string `yaml:"metrics_url"`

    // 対象 namespace (空の場合は全て)
    TargetNamespaces []string `yaml:"target_namespaces"`

    // Traefik の namespace (入口 SLO 用)
    TraefikNamespace string `yaml:"traefik_namespace"`
}

// デフォルト設定
var DefaultSLOConfig = SLOConfig{
    Enabled:         false,
    CollectInterval: 30 * time.Second,
    Linkerd: LinkerdConfig{
        Enabled:          false,
        MetricsURL:       "http://metrics-api.linkerd-viz:8085/metrics",
        TargetNamespaces: []string{"atlhyper", "atlantis", "nginx", "geass", "elastic", "redis"},
        TraefikNamespace: "kube-system",
    },
}
```

### 4.4 Linkerd Collector

```go
// source/slo/linkerd.go

package slo

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

type LinkerdCollector struct {
    cfg    LinkerdConfig
    client *http.Client

    // 前回値 (RPS 計算用)
    prevMetrics map[string]float64
    prevTime    time.Time
}

func NewLinkerdCollector(cfg LinkerdConfig) *LinkerdCollector {
    return &LinkerdCollector{
        cfg: cfg,
        client: &http.Client{
            Timeout: 10 * time.Second,
        },
        prevMetrics: make(map[string]float64),
    }
}

func (c *LinkerdCollector) Collect(ctx context.Context) (*SLOSnapshot, error) {
    // 1. メトリクス取得
    req, err := http.NewRequestWithContext(ctx, "GET", c.cfg.MetricsURL, nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("fetch metrics: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    // 2. パース
    metrics := ParsePrometheusMetrics(string(body))

    // 3. 入口 SLO 抽出
    ingress := c.extractIngressSLO(metrics)

    // 4. サービス間 SLO 抽出
    services := c.extractServiceSLO(metrics)

    return &SLOSnapshot{
        Ingress:   ingress,
        Services:  services,
        Timestamp: time.Now(),
    }, nil
}

func (c *LinkerdCollector) extractIngressSLO(metrics []PrometheusMetric) []ServiceSLO {
    // Traefik の inbound トラフィックを抽出
    // direction=inbound, namespace=kube-system

    sloMap := make(map[string]*ServiceSLO)

    for _, m := range metrics {
        if m.Name != "request_total" && m.Name != "response_total" {
            continue
        }
        if m.Labels["direction"] != "inbound" {
            continue
        }
        if m.Labels["namespace"] != c.cfg.TraefikNamespace {
            continue
        }

        authority := m.Labels["authority"]
        if authority == "" || strings.HasSuffix(authority, ".svc.cluster.local") {
            continue // 内部通信は除外
        }

        key := authority
        if _, ok := sloMap[key]; !ok {
            sloMap[key] = &ServiceSLO{
                Type:      SLOTypeIngress,
                Source:    authority,
                Target:    m.Labels["dst_service"],
                Namespace: m.Labels["dst_namespace"],
                Timestamp: time.Now(),
            }
        }

        if m.Name == "request_total" {
            sloMap[key].TotalRequests += int64(m.Value)
        }
        if m.Name == "response_total" && m.Labels["classification"] == "success" {
            sloMap[key].SuccessRequests += int64(m.Value)
        }
    }

    // 成功率計算
    result := make([]ServiceSLO, 0, len(sloMap))
    for _, slo := range sloMap {
        if slo.TotalRequests > 0 {
            slo.SuccessRate = float64(slo.SuccessRequests) / float64(slo.TotalRequests) * 100
        }
        result = append(result, *slo)
    }

    return result
}

func (c *LinkerdCollector) extractServiceSLO(metrics []PrometheusMetric) []ServiceSLO {
    // 各サービスの outbound トラフィックを抽出

    sloMap := make(map[string]*ServiceSLO)

    for _, m := range metrics {
        if m.Name != "request_total" && m.Name != "response_total" {
            continue
        }
        if m.Labels["direction"] != "outbound" {
            continue
        }

        dstNs := m.Labels["dst_namespace"]
        if !c.isTargetNamespace(dstNs) {
            continue
        }

        srcDeploy := m.Labels["deployment"]
        dstService := m.Labels["dst_service"]
        key := fmt.Sprintf("%s->%s", srcDeploy, dstService)

        if _, ok := sloMap[key]; !ok {
            sloMap[key] = &ServiceSLO{
                Type:      SLOTypeService,
                Source:    srcDeploy,
                Target:    dstService,
                Namespace: dstNs,
                Timestamp: time.Now(),
            }
        }

        if m.Name == "request_total" {
            sloMap[key].TotalRequests += int64(m.Value)
        }
        if m.Name == "response_total" && m.Labels["classification"] == "success" {
            sloMap[key].SuccessRequests += int64(m.Value)
        }
    }

    result := make([]ServiceSLO, 0, len(sloMap))
    for _, slo := range sloMap {
        if slo.TotalRequests > 0 {
            slo.SuccessRate = float64(slo.SuccessRequests) / float64(slo.TotalRequests) * 100
        }
        result = append(result, *slo)
    }

    return result
}

func (c *LinkerdCollector) isTargetNamespace(ns string) bool {
    if len(c.cfg.TargetNamespaces) == 0 {
        return true
    }
    for _, target := range c.cfg.TargetNamespaces {
        if target == ns {
            return true
        }
    }
    return false
}
```

---

## 5. Master 変更

### 5.1 データモデル拡張

```go
// database/model/slo.go

// SLORaw 生 SLO データ
type SLORaw struct {
    ID        int64     `gorm:"primaryKey"`
    ClusterID string    `gorm:"index"`

    // 新規フィールド
    Type      string    `gorm:"index"`  // ingress / service
    Source    string    `gorm:"index"`  // ドメイン or 送信元サービス
    Target    string    `gorm:"index"`  // 宛先サービス
    Namespace string    `gorm:"index"`

    // 指標
    SuccessRate   float64
    LatencyP50    float64
    LatencyP99    float64
    RequestRate   float64
    TotalRequests int64

    Timestamp time.Time `gorm:"index"`
}
```

### 5.2 API 拡張

```go
// GET /api/v2/clusters/{id}/slo/overview
type SLOOverviewResponse struct {
    Ingress  []ServiceSLO `json:"ingress"`   // 入口 SLO
    Services []ServiceSLO `json:"services"`  // サービス間 SLO
}

// GET /api/v2/clusters/{id}/slo/detail
// Query: type=ingress|service, source=xxx, target=xxx, range=1h|24h|7d
```

---

## 6. フロントエンド変更

### 6.1 SLO ダッシュボード

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SLO Dashboard                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─ Tab ──────────────────────────────────────────────────────────────────┐│
│  │ [入口 SLO]  [サービス間 SLO]                                           ││
│  └────────────────────────────────────────────────────────────────────────┘│
│                                                                             │
│  ▼ 入口 SLO                                                                │
│  ┌─────────────────────────────────────────────────────────────────────┐  │
│  │ ドメイン              宛先             成功率    P99     RPS        │  │
│  │ api.example.com      backend-svc      99.5%    45ms    1.2k       │  │
│  │ web.example.com      frontend-svc     99.9%    32ms    3.5k       │  │
│  │ admin.example.com    admin-svc        99.8%    28ms    0.5k       │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ▼ サービス間 SLO                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐  │
│  │ 送信元        宛先           Namespace    成功率    P99     RPS     │  │
│  │ web          master         atlhyper    99.8%    12ms    2.1k     │  │
│  │ master       agent          atlhyper    99.9%     8ms    1.8k     │  │
│  │ atlantis     elastic        elastic     98.5%    25ms    0.9k ⚠️  │  │
│  │ geass        redis          redis       99.9%     3ms    5.2k     │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. 設定例

### 7.1 Agent 設定

```yaml
# agent-config.yaml

cluster_id: "production"
master_url: "http://atlhyper-master:8081"

slo:
  enabled: true
  collect_interval: 30s

  linkerd:
    enabled: true
    metrics_url: "http://metrics-api.linkerd-viz:8085/metrics"
    target_namespaces:
      - atlhyper
      - atlantis
      - nginx
      - geass
      - elastic
      - redis
    traefik_namespace: "kube-system"
```

---

## 8. 実施計画

### Phase 1: Agent 実装 (1週間)

| タスク | 説明 |
|--------|------|
| Prometheus パーサー実装 | テキストフォーマットのパース |
| LinkerdCollector 実装 | 入口 / サービス間 SLO 抽出 |
| SLO Pusher 実装 | Master への送信 |
| 単体テスト | モックメトリクスでテスト |

### Phase 2: Master 実装 (1週間)

| タスク | 説明 |
|--------|------|
| データモデル拡張 | Type/Source/Target フィールド追加 |
| Ingest API 拡張 | SLO データ受信 |
| Query API 拡張 | Type 別クエリ対応 |
| DB マイグレーション | テーブル構造変更 |

### Phase 3: フロントエンド (1週間)

| タスク | 説明 |
|--------|------|
| SLO ダッシュボード更新 | タブ切り替え UI |
| サービス間 SLO 一覧 | 新規コンポーネント |
| トレンドグラフ更新 | サービス間対応 |

---

## 9. 付録

### 9.1 Prometheus メトリクス例

```prometheus
# 入口トラフィック (Traefik inbound)
request_total{
  direction="inbound",
  namespace="kube-system",
  deployment="traefik",
  authority="api.example.com",
  dst_namespace="atlhyper",
  dst_service="backend"
} 12345

response_total{
  direction="inbound",
  namespace="kube-system",
  deployment="traefik",
  authority="api.example.com",
  classification="success"
} 12300

# サービス間トラフィック
request_total{
  direction="outbound",
  namespace="atlhyper",
  deployment="web",
  dst_namespace="atlhyper",
  dst_service="master"
} 5678

response_latency_ms_bucket{
  direction="outbound",
  namespace="atlhyper",
  deployment="web",
  dst_namespace="atlhyper",
  dst_service="master",
  le="10"
} 5000
```

### 9.2 関連ドキュメント

- [linkerd-deployment-guide.md](linkerd-deployment-guide.md) - Linkerd 部署手顺書
- [Linkerd Proxy Metrics](https://linkerd.io/2/reference/proxy-metrics/)
