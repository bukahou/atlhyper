// atlhyper_master/dto/ui/deployment.go
// Deployment UI DTOs
package ui

import (
	"time"

	modelpod "AtlHyper/model/k8s"
)

// ====================== Overview ======================

// DeploymentOverviewDTO - 概览页
type DeploymentOverviewDTO struct {
	Cards DeploymentOverviewCards  `json:"cards"`
	Rows  []DeploymentRowSimple    `json:"rows"`
}

type DeploymentOverviewCards struct {
	TotalDeployments int `json:"totalDeployments"`
	Namespaces       int `json:"namespaces"`
	TotalReplicas    int `json:"totalReplicas"`
	ReadyReplicas    int `json:"readyReplicas"`
}

type DeploymentRowSimple struct {
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	Image       string    `json:"image"`
	Replicas    string    `json:"replicas"`
	LabelCount  int       `json:"labelCount"`
	AnnoCount   int       `json:"annoCount"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ====================== Detail ======================

// DeploymentDetailDTO - 详情页
type DeploymentDetailDTO struct {
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Strategy    string    `json:"strategy"`
	Replicas    int32     `json:"replicas"`
	Updated     int32     `json:"updated"`
	Ready       int32     `json:"ready"`
	Available   int32     `json:"available"`
	Unavailable int32     `json:"unavailable,omitempty"`
	Paused      bool      `json:"paused,omitempty"`
	Selector    string    `json:"selector,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	Age         string    `json:"age,omitempty"`

	Spec       DeploymentSpecDTO       `json:"spec"`
	Template   DeploymentTemplateDTO   `json:"template"`
	Status     DeploymentStatusDTO     `json:"status"`
	Conditions []DeploymentCondDTO     `json:"conditions,omitempty"`
	Rollout    *DeploymentRolloutDTO   `json:"rollout,omitempty"`
	ReplicaSets []ReplicaSetBriefDTO   `json:"replicaSets,omitempty"`

	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type DeploymentSpecDTO struct {
	Replicas                *int32            `json:"replicas,omitempty"`
	MinReadySeconds         int32             `json:"minReadySeconds,omitempty"`
	RevisionHistoryLimit    *int32            `json:"revisionHistoryLimit,omitempty"`
	ProgressDeadlineSeconds *int32            `json:"progressDeadlineSeconds,omitempty"`
	StrategyType            string            `json:"strategyType,omitempty"`
	MaxUnavailable          string            `json:"maxUnavailable,omitempty"`
	MaxSurge                string            `json:"maxSurge,omitempty"`
	MatchLabels             map[string]string `json:"matchLabels,omitempty"`
}

type DeploymentTemplateDTO struct {
	Labels             map[string]string    `json:"labels,omitempty"`
	Annotations        map[string]string    `json:"annotations,omitempty"`
	Containers         []modelpod.Container `json:"containers"`
	Volumes            []modelpod.Volume    `json:"volumes,omitempty"`
	ServiceAccountName string               `json:"serviceAccountName,omitempty"`
	NodeSelector       map[string]string    `json:"nodeSelector,omitempty"`
	HostNetwork        bool                 `json:"hostNetwork,omitempty"`
	DNSPolicy          string               `json:"dnsPolicy,omitempty"`
	RuntimeClassName   string               `json:"runtimeClassName,omitempty"`
	ImagePullSecrets   []string             `json:"imagePullSecrets,omitempty"`
}

type DeploymentStatusDTO struct {
	ObservedGeneration  int64  `json:"observedGeneration,omitempty"`
	Replicas            int32  `json:"replicas"`
	UpdatedReplicas     int32  `json:"updatedReplicas,omitempty"`
	ReadyReplicas       int32  `json:"readyReplicas,omitempty"`
	AvailableReplicas   int32  `json:"availableReplicas,omitempty"`
	UnavailableReplicas int32  `json:"unavailableReplicas,omitempty"`
	CollisionCount      *int32 `json:"collisionCount,omitempty"`
}

type DeploymentCondDTO struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastUpdateTime     time.Time `json:"lastUpdateTime,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

type DeploymentRolloutDTO struct {
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
