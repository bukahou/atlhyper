// ui_interfaces/deployment/dto_detail.go
package deployment

import (
	"time"

	modelpod "AtlHyper/model/pod"
)

// DeploymentDetailDTO —— 详情页（扁平化，但保留核心结构）
type DeploymentDetailDTO struct {
	// Summary
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Strategy    string    `json:"strategy"`
	Replicas    int32     `json:"replicas"`               // 期望
	Updated     int32     `json:"updated"`
	Ready       int32     `json:"ready"`
	Available   int32     `json:"available"`
	Unavailable int32     `json:"unavailable,omitempty"`
	Paused      bool      `json:"paused,omitempty"`
	Selector    string    `json:"selector,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	Age         string    `json:"age,omitempty"`

	// Spec（关键字段）
	Spec SpecDTO `json:"spec"`

	// Template（直接展示给 UI）
	Template TemplateDTO `json:"template"`

	// Status / Conditions
	Status     StatusDTO     `json:"status"`
	Conditions []ConditionDTO`json:"conditions,omitempty"`

	// Rollout（如有）
	Rollout *RolloutDTO `json:"rollout,omitempty"`

	// 相关 RS（简要）
	ReplicaSets []ReplicaSetBriefDTO `json:"replicaSets,omitempty"`

	// 元数据
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ---- 子 DTO ----

type SpecDTO struct {
	Replicas                *int32  `json:"replicas,omitempty"`
	MinReadySeconds         int32   `json:"minReadySeconds,omitempty"`
	RevisionHistoryLimit    *int32  `json:"revisionHistoryLimit,omitempty"`
	ProgressDeadlineSeconds *int32  `json:"progressDeadlineSeconds,omitempty"`
	StrategyType            string  `json:"strategyType,omitempty"`
	MaxUnavailable          string  `json:"maxUnavailable,omitempty"`
	MaxSurge                string  `json:"maxSurge,omitempty"`
	// 选择器（只保留最常用的 MatchLabels，足够 UI 展示）
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

type TemplateDTO struct {
	Labels       map[string]string    `json:"labels,omitempty"`
	Annotations  map[string]string    `json:"annotations,omitempty"`
	Containers   []modelpod.Container `json:"containers"`
	Volumes      []modelpod.Volume    `json:"volumes,omitempty"`
	// 常用约束
	ServiceAccountName string            `json:"serviceAccountName,omitempty"`
	NodeSelector       map[string]string `json:"nodeSelector,omitempty"`
	HostNetwork        bool              `json:"hostNetwork,omitempty"`
	DNSPolicy          string            `json:"dnsPolicy,omitempty"`
	RuntimeClassName   string            `json:"runtimeClassName,omitempty"`
	ImagePullSecrets   []string          `json:"imagePullSecrets,omitempty"`
}

type StatusDTO struct {
	ObservedGeneration  int64  `json:"observedGeneration,omitempty"`
	Replicas            int32  `json:"replicas"`
	UpdatedReplicas     int32  `json:"updatedReplicas,omitempty"`
	ReadyReplicas       int32  `json:"readyReplicas,omitempty"`
	AvailableReplicas   int32  `json:"availableReplicas,omitempty"`
	UnavailableReplicas int32  `json:"unavailableReplicas,omitempty"`
	CollisionCount      *int32 `json:"collisionCount,omitempty"`
}

type ConditionDTO struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastUpdateTime     time.Time `json:"lastUpdateTime,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

type RolloutDTO struct {
	Phase   string   `json:"phase"`
	Message string   `json:"message,omitempty"`
	Badges  []string `json:"badges,omitempty"`
}

type ReplicaSetBriefDTO struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Revision  string    `json:"revision,omitempty"`
	Replicas  int32     `json:"replicas"`
	Ready     int32     `json:"ready"`
	Available int32     `json:"available"`
	CreatedAt time.Time `json:"createdAt"`
	Age       string    `json:"age"`
}
