// atlhyper_master_v2/aiops/baseline/extractor_enhanced_test.go
// Enhanced 层（OTel 信号）指标提取 + 确定性异常测试
package baseline

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/log"
	"AtlHyper/model_v3/metrics"
)

// ==================== 共用 Mock 数据 ====================

func makeOTelSnapshot() *cluster.OTelSnapshot {
	return &cluster.OTelSnapshot{
		// APM 服务指标
		APMServices: []apm.APMService{
			// 异常：error_rate = 1-0.82 = 0.18 > 15%, P99 = 6000 > 5000ms
			{Name: "api-gateway", Namespace: "default", SuccessRate: 0.82, P99Ms: 6000, RPS: 150},
			// 正常
			{Name: "user-svc", Namespace: "default", SuccessRate: 0.99, P99Ms: 200, RPS: 80},
		},
		// APM 拓扑
		APMTopology: &apm.Topology{
			Nodes: []apm.TopologyNode{
				{Id: "api-gateway", Name: "api-gateway", Namespace: "default"},
				{Id: "user-svc", Name: "user-svc", Namespace: "default"},
			},
			Edges: []apm.TopologyEdge{
				{Source: "api-gateway", Target: "user-svc", CallCount: 1000, AvgMs: 50, ErrorRate: 0.02},
			},
		},
		// 日志摘要（含按服务的错误数）
		LogsSummary: &log.Summary{
			SeverityCounts: map[string]int64{"ERROR": 600, "WARN": 200, "INFO": 5000},
			ServiceErrors: []log.ServiceErrorCount{
				{Namespace: "default", Service: "api-gateway", Count: 2},
			},
		},
		// OTel Node 指标（磁盘 + PSI）
		MetricsNodes: []metrics.NodeMetrics{
			{
				NodeName: "node-1",
				CPU:      metrics.NodeCPU{UsagePct: 65},
				Memory:   metrics.NodeMemory{UsagePct: 70},
				Disks:    []metrics.NodeDisk{{MountPoint: "/", UsagePct: 92}},
				PSI:      metrics.NodePSI{CPUSomePct: 30, MemSomePct: 5, IOSomePct: 45},
			},
		},
	}
}

// ==================== APM 指标提取 ====================

func TestExtractAPMMetrics(t *testing.T) {
	otel := makeOTelSnapshot()
	points := extractAPMMetrics(otel)

	// 2 个服务 × 3 个指标 = 6 个点
	if len(points) != 6 {
		t.Fatalf("expected 6 points, got %d", len(points))
	}

	// api-gateway 指标
	gwMetrics := indexPoints(points, "default/service/api-gateway")
	// error_rate = 1 - 0.82 = 0.18
	assertMetricApprox(t, gwMetrics, "apm_error_rate", 0.18)
	assertMetric(t, gwMetrics, "apm_p99_latency", 6000)
	assertMetric(t, gwMetrics, "apm_rps", 150)

	// user-svc 指标
	usMetrics := indexPoints(points, "default/service/user-svc")
	// error_rate = 1 - 0.99 = 0.01
	assertMetricApprox(t, usMetrics, "apm_error_rate", 0.01)
	assertMetric(t, usMetrics, "apm_p99_latency", 200)
	assertMetric(t, usMetrics, "apm_rps", 80)
}

func TestExtractAPMMetrics_NilOTel(t *testing.T) {
	points := extractAPMMetrics(nil)
	if len(points) != 0 {
		t.Fatalf("nil otel should return empty, got %d", len(points))
	}
}

func TestExtractAPMMetrics_EmptyServices(t *testing.T) {
	otel := &cluster.OTelSnapshot{APMServices: nil}
	points := extractAPMMetrics(otel)
	if len(points) != 0 {
		t.Fatalf("empty services should return empty, got %d", len(points))
	}
}

// ==================== 日志指标提取 ====================

func TestExtractLogMetrics(t *testing.T) {
	otel := makeOTelSnapshot()
	points := extractLogMetrics(otel)

	// 全局指标（来自 Summary）
	globalMetrics := indexPoints(points, "_cluster/logs/global")
	assertMetric(t, globalMetrics, "log_error_count", 600)
	assertMetric(t, globalMetrics, "log_warn_count", 200)

	// 服务级指标（来自按服务的错误数）
	gwMetrics := indexPoints(points, "default/service/api-gateway")
	assertMetric(t, gwMetrics, "log_error_count", 2)
}

