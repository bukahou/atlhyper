package node

import (
	"sort"
	"time"

	modelnode "AtlHyper/model/node"

	corev1 "k8s.io/api/core/v1"
)

func buildSkeleton(n *corev1.Node) modelnode.Node {
	creation := n.CreationTimestamp.Time

	return modelnode.Node{
		Summary: modelnode.NodeSummary{
			Name:         n.Name,
			Roles:        rolesFromLabels(n.Labels),
			Ready:        readyCondition(n.Status.Conditions),
			Schedulable:  !n.Spec.Unschedulable,
			Age:          fmtAge(creation),
			CreationTime: creation,
			Badges:       deriveBadges(n.Status.Conditions),
			// Reason/Message：当 Ready!=True 时汇总（简单取 Ready 条件）
			Reason:  condReason(n.Status.Conditions, corev1.NodeReady),
			Message: condMessage(n.Status.Conditions, corev1.NodeReady),
		},
		Spec: modelnode.NodeSpec{
			PodCIDRs:     n.Spec.PodCIDRs,
			ProviderID:   n.Spec.ProviderID,
			Unschedulable: n.Spec.Unschedulable,
		},
		Capacity:    extractNodeResources(n.Status.Capacity),
		Allocatable: extractNodeResources(n.Status.Allocatable),
		Addresses:   extractAddresses(n.Status.Addresses),
		Info: modelnode.NodeInfo{
			OSImage:                 n.Status.NodeInfo.OSImage,
			OperatingSystem:         n.Status.NodeInfo.OperatingSystem,
			Architecture:            n.Status.NodeInfo.Architecture,
			KernelVersion:           n.Status.NodeInfo.KernelVersion,
			ContainerRuntimeVersion: n.Status.NodeInfo.ContainerRuntimeVersion,
			KubeletVersion:          n.Status.NodeInfo.KubeletVersion,
			KubeProxyVersion:        n.Status.NodeInfo.KubeProxyVersion,
		},
		Conditions: convertConditions(n.Status.Conditions),
		Taints:     convertTaints(n.Spec.Taints),
		Labels:     n.Labels, // 如需瘦身可在此裁剪
	}
}

func convertConditions(src []corev1.NodeCondition) []modelnode.NodeCondition {
	out := make([]modelnode.NodeCondition, 0, len(src))
	for _, c := range src {
		out = append(out, modelnode.NodeCondition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			Reason:             c.Reason,
			Message:            c.Message,
			LastHeartbeatTime:  c.LastHeartbeatTime.Time,
			LastTransitionTime: c.LastTransitionTime.Time,
		})
	}
	return out
}

func convertTaints(src []corev1.Taint) []modelnode.Taint {
	out := make([]modelnode.Taint, 0, len(src))
	for i := range src {
		t := src[i]
		var ta *time.Time
		if t.TimeAdded != nil {
			tm := t.TimeAdded.Time
			ta = &tm
		}
		out = append(out, modelnode.Taint{
			Key:       t.Key,
			Value:     t.Value,
			Effect:    string(t.Effect),
			TimeAdded: ta,
		})
	}
	return out
}

func extractAddresses(addrs []corev1.NodeAddress) modelnode.NodeAddresses {
	na := modelnode.NodeAddresses{}
	all := make([]modelnode.Addr, 0, len(addrs))
	for _, a := range addrs {
		all = append(all, modelnode.Addr{Type: string(a.Type), Address: a.Address})
		switch a.Type {
		case corev1.NodeHostName:
			na.Hostname = a.Address
		case corev1.NodeInternalIP:
			na.InternalIP = a.Address
		case corev1.NodeExternalIP:
			na.ExternalIP = a.Address
		}
	}
	na.All = all
	return na
}

func rolesFromLabels(labels map[string]string) []string {
	if len(labels) == 0 {
		return nil
	}
	var roles []string
	for k := range labels {
		// node-role.kubernetes.io/<role>
		const p = "node-role.kubernetes.io/"
		if len(k) > len(p) && k[:len(p)] == p {
			roles = append(roles, k[len(p):])
		}
	}
	if len(roles) == 0 {
		return nil
	}
	sort.Strings(roles)
	return roles
}

func readyCondition(conds []corev1.NodeCondition) string {
	for _, c := range conds {
		if c.Type == corev1.NodeReady {
			return string(c.Status)
		}
	}
	return "Unknown"
}

func condReason(conds []corev1.NodeCondition, t corev1.NodeConditionType) string {
	for _, c := range conds {
		if c.Type == t && c.Status != corev1.ConditionTrue {
			return c.Reason
		}
	}
	return ""
}
func condMessage(conds []corev1.NodeCondition, t corev1.NodeConditionType) string {
	for _, c := range conds {
		if c.Type == t && c.Status != corev1.ConditionTrue {
			return c.Message
		}
	}
	return ""
}
