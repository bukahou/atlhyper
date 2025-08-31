package pod

import (
	model "AtlHyper/model/pod"
	"time"
)

// PodDetailDTO —— 扁平化后的 Pod 详情（前端友好，少嵌套）
type PodDetailDTO struct {
	// 基本信息（原 summary）
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Controller  string    `json:"controller,omitempty"` // 形如 "ReplicaSet/media-669f7c4b95"
	Phase       string    `json:"phase"`
	Ready       string    `json:"ready"`   // "1/1"
	Restarts    int32     `json:"restarts"`
	StartTime   time.Time `json:"startTime"`
	Age         string    `json:"age,omitempty"`
	Node        string    `json:"node"`
	PodIP       string    `json:"podIP,omitempty"`
	QoSClass    string    `json:"qosClass,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	Message     string    `json:"message,omitempty"`
	Badges      []string  `json:"badges,omitempty"`

	// 调度/策略（原 spec 精简）
	RestartPolicy                 string `json:"restartPolicy,omitempty"`
	PriorityClassName             string `json:"priorityClassName,omitempty"`
	RuntimeClassName              string `json:"runtimeClassName,omitempty"`
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
	// 透传大对象给详情页弹层（可选）
	Tolerations               any `json:"tolerations,omitempty"`
	Affinity                  any `json:"affinity,omitempty"`
	TopologySpreadConstraints any `json:"topologySpreadConstraints,omitempty"`
	NodeSelector              map[string]string `json:"nodeSelector,omitempty"`

	// 网络（扁平）
	HostNetwork        bool     `json:"hostNetwork"`
	HostIP             string   `json:"hostIP,omitempty"`
	PodIPs             []string `json:"podIPs,omitempty"`
	DNSPolicy          string   `json:"dnsPolicy,omitempty"`
	ServiceAccountName string   `json:"serviceAccountName,omitempty"`

	// 指标（扁平）
	CPUUsage    string  `json:"cpuUsage,omitempty"`    // e.g. "0"
	CPULimit    string  `json:"cpuLimit,omitempty"`    // e.g. "1k"
	CPUUtilPct  float64 `json:"cpuUtilPct,omitempty"`  // 0-100，可选
	MemUsage    string  `json:"memUsage,omitempty"`    // e.g. "19232Ki"
	MemLimit    string  `json:"memLimit,omitempty"`    // e.g. "1Gi"
	MemUtilPct  float64 `json:"memUtilPct,omitempty"`  // 0-100

	// 容器（瘦身）
	Containers []ContainerDTO `json:"containers"`

	// 卷（瘦身）
	Volumes []VolumeDTO `json:"volumes,omitempty"`
}

// ContainerDTO —— 只保留常用展示字段；探针、SEC 上屏但不再多级嵌套
type ContainerDTO struct {
	Name            string           `json:"name"`
	Image           string           `json:"image"`
	ImagePullPolicy string           `json:"imagePullPolicy,omitempty"`
	Ports           []ContainerPort  `json:"ports,omitempty"`
	Envs            []EnvKV          `json:"envs,omitempty"`
	VolumeMounts    []VolumeMountDTO `json:"volumeMounts,omitempty"`
	Requests        map[string]string`json:"requests,omitempty"` // {"cpu":"50m","memory":"64Mi"}
	Limits          map[string]string`json:"limits,omitempty"`   // {"cpu":"1","memory":"1Gi"}

	// 探针与安全上下文直接透传，避免再嵌套多级结构
	ReadinessProbe any `json:"readinessProbe,omitempty"`
	LivenessProbe  any `json:"livenessProbe,omitempty"`
	StartupProbe   any `json:"startupProbe,omitempty"`
	Security       any `json:"securityContext,omitempty"`

	// 状态摘要
	State        string `json:"state,omitempty"`
	RestartCount int32  `json:"restartCount,omitempty"`
	LastReason   string `json:"lastTerminatedReason,omitempty"`
}

type ContainerPort struct {
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol"`          // TCP/UDP/SCTP
	Name          string `json:"name,omitempty"`    // 可选
}