// 按服务的错误数落在各服务实体上（log_error_count），与集群级计数互不依赖
func TestExtractLogMetrics_ServiceErrors(t *testing.T) {
	tests := []struct {
		name    string
		summary *log.Summary
		want    map[string]float64 // 服务实体键 → log_error_count
	}{
		{
			name: "按服务的错误数落在各服务实体上",
			summary: &log.Summary{
				SeverityCounts: map[string]int64{"ERROR": 45},
				ServiceErrors: []log.ServiceErrorCount{
					{Namespace: "shop", Service: "checkout", Count: 42},
					{Namespace: "jobs", Service: "worker", Count: 3},
				},
			},
			want: map[string]float64{"shop/service/checkout": 42, "jobs/service/worker": 3},
		},
		{
			name:    "旧版 Agent 不带按服务的计数：不发，也不出错",
			summary: &log.Summary{SeverityCounts: map[string]int64{"ERROR": 7}},
			want:    map[string]float64{},
		},
		{
			name:    "没有服务出错",
			summary: &log.Summary{SeverityCounts: map[string]int64{"ERROR": 7}, ServiceErrors: []log.ServiceErrorCount{}},
			want:    map[string]float64{},
		},
		{
			name:    "条数为 0 不发（与集群级计数一致）",
			summary: &log.Summary{ServiceErrors: []log.ServiceErrorCount{{Namespace: "shop", Service: "checkout", Count: 0}}},
			want:    map[string]float64{},
		},
		{
			name:    "没有集群级计数也照发按服务的",
			summary: &log.Summary{ServiceErrors: []log.ServiceErrorCount{{Namespace: "shop", Service: "checkout", Count: 5}}},
			want:    map[string]float64{"shop/service/checkout": 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serviceLogErrorCounts(t, extractLogMetrics(&cluster.OTelSnapshot{LogsSummary: tt.summary}))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("服务级 log_error_count = %v, want %v", got, tt.want)
			}
		})
	}
}

// 同一个服务的日志指标与 APM 指标落在同一个实体上：两边都按 aiops.EntityKey(命名空间, "service", 服务名) 拼，
// 命名空间为空时也一样（落到集群级占位）；对不上 K8s Service 的服务两边都照常发出，⛔ 另做映射或丢弃
func TestExtractLogMetrics_ServiceKeyMatchesAPM(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
	}{
		{"有命名空间", "shop"},
		{"命名空间为空", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otel := &cluster.OTelSnapshot{
				APMServices: []apm.APMService{{Name: "checkout", Namespace: tt.namespace, SuccessRate: 0.99}},
				LogsSummary: &log.Summary{ServiceErrors: []log.ServiceErrorCount{
					{Namespace: tt.namespace, Service: "checkout", Count: 9},
				}},
			}
			apmKeys := map[string]bool{}
			for _, p := range extractAPMMetrics(otel) {
				apmKeys[p.EntityKey] = true
			}
			logKeys := serviceLogErrorCounts(t, extractLogMetrics(otel))
			if len(apmKeys) != 1 || len(logKeys) != 1 {
				t.Fatalf("APM 实体 %v，日志实体 %v，各应只有一个", apmKeys, logKeys)
			}
			for key := range logKeys {
				if !apmKeys[key] {
					t.Errorf("日志实体键 %q 与 APM 实体键 %v 不一致", key, apmKeys)
				}
			}
		})
	}
}

// 快照按 JSON 从 Agent 传来：旧版 Agent 没有 serviceErrors，照常解出、照常提取集群级指标，不发服务级；
// 新版 Agent 的字段名是 camelCase
func TestExtractLogMetrics_AgentPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    map[string]float64 // 服务实体键 → log_error_count
	}{
		{
			name:    "旧版 Agent",
			payload: `{"logsSummary":{"totalEntries":120,"severityCounts":{"ERROR":7},"topServices":[],"latestAt":"2026-11-20T09:00:00Z"}}`,
			want:    map[string]float64{},
		},
		{
			name: "新版 Agent",
			payload: `{"logsSummary":{"totalEntries":120,"severityCounts":{"ERROR":7},"topServices":[],"latestAt":"2026-11-20T09:00:00Z",` +
				`"serviceErrors":[{"namespace":"shop","service":"checkout","count":5}]}}`,
			want: map[string]float64{"shop/service/checkout": 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var otel cluster.OTelSnapshot
			if err := json.Unmarshal([]byte(tt.payload), &otel); err != nil {
				t.Fatalf("快照应能解出: %v", err)
			}
			points := extractLogMetrics(&otel)
			assertMetric(t, indexPoints(points, "_cluster/logs/global"), "log_error_count", 7)
			if got := serviceLogErrorCounts(t, points); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("服务级 log_error_count = %v, want %v", got, tt.want)
			}
		})
	}
}

