// atlhyper_master_v2/model/pv.go
// PersistentVolume Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// PVItem PersistentVolume 列表项（集群级，无 Namespace）
type PVItem struct {
	Name          string   `json:"name"`
	Capacity      string   `json:"capacity"`
	Phase         string   `json:"phase"`
	StorageClass  string   `json:"storageClass"`
	AccessModes   []string `json:"accessModes"`
	ReclaimPolicy string   `json:"reclaimPolicy"`
	CreatedAt     string   `json:"createdAt"`
	Age           string   `json:"age"`
}
