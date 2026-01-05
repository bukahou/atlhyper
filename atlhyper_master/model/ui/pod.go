// atlhyper_master/dto/ui/pod.go
// Pod UI DTOs
package ui

import (
	model "AtlHyper/model/k8s"
	"time"
)

// ====================== Overview ======================

// PodOverviewDTO - Pod 概览页返回结构
type PodOverviewDTO struct {
	Cards PodCards          `json:"cards"`
	Pods  []PodOverviewItem `json:"pods"`
}

type PodCards struct {
	Running int `json:"running"`
	Pending int `json:"pending"`
	Failed  int `json:"failed"`
	Unknown int `json:"unknown"`
}

type PodOverviewItem struct {
	Namespace  string    `json:"namespace"`
	Deployment string    `json:"deployment,omitempty"`
	Name       string    `json:"name"`
	Ready      string    `json:"ready"`
	Phase      string    `json:"phase"`
	Restarts   int32     `json:"restarts"`
	CPU        float64   `json:"cpu"`
	CPUPercent float64   `json:"cpuPercent"`
	Memory     int       `json:"memory"`
	MemPercent float64   `json:"memPercent"`
	CPUText        string `json:"cpuText,omitempty"`
	CPUPercentText string `json:"cpuPercentText,omitempty"`
	MemoryText     string `json:"memoryText,omitempty"`
	MemPercentText string `json:"memPercentText,omitempty"`
	StartTime  time.Time `json:"startTime"`
	Node       string    `json:"node"`
}

// ====================== Detail ======================

// PodDetailDTO - 扁平化后的 Pod 详情
type PodDetailDTO struct {
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Controller  string    `json:"controller,omitempty"`
	Phase       string    `json:"phase"`
	Ready       string    `json:"ready"`
	Restarts    int32     `json:"restarts"`
	StartTime   time.Time `json:"startTime"`
	Age         string    `json:"age,omitempty"`
	Node        string    `json:"node"`
	PodIP       string    `json:"podIP,omitempty"`
	QoSClass    string    `json:"qosClass,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	Message     string    `json:"message,omitempty"`
	Badges      []string  `json:"badges,omitempty"`

	RestartPolicy                 string `json:"restartPolicy,omitempty"`
	PriorityClassName             string `json:"priorityClassName,omitempty"`
	RuntimeClassName              string `json:"runtimeClassName,omitempty"`
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
	Tolerations               any `json:"tolerations,omitempty"`
	Affinity                  any `json:"affinity,omitempty"`
	TopologySpreadConstraints any `json:"topologySpreadConstraints,omitempty"`
	NodeSelector              map[string]string `json:"nodeSelector,omitempty"`

	HostNetwork        bool     `json:"hostNetwork"`
	HostIP             string   `json:"hostIP,omitempty"`
	PodIPs             []string `json:"podIPs,omitempty"`
	DNSPolicy          string   `json:"dnsPolicy,omitempty"`
	ServiceAccountName string   `json:"serviceAccountName,omitempty"`

	CPUUsage    string  `json:"cpuUsage,omitempty"`
	CPULimit    string  `json:"cpuLimit,omitempty"`
	CPUUtilPct  float64 `json:"cpuUtilPct,omitempty"`
	MemUsage    string  `json:"memUsage,omitempty"`
	MemLimit    string  `json:"memLimit,omitempty"`
	MemUtilPct  float64 `json:"memUtilPct,omitempty"`

	Containers []PodContainerDTO `json:"containers"`
	Volumes    []PodVolumeDTO    `json:"volumes,omitempty"`
}

type PodContainerDTO struct {
	Name            string              `json:"name"`
	Image           string              `json:"image"`
	ImagePullPolicy string              `json:"imagePullPolicy,omitempty"`
	Ports           []PodContainerPort  `json:"ports,omitempty"`
	Envs            []PodEnvKV          `json:"envs,omitempty"`
	VolumeMounts    []PodVolumeMountDTO `json:"volumeMounts,omitempty"`
	Requests        map[string]string   `json:"requests,omitempty"`
	Limits          map[string]string   `json:"limits,omitempty"`
	ReadinessProbe  any                 `json:"readinessProbe,omitempty"`
	LivenessProbe   any                 `json:"livenessProbe,omitempty"`
	StartupProbe    any                 `json:"startupProbe,omitempty"`
	Security        any                 `json:"securityContext,omitempty"`
	State           string              `json:"state,omitempty"`
	RestartCount    int32               `json:"restartCount,omitempty"`
	LastReason      string              `json:"lastTerminatedReason,omitempty"`
}

type PodContainerPort struct {
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol"`
	Name          string `json:"name,omitempty"`
}

type PodEnvKV struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type PodVolumeMountDTO struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
	SubPath   string `json:"subPath,omitempty"`
}

