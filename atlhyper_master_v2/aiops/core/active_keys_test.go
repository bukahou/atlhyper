package core

import (
	"reflect"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/aiops/baseline"
	"AtlHyper/atlhyper_master_v2/aiops/correlator"
	"AtlHyper/atlhyper_master_v2/aiops/risk"
	"AtlHyper/atlhyper_master_v2/datahub/memory"
	"AtlHyper/model_v3/cluster"
	logmodel "AtlHyper/model_v3/log"
)

// 活跃实体：有错误日志的服务与 APM 服务一样算作活跃实体（同一拼法）；旧版 Agent 没有按服务的计数时只有 logs/global
func TestExtractActiveEntityKeys_LogServices(t *testing.T) {
	tests := []struct {
		name    string
		summary *logmodel.Summary
		want    map[string]bool
	}{
		{
			name: "有错误日志的服务",
			summary: &logmodel.Summary{ServiceErrors: []logmodel.ServiceErrorCount{
				{Namespace: "jobs", Service: "worker", Count: 5},
			}},
			want: map[string]bool{"_cluster/logs/global": true, "jobs/service/worker": true},
		},
		{
			name:    "旧版 Agent",
			summary: &logmodel.Summary{SeverityCounts: map[string]int64{"ERROR": 3}},
			want:    map[string]bool{"_cluster/logs/global": true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractActiveEntityKeys(&cluster.ClusterSnapshot{}, &cluster.OTelSnapshot{LogsSummary: tt.summary})
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("活跃实体 = %v, want %v", got, tt.want)
			}
		})
	}
}

// 只靠错误日志露面的服务（没有同名 K8s Service，也没有 APM 数据）：每轮快照都不能被当成已消失的实体清掉基线，
// 否则 log_error_count 每轮从头学起，永远停在冷启动
func TestOnSnapshot_LogOnlyServiceKeepsBaseline(t *testing.T) {
	const clusterID = "c1"
	store := memory.NewMemoryStore(time.Hour, time.Hour, time.Hour)
	snap := &cluster.ClusterSnapshot{OTel: &cluster.OTelSnapshot{LogsSummary: &logmodel.Summary{
		ServiceErrors: []logmodel.ServiceErrorCount{{Namespace: "jobs", Service: "worker", Count: 5}},
	}}}
	if err := store.SetSnapshot(clusterID, snap); err != nil {
		t.Fatal(err)
	}
	e := &engine{
		store:        store,
		corr:         correlator.NewCorrelator(),
		stateManager: baseline.NewStateManager(nil),
		scorer:       risk.NewScorer(nil),
		anomalyCache: make(map[string][]*aiops.AnomalyResult),
	}

	const rounds = 3
	for range rounds {
		e.OnSnapshot(clusterID)
	}

	key := aiops.EntityKey("jobs", "service", "worker")
	var learned int64
	for _, s := range e.GetBaseline(clusterID, key).States {
		if s.MetricName == "log_error_count" {
			learned = s.Count
		}
	}
	if learned != rounds {
		t.Errorf("%s 的 log_error_count 基线学了 %d 轮，want %d", key, learned, rounds)
	}
}
