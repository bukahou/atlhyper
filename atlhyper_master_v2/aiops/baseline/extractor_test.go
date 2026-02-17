// atlhyper_master_v2/aiops/baseline/extractor_test.go
// 容器级指标提取 + 确定性异常直注 + Event 关联信号 测试
package baseline

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/model_v2"
)

// ==================== Phase 1: extractPodMetrics 容器级指标 ====================

func TestExtractPodMetrics_AllHealthy(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "web-abc", "Running", 0,
				makeContainer("nginx", "running", "", true, 0, ""),
				makeContainer("sidecar", "running", "", true, 0, ""),
				makeContainer("init-done", "running", "", true, 0, ""),
			),
		},
	}

	points := extractPodMetrics(snap)
	metrics := indexPoints(points, "default/pod/web-abc")

	assertMetric(t, metrics, "not_ready_containers", 0)
	assertMetric(t, metrics, "max_container_restarts", 0)
	assertMetric(t, metrics, "is_running", 1)
	assertMetric(t, metrics, "restart_count", 0)
}

func TestExtractPodMetrics_OneCrashLoopBackOff(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "mysql-0", "Running", 5,
				makeContainer("mysql", "waiting", "CrashLoopBackOff", false, 5, ""),
				makeContainer("sidecar", "running", "", true, 0, ""),
			),
		},
	}

	points := extractPodMetrics(snap)
	metrics := indexPoints(points, "default/pod/mysql-0")

	assertMetric(t, metrics, "not_ready_containers", 1)
	assertMetric(t, metrics, "max_container_restarts", 5)
	assertMetric(t, metrics, "is_running", 1) // Pod Phase 仍为 Running
}

func TestExtractPodMetrics_NoContainers(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("kube-system", "pending-pod", "Pending", 0),
		},
	}

	points := extractPodMetrics(snap)
	metrics := indexPoints(points, "kube-system/pod/pending-pod")

	assertMetric(t, metrics, "not_ready_containers", 0)
	assertMetric(t, metrics, "max_container_restarts", 0)
	assertMetric(t, metrics, "is_running", 0)
}

// ==================== Phase 2: ExtractDeterministicAnomalies 容器异常 ====================

func TestExtractDeterministic_CrashLoopBackOff(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "crash-pod", "Running", 10,
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 10, ""),
				makeContainer("sidecar", "running", "", true, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/crash-pod", "container_anomaly")
	if found == nil {
		t.Fatal("CrashLoopBackOff 应生成 container_anomaly 结果")
	}
	if found.Score != 0.90 {
		t.Errorf("CrashLoopBackOff score 应为 0.90, got %.2f", found.Score)
	}
	if !found.IsAnomaly {
		t.Error("确定性异常 IsAnomaly 应为 true")
	}
}

func TestExtractDeterministic_OOMKilled(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "oom-pod", "Running", 3,
				makeContainer("app", "waiting", "OOMKilled", false, 3, "OOMKilled"),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/oom-pod", "container_anomaly")
	if found == nil {
		t.Fatal("OOMKilled 应生成 container_anomaly 结果")
	}
	if found.Score != 0.95 {
		t.Errorf("OOMKilled score 应为 0.95, got %.2f", found.Score)
	}
}

