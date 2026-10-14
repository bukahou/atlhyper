// 确定性异常的 Detail：时间线与 AI 上下文靠它说清「是什么触发的」
package baseline

import (
	"strings"
	"testing"
	"time"

	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/cluster"
)

// 同一 Pod 多条 critical event → 取最近一条，Detail 含 reason 与 message
func TestExtractEventAnomalies_DetailUsesLatestEvent(t *testing.T) {
	now := time.Now()
	older := makeEvent("Warning", "BackOff", "Pod", "default", "p", now.Add(-4*time.Minute))
	older.Message = "Back-off restarting failed container"
	newer := makeEvent("Warning", "Unhealthy", "Pod", "default", "p", now.Add(-1*time.Minute))
	newer.Message = "Readiness probe failed: " + strings.Repeat("x", 300)
	snap := &cluster.ClusterSnapshot{
		Pods:   []cluster.Pod{makePod("default", "p", "Running", 0, makeContainer("app", "running", "", true, 0, ""))},
		Events: []cluster.Event{older, newer},
	}

	r := findResult(extractEventAnomalies(snap, now.Unix()), "default/pod/p", "critical_event")
	if r == nil {
		t.Fatal("应生成 critical_event")
	}
	if !strings.HasPrefix(r.Detail, "Unhealthy: Readiness probe failed: ") {
		t.Errorf("应取最近一条事件，Detail = %q", r.Detail)
	}
	if n := len([]rune(r.Detail)); n > 160+len("Unhealthy: ")+1 {
		t.Errorf("message 应截断，Detail 长 %d", n)
	}
}

func TestExtractContainerAnomalies_DetailNamesContainer(t *testing.T) {
	snap := &cluster.ClusterSnapshot{Pods: []cluster.Pod{
		makePod("default", "p", "Running", 3,
			makeContainer("sidecar", "running", "", true, 0, ""),
			makeContainer("app", "waiting", "CrashLoopBackOff", false, 3, ""),
		),
	}}
	r := findResult(extractContainerAnomalies(snap, time.Now().Unix()), "default/pod/p", "container_anomaly")
	if r == nil || r.Detail != "容器 app: CrashLoopBackOff" {
		t.Errorf("Detail 应指明最严重的容器，got %+v", r)
	}
}

func TestExtractDeploymentImpact_DetailNamesDeployment(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Deployments: []cluster.Deployment{makeDeployment("default", "web", 4, 1)},
		ReplicaSets: []cluster.ReplicaSet{makeReplicaSet("default", "web-rs", "web")},
		Pods: []cluster.Pod{
			makePodWithOwner("default", "web-1", "Running", 5, "ReplicaSet", "web-rs",
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 5, "")),
		},
	}
	r := findResult(extractDeploymentImpact(snap, time.Now().Unix()), "default/pod/web-1", "deployment_impact")
	if r == nil || r.Detail != "Deployment default/web 不可用 75%" {
		t.Errorf("got %+v", r)
	}
}

func TestExtractNodePressure_Detail(t *testing.T) {
	node := cluster.Node{Metrics: &cluster.NodeResourceUsage{}}
	node.Summary.Name = "n1"
	node.Metrics.Pressure.MemoryPressure = true
	snap := &cluster.ClusterSnapshot{Nodes: []cluster.Node{node}}
	r := findResult(extractNodePressure(snap, time.Now().Unix()), "_cluster/node/n1", "memory_pressure")
	if r == nil || r.Detail != "Node condition MemoryPressure=True" {
		t.Errorf("got %+v", r)
	}
}

func TestExtractOTelDeterministic_Detail(t *testing.T) {
	otel := &cluster.OTelSnapshot{
		APMServices: []apm.APMService{{Name: "api", Namespace: "default", SuccessRate: 0.8, P99Ms: 6200}},
	}
	results := ExtractOTelDeterministicAnomalies(otel)
	if r := findResult(results, "default/service/api", "apm_high_error_rate"); r == nil || r.Detail != "错误率 20.0%" {
		t.Errorf("error rate detail: %+v", r)
	}
	if r := findResult(results, "default/service/api", "apm_high_p99_latency"); r == nil || r.Detail != "P99 6200ms" {
		t.Errorf("p99 detail: %+v", r)
	}
}
