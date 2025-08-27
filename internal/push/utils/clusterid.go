package utils

import (
	"NeuroController/internal/utils"
	"context"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetClusterID 获取集群唯一 ID
// 优先从 kube-system Namespace 的 UID 获取
func GetClusterID() string {
	// 确保客户端已初始化
	client := utils.GetCoreClient()

	ns, err := client.CoreV1().Namespaces().Get(context.Background(), "kube-system", metav1.GetOptions{})
	if err != nil {
		log.Printf("❌ 获取 kube-system namespace 失败: %v", err)
		return "unknown-cluster"
	}

	if ns.UID == "" {
		log.Printf("⚠️ kube-system namespace UID 为空")
		return "unknown-cluster"
	}

	log.Printf("✅ 成功获取 ClusterID: %s", ns.UID)
	return string(ns.UID)
}
