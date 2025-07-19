// =======================================================================================
// 📄 node_api.go (interfaces/ui_api)
//
// ✨ 文件功能说明：
//     接入 internal/query/node 内容，实现 Node 资源相关的逻辑接口：
//     - 获取全部节点列表
//     - 获取节点统计资源使用情况 (CPU / Memory / DiskPressure)
//
// ❌ 不直接依赖 HTTP / gin，用于被 external 层 handler 调用
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package uiapi

import (
	"context"

	"NeuroController/internal/query/node"

	corev1 "k8s.io/api/core/v1"
)

// GetAllNodes 返回全部节点列表
func GetAllNodes(ctx context.Context) ([]corev1.Node, error) {
	return node.ListAllNodes(ctx)
}

// GetNodeMetricsSummary 返回所有节点的资源统计信息
func GetNodeMetricsSummary(ctx context.Context) (*node.NodeMetricsSummary, error) {
	return node.GetNodeMetricsSummary(ctx)
}

// GetNodeOverview 返回节点总览信息（包含总数、Ready 数、总 CPU / 内存 等）
func GetNodeOverview(ctx context.Context) (*node.NodeOverviewResult, error) {
	return node.GetNodeOverview(ctx)
}

// GetNodeDetail 获取指定名称的 Node 的完整原始信息（用于详情页展示）
func GetNodeDetail(ctx context.Context, name string) (*corev1.Node, error) {
	return node.GetNodeDetail(ctx, name)
}
