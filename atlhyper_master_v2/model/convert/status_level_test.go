package convert

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/cluster"
)

func readyPod(phase string, containers ...cluster.PodContainerDetail) *cluster.Pod {
	return &cluster.Pod{
		Status:     cluster.PodStatus{Phase: phase, Conditions: []cluster.PodCondition{{Type: "Ready", Status: "True"}}},
		Containers: containers,
	}
}

// 状态的显示等级由后端判定；阶段是 Running 的 Pod 也可能有容器在 CrashLoopBackOff
func TestPodStatusLevel(t *testing.T) {
	crashing := cluster.PodContainerDetail{State: "waiting", StateReason: "CrashLoopBackOff"}
	running := cluster.PodContainerDetail{State: "running", Ready: true}
	cases := []struct {
		name       string
		pod        *cluster.Pod
		want       model.StatusLevel
		wantReason string
	}{
		{"运行且就绪", readyPod("Running", running), model.StatusSuccess, ""},
		{"运行但有容器崩溃循环", readyPod("Running", running, crashing), model.StatusError, "CrashLoopBackOff"},
		{"运行但未就绪", &cluster.Pod{Status: cluster.PodStatus{Phase: "Running"}}, model.StatusWarning, ""},
		{"等待调度", readyPod("Pending"), model.StatusWarning, ""},
		{"拉镜像失败", readyPod("Pending", cluster.PodContainerDetail{State: "waiting", StateReason: "ImagePullBackOff"}), model.StatusError, "ImagePullBackOff"},
		{"已完成", readyPod("Succeeded"), model.StatusInfo, ""},
		{"失败", readyPod("Failed"), model.StatusError, ""},
		{"未知", readyPod("Unknown"), model.StatusDefault, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			item := PodItem(c.pod)
			if item.StatusLevel != c.want || item.StatusReason != c.wantReason {
				t.Errorf("level/reason = %s/%q, want %s/%q", item.StatusLevel, item.StatusReason, c.want, c.wantReason)
			}
		})
	}
}

func TestContainerStateLevel(t *testing.T) {
	cases := []struct {
		c    cluster.PodContainerDetail
		want model.StatusLevel
	}{
		{cluster.PodContainerDetail{State: "running", Ready: true}, model.StatusSuccess},
		{cluster.PodContainerDetail{State: "running"}, model.StatusWarning},
		{cluster.PodContainerDetail{State: "waiting", StateReason: "ContainerCreating"}, model.StatusWarning},
		{cluster.PodContainerDetail{State: "waiting", StateReason: "ErrImagePull"}, model.StatusError},
		{cluster.PodContainerDetail{State: "terminated", StateReason: "Completed"}, model.StatusInfo},
		{cluster.PodContainerDetail{State: "terminated", StateReason: "OOMKilled"}, model.StatusError},
	}
	for _, c := range cases {
		if got := containerStateLevel(&c.c); got != c.want {
			t.Errorf("%s/%s = %s, want %s", c.c.State, c.c.StateReason, got, c.want)
		}
	}
}

func TestNamespaceStatusLevel(t *testing.T) {
	if namespaceStatusLevel("Active") != model.StatusSuccess || namespaceStatusLevel("Terminating") != model.StatusInfo || namespaceStatusLevel("") != model.StatusDefault {
		t.Error("命名空间阶段 → 等级不对")
	}
}
