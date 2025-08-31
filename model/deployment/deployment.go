// model/deployment/deployment.go
package deployment

import (
	"time"

	modelpod "AtlHyper/model/pod"
)

// ====================== 顶层：Deployment（直接对外发送/返回的结构） ======================

type Deployment struct {
	Summary      DeploymentSummary   `json:"summary"`                 // 概要（列表常用字段）
	Spec         DeploymentSpec      `json:"spec"`                    // 关键编排策略
	Template     PodTemplate         `json:"template"`                // Pod 模板（核心：容器/卷/调度约束）
	Status       DeploymentStatus    `json:"status"`                  // 状态详情、条件
	Rollout      *Rollout            `json:"rollout,omitempty"`       // 友好化的发布状态（可选，便于 UI）
	ReplicaSets  []ReplicaSetBrief   `json:"replicaSets,omitempty"`   // 相关 RS 简要（可选，用于溯源/回滚）
	Annotations  map[string]string   `json:"annotations,omitempty"`   // 选择性透传注解（可按需裁剪）
	Labels       map[string]string   `json:"labels,omitempty"`        // 顶层标签
}

// ====================== Summary ======================

type DeploymentSummary struct {
	Name        string    `json:"name"`                    // 部署名
	Namespace   string    `json:"namespace"`               // 命名空间
	Strategy    string    `json:"strategy"`                // Recreate/RollingUpdate
	Replicas    int32     `json:"replicas"`                // 期望副本（spec.replicas）
	Updated     int32     `json:"updated"`                 // 已更新副本（status.updatedReplicas）
	Ready       int32     `json:"ready"`                   // 就绪副本（status.readyReplicas）
	Available   int32     `json:"available"`               // 可用副本（status.availableReplicas）
	Unavailable int32     `json:"unavailable,omitempty"`  // 不可用副本（status.unavailableReplicas）
	Paused      bool      `json:"paused,omitempty"`        // 是否暂停发布（spec.paused）
	CreatedAt   time.Time `json:"createdAt"`               // 创建时间
	Age         string    `json:"age"`                     // 运行时长（派生显示）
	Selector    string    `json:"selector,omitempty"`      // 选择器的字符串化（便于列表展示）
}

// ====================== Spec ======================

type DeploymentSpec struct {
	Replicas                   *int32                 `json:"replicas,omitempty"`                   // 期望副本（nil 表示默认 1）
	Selector                   LabelSelector          `json:"selector"`                             // 匹配 Pod 的选择器
	Strategy                   *Strategy              `json:"strategy,omitempty"`                   // 发布策略
	MinReadySeconds            int32                  `json:"minReadySeconds,omitempty"`            // 最小就绪秒
	RevisionHistoryLimit       *int32                 `json:"revisionHistoryLimit,omitempty"`       // 保留 RS 历史条数
	ProgressDeadlineSeconds    *int32                 `json:"progressDeadlineSeconds,omitempty"`    // 进度超时（秒）
}

// 发布策略
type Strategy struct {
	Type          string                 `json:"type"`                    // "Recreate" / "RollingUpdate"
	RollingUpdate *RollingUpdateStrategy `json:"rollingUpdate,omitempty"` // 滚动更新参数（仅在 Type=RollingUpdate 时有效）
}

// 滚动更新的两个关键阈值（MaxUnavailable/MaxSurge 支持百分比或绝对值，这里统一为字符串）
type RollingUpdateStrategy struct {
	MaxUnavailable string `json:"maxUnavailable,omitempty"` // 如 "25%" 或 "1"
	MaxSurge       string `json:"maxSurge,omitempty"`       // 如 "25%" 或 "1"
}

// 选择器（MatchLabels + MatchExpressions）
type LabelSelector struct {
	MatchLabels      map[string]string   `json:"matchLabels,omitempty"`
	MatchExpressions []LabelExpr         `json:"matchExpressions,omitempty"`
}

type LabelExpr struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"` // In/NotIn/Exists/DoesNotExist
	Values   []string `json:"values,omitempty"`
}

// ====================== Pod Template ======================

type PodTemplate struct {
	Labels              map[string]string        `json:"labels,omitempty"`              // 模板标签
	Annotations         map[string]string        `json:"annotations,omitempty"`         // 模板注解（可裁剪）
	Containers          []modelpod.Container     `json:"containers"`                    // 业务容器（沿用 Pod 模型）
	// InitContainers    []modelpod.Container   `json:"initContainers,omitempty"`      // 如需展示可解开
	// EphemeralContainers []modelpod.Container `json:"ephemeralContainers,omitempty"` // 如需展示可解开
	Volumes             []modelpod.Volume        `json:"volumes,omitempty"`             // 卷（沿用 Pod 模型）

	// 调度/运行约束（与 corev1.PodSpec 对齐，尽量轻量化）
	ServiceAccountName  string                   `json:"serviceAccountName,omitempty"`
	NodeSelector        map[string]string        `json:"nodeSelector,omitempty"`
	Tolerations         any                      `json:"tolerations,omitempty"`          // 透传 corev1.Toleration[]
	Affinity            any                      `json:"affinity,omitempty"`             // 透传 *corev1.Affinity
	RuntimeClassName    string                   `json:"runtimeClassName,omitempty"`
	ImagePullSecrets    []string                 `json:"imagePullSecrets,omitempty"`
	HostNetwork         bool                     `json:"hostNetwork,omitempty"`
	DNSPolicy           string                   `json:"dnsPolicy,omitempty"`
	// 其他常见 PodSpec 字段可按需补充（Priority, TopologySpreadConstraints, SecurityContext 等）
}

// ====================== Status / Conditions ======================

type DeploymentStatus struct {
	ObservedGeneration  int64             `json:"observedGeneration,omitempty"` // 控制器已观测到的 generation
	Replicas            int32             `json:"replicas"`                     // 当前 RS 总副本
	UpdatedReplicas     int32             `json:"updatedReplicas,omitempty"`
	ReadyReplicas       int32             `json:"readyReplicas,omitempty"`
	AvailableReplicas   int32             `json:"availableReplicas,omitempty"`
	UnavailableReplicas int32             `json:"unavailableReplicas,omitempty"`
	CollisionCount      *int32            `json:"collisionCount,omitempty"`
	Conditions          []Condition       `json:"conditions,omitempty"`         // Progressing/Available 等
}

type Condition struct {
	Type               string    `json:"type"`               // Available / Progressing 等
	Status             string    `json:"status"`             // True/False/Unknown
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastUpdateTime     time.Time `json:"lastUpdateTime,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

// Rollout：对 Status 的友好归纳（便于 UI/告警）
type Rollout struct {
	Phase   string   `json:"phase"`              // Progressing/Complete/Paused/Degraded 等
	Message string   `json:"message,omitempty"`  // 关键说明
	Badges  []string `json:"badges,omitempty"`   // UI 徽标（如 “Paused”“Timeout”“NoProgress”等）
}

// ====================== 相关副本集（可选） ======================

type ReplicaSetBrief struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Revision  string    `json:"revision,omitempty"`   // 来自 annotation: deployment.kubernetes.io/revision
	Replicas  int32     `json:"replicas"`
	Ready     int32     `json:"ready"`
	Available int32     `json:"available"`
	CreatedAt time.Time `json:"createdAt"`
	Age       string    `json:"age"`
}
