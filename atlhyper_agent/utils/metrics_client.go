package utils

import (
	"log"
	"sync"

	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	metricsOnce    sync.Once
	metricsClient  *metricsclient.Clientset
	metricsInitErr error
)

// ✅ InitMetricsClient 初始化 metrics.k8s.io 客户端（线程安全）
// - 使用内部共享的 REST 配置
// - 若未安装 metrics-server，不会阻止主流程
func InitMetricsClient() {
	metricsOnce.Do(func() {
		cfg := GetRestConfig()
		client, err := metricsclient.NewForConfig(cfg)
		if err != nil {
			log.Printf("⚠️ [InitMetricsClient] 初始化失败，可能未部署 metrics-server: %v", err)
			metricsInitErr = err
			return
		}
		metricsClient = client
		log.Println("✅ [InitMetricsClient] 成功初始化 metrics.k8s.io 客户端")
	})
}

// ✅ GetMetricsClient 获取已初始化的 metrics client 实例（可能为 nil）
func GetMetricsClient() *metricsclient.Clientset {
	return metricsClient
}

// ✅ HasMetricsServer 检测当前环境是否成功连接 metrics-server
func HasMetricsServer() bool {
	return metricsClient != nil && metricsInitErr == nil
}