// serviceLogErrorCounts 取出落在服务实体上的日志指标（实体键 → 值）；服务实体上只该有 log_error_count
func serviceLogErrorCounts(t *testing.T, points []aiops.MetricDataPoint) map[string]float64 {
	t.Helper()
	got := map[string]float64{}
	for _, p := range points {
		if aiops.ExtractEntityType(p.EntityKey) != "service" {
			continue
		}
		if p.MetricName != "log_error_count" {
			t.Errorf("服务实体 %s 上出现了 %s", p.EntityKey, p.MetricName)
			continue
		}
		got[p.EntityKey] = p.Value
	}
	return got
}

func TestExtractLogMetrics_NilOTel(t *testing.T) {
	points := extractLogMetrics(nil)
	if len(points) != 0 {
		t.Fatalf("nil otel should return empty, got %d", len(points))
	}
}

func TestExtractLogMetrics_NilSummary(t *testing.T) {
	otel := &cluster.OTelSnapshot{LogsSummary: nil}
	points := extractLogMetrics(otel)
	if len(points) != 0 {
		t.Fatalf("nil summary should return empty, got %d", len(points))
	}
}

// ==================== Enhanced Node 指标 ====================

func TestExtractEnhancedNodeMetrics(t *testing.T) {
	otel := makeOTelSnapshot()
	points := extractEnhancedNodeMetrics(otel)

	nodeMetrics := indexPoints(points, "_cluster/node/node-1")

	// 磁盘使用率（GetPrimaryDisk = "/" → 92%）
	assertMetric(t, nodeMetrics, "disk_usage", 92)

	// PSI 指标
	assertMetric(t, nodeMetrics, "psi_cpu", 30)
	assertMetric(t, nodeMetrics, "psi_memory", 5)
	assertMetric(t, nodeMetrics, "psi_io", 45)
}

func TestExtractEnhancedNodeMetrics_NilOTel(t *testing.T) {
	points := extractEnhancedNodeMetrics(nil)
	if len(points) != 0 {
		t.Fatalf("nil otel should return empty, got %d", len(points))
	}
}

func TestExtractEnhancedNodeMetrics_NoDisk(t *testing.T) {
	otel := &cluster.OTelSnapshot{
		MetricsNodes: []metrics.NodeMetrics{
			{
				NodeName: "node-2",
				Disks:    nil, // 无磁盘数据
				PSI:      metrics.NodePSI{CPUSomePct: 10, MemSomePct: 20, IOSomePct: 30},
			},
		},
	}
	points := extractEnhancedNodeMetrics(otel)
	nodeMetrics := indexPoints(points, "_cluster/node/node-2")

	// 无磁盘 → 无 disk_usage 指标
	if _, ok := nodeMetrics["disk_usage"]; ok {
		t.Error("无磁盘数据时不应生成 disk_usage 指标")
	}

	// PSI 仍然存在
	assertMetric(t, nodeMetrics, "psi_cpu", 10)
	assertMetric(t, nodeMetrics, "psi_memory", 20)
	assertMetric(t, nodeMetrics, "psi_io", 30)
}

// ==================== OTel 确定性异常 ====================

func TestExtractOTelDeterministicAnomalies(t *testing.T) {
	otel := makeOTelSnapshot()
	results := ExtractOTelDeterministicAnomalies(otel)

	// 预期 3 条异常:
	// 1. api-gateway: error_rate=18% > 15% → apm_high_error_rate
	// 2. api-gateway: P99=6000ms > 5000ms → apm_high_p99_latency
	// 3. logs:global: ERROR=600 > 500 → log_error_spike
	if len(results) != 3 {
		t.Fatalf("expected 3 anomalies, got %d", len(results))
		for _, r := range results {
			t.Logf("  %s / %s", r.EntityKey, r.MetricName)
		}
	}

	// APM 高错误率
	apmErr := findResult(results, "default/service/api-gateway", "apm_high_error_rate")
	if apmErr == nil {
		t.Fatal("应检测到 api-gateway apm_high_error_rate")
	}
	if apmErr.Score < 0.7 {
		t.Errorf("apm_high_error_rate score 应 >= 0.7, got %.2f", apmErr.Score)
	}

	// APM 高延迟
	apmLat := findResult(results, "default/service/api-gateway", "apm_high_p99_latency")
	if apmLat == nil {
		t.Fatal("应检测到 api-gateway apm_high_p99_latency")
	}
	if apmLat.Score < 0.7 {
		t.Errorf("apm_high_p99_latency score 应 >= 0.7, got %.2f", apmLat.Score)
	}

	// 日志错误尖峰
	logSpike := findResult(results, "_cluster/logs/global", "log_error_spike")
	if logSpike == nil {
		t.Fatal("应检测到 logs:global log_error_spike")
	}
	if logSpike.Score < 0.7 {
		t.Errorf("log_error_spike score 应 >= 0.7, got %.2f", logSpike.Score)
	}

	// user-svc 不应有异常（正常服务）
	if findResult(results, "default/service/user-svc", "apm_high_error_rate") != nil {
		t.Error("user-svc 不应有 apm_high_error_rate")
	}
}

