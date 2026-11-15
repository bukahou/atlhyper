package query

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/model_v3/cluster"
)

func crashPod(name string, restarts int32) cluster.Pod {
	pod := cluster.Pod{}
	pod.Summary.Name, pod.Summary.Namespace = name, "shop"
	pod.Status.Phase, pod.Status.Ready, pod.Status.Restarts = "Running", "0/1", restarts
	pod.Containers = []cluster.PodContainerDetail{{Name: "app", State: "waiting", StateReason: "CrashLoopBackOff"}}
	return pod
}

func detailWithMembers(subject string, members ...aiops.IncidentMember) *aiops.IncidentDetail {
	detail := &aiops.IncidentDetail{Members: members}
	detail.Subject = subject
	return detail
}

// 成员按快照补状态：还在的 Pod 带状态、重启、就绪；已删除的 present=false；影响数字取工作负载副本与重启合计
func TestEnrichIncidentDetail_MembersAndImpact(t *testing.T) {
	snap := &cluster.ClusterSnapshot{Pods: []cluster.Pod{crashPod("api-a", 6), crashPod("api-b", 4)}}
	deployment := cluster.Deployment{}
	deployment.Summary.Name, deployment.Summary.Namespace, deployment.Summary.Replicas, deployment.Summary.Available = "api", "shop", 3, 0
	snap.Deployments = []cluster.Deployment{deployment}

	detail := detailWithMembers("shop/deployment/api",
		aiops.IncidentMember{EntityKey: "shop/pod/api-a", EntityType: "pod", IsLead: true},
		aiops.IncidentMember{EntityKey: "shop/pod/api-b", EntityType: "pod"},
		aiops.IncidentMember{EntityKey: "shop/pod/api-gone", EntityType: "pod"},
	)
	detail.Headline = &aiops.IncidentHeadline{Signal: aiops.SignalContainerCrash, AffectedMembers: 2, TotalMembers: 3}
	enrichIncidentDetail(detail, snap)
	if detail.Headline.TotalMembers != 2 {
		t.Errorf("结论的总数应按当前还在的成员算（已删除的旧 Pod 不算），实际 %d", detail.Headline.TotalMembers)
	}

	a := detail.Members[0]
	if !a.Present || a.Status != "CrashLoopBackOff" || a.Restarts == nil || *a.Restarts != 6 || a.Ready != "0/1" {
		t.Errorf("还在的 Pod 应补上状态: %+v", a)
	}
	if gone := detail.Members[2]; gone.Present || gone.Status != "" || gone.Restarts != nil {
		t.Errorf("已删除的 Pod 应标 present=false 且不带状态: %+v", gone)
	}
	if detail.SubjectPresent == nil || !*detail.SubjectPresent {
		t.Errorf("Deployment 在快照里，subjectPresent 应为 true: %v", detail.SubjectPresent)
	}
	gone := detailWithMembers("shop/deployment/deleted")
	enrichIncidentDetail(gone, snap)
	if gone.SubjectPresent == nil || *gone.SubjectPresent {
		t.Errorf("已删除的 Deployment，subjectPresent 应为 false: %v", gone.SubjectPresent)
	}
	unknown := detailWithMembers("shop/logs/x")
	enrichIncidentDetail(unknown, snap)
	if unknown.SubjectPresent != nil {
		t.Errorf("快照里没有的类型应为 nil（不知道）: %v", *unknown.SubjectPresent)
	}
	impact := detail.Impact
	if impact == nil || impact.RestartsTotal == nil || *impact.RestartsTotal != 10 ||
		impact.DesiredReplicas == nil || *impact.DesiredReplicas != 3 || impact.AvailableReplicas == nil || *impact.AvailableReplicas != 0 {
		t.Errorf("影响数字不对: %+v", impact)
	}
}

// 主体不是工作负载时没有副本数；没有成员 Pod 时没有重启合计；快照不可用时不动
func TestEnrichIncidentDetail_NotApplicable(t *testing.T) {
	node := cluster.Node{}
	node.Summary.Name, node.Summary.Ready = "worker-1", "True"
	snap := &cluster.ClusterSnapshot{Nodes: []cluster.Node{node}}
	detail := detailWithMembers("_cluster/node/worker-1", aiops.IncidentMember{EntityKey: "_cluster/node/worker-1", EntityType: "node"})
	enrichIncidentDetail(detail, snap)
	if !detail.Members[0].Present || detail.Members[0].Status != "True" {
		t.Errorf("节点成员应补上就绪状态: %+v", detail.Members[0])
	}
	if detail.Impact == nil || detail.Impact.DesiredReplicas != nil || detail.Impact.RestartsTotal != nil {
		t.Errorf("不适用的影响项应为 nil: %+v", detail.Impact)
	}

	untouched := detailWithMembers("shop/deployment/api", aiops.IncidentMember{EntityKey: "shop/pod/x", EntityType: "pod"})
	enrichIncidentDetail(untouched, nil)
	if untouched.Impact != nil || untouched.Members[0].Present {
		t.Errorf("快照不可用时不动: %+v", untouched)
	}
}

func TestPodDisplayStatus(t *testing.T) {
	pod := crashPod("x", 0)
	if got := podDisplayStatus(&pod); got != "CrashLoopBackOff" {
		t.Errorf("等待中的容器原因优先，得到 %s", got)
	}
	pod.Containers = []cluster.PodContainerDetail{{State: "running"}}
	if got := podDisplayStatus(&pod); got != "Running" {
		t.Errorf("没有原因时用阶段，得到 %s", got)
	}
}
