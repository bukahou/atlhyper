// atlhyper_master_v2/service/query/aiops_incident_view.go
// 事件详情：按当前集群快照补上成员状态与影响数字（见设计文档「功能四：事件详情页改版」）
package query

import (
	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/model_v3/cluster"
)

// enrichIncidentDetail 用快照补成员的当前状态（是否还在、状态、重启、就绪）与影响数字。
// 快照不可用时不动成员、影响为 nil
func enrichIncidentDetail(detail *aiops.IncidentDetail, snap *cluster.ClusterSnapshot) {
	if detail == nil || snap == nil {
		return
	}
	pods := make(map[string]*cluster.Pod, len(snap.Pods))
	for i := range snap.Pods {
		pod := &snap.Pods[i]
		pods[aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)] = pod
	}
	nodes := make(map[string]*cluster.Node, len(snap.Nodes))
	for i := range snap.Nodes {
		node := &snap.Nodes[i]
		nodes[aiops.EntityKey("", "node", node.Summary.Name)] = node
	}

	restartsTotal, anyPod, presentCount := 0, false, 0
	for i := range detail.Members {
		member := &detail.Members[i]
		switch member.EntityType {
		case "pod":
			pod, ok := pods[member.EntityKey]
			if !ok {
				continue
			}
			restarts := int(pod.Status.Restarts)
			member.Present, member.Status, member.Restarts, member.Ready = true, podDisplayStatus(pod), &restarts, pod.Status.Ready
			restartsTotal += restarts
			anyPod = true
		case "node":
			if node, ok := nodes[member.EntityKey]; ok {
				member.Present, member.Status = true, node.Summary.Ready
			}
		default:
			// 服务等其他类型：快照里没有可展示的状态，按仍存在处理
			member.Present = true
		}
		if member.Present {
			presentCount++
		}
	}

	// 结论里的「N / 总数」：成员表会留着已删除的旧 Pod（重建前的名字），总数按当前还在的成员算，
	// 否则会出现「3 / 6 个 Pod」这种把旧名字算进去的说法
	if detail.Headline != nil && presentCount > 0 {
		detail.Headline.TotalMembers = max(presentCount, detail.Headline.AffectedMembers)
	}

	impact := &aiops.IncidentImpact{}
	if anyPod {
		impact.RestartsTotal = &restartsTotal
	}
	impact.DesiredReplicas, impact.AvailableReplicas = workloadReplicas(detail.Subject, snap)
	detail.Impact = impact
	detail.SubjectPresent = subjectPresent(detail.Subject, snap)
}

// subjectPresent 主体的对象在快照里还在不在；类型不在快照里（如 logs）时为 nil（不知道）
func subjectPresent(subjectKey string, snap *cluster.ClusterSnapshot) *bool {
	namespace, subjectType, name := aiops.SplitEntityKey(subjectKey)
	found := false
	switch subjectType {
	case "deployment":
		for _, d := range snap.Deployments {
			found = found || (d.Summary.Namespace == namespace && d.Summary.Name == name)
		}
	case "statefulset":
		for _, s := range snap.StatefulSets {
			found = found || (s.Summary.Namespace == namespace && s.Summary.Name == name)
		}
	case "daemonset":
		for _, d := range snap.DaemonSets {
			found = found || (d.Summary.Namespace == namespace && d.Summary.Name == name)
		}
	case "pod":
		for _, p := range snap.Pods {
			found = found || (p.Summary.Namespace == namespace && p.Summary.Name == name)
		}
	case "service":
		for _, s := range snap.Services {
			found = found || (s.Summary.Namespace == namespace && s.Summary.Name == name)
		}
	case "node":
		for _, n := range snap.Nodes {
			found = found || n.Summary.Name == name
		}
	default:
		return nil
	}
	return &found
}

// podDisplayStatus 与 kubectl 的 STATUS 一列相近：容器在等待且有原因时用原因（CrashLoopBackOff 等），否则用 Pod 的原因或阶段
func podDisplayStatus(pod *cluster.Pod) string {
	for _, c := range pod.Containers {
		if c.State == "waiting" && c.StateReason != "" {
			return c.StateReason
		}
	}
	for _, c := range pod.Containers {
		if c.State == "terminated" && c.StateReason != "" {
			return c.StateReason
		}
	}
	if pod.Status.Reason != "" {
		return pod.Status.Reason
	}
	return pod.Status.Phase
}

// workloadReplicas 主体是 Deployment / StatefulSet / DaemonSet 时的期望与可用副本数；其余为 nil
func workloadReplicas(subjectKey string, snap *cluster.ClusterSnapshot) (desired, available *int) {
	namespace, subjectType, name := aiops.SplitEntityKey(subjectKey)
	pair := func(d, a int32) (*int, *int) {
		desiredValue, availableValue := int(d), int(a)
		return &desiredValue, &availableValue
	}
	switch subjectType {
	case "deployment":
		for _, d := range snap.Deployments {
			if d.Summary.Namespace == namespace && d.Summary.Name == name {
				return pair(d.Summary.Replicas, d.Summary.Available)
			}
		}
	case "statefulset":
		for _, s := range snap.StatefulSets {
			if s.Summary.Namespace == namespace && s.Summary.Name == name {
				return pair(s.Summary.Replicas, s.Summary.Available)
			}
		}
	case "daemonset":
		for _, d := range snap.DaemonSets {
			if d.Summary.Namespace == namespace && d.Summary.Name == name {
				return pair(d.Summary.DesiredNumberScheduled, d.Summary.NumberAvailable)
			}
		}
	}
	return nil, nil
}