type EnvKV struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type VolumeMountDTO struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
	SubPath   string `json:"subPath,omitempty"`
}

type VolumeDTO struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	// 关键来源字段保持一个层级；如果前端需要更多就看 Type 用弹层再展开 SourceRaw
	SourceBrief string `json:"sourceBrief,omitempty"` // 如 "configMap:name" / "pvc:claimName"
	SourceRaw   any    `json:"sourceRaw,omitempty"`   // 原始结构可选透出
}

// ---- 转换函数：从 model/pod.Pod → PodDetailDTO ----
func FromModel(p model.Pod) PodDetailDTO {
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

	// Controller 展示成 "Kind/Name"
	if p.Summary.ControlledBy != nil {
		dto.Controller = p.Summary.ControlledBy.Kind + "/" + p.Summary.ControlledBy.Name
	}

	// Metrics 扁平
	if p.Metrics != nil {
		dto.CPUUsage = p.Metrics.CPU.Usage
		dto.CPULimit = p.Metrics.CPU.Limit
		dto.CPUUtilPct = p.Metrics.CPU.UtilPct
		dto.MemUsage = p.Metrics.Memory.Usage
		dto.MemLimit = p.Metrics.Memory.Limit
		dto.MemUtilPct = p.Metrics.Memory.UtilPct
	}

	// 容器
	dto.Containers = make([]ContainerDTO, 0, len(p.Containers))
	for _, c := range p.Containers {
		cd := ContainerDTO{
			Name:            c.Name,
			Image:           c.Image,
			ImagePullPolicy: c.ImagePullPolicy,
			Requests:        c.Resources.Requests,
			Limits:          c.Resources.Limits,
			ReadinessProbe:  nil,
			LivenessProbe:   nil,
			StartupProbe:    nil,
			Security:        c.SecurityContext,
			State:           c.Status.State,
			RestartCount:    c.Status.RestartCount,
			LastReason:      c.Status.LastTerminatedReason,
		}
		// 端口
		if len(c.Ports) > 0 {
			ports := make([]ContainerPort, 0, len(c.Ports))
			for _, p := range c.Ports {
				ports = append(ports, ContainerPort{
					ContainerPort: p.ContainerPort,
					Protocol:      p.Protocol,
					Name:          p.Name,
				})
			}
			cd.Ports = ports
		}
		// 环境变量
		if len(c.Env) > 0 {
			envs := make([]EnvKV, 0, len(c.Env))
			for _, e := range c.Env {
				envs = append(envs, EnvKV{Name: e.Name, Value: e.Value})
			}
			cd.Envs = envs
		}
		// 挂载
		if len(c.VolumeMounts) > 0 {
			vm := make([]VolumeMountDTO, 0, len(c.VolumeMounts))
			for _, m := range c.VolumeMounts {
				vm = append(vm, VolumeMountDTO{
					Name: m.Name, MountPath: m.MountPath, ReadOnly: m.ReadOnly, SubPath: m.SubPath,
				})
			}
			cd.VolumeMounts = vm
		}
		// 探针（保持一层）
		if c.Probes != nil {
			cd.ReadinessProbe = c.Probes.Readiness
			cd.LivenessProbe = c.Probes.Liveness
			cd.StartupProbe = c.Probes.Startup
		}

		dto.Containers = append(dto.Containers, cd)
	}

	// 卷
	if len(p.Volumes) > 0 {
		dto.Volumes = make([]VolumeDTO, 0, len(p.Volumes))
		for _, v := range p.Volumes {
			brief := "" // 简单生成来源摘要，前端表格可直接展示
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
			dto.Volumes = append(dto.Volumes, VolumeDTO{
				Name:        v.Name,
				Type:        v.Type,
				SourceBrief: brief,
				SourceRaw:   v.Source, // 需要时前端再展开
			})
		}
	}

	return dto
}
