package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

func TestNodeItem_UnitConversion(t *testing.T) {
	src := model_v2.Node{
		Summary: model_v2.NodeSummary{
			Name:        "node-1",
			Ready:       "True",
			Schedulable: true,
		},
		Capacity: model_v2.NodeResources{
			CPU:    "4",
			Memory: "8Gi",
		},
		Addresses: model_v2.NodeAddresses{InternalIP: "10.0.0.1"},
		Info: model_v2.NodeInfo{
			OSImage:      "Ubuntu 22.04",
			Architecture: "amd64",
		},
	}

	result := NodeItem(&src)

	if result.Name != "node-1" {
		t.Errorf("Name: got %q, want %q", result.Name, "node-1")
	}
	if !result.Ready {
		t.Error("Ready: got false, want true")
	}
	if result.CPUCores != 4.0 {
		t.Errorf("CPUCores: got %f, want 4.0", result.CPUCores)
	}
	if result.MemoryGiB != 8.0 {
		t.Errorf("MemoryGiB: got %f, want 8.0", result.MemoryGiB)
	}
}

func TestNodeItem_MillicoreCPU(t *testing.T) {
	src := model_v2.Node{
		Capacity: model_v2.NodeResources{CPU: "500m"},
	}
	result := NodeItem(&src)
	if result.CPUCores != 0.5 {
		t.Errorf("CPUCores: got %f, want 0.5", result.CPUCores)
	}
}

func TestNodeDetail_FieldMapping(t *testing.T) {
	src := model_v2.Node{
		Summary: model_v2.NodeSummary{
			Name:         "master-1",
			Roles:        []string{"control-plane"},
			Ready:        "True",
			Schedulable:  false,
			Age:          "30d",
			CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			Badges:       []string{"control-plane"},
		},
		Spec: model_v2.NodeSpec{
			PodCIDRs: []string{"10.244.0.0/24"},
		},
		Capacity: model_v2.NodeResources{
			CPU:    "8",
			Memory: "32Gi",
			Pods:   "110",
		},
		Allocatable: model_v2.NodeResources{
			CPU:    "7500m",
			Memory: "30Gi",
		},
		Addresses: model_v2.NodeAddresses{
			Hostname:   "master-1",
			InternalIP: "192.168.1.10",
		},
		Info: model_v2.NodeInfo{
			KubeletVersion:          "v1.28.5",
			ContainerRuntimeVersion: "containerd://1.7.2",
			KernelVersion:           "5.15.0",
		},
		Conditions: []model_v2.NodeCondition{
			{Type: "Ready", Status: "True", Reason: "KubeletReady"},
		},
		Taints: []model_v2.NodeTaint{
			{Key: "node-role.kubernetes.io/control-plane", Effect: "NoSchedule"},
		},
		Metrics: &model_v2.NodeMetrics{
			CPU:    model_v2.NodeResourceMetric{Usage: "2000m", UtilPct: 25.0},
			Memory: model_v2.NodeResourceMetric{Usage: "16Gi", UtilPct: 50.0},
			Pods:   model_v2.PodCountMetric{Used: 35, Capacity: 110, UtilPct: 31.8},
			Pressure: model_v2.PressureFlags{
				MemoryPressure: false,
				DiskPressure:   false,
			},
		},
	}

	result := NodeDetail(&src)

	if result.CPUCapacityCores != 8.0 {
		t.Errorf("CPUCapacityCores: got %f, want 8.0", result.CPUCapacityCores)
	}
	if result.MemCapacityGiB != 32.0 {
		t.Errorf("MemCapacityGiB: got %f, want 32.0", result.MemCapacityGiB)
	}
	if result.CPUAllocatableCores != 7.5 {
		t.Errorf("CPUAllocatableCores: got %f, want 7.5", result.CPUAllocatableCores)
	}
	if result.PodsCapacity != 110 {
		t.Errorf("PodsCapacity: got %d, want 110", result.PodsCapacity)
	}
	if result.CPUUsageCores != 2.0 {
		t.Errorf("CPUUsageCores: got %f, want 2.0", result.CPUUsageCores)
	}
	if result.MemUsageGiB != 16.0 {
		t.Errorf("MemUsageGiB: got %f, want 16.0", result.MemUsageGiB)
	}
	if result.PodsUsed != 35 {
		t.Errorf("PodsUsed: got %d, want 35", result.PodsUsed)
	}
	if len(result.Conditions) != 1 {
		t.Fatalf("Conditions: got %d, want 1", len(result.Conditions))
	}
	if result.Conditions[0].Type != "Ready" {
		t.Errorf("Condition Type: got %q, want %q", result.Conditions[0].Type, "Ready")
	}
	if len(result.Taints) != 1 {
		t.Fatalf("Taints: got %d, want 1", len(result.Taints))
	}
}

func TestNodeDetail_NilMetrics(t *testing.T) {
	src := model_v2.Node{
		Summary: model_v2.NodeSummary{Name: "node-no-metrics"},
	}
	result := NodeDetail(&src)
	if result.CPUUsageCores != 0 {
		t.Errorf("CPUUsageCores should be 0 without metrics, got %f", result.CPUUsageCores)
	}
}

func TestNodeItems_NilInput(t *testing.T) {
	result := NodeItems(nil)
	if result == nil {
		t.Error("should return non-nil empty slice")
	}
}

func TestCpuToFloat(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"4", 4.0},
		{"500m", 0.5},
		{"4000m", 4.0},
		{"", 0.0},
	}
	for _, tt := range tests {
		got := cpuToFloat(tt.input)
		if got != tt.want {
			t.Errorf("cpuToFloat(%q): got %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestMemToGiB(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"8Gi", 8.0},
		{"1073741824", 1.0}, // 1 GiB in bytes
		{"", 0.0},
	}
	for _, tt := range tests {
		got := memToGiB(tt.input)
		if got != tt.want {
			t.Errorf("memToGiB(%q): got %f, want %f", tt.input, got, tt.want)
		}
	}
}
