// internal/push/starter.go
package push

import (
	"time"

	"AtlHyper/atlhyper_agent/external/control"
	pcfg "AtlHyper/atlhyper_agent/external/push/config"
	Pusher "AtlHyper/atlhyper_agent/external/push/pusher"
	"AtlHyper/atlhyper_agent/external/push/utils"
)

func StartPusher() {
	clusterID := utils.GetClusterID()
	eventsPath := pcfg.PathEventsCleaned
	metricsPath := pcfg.PathMetricsSnapshot
	podListPath := pcfg.PathPodList
	nodeListPath := pcfg.PathNodeList
	serviceListPath := pcfg.PathServiceList
	namespaceListPath := pcfg.PathNamespaceList
	ingressListPath := pcfg.PathIngressList
	deploymentListPath := pcfg.PathDeploymentList
	configMapListPath := pcfg.PathConfigMapList
	opsPath := pcfg.PathOps

	// 支持用环境变量覆盖频率（单位：秒）
	eventsInterval := 5*time.Second
	metricsInterval := 5*time.Second
	podListInterval := 25*time.Second
	nodeListInterval := 30*time.Second
	serviceListInterval := 35*time.Second
	namespaceListInterval := 40*time.Second
	ingressListInterval := 45*time.Second
	deploymentListInterval := 50*time.Second
	configMapListInterval := 55*time.Second

	// 分别启动两个上报循环
	Pusher.StartEventsPusher(clusterID, eventsPath, eventsInterval)
	Pusher.StartMetricsPusher(clusterID, metricsPath, metricsInterval)
	Pusher.StartPodListPusher(clusterID, podListPath, podListInterval)
	Pusher.StartNodeListPusher(clusterID, nodeListPath, nodeListInterval)
	Pusher.StartServiceListPusher(clusterID, serviceListPath, serviceListInterval)
	Pusher.StartNamespaceListPusher(clusterID, namespaceListPath, namespaceListInterval)
	Pusher.StartIngressListPusher(clusterID, ingressListPath, ingressListInterval)
	Pusher.StartDeploymentListPusher(clusterID, deploymentListPath, deploymentListInterval)
	Pusher.StartConfigMapListPusher(clusterID, configMapListPath, configMapListInterval)
	control.StartControlLoop(clusterID, opsPath) 

}