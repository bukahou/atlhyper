package model_v2

// ============================================================
// PersistentVolume 模型
// ============================================================

// PersistentVolume K8s PV 资源模型
//
// PersistentVolume 是集群级别的存储资源。
type PersistentVolume struct {
	CommonMeta

	// 容量
	Capacity string `json:"capacity"` // 如 "10Gi"

	// 状态
	Phase string `json:"phase"` // Available, Bound, Released, Failed

	// 配置
	StorageClass  string   `json:"storage_class,omitempty"` // 存储类
	AccessModes   []string `json:"access_modes,omitempty"`  // ReadWriteOnce, ReadOnlyMany, ReadWriteMany
	ReclaimPolicy string   `json:"reclaim_policy,omitempty"` // Retain, Recycle, Delete

	// 卷来源与绑定
	VolumeSourceType string `json:"volume_source_type,omitempty"` // NFS, HostPath, CSI, Local 等
	ClaimRefName     string `json:"claim_ref_name,omitempty"`     // 绑定的 PVC 名
	ClaimRefNS       string `json:"claim_ref_namespace,omitempty"` // 绑定的 PVC namespace
}

// IsBound 判断是否已绑定
func (p *PersistentVolume) IsBound() bool {
	return p.Phase == "Bound"
}

// IsAvailable 判断是否可用
func (p *PersistentVolume) IsAvailable() bool {
	return p.Phase == "Available"
}

// ============================================================
// PersistentVolumeClaim 模型
// ============================================================

// PersistentVolumeClaim K8s PVC 资源模型
//
// PersistentVolumeClaim 是用户对存储的请求。
type PersistentVolumeClaim struct {
	CommonMeta

	// 状态
	Phase      string `json:"phase"`       // Pending, Bound, Lost
	VolumeName string `json:"volume_name"` // 绑定的 PV 名称

	// 配置
	StorageClass      string   `json:"storage_class,omitempty"`      // 存储类
	AccessModes       []string `json:"access_modes,omitempty"`       // 访问模式
	RequestedCapacity string   `json:"requested_capacity,omitempty"` // 请求容量
	ActualCapacity    string   `json:"actual_capacity,omitempty"`    // 实际容量（绑定后）
	VolumeMode        string   `json:"volume_mode,omitempty"`        // Filesystem, Block
}

// IsBound 判断是否已绑定
func (p *PersistentVolumeClaim) IsBound() bool {
	return p.Phase == "Bound"
}

// IsPending 判断是否等待中
func (p *PersistentVolumeClaim) IsPending() bool {
	return p.Phase == "Pending"
}

// IsLost 判断是否丢失
func (p *PersistentVolumeClaim) IsLost() bool {
	return p.Phase == "Lost"
}
