// model/k8s/deployment.go
// Deployment 资源模型
package k8s

import "time"

// ====================== 顶层：Deployment ======================

type Deployment struct {
	Summary     DeploymentSummary   `json:"summary"`
	Spec        DeploymentSpec      `json:"spec"`
	Template    PodTemplate         `json:"template"`
	Status      DeploymentStatus    `json:"status"`
	Rollout     *Rollout            `json:"rollout,omitempty"`
	ReplicaSets []ReplicaSetBrief   `json:"replicaSets,omitempty"`
	Annotations map[string]string   `json:"annotations,omitempty"`
	Labels      map[string]string   `json:"labels,omitempty"`
}

// ====================== Summary ======================

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

// ====================== Spec ======================

type DeploymentSpec struct {
	Replicas                *int32        `json:"replicas,omitempty"`
	Selector                LabelSelector `json:"selector"`
	Strategy                *Strategy     `json:"strategy,omitempty"`
	MinReadySeconds         int32         `json:"minReadySeconds,omitempty"`
	RevisionHistoryLimit    *int32        `json:"revisionHistoryLimit,omitempty"`
	ProgressDeadlineSeconds *int32        `json:"progressDeadlineSeconds,omitempty"`
}

type Strategy struct {
	Type          string                 `json:"type"`
	RollingUpdate *RollingUpdateStrategy `json:"rollingUpdate,omitempty"`
}

type RollingUpdateStrategy struct {
	MaxUnavailable string `json:"maxUnavailable,omitempty"`
	MaxSurge       string `json:"maxSurge,omitempty"`
}

type LabelSelector struct {
	MatchLabels      map[string]string `json:"matchLabels,omitempty"`
	MatchExpressions []LabelExpr       `json:"matchExpressions,omitempty"`
}

type LabelExpr struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// ====================== Pod Template ======================

type PodTemplate struct {
	Labels             map[string]string `json:"labels,omitempty"`
	Annotations        map[string]string `json:"annotations,omitempty"`
	Containers         []Container       `json:"containers"`
	Volumes            []Volume          `json:"volumes,omitempty"`
	ServiceAccountName string            `json:"serviceAccountName,omitempty"`
	NodeSelector       map[string]string `json:"nodeSelector,omitempty"`
	Tolerations        any               `json:"tolerations,omitempty"`
	Affinity           any               `json:"affinity,omitempty"`
	RuntimeClassName   string            `json:"runtimeClassName,omitempty"`
	ImagePullSecrets   []string          `json:"imagePullSecrets,omitempty"`
	HostNetwork        bool              `json:"hostNetwork,omitempty"`
	DNSPolicy          string            `json:"dnsPolicy,omitempty"`
}

// ====================== Status / Conditions ======================

type DeploymentStatus struct {
	ObservedGeneration  int64       `json:"observedGeneration,omitempty"`
	Replicas            int32       `json:"replicas"`
	UpdatedReplicas     int32       `json:"updatedReplicas,omitempty"`
	ReadyReplicas       int32       `json:"readyReplicas,omitempty"`
	AvailableReplicas   int32       `json:"availableReplicas,omitempty"`
	UnavailableReplicas int32       `json:"unavailableReplicas,omitempty"`
	CollisionCount      *int32      `json:"collisionCount,omitempty"`
	Conditions          []Condition `json:"conditions,omitempty"`
}

type Condition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastUpdateTime     time.Time `json:"lastUpdateTime,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

type Rollout struct {
	Phase   string   `json:"phase"`
	Message string   `json:"message,omitempty"`
	Badges  []string `json:"badges,omitempty"`
}

// ====================== 相关副本集 ======================

type ReplicaSetBrief struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Revision  string    `json:"revision,omitempty"`
	Replicas  int32     `json:"replicas"`
	Ready     int32     `json:"ready"`
	Available int32     `json:"available"`
	CreatedAt time.Time `json:"createdAt"`
	Age       string    `json:"age"`
}
