// =======================================================================================
// 📄 handler.go（external/uiapi/cluster）
//
// ✨ 文件说明：
//     实现集群总览接口的 HTTP handler，处理 GET /uiapi/cluster/overview 请求，
//     调用 uiapi 层获取当前 Kubernetes 集群的全局概况（节点、Pod、命名空间等）。
//
// ✅ 接口用途：
//     - 前端首页或 Dashboard 页面展示集群健康状态与资源总览
//
// 📦 依赖模块：
//     - interfaces/ui_api/cluster_api.go 中的 GetClusterOverview(ctx)
//
// 📍 请求方式：
//     GET /uiapi/cluster/overview
//
// 🧪 示例响应：
//     {
//       "nodeCount": 5,
//       "podCount": 43,
//       "k8sVersion": "v1.29.2",
//       "healthyNodeCount": 5,
//       ...
//     }
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package cluster

import (
	"NeuroController/external/server/api/response"

	"github.com/gin-gonic/gin"
)


func ClusterOverviewHandler(c *gin.Context) {
	// 提取上下文，用于 traceID 注入、超时控制、日志记录等
	ctx := c.Request.Context()

	// 调用聚合函数
	payload, err := BuildClusterOverviewAggregated(ctx, c.Query("cluster_id"))
	if err != nil {
		// 统一错误响应
		response.Error(c, err.Error())
		return
	}

	// 成功响应
	response.Success(c, "获取集群概览成功", payload)
}