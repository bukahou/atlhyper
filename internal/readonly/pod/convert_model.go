package pod

import (
	"fmt"
	"time"

	modelpod "NeuroController/model/pod"

	corev1 "k8s.io/api/core/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// buildSkeleton —— 基于 corev1.Pod 构建“骨架”（不含 metrics）
func buildSkeleton(p *corev1.Pod) modelpod.Pod {
	readyOK, total := readyCount(p.Status.ContainerStatuses, len(p.Spec.Containers))
	restarts := totalRestarts(p.Status.ContainerStatuses)

	startTime := time.Time{}
	if p.Status.StartTime != nil {
		startTime = p.Status.StartTime.Time
	}

	return modelpod.Pod{
		Summary: modelpod.PodSummary{
			Name:         p.Name,
			Namespace:    p.Namespace,
			ControlledBy: firstOwner(p.OwnerReferences),
			Phase:        string(p.Status.Phase),
			Ready:        fmt.Sprintf("%d/%d", readyOK, total),
			Restarts:     int32(restarts),
			StartTime:    startTime,
			Age:          fmtAge(startTime),
			Node:         p.Spec.NodeName,
			PodIP:        p.Status.PodIP,
			QoSClass:     string(p.Status.QOSClass),
			Reason:       p.Status.Reason,
			Message:      p.Status.Message,
			Badges:       deriveBadges(p),
		},
		Spec:       buildSpec(p),
		Containers: mapContainers(p.Spec.Containers, p.Status.ContainerStatuses),
		Volumes:    mapVolumes(p.Spec.Volumes),
		Network:    buildNetwork(p),
	}
}

// attachMetrics —— 在骨架上填充 metrics（依据 PodMetrics 与其 spec.containers）
func attachMetrics(dst *modelpod.Pod, pm *metricsv1beta1.PodMetrics, specContainers interface{}) {
	if pm == nil {
		return
	}
	// specContainers 是 cp.Spec.Containers（corev1.Container 切片）
	dst.Metrics = buildPodMetrics(pm, specContainers.([]corev1.Container))
}

// （保留给包内/测试使用；外部不导出）
func toModel(p *corev1.Pod, pm *metricsv1beta1.PodMetrics) modelpod.Pod {
	m := buildSkeleton(p)
	if pm != nil {
		attachMetrics(&m, pm, p.Spec.Containers)
	}
	return m
}

func buildSpec(p *corev1.Pod) modelpod.PodSpec {
	return modelpod.PodSpec{
		RestartPolicy:                 string(p.Spec.RestartPolicy),
		PriorityClassName:             p.Spec.PriorityClassName,
		NodeSelector:                  p.Spec.NodeSelector,
		Tolerations:                   p.Spec.Tolerations,
		Affinity:                      p.Spec.Affinity,
		TopologySpreadConstraints:     p.Spec.TopologySpreadConstraints,
		RuntimeClassName:              stringPtrValue(p.Spec.RuntimeClassName),
		TerminationGracePeriodSeconds: p.Spec.TerminationGracePeriodSeconds,
	}
}
