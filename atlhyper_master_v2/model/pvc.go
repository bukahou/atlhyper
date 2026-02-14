// atlhyper_master_v2/model/pvc.go
// PersistentVolumeClaim Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// PVCItem PersistentVolumeClaim 列表项
type PVCItem struct {
	Name              string   `json:"name"`
	Namespace         string   `json:"namespace"`
	Phase             string   `json:"phase"`
	VolumeName        string   `json:"volumeName"`
	StorageClass      string   `json:"storageClass"`
	AccessModes       []string `json:"accessModes"`
	RequestedCapacity string   `json:"requestedCapacity"`
	ActualCapacity    string   `json:"actualCapacity"`
	CreatedAt         string   `json:"createdAt"`
	Age               string   `json:"age"`
}