type PodVolumeDTO struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	SourceBrief string `json:"sourceBrief,omitempty"`
	SourceRaw   any    `json:"sourceRaw,omitempty"`
}

// ====================== Conversion ======================

// PodFromModel converts model.Pod to PodDetailDTO
func PodFromModel(p model.Pod) PodDetailDTO {
	dto := PodDetailDTO{
		Name:      p.Summary.Name,
		Namespace: p.Summary.Namespace,
		Phase:     p.Summary.Phase,
		Ready:     p.Summary.Ready,
		Restarts:  p.Summary.Restarts,
		StartTime: p.Summary.StartTime,
		Age:       p.Summary.Age,
		Node:      p.Summary.Node,
		PodIP:     p.Summary.PodIP,
		QoSClass:  p.Summary.QoSClass,
		Reason:    p.Summary.Reason,
		Message:   p.Summary.Message,
		Badges:    p.Summary.Badges,

		RestartPolicy:                 p.Spec.RestartPolicy,
		PriorityClassName:             p.Spec.PriorityClassName,
		RuntimeClassName:              p.Spec.RuntimeClassName,
		TerminationGracePeriodSeconds: p.Spec.TerminationGracePeriodSeconds,
		Tolerations:                   p.Spec.Tolerations,
		Affinity:                      p.Spec.Affinity,
		TopologySpreadConstraints:     p.Spec.TopologySpreadConstraints,
		NodeSelector:                  p.Spec.NodeSelector,

		HostNetwork:        p.Network.HostNetwork,
		HostIP:             p.Network.HostIP,
		PodIPs:             p.Network.PodIPs,
		DNSPolicy:          p.Network.DNSPolicy,
		ServiceAccountName: p.Network.ServiceAccountName,
	}

	if p.Summary.ControlledBy != nil {
		dto.Controller = p.Summary.ControlledBy.Kind + "/" + p.Summary.ControlledBy.Name
	}

	if p.Metrics != nil {
		dto.CPUUsage = p.Metrics.CPU.Usage
		dto.CPULimit = p.Metrics.CPU.Limit
		dto.CPUUtilPct = p.Metrics.CPU.UtilPct
		dto.MemUsage = p.Metrics.Memory.Usage
		dto.MemLimit = p.Metrics.Memory.Limit
		dto.MemUtilPct = p.Metrics.Memory.UtilPct
	}

	dto.Containers = make([]PodContainerDTO, 0, len(p.Containers))
	for _, c := range p.Containers {
		cd := PodContainerDTO{
			Name:            c.Name,
			Image:           c.Image,
			ImagePullPolicy: c.ImagePullPolicy,
			Requests:        c.Resources.Requests,
			Limits:          c.Resources.Limits,
			Security:        c.SecurityContext,
			State:           c.Status.State,
			RestartCount:    c.Status.RestartCount,
			LastReason:      c.Status.LastTerminatedReason,
		}
		if len(c.Ports) > 0 {
			ports := make([]PodContainerPort, 0, len(c.Ports))
			for _, p := range c.Ports {
				ports = append(ports, PodContainerPort{
					ContainerPort: p.ContainerPort,
					Protocol:      p.Protocol,
					Name:          p.Name,
				})
			}
			cd.Ports = ports
		}
		if len(c.Env) > 0 {
			envs := make([]PodEnvKV, 0, len(c.Env))
			for _, e := range c.Env {
				envs = append(envs, PodEnvKV{Name: e.Name, Value: e.Value})
			}
			cd.Envs = envs
		}
		if len(c.VolumeMounts) > 0 {
			vm := make([]PodVolumeMountDTO, 0, len(c.VolumeMounts))
			for _, m := range c.VolumeMounts {
				vm = append(vm, PodVolumeMountDTO{
					Name: m.Name, MountPath: m.MountPath, ReadOnly: m.ReadOnly, SubPath: m.SubPath,
				})
			}
			cd.VolumeMounts = vm
		}
		if c.Probes != nil {
			cd.ReadinessProbe = c.Probes.Readiness
			cd.LivenessProbe = c.Probes.Liveness
			cd.StartupProbe = c.Probes.Startup
		}
		dto.Containers = append(dto.Containers, cd)
	}

	if len(p.Volumes) > 0 {
		dto.Volumes = make([]PodVolumeDTO, 0, len(p.Volumes))
		for _, v := range p.Volumes {
			brief := ""
			switch v.Type {
			case "configMap":
				brief = "configMap"
			case "secret":
				brief = "secret"
			case "pvc":
				brief = "pvc"
			case "hostPath":
				brief = "hostPath"
			case "projected":
				brief = "projected"
			}
			dto.Volumes = append(dto.Volumes, PodVolumeDTO{
				Name:        v.Name,
				Type:        v.Type,
				SourceBrief: brief,
				SourceRaw:   v.Source,
			})
		}
	}

	return dto
}
