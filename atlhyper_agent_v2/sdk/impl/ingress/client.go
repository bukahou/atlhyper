// Package ingress Ingress 路由采集客户端实现
//
// client.go - IngressClient 接口实现入口
//
// 本文件实现 sdk.IngressClient 接口。
// 指标采集已迁移到 OTel Collector (sdk/impl/otel/)，
// 本包仅保留路由配置采集 (route_collector.go)。
//
// 架构位置:
//
//	SLORepository
//	    ↓ 调用
//	IngressClient (本包)
//	    ↓ 使用
//	K8s Dynamic API → IngressRoute CRD / 标准 Ingress
package ingress

import (
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
)

var log = logger.Module("IngressClient")

// client IngressClient 实现
type client struct {
	k8sClient sdk.K8sClient
}

// NewIngressClient 创建 IngressClient
//
// 仅用于路由配置采集 (CollectRoutes)。
// 指标采集已迁移到 OTelClient。
func NewIngressClient(k8sClient sdk.K8sClient) sdk.IngressClient {
	return &client{
		k8sClient: k8sClient,
	}
}
