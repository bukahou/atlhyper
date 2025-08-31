// internal/readonly/deployment/convert_template.go
package deployment

import (
	modeldep "AtlHyper/model/deployment"
	modelpod "AtlHyper/model/pod"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func toPodTemplate(tpl *corev1.PodTemplateSpec) modeldep.PodTemplate {
	if tpl == nil {
		return modeldep.PodTemplate{}
	}
	pt := modeldep.PodTemplate{
		Labels:             copyStrMap(tpl.Labels),
		Annotations:        pickPodTemplateAnnotations(tpl.Annotations),
		Containers:         mapContainers(tpl.Spec.Containers),
		Volumes:            mapVolumes(tpl.Spec.Volumes),
		ServiceAccountName: tpl.Spec.ServiceAccountName,
		NodeSelector:       copyStrMap(tpl.Spec.NodeSelector),
		Tolerations:        tpl.Spec.Tolerations,  // 透传
		Affinity:           tpl.Spec.Affinity,     // 透传
		RuntimeClassName:   strPtr(tpl.Spec.RuntimeClassName),
		ImagePullSecrets:   toSecretNames(tpl.Spec.ImagePullSecrets),
		HostNetwork:        tpl.Spec.HostNetwork,
		DNSPolicy:          string(tpl.Spec.DNSPolicy),
	}
	return pt
}

// ========== 容器（模板内：没有运行时状态） ==========

func mapContainers(specs []corev1.Container) []modelpod.Container {
	out := make([]modelpod.Container, 0, len(specs))
	for _, c := range specs {
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
			// 模板内无运行状态，Status 留空即可
		})
	}
	return out
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
			ev.Value = "<FromRef>"
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
	if q := r.Requests.Cpu(); q != nil && !q.IsZero() {
		req["cpu"] = q.String()
	}
	if q := r.Requests.Memory(); q != nil && !q.IsZero() {
		req["memory"] = q.String()
	}
	if q := r.Limits.Cpu(); q != nil && !q.IsZero() {
		lim["cpu"] = q.String()
	}
	if q := r.Limits.Memory(); q != nil && !q.IsZero() {
		lim["memory"] = q.String()
	}
	return modelpod.Resources{Requests: req, Limits: lim}
}

// ========== 卷（与 pod/convert_volumes.go 风格一致的精简映射） ==========

func mapVolumes(vols []corev1.Volume) []modelpod.Volume {
	out := make([]modelpod.Volume, 0, len(vols))
	for _, v := range vols {
		t, src := volumeTypeAndSource(&v)
		out = append(out, modelpod.Volume{
			Name:   v.Name,
			Type:   t,
			Source: src,
		})
	}
	return out
}

func volumeTypeAndSource(v *corev1.Volume) (string, any) {
	vs := v.VolumeSource
	switch {
	case vs.ConfigMap != nil:
		return "configMap", map[string]string{"name": vs.ConfigMap.Name}
	case vs.Secret != nil:
		return "secret", map[string]string{"secretName": vs.Secret.SecretName}
	case vs.PersistentVolumeClaim != nil:
		return "persistentVolumeClaim", map[string]string{"claimName": vs.PersistentVolumeClaim.ClaimName}
	case vs.EmptyDir != nil:
		return "emptyDir", map[string]any{
			"medium":    string(vs.EmptyDir.Medium),
			"sizeLimit": quantityOrEmpty(vs.EmptyDir.SizeLimit),
		}
	case vs.HostPath != nil:
		hpt := ""
		if vs.HostPath.Type != nil {
			hpt = string(*vs.HostPath.Type)
		}
		return "hostPath", map[string]string{"path": vs.HostPath.Path, "type": hpt}
	case vs.Projected != nil:
		return "projected", map[string]any{"sources": len(vs.Projected.Sources)}
	case vs.DownwardAPI != nil:
		return "downwardAPI", map[string]int{"items": len(vs.DownwardAPI.Items)}
	case vs.CSI != nil:
		m := map[string]any{"driver": vs.CSI.Driver}
		if vs.CSI.ReadOnly != nil {
			m["readOnly"] = *vs.CSI.ReadOnly
		}
		if vs.CSI.FSType != nil {
			m["fsType"] = *vs.CSI.FSType
		}
		if len(vs.CSI.VolumeAttributes) > 0 {
			m["attrs"] = len(vs.CSI.VolumeAttributes)
		}
		if vs.CSI.NodePublishSecretRef != nil {
			m["secretRef"] = vs.CSI.NodePublishSecretRef.Name
		}
		return "csi", m
	case vs.NFS != nil:
		return "nfs", map[string]string{"server": vs.NFS.Server, "path": vs.NFS.Path}
	default:
		return "other", nil
	}
}

func quantityOrEmpty(q *resource.Quantity) string {
    if q == nil || q.IsZero() {
        return ""
    }
    return q.String()
}

func toSecretNames(secs []corev1.LocalObjectReference) []string {
	if len(secs) == 0 {
		return nil
	}
	out := make([]string, 0, len(secs))
	for _, s := range secs {
		out = append(out, s.Name)
	}
	return out
}
