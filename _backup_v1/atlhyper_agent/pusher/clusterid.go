// logic/pusher/clusterid.go
// 集群ID获取
package pusher

import (
	"context"
	"log"
	"strings"
	"sync"

	"AtlHyper/atlhyper_agent/config"
	"AtlHyper/atlhyper_agent/sdk"
)

var (
	clusterID     string
	clusterIDOnce sync.Once
)

// GetClusterID 返回集群唯一 ID：优先配置 → 其次 SDK 获取 → 最后兜底
func GetClusterID() string {
	clusterIDOnce.Do(func() {
		// 1) 配置优先
		if env := strings.TrimSpace(config.GlobalConfig.Cluster.ClusterID); env != "" {
			clusterID = env
			log.Printf("[pusher] cluster_id resolved from config: %s", clusterID)
			return
		}

		// 2) 通过 SDK 获取（kube-system Namespace UID）
		id, err := sdk.Get().Cluster().GetClusterID(context.Background())
		if err != nil {
			log.Printf("[pusher] fetch cluster_id via SDK failed: %v", err)
		}
		if id != "" && id != "unknown-cluster" {
			clusterID = id
			log.Printf("[pusher] cluster_id resolved from SDK: %s", clusterID)
			return
		}

		// 3) 兜底
		clusterID = "unknown-cluster"
		log.Printf("[pusher] cluster_id fallback: %s", clusterID)
	})
	return clusterID
}
