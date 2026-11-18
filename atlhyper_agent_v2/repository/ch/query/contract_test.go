package query

import (
	"reflect"
	"testing"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		name     string
		actual   []string
		expected []string
		want     []string
	}{
		{"全部符合", []string{"Server", "Client"}, []string{"Server", "Client", "Internal"}, nil},
		{"出现旧版格式", []string{"SPAN_KIND_SERVER", "Client"}, []string{"Server", "Client"}, []string{"SPAN_KIND_SERVER"}},
		{"大小写视为不同", []string{"info"}, []string{"INFO"}, []string{"info"}},
		{"实际为空", nil, []string{"INFO"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := diff(tt.actual, tt.expected); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("diff() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEnumContracts_Registered 保证三类契约都已登记（新增信号源时同步更新此处）。
func TestEnumContracts_Registered(t *testing.T) {
	want := map[string]bool{
		"otel_traces.SpanKind":   false,
		"otel_traces.StatusCode": false,
		"otel_logs.SeverityText": false,
	}
	for _, c := range enumContracts {
		key := c.table + "." + c.column
		if _, ok := want[key]; ok {
			want[key] = true
		}
		if len(c.expected) == 0 {
			t.Errorf("%s 的预期集合为空", key)
		}
	}
	for k, seen := range want {
		if !seen {
			t.Errorf("契约 %s 未登记", k)
		}
	}
}

// 清单里有的指标只在特定硬件上才有（如树莓派的欠压告警位）。
// 集群里没有这种硬件时它必然缺席，这不是采集配置漂移，不能按漂移报错。
func TestMissingMetrics_SeparatesHardwareDependentFromDrift(t *testing.T) {
	collected := make(map[string]struct{})
	for _, name := range NodeExporterMetrics {
		collected[name] = struct{}{}
	}
	delete(collected, "node_hwmon_in_lcrit_alarm_volts") // 硬件相关：没有树莓派
	delete(collected, "node_memory_MemTotal_bytes")      // 任何 Linux 节点都有：缺了就是漂移

	drifted, hardwareAbsent := missingMetrics(collected)

	if len(drifted) != 1 || drifted[0] != "node_memory_MemTotal_bytes" {
		t.Errorf("漂移 = %v, want [node_memory_MemTotal_bytes]", drifted)
	}
	if len(hardwareAbsent) != 1 || hardwareAbsent[0] != "node_hwmon_in_lcrit_alarm_volts" {
		t.Errorf("硬件不提供 = %v, want [node_hwmon_in_lcrit_alarm_volts]", hardwareAbsent)
	}
}

func TestMissingMetrics_AllCollected(t *testing.T) {
	collected := make(map[string]struct{})
	for _, name := range NodeExporterMetrics {
		collected[name] = struct{}{}
	}
	drifted, hardwareAbsent := missingMetrics(collected)
	if len(drifted) != 0 || len(hardwareAbsent) != 0 {
		t.Errorf("全部采集到时不应有缺失，实际 %v %v", drifted, hardwareAbsent)
	}
}

// 标为硬件相关的指标必须在清单里，否则是写错了名字
func TestHardwareDependentMetrics_AreInCatalog(t *testing.T) {
	catalog := make(map[string]struct{})
	for _, name := range NodeExporterMetrics {
		catalog[name] = struct{}{}
	}
	for name := range hardwareDependentMetrics {
		if _, ok := catalog[name]; !ok {
			t.Errorf("%s 标为硬件相关，但不在 NodeExporterMetrics 清单里", name)
		}
	}
}
