// logic/pusher/sources.go
// DataSource 适配器 - 连接 snapshot 模块和 pusher
package pusher

import (
	"context"

	"AtlHyper/atlhyper_agent/source/event/datahub"
	"AtlHyper/atlhyper_agent/source/metrics"
	"AtlHyper/atlhyper_agent/source/snapshot/configmap"
	"AtlHyper/atlhyper_agent/source/snapshot/deployment"
	"AtlHyper/atlhyper_agent/source/snapshot/ingress"
	"AtlHyper/atlhyper_agent/source/snapshot/namespace"
	"AtlHyper/atlhyper_agent/source/snapshot/node"
	"AtlHyper/atlhyper_agent/source/snapshot/pod"
	"AtlHyper/atlhyper_agent/source/snapshot/service"
	"AtlHyper/model/collect"
	"AtlHyper/model/transport"
)

// 类型别名
type LogEvent = transport.LogEvent

// GetCleanedEvents 获取清洗后的事件
var GetCleanedEvents = datahub.GetCleanedEvents

// PodSource Pod 数据源
type PodSource struct{}

func (s *PodSource) Name() string { return "pod" }
func (s *PodSource) Fetch(ctx context.Context) (any, error) {
	pods, err := pod.ListPods(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"pods": pods}, nil // 包装为 {"pods": [...]}
}

// NodeSource Node 数据源
type NodeSource struct{}

func (s *NodeSource) Name() string { return "node" }
func (s *NodeSource) Fetch(ctx context.Context) (any, error) {
	nodes, err := node.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"nodes": nodes}, nil // 包装为 {"nodes": [...]}
}

// ServiceSource Service 数据源
type ServiceSource struct{}

func (s *ServiceSource) Name() string { return "service" }
func (s *ServiceSource) Fetch(ctx context.Context) (any, error) {
	services, err := service.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"services": services}, nil // 包装为 {"services": [...]}
}

// NamespaceSource Namespace 数据源
type NamespaceSource struct{}

func (s *NamespaceSource) Name() string { return "namespace" }
func (s *NamespaceSource) Fetch(ctx context.Context) (any, error) {
	namespaces, err := namespace.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"namespaces": namespaces}, nil // 包装为 {"namespaces": [...]}
}

// DeploymentSource Deployment 数据源
type DeploymentSource struct{}

func (s *DeploymentSource) Name() string { return "deployment" }
func (s *DeploymentSource) Fetch(ctx context.Context) (any, error) {
	deployments, err := deployment.ListDeployments(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"deployments": deployments}, nil // 包装为 {"deployments": [...]}
}

// IngressSource Ingress 数据源
type IngressSource struct{}

func (s *IngressSource) Name() string { return "ingress" }
func (s *IngressSource) Fetch(ctx context.Context) (any, error) {
	ingresses, err := ingress.ListIngresses(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ingresses": ingresses}, nil // 包装为 {"ingresses": [...]}
}

// ConfigMapSource ConfigMap 数据源
type ConfigMapSource struct{}

func (s *ConfigMapSource) Name() string { return "configmap" }
func (s *ConfigMapSource) Fetch(ctx context.Context) (any, error) {
	configmaps, err := configmap.ListConfigMaps(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"configmaps": configmaps}, nil // 包装为 {"configmaps": [...]}
}

// MetricsSource Metrics 数据源
type MetricsSource struct{}

func (s *MetricsSource) Name() string { return "metrics" }
func (s *MetricsSource) Fetch(ctx context.Context) (any, error) {
	snapshots := metrics.GetAllMetricsSnapshots()
	if len(snapshots) == 0 {
		return nil, nil
	}
	// 转换 map 为 slice，保持与旧 agent 一致的格式: {"snapshots": [...]}
	list := make([]collect.NodeMetricsSnapshot, 0, len(snapshots))
	for _, snap := range snapshots {
		list = append(list, snap)
	}
	return map[string]any{"snapshots": list}, nil
}

// EventsSource Events 数据源（增量推送）
type EventsSource struct {
	lastSentMap map[string]int64 // key -> timestamp_ms
}

func NewEventsSource() *EventsSource {
	return &EventsSource{
		lastSentMap: make(map[string]int64),
	}
}

func (s *EventsSource) Name() string { return "events" }

func (s *EventsSource) Fetch(ctx context.Context) (any, error) {
	events := GetCleanedEvents()
	if len(events) == 0 {
		return nil, nil
	}

	// 增量过滤：只发送新事件
	newEvents := make([]LogEvent, 0, len(events))
	currentKeys := make(map[string]struct{}, len(events))

	for _, ev := range events {
		key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode + "|" + ev.Message
		currentKeys[key] = struct{}{}
		tsMs := ev.Timestamp.UnixMilli()
		if lastTs, ok := s.lastSentMap[key]; !ok || tsMs > lastTs {
			newEvents = append(newEvents, ev)
			s.lastSentMap[key] = tsMs
		}
	}

	// 清理不再出现的 key
	for k := range s.lastSentMap {
		if _, still := currentKeys[k]; !still {
			delete(s.lastSentMap, k)
		}
	}

	if len(newEvents) == 0 {
		return nil, nil
	}

	// 返回包裹格式 {"events": [...]}
	return map[string]any{"events": newEvents}, nil
}