func TestExtractDeterministic_ImagePullBackOff(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "img-pod", "Pending", 0,
				makeContainer("app", "waiting", "ImagePullBackOff", false, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/img-pod", "container_anomaly")
	if found == nil {
		t.Fatal("ImagePullBackOff 应生成 container_anomaly 结果")
	}
	if found.Score != 0.70 {
		t.Errorf("ImagePullBackOff score 应为 0.70, got %.2f", found.Score)
	}
}

func TestExtractDeterministic_AllNormal(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "healthy-pod", "Running", 0,
				makeContainer("app", "running", "", true, 0, ""),
				makeContainer("sidecar", "running", "", true, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/healthy-pod", "container_anomaly")
	if found != nil {
		t.Errorf("全正常 Pod 不应生成 container_anomaly, got score=%.2f", found.Score)
	}
}

func TestExtractDeterministic_MultiContainer_TakesWorst(t *testing.T) {
	// 一个容器 ImagePullBackOff(0.70), 另一个 OOMKilled(0.95) → 取最严重
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "multi-bad", "Running", 5,
				makeContainer("app", "waiting", "ImagePullBackOff", false, 0, ""),
				makeContainer("worker", "waiting", "OOMKilled", false, 5, "OOMKilled"),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/multi-bad", "container_anomaly")
	if found == nil {
		t.Fatal("多容器异常应生成结果")
	}
	if found.Score != 0.95 {
		t.Errorf("应取最严重容器 (OOMKilled=0.95), got %.2f", found.Score)
	}
}

func TestExtractDeterministic_TerminatedOOMKilled(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "term-oom", "Running", 1,
				makeContainer("app", "terminated", "", false, 1, "OOMKilled"),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/term-oom", "container_anomaly")
	if found == nil {
		t.Fatal("terminated + LastTerminationReason=OOMKilled 应生成结果")
	}
	if found.Score != 0.95 {
		t.Errorf("OOMKilled score 应为 0.95, got %.2f", found.Score)
	}
}

// ==================== Phase 2: classifyContainerAnomaly ====================

func TestClassifyContainerAnomaly(t *testing.T) {
	tests := []struct {
		name       string
		container  model_v2.PodContainerDetail
		wantReason string
	}{
		{
			"waiting_CrashLoopBackOff",
			model_v2.PodContainerDetail{State: "waiting", StateReason: "CrashLoopBackOff"},
			"CrashLoopBackOff",
		},
		{
			"waiting_OOMKilled",
			model_v2.PodContainerDetail{State: "waiting", StateReason: "OOMKilled"},
			"OOMKilled",
		},
		{
			"waiting_ImagePullBackOff",
			model_v2.PodContainerDetail{State: "waiting", StateReason: "ImagePullBackOff"},
			"ImagePullBackOff",
		},
		{
			"waiting_ErrImagePull",
			model_v2.PodContainerDetail{State: "waiting", StateReason: "ErrImagePull"},
			"ErrImagePull",
		},
		{
			"waiting_CreateContainerConfigError",
			model_v2.PodContainerDetail{State: "waiting", StateReason: "CreateContainerConfigError"},
			"CreateContainerConfigError",
		},
		{
			"terminated_OOMKilled",
			model_v2.PodContainerDetail{State: "terminated", LastTerminationReason: "OOMKilled"},
			"OOMKilled",
		},
		{
			"running_normal",
			model_v2.PodContainerDetail{State: "running", Ready: true},
			"",
		},
		{
			"waiting_ContainerCreating",
			model_v2.PodContainerDetail{State: "waiting", StateReason: "ContainerCreating"},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyContainerAnomaly(&tt.container)
			if got != tt.wantReason {
				t.Errorf("classifyContainerAnomaly() = %q, want %q", got, tt.wantReason)
			}
		})
	}
}

// ==================== Phase 3: Event 关联异常 ====================

func TestExtractEventAnomalies_CriticalPodEvent(t *testing.T) {
	now := time.Now()
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "crash-pod", "Running", 5,
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 5, ""),
			),
		},
		Events: []model_v2.Event{
			makeEvent("Warning", "BackOff", "Pod", "default", "crash-pod", now.Add(-2*time.Minute)),
		},
	}

	results := extractEventAnomalies(snap, now.Unix())
	found := findResult(results, "default/pod/crash-pod", "critical_event")
	if found == nil {
		t.Fatal("关联 Critical Event 应生成 critical_event 结果")
	}
	if found.Score != 0.85 {
		t.Errorf("critical_event score 应为 0.85, got %.2f", found.Score)
	}
}

func TestExtractEventAnomalies_OldEvent(t *testing.T) {
	now := time.Now()
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "old-pod", "Running", 0,
				makeContainer("app", "running", "", true, 0, ""),
			),
		},
		Events: []model_v2.Event{
			makeEvent("Warning", "BackOff", "Pod", "default", "old-pod", now.Add(-10*time.Minute)),
		},
	}

	results := extractEventAnomalies(snap, now.Unix())
	found := findResult(results, "default/pod/old-pod", "critical_event")
	if found != nil {
		t.Error("超过 5 分钟的 Event 不应生成结果")
	}
}

func TestExtractEventAnomalies_PodNotInSnapshot(t *testing.T) {
	now := time.Now()
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{}, // 无 Pod
		Events: []model_v2.Event{
			makeEvent("Warning", "BackOff", "Pod", "default", "ghost-pod", now.Add(-1*time.Minute)),
		},
	}

	results := extractEventAnomalies(snap, now.Unix())
	if len(results) != 0 {
		t.Errorf("关联 Pod 不在快照中时不应生成结果, got %d", len(results))
	}
}

