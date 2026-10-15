package enricher

import (
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v3/cluster"
)

func podWith(ns, name string, containers ...cluster.PodContainerDetail) cluster.Pod {
	p := cluster.Pod{Containers: containers}
	p.Summary.Name, p.Summary.Namespace = name, ns
	return p
}

// A10：后台分析不能调工具，只看得到风险与时间线 → 只能写「原因未知」。
// 把快照里已有的容器状态（当前状态 / 重启次数 / 上次退出）带进上下文，零额外查询
func TestBuildPodStateContext(t *testing.T) {
	snap := &cluster.ClusterSnapshot{Pods: []cluster.Pod{
		podWith("smoke", "crash-1", cluster.PodContainerDetail{
			Name: "app", State: "waiting", StateReason: "CrashLoopBackOff", RestartCount: 5,
			LastTerminationReason: "Error", LastTerminationMessage: "simulated failure: config missing",
			LastTerminationTime: "2026-10-12T13:05:54Z",
		}),
		podWith("other", "unrelated", cluster.PodContainerDetail{Name: "x", State: "running"}),
	}}
	entities := []*database.AIOpsIncidentEntity{
		{EntityKey: "smoke/pod/crash-1", Role: "root_cause"},
		{EntityKey: "smoke/service/web", Role: "affected"}, // 非 Pod 实体不处理
	}

	got := buildPodStateContext(snap, entities)
	for _, want := range []string{"smoke/pod/crash-1", "app", "CrashLoopBackOff", "重启 5 次", "上次退出: Error", "2026-10-12T13:05:54Z", "simulated failure: config missing"} {
		if !strings.Contains(got, want) {
			t.Errorf("缺少 %q：\n%s", want, got)
		}
	}
	if strings.Contains(got, "unrelated") {
		t.Errorf("不得带入与事件无关的 Pod：\n%s", got)
	}
}

func TestBuildPodStateContext_EmptyWhenNoPodOrNoSnapshot(t *testing.T) {
	entities := []*database.AIOpsIncidentEntity{{EntityKey: "smoke/pod/gone", Role: "root_cause"}}
	if got := buildPodStateContext(nil, entities); got != "" {
		t.Errorf("无快照应为空：%q", got)
	}
	if got := buildPodStateContext(&cluster.ClusterSnapshot{}, entities); got != "" {
		t.Errorf("快照里找不到 Pod 应为空：%q", got)
	}
}

// 终止信息可能很长（栈、配置内容）：截断，避免撑大提示词
func TestBuildPodStateContext_TruncatesMessage(t *testing.T) {
	long := strings.Repeat("x", 2000)
	snap := &cluster.ClusterSnapshot{Pods: []cluster.Pod{
		podWith("ns", "p", cluster.PodContainerDetail{Name: "c", LastTerminationReason: "Error", LastTerminationMessage: long}),
	}}
	got := buildPodStateContext(snap, []*database.AIOpsIncidentEntity{{EntityKey: "ns/pod/p"}})
	if len(got) > 600 {
		t.Errorf("应截断终止信息，长度 %d", len(got))
	}
}
