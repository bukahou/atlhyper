// atlhyper_master_v2/deployer/interfaces.go
// Deployer 模块接口定义
package deployer

import "context"

// Deployer manages the CD lifecycle: polling, rendering, deploying
type Deployer interface {
	// Start begins the polling loop
	Start(ctx context.Context) error
	// Stop gracefully stops the polling loop
	Stop() error

	// SyncNow triggers an immediate sync for the given path
	SyncNow(ctx context.Context, path string) error
	// GetPathStatus returns sync status for all configured paths
	GetPathStatus(ctx context.Context) ([]PathStatus, error)
}