func TestExtractEventAnomalies_NormalEvent(t *testing.T) {
	now := time.Now()
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "normal-pod", "Running", 0,
				makeContainer("app", "running", "", true, 0, ""),
			),
		},
		Events: []model_v2.Event{
			makeEvent("Normal", "Scheduled", "Pod", "default", "normal-pod", now.Add(-1*time.Minute)),
		},
	}

	results := extractEventAnomalies(snap, now.Unix())
	if len(results) != 0 {
		t.Errorf("Normal Event 不应生成结果, got %d", len(results))
	}
}

func TestExtractEventAnomalies_NonPodEvent(t *testing.T) {
	now := time.Now()
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "some-pod", "Running", 0,
				makeContainer("app", "running", "", true, 0, ""),
			),
		},
		Events: []model_v2.Event{
			makeEvent("Warning", "NodeNotReady", "Node", "", "worker-1", now.Add(-1*time.Minute)),
		},
	}

	results := extractEventAnomalies(snap, now.Unix())
	if len(results) != 0 {
		t.Errorf("非 Pod Event 不应生成 Pod 结果, got %d", len(results))
	}
}

func TestExtractEventAnomalies_DeduplicatePerPod(t *testing.T) {
	now := time.Now()
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			makePod("default", "dup-pod", "Running", 3,
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 3, ""),
			),
		},
		Events: []model_v2.Event{
			makeEvent("Warning", "BackOff", "Pod", "default", "dup-pod", now.Add(-1*time.Minute)),
			makeEvent("Warning", "Failed", "Pod", "default", "dup-pod", now.Add(-2*time.Minute)),
		},
	}

	results := extractEventAnomalies(snap, now.Unix())
	count := 0
	for _, r := range results {
		if r.EntityKey == "default/pod/dup-pod" && r.MetricName == "critical_event" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("每个 Pod 只应报告一次 critical_event, got %d", count)
	}
}

// ==================== 辅助函数 ====================

func makePod(namespace, name, phase string, restarts int32, containers ...model_v2.PodContainerDetail) model_v2.Pod {
	return model_v2.Pod{
		Summary: model_v2.PodSummary{
			Name:      name,
			Namespace: namespace,
		},
		Status: model_v2.PodStatus{
			Phase:    phase,
			Restarts: restarts,
		},
		Containers: containers,
	}
}

func makeContainer(name, state, stateReason string, ready bool, restarts int32, lastTermReason string) model_v2.PodContainerDetail {
	return model_v2.PodContainerDetail{
		Name:                  name,
		State:                 state,
		StateReason:           stateReason,
		Ready:                 ready,
		RestartCount:          restarts,
		LastTerminationReason: lastTermReason,
	}
}

func makeEvent(typ, reason, involvedKind, involvedNS, involvedName string, lastTimestamp time.Time) model_v2.Event {
	return model_v2.Event{
		CommonMeta: model_v2.CommonMeta{
			Name:      reason + "-event",
			Namespace: involvedNS,
		},
		Type:           typ,
		Reason:         reason,
		LastTimestamp:   lastTimestamp,
		InvolvedObject: model_v2.ResourceRef{Kind: involvedKind, Namespace: involvedNS, Name: involvedName},
	}
}

// indexPoints 将指标点按 metricName 索引
func indexPoints(points []aiops.MetricDataPoint, entityKey string) map[string]float64 {
	m := make(map[string]float64)
	for _, p := range points {
		if p.EntityKey == entityKey {
			m[p.MetricName] = p.Value
		}
	}
	return m
}

func assertMetric(t *testing.T, metrics map[string]float64, name string, expected float64) {
	t.Helper()
	got, ok := metrics[name]
	if !ok {
		t.Errorf("缺少指标 %s", name)
		return
	}
	if got != expected {
		t.Errorf("指标 %s = %.2f, want %.2f", name, got, expected)
	}
}

// findResult 在结果中查找匹配的异常
func findResult(results []*aiops.AnomalyResult, entityKey, metricName string) *aiops.AnomalyResult {
	for _, r := range results {
		if r.EntityKey == entityKey && r.MetricName == metricName {
			return r
		}
	}
	return nil
}