func TestExtractOTelDeterministicAnomalies_NilOTel(t *testing.T) {
	results := ExtractOTelDeterministicAnomalies(nil)
	if len(results) != 0 {
		t.Fatalf("nil otel should return empty, got %d", len(results))
	}
}

func TestExtractOTelDeterministicAnomalies_NoAnomalies(t *testing.T) {
	otel := &cluster.OTelSnapshot{
		APMServices: []apm.APMService{
			// 全部正常
			{Name: "healthy-svc", Namespace: "default", SuccessRate: 0.999, P99Ms: 100, RPS: 50},
		},
		LogsSummary: &log.Summary{
			SeverityCounts: map[string]int64{"ERROR": 10, "WARN": 20}, // 远低于阈值
		},
	}

	results := ExtractOTelDeterministicAnomalies(otel)
	if len(results) != 0 {
		t.Fatalf("全正常数据不应产生异常, got %d", len(results))
		for _, r := range results {
			t.Logf("  %s / %s", r.EntityKey, r.MetricName)
		}
	}
}

func TestExtractOTelDeterministicAnomalies_BoundaryValues(t *testing.T) {
	// 边界测试：刚好在阈值上
	tests := []struct {
		name        string
		successRate float64
		p99Ms       float64
		errorCount  int64
		wantCount   int
	}{
		{"error_rate=14%", 0.86, 100, 0, 0},        // 14% 不触发（需 > 15%）
		{"error_rate=16%", 0.84, 100, 0, 1},        // 16% > 15% 触发
		{"p99=5000ms exact", 0.99, 5000, 0, 0},     // 5000ms 不触发（需 > 5000）
		{"p99=5001ms", 0.99, 5001, 0, 1},           // > 5000ms 触发
		{"log_error=500 exact", 0.99, 100, 500, 0}, // 500 不触发（需 > 500）
		{"log_error=501", 0.99, 100, 501, 1},       // > 500 触发
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otel := &cluster.OTelSnapshot{
				APMServices: []apm.APMService{
					{Name: "test-svc", Namespace: "default", SuccessRate: tt.successRate, P99Ms: tt.p99Ms, RPS: 50},
				},
			}
			if tt.errorCount > 0 {
				otel.LogsSummary = &log.Summary{
					SeverityCounts: map[string]int64{"ERROR": tt.errorCount},
				}
			}
			results := ExtractOTelDeterministicAnomalies(otel)
			if len(results) != tt.wantCount {
				t.Errorf("expected %d anomalies, got %d", tt.wantCount, len(results))
				for _, r := range results {
					t.Logf("  %s / %s (score=%.2f)", r.EntityKey, r.MetricName, r.Score)
				}
			}
		})
	}
}

// ==================== 辅助函数 ====================

// assertMetricApprox 浮点近似比较（容差 0.001）
func assertMetricApprox(t *testing.T, metrics map[string]float64, name string, expected float64) {
	t.Helper()
	got, ok := metrics[name]
	if !ok {
		t.Errorf("缺少指标 %s", name)
		return
	}
	if math.Abs(got-expected) > 0.001 {
		t.Errorf("指标 %s = %.4f, want %.4f", name, got, expected)
	}
}

// dumpPoints 调试用：打印所有指标点
func dumpPoints(points []aiops.MetricDataPoint) {
	for _, p := range points {
		fmt.Printf("  entity=%s metric=%s value=%.2f\n", p.EntityKey, p.MetricName, p.Value)
	}
}
