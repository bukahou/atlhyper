package cluster

import "time"

// Pod K8s Pod 资源模型
type Pod struct {
	Summary        PodSummary           `json:"summary"`
	Spec           PodSpec              `json:"spec"`
	Status         PodStatus            `json:"status"`
	Containers     []PodContainerDetail `json:"containers"`
	InitContainers []PodContainerDetail `json:"initContainers,omitempty"`
	Volumes        []VolumeSpec         `json:"volumes,omitempty"`
	Labels         map[string]string    `json:"labels,omitempty"`
	Annotations    map[string]string    `json:"annotations,omitempty"`
}

// PodSummary Pod 摘要信息
type PodSummary struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	NodeName  string    `json:"nodeName,omitempty"`
	OwnerKind string    `json:"ownerKind,omitempty"`
	OwnerName string    `json:"ownerName,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	Age       string    `json:"age"`
}

// PodSpec Pod 规格
type PodSpec struct {
	RestartPolicy                 string            `json:"restartPolicy,omitempty"`
	ServiceAccountName            string            `json:"serviceAccountName,omitempty"`
	NodeSelector                  map[string]string `json:"nodeSelector,omitempty"`
	Tolerations                   []Toleration      `json:"tolerations,omitempty"`
	Affinity                      *Affinity         `json:"affinity,omitempty"`
	DNSPolicy                     string            `json:"dnsPolicy,omitempty"`
	HostNetwork                   bool              `json:"hostNetwork,omitempty"`
	RuntimeClassName              string            `json:"runtimeClassName,omitempty"`
	PriorityClassName             string            `json:"priorityClassName,omitempty"`
	TerminationGracePeriodSeconds *int64            `json:"terminationGracePeriodSeconds,omitempty"`
	ImagePullSecrets              []string          `json:"imagePullSecrets,omitempty"`
}

// PodStatus Pod 状态
type PodStatus struct {
	Phase       string         `json:"phase"`
	Ready       string         `json:"ready"`
	Restarts    int32          `json:"restarts"`
	QoSClass    string         `json:"qosClass,omitempty"`
	PodIP       string         `json:"podIP,omitempty"`
	PodIPs      []string       `json:"podIPs,omitempty"`
	HostIP      string         `json:"hostIP,omitempty"`
	Reason      string         `json:"reason,omitempty"`
	Message     string         `json:"message,omitempty"`
	Conditions  []PodCondition `json:"conditions,omitempty"`
	CPUUsage    string         `json:"cpuUsage,omitempty"`
	MemoryUsage string         `json:"memoryUsage,omitempty"`
}

// PodCondition Pod 状态条件
type PodCondition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
}

// PodContainerDetail Pod 容器详情（合并 spec + status）
type PodContainerDetail struct {
	Name            string            `json:"name"`
	Image           string            `json:"image"`
	ImagePullPolicy string            `json:"imagePullPolicy,omitempty"`
	Command         []string          `json:"command,omitempty"`
	Args            []string          `json:"args,omitempty"`
	WorkingDir      string            `json:"workingDir,omitempty"`
	Ports           []ContainerPort   `json:"ports,omitempty"`
	Envs            []EnvVar          `json:"envs,omitempty"`
	VolumeMounts    []VolumeMount     `json:"volumeMounts,omitempty"`
	Requests        map[string]string `json:"requests,omitempty"`
	Limits          map[string]string `json:"limits,omitempty"`
	LivenessProbe   *Probe            `json:"livenessProbe,omitempty"`
	ReadinessProbe  *Probe            `json:"readinessProbe,omitempty"`
	StartupProbe    *Probe            `json:"startupProbe,omitempty"`

	State                  string `json:"state"`
	StateReason            string `json:"stateReason,omitempty"`
	StateMessage           string `json:"stateMessage,omitempty"`
	Ready                  bool   `json:"ready"`
	RestartCount           int32  `json:"restartCount"`
	LastTerminationReason  string `json:"lastTerminationReason,omitempty"`
	LastTerminationMessage string `json:"lastTerminationMessage,omitempty"`
	LastTerminationTime    string `json:"lastTerminationTime,omitempty"`
}

func (p *Pod) GetName() string      { return p.Summary.Name }
func (p *Pod) GetNamespace() string { return p.Summary.Namespace }
func (p *Pod) GetNodeName() string  { return p.Summary.NodeName }
func (p *Pod) IsRunning() bool      { return p.Status.Phase == "Running" }
func (p *Pod) IsPending() bool      { return p.Status.Phase == "Pending" }
func (p *Pod) IsFailed() bool       { return p.Status.Phase == "Failed" }
func (p *Pod) IsSucceeded() bool    { return p.Status.Phase == "Succeeded" }
func (p *Pod) HasRestarts() bool    { return p.Status.Restarts > 0 }

func (p *Pod) IsReady() bool {
	for _, c := range p.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			return true
		}
	}
	return false
}
