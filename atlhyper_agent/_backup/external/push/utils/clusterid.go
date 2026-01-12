package utils

import (
	"context"
	"log"
	"strings"
	"sync"

	"AtlHyper/atlhyper_agent/config"
	agentutils "AtlHyper/atlhyper_agent/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	clusterID     string
	clusterIDOnce sync.Once
)

// GetClusterID 返回集群唯一 ID：优先配置 → 其次 kube-system Namespace UID → 最后兜底
// - 缓存一次结果，避免重复打 K8s API
// - 单行结构化日志，便于检索
func GetClusterID() string {
	clusterIDOnce.Do(func() {
		// 1) 配置优先
		if env := strings.TrimSpace(config.GlobalConfig.Cluster.ClusterID); env != "" {
			clusterID = env
			log.Printf("level=info msg=\"cluster_id resolved\" source=config cluster_id=%s", clusterID)
			return
		}

		// 2) kube-system Namespace UID
		client := agentutils.GetCoreClient() // 需确保已初始化
		ns, err := client.CoreV1().Namespaces().Get(context.Background(), "kube-system", metav1.GetOptions{})
		if err == nil && ns.UID != "" {
			clusterID = string(ns.UID)
			log.Printf("level=info msg=\"cluster_id resolved\" source=kube-system-uid cluster_id=%s", clusterID)
			return
		}
		if err != nil {
			log.Printf("level=warn msg=\"fetch kube-system UID failed\" err=%q", err.Error())
		} else {
			log.Printf("level=warn msg=\"kube-system UID empty\"")
		}

		// 3) 兜底：保持明确可见的占位值
		clusterID = "unknown-cluster"
		log.Printf("level=warn msg=\"cluster_id resolved\" source=fallback cluster_id=%s", clusterID)
	})
	return clusterID
}
