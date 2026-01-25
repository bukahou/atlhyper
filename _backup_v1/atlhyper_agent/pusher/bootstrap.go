// logic/pusher/bootstrap.go
// Pusher 启动管理
package pusher

import (
	"context"
	"log"
	"strings"
	"time"

	"AtlHyper/atlhyper_agent/config"
	"AtlHyper/model/transport"
)

// 默认推送间隔
const (
	DefaultEventsInterval     = 5 * time.Second
	DefaultMetricsInterval    = 5 * time.Second
	DefaultPodInterval        = 25 * time.Second
	DefaultNodeInterval       = 30 * time.Second
	DefaultServiceInterval    = 35 * time.Second
	DefaultNamespaceInterval  = 40 * time.Second
	DefaultIngressInterval    = 45 * time.Second
	DefaultDeploymentInterval = 50 * time.Second
	DefaultConfigMapInterval  = 55 * time.Second
)

var globalRegistry *Registry

// StartAllPushers 启动所有数据推送器
func StartAllPushers(ctx context.Context) {
	// 使用 RestClient.BaseURL（与旧 agent 保持一致）
	masterURL := strings.TrimRight(config.GlobalConfig.RestClient.BaseURL, "/")
	if masterURL == "" {
		log.Println("[pusher] RestClient.BaseURL not configured, pushers disabled")
		return
	}

	clusterID := GetClusterID()
	log.Printf("[pusher] starting pushers, master=%s cluster=%s", masterURL, clusterID)

	globalRegistry = NewRegistry()

	// 注册所有推送器
	// Events pusher (增量推送，高频)
	registerPusher(clusterID, masterURL, "events", PathEventsCleaned, transport.SourceK8sEvent, NewEventsSource(), DefaultEventsInterval)

	// Metrics pusher (高频推送)
	registerPusher(clusterID, masterURL, "metrics", PathMetricsSnapshot, transport.SourceMetricsSnapshot, &MetricsSource{}, DefaultMetricsInterval)

	// 快照型推送器
	registerPusher(clusterID, masterURL, "pod", PathPodList, transport.SourcePodListSnapshot, &PodSource{}, DefaultPodInterval)
	registerPusher(clusterID, masterURL, "node", PathNodeList, transport.SourceNodeListSnapshot, &NodeSource{}, DefaultNodeInterval)
	registerPusher(clusterID, masterURL, "service", PathServiceList, transport.SourceServiceListSnapshot, &ServiceSource{}, DefaultServiceInterval)
	registerPusher(clusterID, masterURL, "namespace", PathNamespaceList, transport.SourceNamespaceListSnapshot, &NamespaceSource{}, DefaultNamespaceInterval)
	registerPusher(clusterID, masterURL, "ingress", PathIngressList, transport.SourceIngressListSnapshot, &IngressSource{}, DefaultIngressInterval)
	registerPusher(clusterID, masterURL, "deployment", PathDeploymentList, transport.SourceDeploymentListSnapshot, &DeploymentSource{}, DefaultDeploymentInterval)
	registerPusher(clusterID, masterURL, "configmap", PathConfigMapList, transport.SourceConfigMapListSnapshot, &ConfigMapSource{}, DefaultConfigMapInterval)

	// 启动所有推送器
	globalRegistry.StartAll(ctx)
	log.Printf("[pusher] all pushers started")
}

// registerPusher 注册单个推送器
func registerPusher(clusterID, baseURL, name, path, source string, ds DataSource, interval time.Duration) {
	cfg := Config{
		Name:      name,
		ClusterID: clusterID,
		Source:    source,
		Path:      path,
		BaseURL:   baseURL,
		Interval:  interval,
	}
	p := NewGenericPusher(cfg, ds)
	globalRegistry.Register(p)
}

// StopAllPushers 停止所有推送器
func StopAllPushers() {
	if globalRegistry != nil {
		globalRegistry.StopAll()
	}
}

// GetRegistry 获取全局注册表
func GetRegistry() *Registry {
	return globalRegistry
}
