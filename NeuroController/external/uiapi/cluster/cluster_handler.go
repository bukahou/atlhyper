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
	uiapi "NeuroController/interfaces/ui_api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ClusterOverviewHandler 处理集群概览接口 GET /uiapi/cluster/overview
//
// 🧩 处理流程：
//  1. 从 gin.Context 中提取请求上下文（用于日志、trace 等）
//  2. 调用 uiapi 层的 GetClusterOverview 获取当前集群状态概览
//  3. 成功则返回 200 + JSON 数据，失败则返回 500 + 错误描述
//
// ✅ 成功响应示例：
//
//	HTTP/1.1 200 OK
//	Content-Type: application/json
//	{
//	  "nodeCount": 3,
//	  "podCount": 85,
//	  "k8sVersion": "v1.28.3"
//	}
//
// ❌ 失败响应示例：
//
//	HTTP/1.1 500 Internal Server Error
//	Content-Type: application/json
//	{
//	  "error": "无法获取集群概要信息: ..."
//	}
func ClusterOverviewHandler(c *gin.Context) {
	// 提取上下文，用于 traceID 注入、超时控制、日志记录等
	ctx := c.Request.Context()

	// 调用 UI API 接口获取集群概要信息（节点数、Pod 数、版本等）
	overview, err := uiapi.GetClusterOverview(ctx)
	if err != nil {
		// 发生错误时，返回 500 错误信息
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无法获取集群概要信息: " + err.Error(),
		})
		return
	}

	// 正常返回 JSON 格式的集群信息
	c.JSON(http.StatusOK, overview)
}
