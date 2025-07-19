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
	"NeuroController/sync/center/http/uiapi"
	"net/http"

	"github.com/gin-gonic/gin"
)


func ClusterOverviewHandler(c *gin.Context) {
	// 提取上下文，用于 traceID 注入、超时控制、日志记录等


	// 调用 UI API 接口获取集群概要信息（节点数、Pod 数、版本等）
	overview, err := uiapi.GetClusterOverview()
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
