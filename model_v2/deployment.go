package model_v2

import "time"

// ============================================================
// Deployment 模型（与 model/k8s/deployment.go 结构一致）
// ============================================================

// Deployment K8s Deployment 资源模型
type Deployment struct {
	Summary     DeploymentSummary   `json:"summary"`
	Spec        DeploymentSpec      `json:"spec"`
	Template    PodTemplate         `json:"template"`
	Status      DeploymentStatus    `json:"status"`
	Rollout     *DeploymentRollout  `json:"rollout,omitempty"`
	ReplicaSets []ReplicaSetBrief   `json:"replicaSets,omitempty"`
	Annotations map[string]string   `json:"annotations,omitempty"`
	Labels      map[string]string   `json:"labels,omitempty"`
}

// DeploymentSummary 摘要信息
type DeploymentSummary struct {
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Strategy    string    `json:"strategy"`
	Replicas    int32     `json:"replicas"`
	Updated     int32     `json:"updated"`
	Ready       int32     `json:"ready"`
	Available   int32     `json:"available"`
	Unavailable int32     `json:"unavailable,omitempty"`
	Paused      bool      `json:"paused,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	Age         string    `json:"age"`
	Selector    string    `json:"selector,omitempty"`
}

// DeploymentSpec 规格
type DeploymentSpec struct {
	Replicas                *int32                  `json:"replicas,omitempty"`
	Selector                *LabelSelector          `json:"selector,omitempty"`
	Strategy                *DeploymentStrategy     `json:"strategy,omitempty"`
	MinReadySeconds         int32                   `json:"minReadySeconds,omitempty"`
	RevisionHistoryLimit    *int32                  `json:"revisionHistoryLimit,omitempty"`
	ProgressDeadlineSeconds *int32                  `json:"progressDeadlineSeconds,omitempty"`
}

// DeploymentStrategy 更新策略
type DeploymentStrategy struct {
	Type          string                 `json:"type"`
	RollingUpdate *RollingUpdateStrategy `json:"rollingUpdate,omitempty"`
}

// RollingUpdateStrategy 滚动更新策略
type RollingUpdateStrategy struct {
	MaxUnavailable string `json:"maxUnavailable,omitempty"`
	MaxSurge       string `json:"maxSurge,omitempty"`
}

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

// PodTemplate Pod 模板
type PodTemplate struct {
	Labels             map[string]string  `json:"labels,omitempty"`
	Annotations        map[string]string  `json:"annotations,omitempty"`
	Containers         []ContainerDetail  `json:"containers"`
	Volumes            []VolumeSpec       `json:"volumes,omitempty"`
	ServiceAccountName string             `json:"serviceAccountName,omitempty"`
	NodeSelector       map[string]string  `json:"nodeSelector,omitempty"`
	Tolerations        []Toleration       `json:"tolerations,omitempty"`
	Affinity           *Affinity          `json:"affinity,omitempty"`
	RuntimeClassName   string             `json:"runtimeClassName,omitempty"`
	ImagePullSecrets   []string           `json:"imagePullSecrets,omitempty"`
	HostNetwork        bool               `json:"hostNetwork,omitempty"`
	DNSPolicy          string             `json:"dnsPolicy,omitempty"`
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
	ValueFrom string `json:"valueFrom,omitempty"` // 简化：只记录来源描述
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
	Type   string `json:"type"`             // ConfigMap, Secret, EmptyDir, PVC, HostPath, etc.
	Source string `json:"source,omitempty"` // 简化的来源描述
}

// Probe 探针
type Probe struct {
	Type                string `json:"type"` // httpGet, tcpSocket, exec
	Path                string `json:"path,omitempty"`
	Port                int32  `json:"port,omitempty"`
	Command             string `json:"command,omitempty"` // exec 命令
	InitialDelaySeconds int32  `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int32  `json:"periodSeconds,omitempty"`
	TimeoutSeconds      int32  `json:"timeoutSeconds,omitempty"`
	SuccessThreshold    int32  `json:"successThreshold,omitempty"`
	FailureThreshold    int32  `json:"failureThreshold,omitempty"`
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
	NodeAffinity    string `json:"nodeAffinity,omitempty"`    // 简化描述
	PodAffinity     string `json:"podAffinity,omitempty"`     // 简化描述
	PodAntiAffinity string `json:"podAntiAffinity,omitempty"` // 简化描述
}

// DeploymentStatus 状态
type DeploymentStatus struct {
	ObservedGeneration  int64                 `json:"observedGeneration,omitempty"`
	Replicas            int32                 `json:"replicas"`
	UpdatedReplicas     int32                 `json:"updatedReplicas,omitempty"`
	ReadyReplicas       int32                 `json:"readyReplicas,omitempty"`
	AvailableReplicas   int32                 `json:"availableReplicas,omitempty"`
	UnavailableReplicas int32                 `json:"unavailableReplicas,omitempty"`
	CollisionCount      *int32                `json:"collisionCount,omitempty"`
	Conditions          []DeploymentCondition `json:"conditions,omitempty"`
}

// DeploymentCondition 状态条件
type DeploymentCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastUpdateTime     time.Time `json:"lastUpdateTime,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

// DeploymentRollout 滚动更新状态
type DeploymentRollout struct {
	Phase   string   `json:"phase"`
	Message string   `json:"message,omitempty"`
	Badges  []string `json:"badges,omitempty"`
}

// ReplicaSetBrief ReplicaSet 简要信息
type ReplicaSetBrief struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Revision  string    `json:"revision,omitempty"`
	Replicas  int32     `json:"replicas"`
	Ready     int32     `json:"ready"`
	Available int32     `json:"available"`
	Image     string    `json:"image,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	Age       string    `json:"age"`
}

// ============================================================
// 辅助方法
// ============================================================

// GetName 获取名称
func (d *Deployment) GetName() string {
	return d.Summary.Name
}

// GetNamespace 获取命名空间
func (d *Deployment) GetNamespace() string {
	return d.Summary.Namespace
}

// IsHealthy 判断 Deployment 是否健康
func (d *Deployment) IsHealthy() bool {
	return d.Summary.Ready == d.Summary.Replicas && d.Summary.Replicas > 0
}

// IsUpdating 判断是否正在更新
func (d *Deployment) IsUpdating() bool {
	return d.Summary.Updated < d.Summary.Replicas
}

// IsPaused 判断是否暂停
func (d *Deployment) IsPaused() bool {
	return d.Summary.Paused
}

// ============================================================
// ReplicaSet 模型
// ============================================================

// ReplicaSet K8s ReplicaSet 资源模型
type ReplicaSet struct {
	CommonMeta

	Replicas          int32             `json:"replicas"`
	ReadyReplicas     int32             `json:"ready_replicas"`
	AvailableReplicas int32             `json:"available_replicas"`
	Selector          map[string]string `json:"selector,omitempty"`
}
