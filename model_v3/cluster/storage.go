package cluster

import model_v3 "AtlHyper/model_v3"

// PersistentVolume K8s PV 资源模型
type PersistentVolume struct {
	model_v3.CommonMeta
	Capacity         string   `json:"capacity"`
	Phase            string   `json:"phase"`
	StorageClass     string   `json:"storageClass,omitempty"`
	AccessModes      []string `json:"accessModes,omitempty"`
	ReclaimPolicy    string   `json:"reclaimPolicy,omitempty"`
	VolumeSourceType string   `json:"volumeSourceType,omitempty"`
	ClaimRefName     string   `json:"claimRefName,omitempty"`
	ClaimRefNS       string   `json:"claimRefNamespace,omitempty"`
}

func (p *PersistentVolume) IsBound() bool     { return p.Phase == "Bound" }
func (p *PersistentVolume) IsAvailable() bool { return p.Phase == "Available" }

// PersistentVolumeClaim K8s PVC 资源模型
type PersistentVolumeClaim struct {
	model_v3.CommonMeta
	Phase             string   `json:"phase"`
	VolumeName        string   `json:"volumeName"`
	StorageClass      string   `json:"storageClass,omitempty"`
	AccessModes       []string `json:"accessModes,omitempty"`
	RequestedCapacity string   `json:"requestedCapacity,omitempty"`
	ActualCapacity    string   `json:"actualCapacity,omitempty"`
	VolumeMode        string   `json:"volumeMode,omitempty"`
}

func (p *PersistentVolumeClaim) IsBound() bool   { return p.Phase == "Bound" }
func (p *PersistentVolumeClaim) IsPending() bool { return p.Phase == "Pending" }
func (p *PersistentVolumeClaim) IsLost() bool    { return p.Phase == "Lost" }
