// atlhyper_master_v2/model/convert/pod.go
// model_v2.Pod → model.PodItem / model.PodDetail 转换函数
package convert

import (
	"strings"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// PodItem 转换为列表项（扁平）
func PodItem(src *model_v2.Pod) model.PodItem {
	deployment := inferDeployment(src)
	cpuText := src.Status.CPUUsage
	if cpuText == "" {
		cpuText = "-"
	}
	memText := src.Status.MemoryUsage
	if memText == "" {
		memText = "-"
	}

	return model.PodItem{
		Name:       src.Summary.Name,
		Namespace:  src.Summary.Namespace,
		Deployment: deployment,
		Ready:      src.Status.Ready,
		Phase:      src.Status.Phase,
		Restarts:   src.Status.Restarts,
		CPUText:    cpuText,
		MemoryText: memText,
		StartTime:  src.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Node:       src.Summary.NodeName,
		Age:        src.Summary.Age,
	}
}

// PodItems 转换多个 Pod 为列表项
func PodItems(src []model_v2.Pod) []model.PodItem {
	if src == nil {
		return []model.PodItem{}
	}
	result := make([]model.PodItem, len(src))
	for i := range src {
		result[i] = PodItem(&src[i])
	}
	return result
}

// PodDetail 转换为详情（扁平）
func PodDetail(src *model_v2.Pod) model.PodDetail {
	controller := ""
	if src.Summary.OwnerKind != "" && src.Summary.OwnerName != "" {
		controller = src.Summary.OwnerKind + "/" + src.Summary.OwnerName
	}

	containers := make([]model.PodContainerResponse, len(src.Containers))
	for i, c := range src.Containers {
		containers[i] = convertPodContainer(c)
	}

	var volumes []model.PodVolumeResponse
	for _, v := range src.Volumes {
		volumes = append(volumes, model.PodVolumeResponse{
			Name:        v.Name,
			Type:        v.Type,
			SourceBrief: v.Source,
		})
	}

	return model.PodDetail{
		Name:       src.Summary.Name,
		Namespace:  src.Summary.Namespace,
		Controller: controller,
		Phase:      src.Status.Phase,
		Ready:      src.Status.Ready,
		Restarts:   src.Status.Restarts,
		StartTime:  src.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Age:        src.Summary.Age,
		Node:       src.Summary.NodeName,
		PodIP:      src.Status.PodIP,
		HostIP:     src.Status.HostIP,
		QoSClass:   src.Status.QoSClass,
		Reason:     src.Status.Reason,
		Message:    src.Status.Message,

		RestartPolicy:                 src.Spec.RestartPolicy,
		PriorityClassName:             src.Spec.PriorityClassName,
		RuntimeClassName:              src.Spec.RuntimeClassName,
		TerminationGracePeriodSeconds: src.Spec.TerminationGracePeriodSeconds,
		Tolerations:                   src.Spec.Tolerations,
		Affinity:                      src.Spec.Affinity,
		NodeSelector:                  src.Spec.NodeSelector,

		HostNetwork:        src.Spec.HostNetwork,
		PodIPs:             src.Status.PodIPs,
		DNSPolicy:          src.Spec.DNSPolicy,
		ServiceAccountName: src.Spec.ServiceAccountName,

		CPUUsage: src.Status.CPUUsage,
		MemUsage: src.Status.MemoryUsage,

		Containers: containers,
		Volumes:    volumes,
	}
}

func convertPodContainer(c model_v2.PodContainerDetail) model.PodContainerResponse {
	return model.PodContainerResponse{
		Name:                 c.Name,
		Image:                c.Image,
		ImagePullPolicy:      c.ImagePullPolicy,
		Ports:                c.Ports,
		Envs:                 c.Envs,
		VolumeMounts:         c.VolumeMounts,
		Requests:             c.Requests,
		Limits:               c.Limits,
		ReadinessProbe:       c.ReadinessProbe,
		LivenessProbe:        c.LivenessProbe,
		StartupProbe:         c.StartupProbe,
		State:                c.State,
		RestartCount:         c.RestartCount,
		LastTerminatedReason: c.LastTerminationReason,
	}
}

// inferDeployment 从 Pod 的 Owner 推断 Deployment 名称
func inferDeployment(pod *model_v2.Pod) string {
	if pod.Summary.OwnerKind == "ReplicaSet" && pod.Summary.OwnerName != "" {
		// ReplicaSet 命名: deployment-name-hash
		parts := strings.Split(pod.Summary.OwnerName, "-")
		if len(parts) > 1 {
			return strings.Join(parts[:len(parts)-1], "-")
		}
		return pod.Summary.OwnerName
	}
	if pod.Summary.OwnerKind == "Deployment" && pod.Summary.OwnerName != "" {
		return pod.Summary.OwnerName
	}
	if pod.Labels != nil {
		if app, ok := pod.Labels["app"]; ok {
			return app
		}
		if app, ok := pod.Labels["app.kubernetes.io/name"]; ok {
			return app
		}
	}
	return ""
}
