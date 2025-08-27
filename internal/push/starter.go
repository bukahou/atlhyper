// internal/push/starter.go
package push

import (
	"time"

	pcfg "NeuroController/internal/push/config"
	pusher "NeuroController/internal/push/pusher"
	podpusher "NeuroController/internal/push/pusher/pod"
	"NeuroController/internal/push/utils"
)

func StartPusher() {
	clusterID := utils.GetClusterID()
	eventsPath := pcfg.PathEventsCleaned
	metricsPath := pcfg.PathMetricsSnapshot
	podListPath := pcfg.PathPodList

	// 支持用环境变量覆盖频率（单位：秒）
	eventsInterval := 5*time.Second
	metricsInterval := 5*time.Second
	podListInterval := 30*time.Second

	// 分别启动两个上报循环
	pusher.StartEventsPusher(clusterID, eventsPath, eventsInterval)
	pusher.StartMetricsPusher(clusterID, metricsPath, metricsInterval)
	podpusher.StartPodListPusher(clusterID, podListPath, podListInterval)
}

