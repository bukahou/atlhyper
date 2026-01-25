package pod

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func deriveBadges(p *corev1.Pod) []string {
	var out []string

	// Terminating
	if p.DeletionTimestamp != nil {
		out = append(out, "Terminating")
	}
	// Evicted
	if strings.EqualFold(p.Status.Reason, "Evicted") {
		out = append(out, "Evicted")
	}
	// 容器状态
	for _, s := range p.Status.ContainerStatuses {
		if s.State.Waiting != nil {
			switch s.State.Waiting.Reason {
			case "CrashLoopBackOff":
				out = append(out, "CrashLoopBackOff")
			case "ImagePullBackOff", "ErrImagePull":
				out = append(out, "ImagePullBackOff")
			}
		}
		if s.LastTerminationState.Terminated != nil && s.LastTerminationState.Terminated.Reason == "OOMKilled" {
			out = append(out, "OOMKilled")
		}
	}
	// 去重
	if len(out) <= 1 {
		return out
	}
	seen := map[string]struct{}{}
	uniq := out[:0]
	for _, b := range out {
		if _, ok := seen[b]; !ok {
			uniq = append(uniq, b)
			seen[b] = struct{}{}
		}
	}
	return uniq
}
