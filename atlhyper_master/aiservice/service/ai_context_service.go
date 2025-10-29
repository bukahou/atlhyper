// =============================================================
// 📦 文件路径: atlhyper_master/aiservice/service/ai_context_service.go
// =============================================================
// 🧠 模块说明:
//   AI Context Service —— 负责根据 AIService 请求清单，调用底层
//   各类资源接口（Pod / Deployment / Service / Node / ConfigMap / ...），
//   以结构化形式返回数据，供 Builder 汇总成 AIFetchResponse。
// =============================================================

package service

import (
	"context"
	"log"

	model "AtlHyper/model/ai"

	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/configmap"
	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/deployment"
	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/ingress"
	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/metrics"
	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/namespace"
	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/node"
	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/pod"
	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/service"
)

// =============================================================
// 🟢 Pod 相关
// -------------------------------------------------------------
// - 支持多命名空间、多实例
// - 返回扁平化的 PodDetailDTO
// =============================================================
func FetchPods(ctx context.Context, clusterID string, refs []model.ResourceRef) []any {
	var pods []any
	for _, ref := range refs {
		detail, err := pod.GetPodDetail(ctx, clusterID, ref.Namespace, ref.Name)
		if err != nil {
			log.Printf("[AIContext] Skip Pod %s/%s (err=%v)", ref.Namespace, ref.Name, err)
			continue
		}
		pods = append(pods, detail)
	}
	return pods
}

// =============================================================
// 🟣 Deployment 相关
// -------------------------------------------------------------
// - 使用 BuildDeploymentDetail 获取完整结构（含策略、容器、状态）
// =============================================================
func FetchDeployments(ctx context.Context, clusterID string, refs []model.ResourceRef) []any {
	var deploymentsList []any
	for _, ref := range refs {
		detail, err := deployment.BuildDeploymentDetail(ctx, clusterID, ref.Namespace, ref.Name)
		if err != nil {
			log.Printf("[AIContext] Skip Deployment %s/%s (err=%v)", ref.Namespace, ref.Name, err)
			continue
		}
		deploymentsList = append(deploymentsList, detail)
	}
	return deploymentsList
}

// =============================================================
// 🔵 Service 相关
// -------------------------------------------------------------
// - 返回 ServiceDetailDTO（含 selector / ports / clusterIP 等）
// =============================================================
func FetchServices(ctx context.Context, clusterID string, refs []model.ResourceRef) []any {
	var servicesList []any
	for _, ref := range refs {
		detail, err := service.GetServiceDetail(ctx, clusterID, ref.Namespace, ref.Name)
		if err != nil {
			log.Printf("[AIContext] Skip Service %s/%s (err=%v)", ref.Namespace, ref.Name, err)
			continue
		}
		servicesList = append(servicesList, detail)
	}
	return servicesList
}

// =============================================================
// 🟠 Node 相关
// -------------------------------------------------------------
// - 返回 NodeDetailDTO（含 capacity / allocatable / images / conditions）
// =============================================================
func FetchNodes(ctx context.Context, clusterID string, nodeNames []string) []any {
	var nodes []any
	for _, name := range nodeNames {
		detail, err := node.GetNodeDetail(ctx, clusterID, name)
		if err != nil {
			log.Printf("[AIContext] Skip Node %s (err=%v)", name, err)
			continue
		}
		nodes = append(nodes, detail)
	}
	return nodes
}

// =============================================================
// ⚫ ConfigMap 相关
// -------------------------------------------------------------
// - 使用 BuildConfigMapListFullByNamespace
// - 一次拉取命名空间下所有 ConfigMap
// =============================================================
func FetchConfigMaps(ctx context.Context, clusterID string, refs []model.ResourceRef) []any {
	var cfgList []any
	for _, ref := range refs {
		// ⚠️ 此接口以 Namespace 为粒度，不需要 name
		list, err := configmap.BuildConfigMapListFullByNamespace(ctx, clusterID, ref.Namespace)
		if err != nil {
			log.Printf("[AIContext] Skip ConfigMap ns=%s (err=%v)", ref.Namespace, err)
			continue
		}
		cfgList = append(cfgList, list)
	}
	return cfgList
}

// =============================================================
// 🟤 Namespace 相关
// -------------------------------------------------------------
// - 获取 Namespace 详细信息（annotations / labels / resource quota 等）
// =============================================================
func FetchNamespaces(ctx context.Context, clusterID string, refs []model.ResourceRef) []any {
	var nsList []any
	for _, ref := range refs {
		detail, err := namespace.BuildNamespaceDetail(ctx, clusterID, ref.Name)
		if err != nil {
			log.Printf("[AIContext] Skip Namespace %s (err=%v)", ref.Name, err)
			continue
		}
		nsList = append(nsList, detail)
	}
	return nsList
}

// =============================================================
// 🟡 Ingress 相关
// -------------------------------------------------------------
// - 获取完整 Ingress 详情（含 TLS / hosts / paths / service target）
// =============================================================
func FetchIngresses(ctx context.Context, clusterID string, refs []model.ResourceRef) []any {
	var ingList []any
	for _, ref := range refs {
		detail, err := ingress.BuildIngressDetail(ctx, clusterID, ref.Namespace, ref.Name)
		if err != nil {
			log.Printf("[AIContext] Skip Ingress %s/%s (err=%v)", ref.Namespace, ref.Name, err)
			continue
		}
		ingList = append(ingList, detail)
	}
	return ingList
}

// =============================================================
// 🔴 Metrics 相关
// -------------------------------------------------------------
// - 用于 Node 级指标分析（CPU/Memory/Temperature 等）
// =============================================================
func FetchNodeMetrics(ctx context.Context, clusterID string, nodeNames []string) []any {
	var metricsList []any
	for _, nodeID := range nodeNames {
		detail, err := metrics.BuildNodeMetricsDetail(ctx, clusterID, nodeID)
		if err != nil {
			log.Printf("[AIContext] Skip NodeMetrics %s (err=%v)", nodeID, err)
			continue
		}
		metricsList = append(metricsList, detail)
	}
	return metricsList
}
