// atlhyper_master_v2/deployer/types.go
// Deployer 模块类型定义
package deployer

import "time"

// PathStatus represents the sync status of a kustomize path
type PathStatus struct {
	Path          string    `json:"path"`
	Namespace     string    `json:"namespace"`
	InSync        bool      `json:"inSync"`
	ResourceCount int       `json:"resourceCount"`
	LastSyncAt    time.Time `json:"lastSyncAt"`
}
