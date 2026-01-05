package pod

import (
	modelpod "AtlHyper/model/k8s"

	corev1 "k8s.io/api/core/v1"
)

// mapContainers: 规格容器 + 状态容器 => model 容器数组
func mapContainers(specs []corev1.Container, statuses []corev1.ContainerStatus) []modelpod.Container {
	out := make([]modelpod.Container, 0, len(specs))
	idx := indexContainerStatus(statuses)

	for _, c := range specs {
		cs := idx[c.Name]
		out = append(out, modelpod.Container{
			Name:            c.Name,
			Image:           c.Image,
			ImagePullPolicy: string(c.ImagePullPolicy),
			Ports:           mapPorts(c.Ports),
			Env:             mapEnvs(c.Env),
			VolumeMounts:    mapMounts(c.VolumeMounts),
			Resources:       mapResources(c.Resources),
			Probes: &modelpod.Probes{
				Readiness: c.ReadinessProbe,
				Liveness:  c.LivenessProbe,
				Startup:   c.StartupProbe,
			},
			SecurityContext: c.SecurityContext,
			Status: modelpod.ContainerStatus{
				State:                containerStateString(cs),
				RestartCount:         restartCount(cs),
				LastTerminatedReason: lastTerminatedReason(cs),
			},
		})
	}
	return out
}

func indexContainerStatus(arr []corev1.ContainerStatus) map[string]*corev1.ContainerStatus {
	m := make(map[string]*corev1.ContainerStatus, len(arr))
	for i := range arr {
		m[arr[i].Name] = &arr[i]
	}
	return m
}

func containerStateString(s *corev1.ContainerStatus) string {
	if s == nil {
		return ""
	}
	switch {
	case s.State.Running != nil:
		return "Running"
	case s.State.Waiting != nil:
		return "Waiting"
	case s.State.Terminated != nil:
		return "Terminated"
	default:
		return ""
	}
}

func lastTerminatedReason(s *corev1.ContainerStatus) string {
	if s == nil || s.LastTerminationState.Terminated == nil {
		return ""
	}
	return s.LastTerminationState.Terminated.Reason
}

func restartCount(s *corev1.ContainerStatus) int32 {
	if s == nil {
		return 0
	}
	return s.RestartCount
}

func mapPorts(ports []corev1.ContainerPort) []modelpod.ContainerPort {
	out := make([]modelpod.ContainerPort, 0, len(ports))
	for _, p := range ports {
		out = append(out, modelpod.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.ContainerPort,
			Protocol:      string(p.Protocol),
		})
	}
	return out
}

func mapEnvs(envs []corev1.EnvVar) []modelpod.EnvKV {
	out := make([]modelpod.EnvKV, 0, len(envs))
	for _, e := range envs {
		ev := modelpod.EnvKV{Name: e.Name}
		if e.Value != "" {
			ev.Value = e.Value
		} else if e.ValueFrom != nil {
			ev.Value = "<FromRef>" // 不直接暴露 Secret/CM 内容
		}
		out = append(out, ev)
	}
	return out
}

func mapMounts(mounts []corev1.VolumeMount) []modelpod.VolumeMount {
	out := make([]modelpod.VolumeMount, 0, len(mounts))
	for _, m := range mounts {
		out = append(out, modelpod.VolumeMount{
			Name:      m.Name,
			MountPath: m.MountPath,
			ReadOnly:  m.ReadOnly,
			SubPath:   m.SubPath,
		})
	}
	return out
}

func mapResources(r corev1.ResourceRequirements) modelpod.Resources {
	req := map[string]string{}
	lim := map[string]string{}
	if !r.Requests.Cpu().IsZero() {
		req["cpu"] = r.Requests.Cpu().String()
	}
	if !r.Requests.Memory().IsZero() {
		req["memory"] = r.Requests.Memory().String()
	}
	if !r.Limits.Cpu().IsZero() {
		lim["cpu"] = r.Limits.Cpu().String()
	}
	if !r.Limits.Memory().IsZero() {
		lim["memory"] = r.Limits.Memory().String()
	}
	return modelpod.Resources{Requests: req, Limits: lim}
}
