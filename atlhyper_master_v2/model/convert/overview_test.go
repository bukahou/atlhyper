package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v3/cluster"
)

// TestOverview_FieldMapping 验证 Overview 关键字段转换
func TestOverview_FieldMapping(t *testing.T) {
	src := &cluster.ClusterOverview{
		ClusterID: "cluster-1",
		Cards: cluster.OverviewCards{
			ClusterHealth: cluster.ClusterHealth{
				Status:           "Healthy",
				Reason:           "",
				NodeReadyPercent: 100.0,
				PodReadyPercent:  95.0,
			},
			NodeReady: cluster.ResourceReady{
				Total: 6, Ready: 6, Percent: 100.0,
			},
			CPUUsage:  cluster.ResourcePercent{Percent: 45.0},
			MemUsage:  cluster.ResourcePercent{Percent: 60.0},
			Events24h: 12,
		},
		Workloads: cluster.OverviewWorkloads{
			Summary: cluster.WorkloadSummary{
				Deployments:  cluster.WorkloadStatus{Total: 10, Ready: 9},
				DaemonSets:   cluster.WorkloadStatus{Total: 3, Ready: 3},
				StatefulSets: cluster.WorkloadStatus{Total: 2, Ready: 2},
				Jobs:         cluster.JobStatus{Total: 5, Running: 1, Succeeded: 3, Failed: 1},
			},
			PodStatus: cluster.PodStatusDistribution{
				Total:            50,
				Running:          45,
				Pending:          2,
				Failed:           1,
				Succeeded:        2,
				RunningPercent:   90.0,
				PendingPercent:   4.0,
				FailedPercent:    2.0,
				SucceededPercent: 4.0,
			},
			PeakStats: &cluster.PeakStats{
				PeakCPU:     85.0,
				PeakCPUNode: "node-3",
				PeakMem:     92.0,
				PeakMemNode: "node-1",
				HasData:     true,
			},
		},
		Alerts: cluster.OverviewAlerts{
			Trend: []cluster.AlertTrendPoint{
				{At: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC), Kinds: map[string]int{"Pod": 3, "Node": 1}},
			},
			Totals: cluster.AlertTotals{Critical: 1, Warning: 5, Info: 6},
			Recent: []cluster.RecentAlert{
				{Timestamp: "2025-01-15T10:30:00Z", Severity: "warning", Kind: "Pod", Namespace: "default", Name: "test-pod", Message: "OOMKilled", Reason: "OOM"},
			},
		},
		Nodes: cluster.OverviewNodes{
			Usage: []cluster.NodeUsage{
				{Node: "node-1", CPUUsage: 45.0, MemUsage: 60.0},
				{Node: "node-2", CPUUsage: 30.0, MemUsage: 50.0},
			},
		},
	}

	result := Overview(src)

	// ClusterID: cluster_id → clusterId
	if result.ClusterID != "cluster-1" {
		t.Errorf("ClusterID: got %q, want %q", result.ClusterID, "cluster-1")
	}

	// Cards: cluster_health → clusterHealth
	if result.Cards.ClusterHealth.NodeReadyPercent != 100.0 {
		t.Errorf("Cards.ClusterHealth.NodeReadyPercent: got %f, want %f", result.Cards.ClusterHealth.NodeReadyPercent, 100.0)
	}

	// Cards: events_24h → events24h
	if result.Cards.Events24h != 12 {
		t.Errorf("Cards.Events24h: got %d, want %d", result.Cards.Events24h, 12)
	}

	// Workloads: pod_status → podStatus
	if result.Workloads.PodStatus.RunningPercent != 90.0 {
		t.Errorf("Workloads.PodStatus.RunningPercent: got %f, want %f", result.Workloads.PodStatus.RunningPercent, 90.0)
	}

	// Workloads: peak_stats → peakStats
	if result.Workloads.PeakStats == nil {
		t.Fatal("Workloads.PeakStats should not be nil")
	}
	if result.Workloads.PeakStats.PeakCPU != 85.0 {
		t.Errorf("PeakStats.PeakCPU: got %f, want %f", result.Workloads.PeakStats.PeakCPU, 85.0)
	}
	if result.Workloads.PeakStats.PeakCPUNode != "node-3" {
		t.Errorf("PeakStats.PeakCPUNode: got %q, want %q", result.Workloads.PeakStats.PeakCPUNode, "node-3")
	}

	// Nodes: cpu_usage → cpuUsage, mem_usage → memUsage
	if len(result.Nodes.Usage) != 2 {
		t.Fatalf("Nodes.Usage length: got %d, want 2", len(result.Nodes.Usage))
	}
	if result.Nodes.Usage[0].CPUUsage != 45.0 {
		t.Errorf("Nodes.Usage[0].CPUUsage: got %f, want %f", result.Nodes.Usage[0].CPUUsage, 45.0)
	}

	// Alerts: 保持结构不变
	if len(result.Alerts.Trend) != 1 {
		t.Fatalf("Alerts.Trend length: got %d, want 1", len(result.Alerts.Trend))
	}
	if result.Alerts.Totals.Critical != 1 {
		t.Errorf("Alerts.Totals.Critical: got %d, want %d", result.Alerts.Totals.Critical, 1)
	}
}

// TestOverview_NilInput 测试 nil 输入
func TestOverview_NilInput(t *testing.T) {
	result := Overview(nil)
	if result.ClusterID != "" {
		t.Errorf("nil input should return zero value, got ClusterID=%q", result.ClusterID)
	}
	if result.Nodes.Usage == nil {
		t.Error("nil input should have non-nil empty Usage slice")
	}
}

// TestOverview_NilPeakStats 测试 PeakStats 为 nil
func TestOverview_NilPeakStats(t *testing.T) {
	src := &cluster.ClusterOverview{
		Workloads: cluster.OverviewWorkloads{
			PeakStats: nil,
		},
	}
	result := Overview(src)
	if result.Workloads.PeakStats != nil {
		t.Error("PeakStats should be nil when source is nil")
	}
}
