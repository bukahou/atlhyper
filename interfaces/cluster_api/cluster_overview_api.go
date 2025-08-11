// =======================================================================================
//
// ✨ 文件功能说明：
//     提供集群首页概要信息接口，用于前端 UI 展示全局状态（节点、Pod、版本等）。
//
// 📍 API 路由：GET /api/cluster/overview
//
// 📦 依赖模块：
//     - internal/query/cluster.GetClusterOverview()
//     - 外部注入 context 与封装的日志系统
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// 📅 创建时间：2025年7月
// =======================================================================================

package clusterapi

import (
	"NeuroController/internal/query/cluster"
	"context"
)

// GetClusterOverview 提供集群概览数据接口（供 external 层调用）
func GetClusterOverview(ctx context.Context) (interface{}, error) {
	return cluster.GetClusterOverview(ctx)
}
