package model_v2

import "time"

// ============================================================
// StatefulSet 模型（嵌套结构）
// ============================================================

// StatefulSet K8s StatefulSet 资源模型
type StatefulSet struct {
	Summary  StatefulSetSummary  `json:"summary"`
	Spec     StatefulSetSpec     `json:"spec"`
	Template PodTemplate         `json:"template"`
	Status   StatefulSetStatus   `json:"status"`
	Rollout  *WorkloadRollout    `json:"rollout,omitempty"`

	// 元数据
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// StatefulSetSummary StatefulSet 摘要
type StatefulSetSummary struct {
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Replicas    int32     `json:"replicas"`
	Ready       int32     `json:"ready"`
	Current     int32     `json:"current"`
	Updated     int32     `json:"updated"`
	Available   int32     `json:"available"`
	CreatedAt   time.Time `json:"createdAt"`
	Age         string    `json:"age"`
	ServiceName string    `json:"serviceName,omitempty"`
	Selector    string    `json:"selector,omitempty"`
}

// StatefulSetSpec StatefulSet 规格
type StatefulSetSpec struct {
	Replicas             *int32               `json:"replicas,omitempty"`
	ServiceName          string               `json:"serviceName,omitempty"`
	PodManagementPolicy  string               `json:"podManagementPolicy,omitempty"` // OrderedReady, Parallel
	UpdateStrategy       *UpdateStrategy      `json:"updateStrategy,omitempty"`
	RevisionHistoryLimit *int32               `json:"revisionHistoryLimit,omitempty"`
	MinReadySeconds      int32                `json:"minReadySeconds,omitempty"`
	PersistentVolumeClaimRetentionPolicy *PVCRetentionPolicy `json:"persistentVolumeClaimRetentionPolicy,omitempty"`
	Selector             *LabelSelector       `json:"selector,omitempty"`
	VolumeClaimTemplates []VolumeClaimTemplate `json:"volumeClaimTemplates,omitempty"`
}

// UpdateStrategy 更新策略
type UpdateStrategy struct {
	Type          string `json:"type,omitempty"` // RollingUpdate, OnDelete
	Partition     *int32 `json:"partition,omitempty"`
	MaxUnavailable string `json:"maxUnavailable,omitempty"`
	MaxSurge       string `json:"maxSurge,omitempty"`
}

// PVCRetentionPolicy PVC 保留策略
type PVCRetentionPolicy struct {
	WhenDeleted string `json:"whenDeleted,omitempty"` // Retain, Delete
	WhenScaled  string `json:"whenScaled,omitempty"`  // Retain, Delete
}

// VolumeClaimTemplate PVC 模板
type VolumeClaimTemplate struct {
	Name         string            `json:"name"`
	AccessModes  []string          `json:"accessModes,omitempty"`
	StorageClass string            `json:"storageClass,omitempty"`
	Storage      string            `json:"storage,omitempty"`
	Selector     map[string]string `json:"selector,omitempty"`
}

// StatefulSetStatus StatefulSet 状态
type StatefulSetStatus struct {
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
	Replicas           int32  `json:"replicas"`
	ReadyReplicas      int32  `json:"readyReplicas,omitempty"`
	CurrentReplicas    int32  `json:"currentReplicas,omitempty"`
	UpdatedReplicas    int32  `json:"updatedReplicas,omitempty"`
	AvailableReplicas  int32  `json:"availableReplicas,omitempty"`
	CurrentRevision    string `json:"currentRevision,omitempty"`
	UpdateRevision     string `json:"updateRevision,omitempty"`
	CollisionCount     *int32 `json:"collisionCount,omitempty"`
	Conditions         []WorkloadCondition `json:"conditions,omitempty"`
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
	Phase   string   `json:"phase"`             // Progressing, Complete, Degraded, Paused
	Message string   `json:"message,omitempty"`
	Badges  []string `json:"badges,omitempty"`
}

// StatefulSet 辅助方法

func (s *StatefulSet) GetName() string {
	return s.Summary.Name
}

func (s *StatefulSet) GetNamespace() string {
	return s.Summary.Namespace
}

func (s *StatefulSet) IsHealthy() bool {
	return s.Summary.Ready == s.Summary.Replicas && s.Summary.Replicas > 0
}

func (s *StatefulSet) IsUpdating() bool {
	return s.Summary.Updated < s.Summary.Replicas
}

// ============================================================
// DaemonSet 模型（嵌套结构）
// ============================================================

// DaemonSet K8s DaemonSet 资源模型
type DaemonSet struct {
	Summary  DaemonSetSummary  `json:"summary"`
	Spec     DaemonSetSpec     `json:"spec"`
	Template PodTemplate       `json:"template"`
	Status   DaemonSetStatus   `json:"status"`
	Rollout  *WorkloadRollout  `json:"rollout,omitempty"`

	// 元数据
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// DaemonSetSummary DaemonSet 摘要
type DaemonSetSummary struct {
	Name                   string    `json:"name"`
	Namespace              string    `json:"namespace"`
	DesiredNumberScheduled int32     `json:"desiredNumberScheduled"`
	CurrentNumberScheduled int32     `json:"currentNumberScheduled"`
	NumberReady            int32     `json:"numberReady"`
	NumberAvailable        int32     `json:"numberAvailable"`
	NumberUnavailable      int32     `json:"numberUnavailable"`
	NumberMisscheduled     int32     `json:"numberMisscheduled"`
	UpdatedNumberScheduled int32     `json:"updatedNumberScheduled"`
	CreatedAt              time.Time `json:"createdAt"`
	Age                    string    `json:"age"`
	Selector               string    `json:"selector,omitempty"`
}

// DaemonSetSpec DaemonSet 规格
type DaemonSetSpec struct {
	UpdateStrategy       *UpdateStrategy `json:"updateStrategy,omitempty"`
	MinReadySeconds      int32           `json:"minReadySeconds,omitempty"`
	RevisionHistoryLimit *int32          `json:"revisionHistoryLimit,omitempty"`
	Selector             *LabelSelector  `json:"selector,omitempty"`
}

// DaemonSetStatus DaemonSet 状态
type DaemonSetStatus struct {
	ObservedGeneration     int64  `json:"observedGeneration,omitempty"`
	DesiredNumberScheduled int32  `json:"desiredNumberScheduled"`
	CurrentNumberScheduled int32  `json:"currentNumberScheduled"`
	NumberReady            int32  `json:"numberReady"`
	NumberAvailable        int32  `json:"numberAvailable,omitempty"`
	NumberUnavailable      int32  `json:"numberUnavailable,omitempty"`
	NumberMisscheduled     int32  `json:"numberMisscheduled"`
	UpdatedNumberScheduled int32  `json:"updatedNumberScheduled,omitempty"`
	CollisionCount         *int32 `json:"collisionCount,omitempty"`
	Conditions             []WorkloadCondition `json:"conditions,omitempty"`
}

// DaemonSet 辅助方法

func (d *DaemonSet) GetName() string {
	return d.Summary.Name
}

func (d *DaemonSet) GetNamespace() string {
	return d.Summary.Namespace
}

func (d *DaemonSet) IsHealthy() bool {
	return d.Summary.NumberReady == d.Summary.DesiredNumberScheduled && d.Summary.DesiredNumberScheduled > 0
}

func (d *DaemonSet) IsUpdating() bool {
	return d.Summary.UpdatedNumberScheduled < d.Summary.DesiredNumberScheduled
}

func (d *DaemonSet) HasMisscheduled() bool {
	return d.Summary.NumberMisscheduled > 0
}
