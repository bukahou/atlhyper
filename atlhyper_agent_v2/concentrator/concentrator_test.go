package concentrator

import (
	"testing"
	"time"

	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

func TestConcentrator_IngestAndFlush(t *testing.T) {
	c := New()
	now := time.Now().Truncate(time.Minute)

	// 模拟 3 分钟的数据（从过去到现在）
	for i := 0; i < 3; i++ {
		ts := now.Add(-time.Duration(2-i) * time.Minute)
		nodes := []metrics.NodeMetrics{
			{
				NodeName: "worker-1",
				CPU:      metrics.NodeCPU{UsagePct: float64(10 + i*10), Load1: 1.0},
				Memory:   metrics.NodeMemory{UsagePct: float64(50 + i)},
				Disks:    []metrics.NodeDisk{{MountPoint: "/", UsagePct: 30.0}},
				Networks: []metrics.NodeNetwork{{Interface: "eth0", Up: true, RxBytesPerSec: 1000, TxBytesPerSec: 500}},
			},
		}
		sloIngress := []slo.IngressSLO{
			{ServiceKey: "api-gw", RPS: float64(100 + i*10), SuccessRate: 99.5, P99Ms: 15.0},
		}
		c.Ingest(nodes, sloIngress, ts)
	}

	// Flush 节点时序
	nodeSeries := c.FlushNodeSeries()
	if len(nodeSeries) != 1 {
		t.Fatalf("expected 1 node series, got %d", len(nodeSeries))
	}
	if nodeSeries[0].NodeName != "worker-1" {
		t.Errorf("expected node name 'worker-1', got '%s'", nodeSeries[0].NodeName)
	}
	if len(nodeSeries[0].Points) != 3 {
		t.Fatalf("expected 3 node points, got %d", len(nodeSeries[0].Points))
	}
	// 验证第一个点
	if nodeSeries[0].Points[0].CPUPct != 10.0 {
		t.Errorf("expected CPUPct 10.0, got %f", nodeSeries[0].Points[0].CPUPct)
	}
	// 验证最后一个点
	lastPt := nodeSeries[0].Points[2]
	if lastPt.CPUPct != 30.0 {
		t.Errorf("expected CPUPct 30.0, got %f", lastPt.CPUPct)
	}
	if lastPt.DiskPct != 30.0 {
		t.Errorf("expected DiskPct 30.0, got %f", lastPt.DiskPct)
	}
	if lastPt.NetRxBps != 1000.0 {
		t.Errorf("expected NetRxBps 1000.0, got %f", lastPt.NetRxBps)
	}

	// Flush SLO 时序
	sloSeries := c.FlushSLOSeries()
	if len(sloSeries) != 1 {
		t.Fatalf("expected 1 SLO series, got %d", len(sloSeries))
	}
	if sloSeries[0].ServiceName != "api-gw" {
		t.Errorf("expected service name 'api-gw', got '%s'", sloSeries[0].ServiceName)
	}
	if len(sloSeries[0].Points) != 3 {
		t.Fatalf("expected 3 SLO points, got %d", len(sloSeries[0].Points))
	}
	if sloSeries[0].Points[0].RPS != 100.0 {
		t.Errorf("expected RPS 100.0, got %f", sloSeries[0].Points[0].RPS)
	}
}

func TestConcentrator_GAUGESemantic(t *testing.T) {
	// 同一分钟内多次写入，取最新值
	c := New()
	now := time.Now().Truncate(time.Minute)

	for i := 0; i < 6; i++ {
		ts := now.Add(time.Duration(i*10) * time.Second) // 同一分钟内 6 次
		nodes := []metrics.NodeMetrics{
			{
				NodeName: "node-1",
				CPU:      metrics.NodeCPU{UsagePct: float64(10 + i)},
				Memory:   metrics.NodeMemory{UsagePct: 50.0},
			},
		}
		c.Ingest(nodes, nil, ts)
	}

	series := c.FlushNodeSeries()
	if len(series) != 1 {
		t.Fatalf("expected 1 series, got %d", len(series))
	}
	// 应该只有 1 个数据点（同一分钟），值为最后写入的值
	if len(series[0].Points) != 1 {
		t.Fatalf("expected 1 point (same minute), got %d", len(series[0].Points))
	}
	if series[0].Points[0].CPUPct != 15.0 { // 10 + 5
		t.Errorf("expected CPUPct 15.0 (last write), got %f", series[0].Points[0].CPUPct)
	}
}

func TestConcentrator_RingOverwrite(t *testing.T) {
	// 超过 60 分钟后旧数据被覆盖
	c := New()
	now := time.Now().Truncate(time.Minute)

	// 写入 70 分钟的数据（从过去到现在）
	for i := 0; i < 70; i++ {
		ts := now.Add(-time.Duration(69-i) * time.Minute)
		nodes := []metrics.NodeMetrics{
			{
				NodeName: "node-1",
				CPU:      metrics.NodeCPU{UsagePct: float64(i)},
				Memory:   metrics.NodeMemory{UsagePct: 50.0},
			},
		}
		c.Ingest(nodes, nil, ts)
	}

	series := c.FlushNodeSeries()
	if len(series) != 1 {
		t.Fatalf("expected 1 series, got %d", len(series))
	}
	// 应该最多 60 个点（环形缓冲区容量）
	if len(series[0].Points) > ringCapacity {
		t.Errorf("expected at most %d points, got %d", ringCapacity, len(series[0].Points))
	}
	// 数据从 now-69 到 now 写入（70 个点），环容量 60
	// flush 窗口 [now-59, now] = 60 个点
	// 对应的是 i=10..69（CPUPct = 10..69）
	if len(series[0].Points) > 0 {
		firstPt := series[0].Points[0]
		if firstPt.CPUPct != 10.0 {
			t.Errorf("expected first CPUPct 10.0 (oldest surviving), got %f", firstPt.CPUPct)
		}
		lastPt := series[0].Points[len(series[0].Points)-1]
		if lastPt.CPUPct != 69.0 {
			t.Errorf("expected last CPUPct 69.0, got %f", lastPt.CPUPct)
		}
	}
}

func TestConcentrator_MultipleNodes(t *testing.T) {
	c := New()
	now := time.Now().Truncate(time.Minute)

	nodes := []metrics.NodeMetrics{
		{NodeName: "master-1", CPU: metrics.NodeCPU{UsagePct: 20}, Memory: metrics.NodeMemory{UsagePct: 40}},
		{NodeName: "worker-1", CPU: metrics.NodeCPU{UsagePct: 60}, Memory: metrics.NodeMemory{UsagePct: 70}},
		{NodeName: "worker-2", CPU: metrics.NodeCPU{UsagePct: 80}, Memory: metrics.NodeMemory{UsagePct: 90}},
	}
	c.Ingest(nodes, nil, now)

	series := c.FlushNodeSeries()
	if len(series) != 3 {
		t.Fatalf("expected 3 node series, got %d", len(series))
	}
}

func TestConcentrator_EmptyFlush(t *testing.T) {
	c := New()
	nodeSeries := c.FlushNodeSeries()
	sloSeries := c.FlushSLOSeries()

	if len(nodeSeries) != 0 {
		t.Errorf("expected 0 node series, got %d", len(nodeSeries))
	}
	if len(sloSeries) != 0 {
		t.Errorf("expected 0 SLO series, got %d", len(sloSeries))
	}
}
