// Package cluster 定义 K8s 集群资源数据模型
//
// 包含所有 K8s 资源类型、集群快照、概览。
// Agent ↔ Master 通信的核心数据结构。
package cluster

// ============================================================
// 集群域内共用类型
// ============================================================

// LabelSelector 标签选择器
type LabelSelector struct {
	MatchLabels      map[string]string `json:"matchLabels,omitempty"`
	MatchExpressions []LabelExpr       `json:"matchExpressions,omitempty"`
}

// LabelExpr 标签表达式
type LabelExpr struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// Toleration 容忍
type Toleration struct {
	Key               string `json:"key,omitempty"`
	Operator          string `json:"operator,omitempty"`
	Value             string `json:"value,omitempty"`
	Effect            string `json:"effect,omitempty"`
	TolerationSeconds *int64 `json:"tolerationSeconds,omitempty"`
}

// Affinity 亲和性（简化版）
type Affinity struct {
	NodeAffinity    string `json:"nodeAffinity,omitempty"`
	PodAffinity     string `json:"podAffinity,omitempty"`
	PodAntiAffinity string `json:"podAntiAffinity,omitempty"`
}

// WorkloadCondition 工作负载状态条件（通用）
type WorkloadCondition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
	LastUpdateTime     string `json:"lastUpdateTime,omitempty"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
}

// WorkloadRollout 工作负载发布状态
type WorkloadRollout struct {
	Phase   string   `json:"phase"`
	Message string   `json:"message,omitempty"`
	Badges  []string `json:"badges,omitempty"`
}

// ============================================================
// Pod 模板及容器（Deployment/StatefulSet/DaemonSet/Job 共用）
// ============================================================

// PodTemplate Pod 模板
type PodTemplate struct {
	Labels             map[string]string `json:"labels,omitempty"`
	Annotations        map[string]string `json:"annotations,omitempty"`
	Containers         []ContainerDetail `json:"containers"`
	Volumes            []VolumeSpec      `json:"volumes,omitempty"`
	ServiceAccountName string            `json:"serviceAccountName,omitempty"`
	NodeSelector       map[string]string `json:"nodeSelector,omitempty"`
	Tolerations        []Toleration      `json:"tolerations,omitempty"`
	Affinity           *Affinity         `json:"affinity,omitempty"`
	RuntimeClassName   string            `json:"runtimeClassName,omitempty"`
	ImagePullSecrets   []string          `json:"imagePullSecrets,omitempty"`
	HostNetwork        bool              `json:"hostNetwork,omitempty"`
	DNSPolicy          string            `json:"dnsPolicy,omitempty"`
}

// ContainerDetail 容器详情
type ContainerDetail struct {
	Name            string            `json:"name"`
	Image           string            `json:"image"`
	ImagePullPolicy string            `json:"imagePullPolicy,omitempty"`
	Ports           []ContainerPort   `json:"ports,omitempty"`
	Envs            []EnvVar          `json:"envs,omitempty"`
	VolumeMounts    []VolumeMount     `json:"volumeMounts,omitempty"`
	Requests        map[string]string `json:"requests,omitempty"`
	Limits          map[string]string `json:"limits,omitempty"`
	LivenessProbe   *Probe            `json:"livenessProbe,omitempty"`
	ReadinessProbe  *Probe            `json:"readinessProbe,omitempty"`
	StartupProbe    *Probe            `json:"startupProbe,omitempty"`
	Command         []string          `json:"command,omitempty"`
	Args            []string          `json:"args,omitempty"`
	WorkingDir      string            `json:"workingDir,omitempty"`
}

// ContainerPort 容器端口
type ContainerPort struct {
	Name          string `json:"name,omitempty"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
	HostPort      int32  `json:"hostPort,omitempty"`
}

// EnvVar 环境变量
type EnvVar struct {
	Name      string `json:"name"`
	Value     string `json:"value,omitempty"`
	ValueFrom string `json:"valueFrom,omitempty"`
}

// VolumeMount 卷挂载
type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	SubPath   string `json:"subPath,omitempty"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

// VolumeSpec 卷定义
type VolumeSpec struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Source string `json:"source,omitempty"`
}

// Probe 探针
type Probe struct {
	Type                string `json:"type"`
	Path                string `json:"path,omitempty"`
	Port                int32  `json:"port,omitempty"`
	Command             string `json:"command,omitempty"`
	InitialDelaySeconds int32  `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int32  `json:"periodSeconds,omitempty"`
	TimeoutSeconds      int32  `json:"timeoutSeconds,omitempty"`
	SuccessThreshold    int32  `json:"successThreshold,omitempty"`
	FailureThreshold    int32  `json:"failureThreshold,omitempty"`
}
