// =============================================================
// 📦 文件路径: atlhyper_master/aiservice/builder/context_assembler.go
// =============================================================
// 🧠 模块说明:
//   本文件负责 AI 上下文数据的组装逻辑。
//   它不再直接依赖各类资源接口，而是调用 service 层的封装函数，
//   统一构建结构化上下文数据以供 AIService 分析使用。
// =============================================================

package builder

import (
	"context"

	"AtlHyper/atlhyper_master/aiservice/model"
	"AtlHyper/atlhyper_master/aiservice/service"
)

// =============================================================
// 🧩 BuildAIContext
// -------------------------------------------------------------
// 按 AIService 请求清单 (AIFetchRequest) 构建完整的上下文数据，
// 调用 service 层获取资源详情，并汇总成 AIFetchResponse。
// - 支持按需选择资源类型（Pods, Deployments, Services, Nodes, Metrics 等）
// - 对单个失败项具备容错能力
// =============================================================
func BuildAIContext(ctx context.Context, req model.AIFetchRequest) (*model.AIFetchResponse, error) {
	out := &model.AIFetchResponse{
		ClusterID: req.ClusterID,
	}

	// =========================================================
	// 🟢 Pods
	// ---------------------------------------------------------
	if len(req.Pods) > 0 {
		out.Pods = service.FetchPods(ctx, req.ClusterID, req.Pods)
	}

	// =========================================================
	// 🟣 Deployments
	// ---------------------------------------------------------
	if len(req.Deployments) > 0 {
		out.Deployments = service.FetchDeployments(ctx, req.ClusterID, req.Deployments)
	}

	// =========================================================
	// 🔵 Services
	// ---------------------------------------------------------
	if len(req.Services) > 0 {
		out.Services = service.FetchServices(ctx, req.ClusterID, req.Services)
	}

	// =========================================================
	// 🟠 Nodes（含 Metrics）
	// ---------------------------------------------------------
	if len(req.Nodes) > 0 {
		out.Nodes = service.FetchNodes(ctx, req.ClusterID, req.Nodes)
		out.Metrics = service.FetchNodeMetrics(ctx, req.ClusterID, req.Nodes)
	}

	// =========================================================
	// ⚫ ConfigMaps（按命名空间拉取）
	// ---------------------------------------------------------
	if len(req.ConfigMaps) > 0 {
		out.ConfigMaps = service.FetchConfigMaps(ctx, req.ClusterID, req.ConfigMaps)
	}

	// =========================================================
	// 🟤 Namespaces
	// ---------------------------------------------------------
	if len(req.Namespaces) > 0 {
		out.Namespaces = service.FetchNamespaces(ctx, req.ClusterID, req.Namespaces)
	}

	// =========================================================
	// 🟡 Ingresses
	// ---------------------------------------------------------
	if len(req.Ingresses) > 0 {
		out.Ingresses = service.FetchIngresses(ctx, req.ClusterID, req.Ingresses)
	}

	// =========================================================
	// ⚪ EndpointSlices（待实现）
	// ---------------------------------------------------------
	if len(req.EndpointSlices) > 0 {
		// TODO: 若未来实现 EndpointSlice 模块，这里可追加 service.FetchEndpointSlices()
	}

	return out, nil
}
