// =======================================================================================
// 📄 handler.go（external/uiapi/node）
//
// ✨ 文件说明：
//     提供 Node 资源的 HTTP 路由处理逻辑，连接 interfaces 层逻辑与外部请求。
//     实现功能包括：
//       - 获取集群所有节点信息
//       - 获取节点资源使用概要（CPU、内存、Ready 状态等）
//
// 📍 路由前缀：/uiapi/node/**
//
// 📦 依赖模块：
//     - interfaces/ui_api/node_api.go
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package node

import (
	uiapi "NeuroController/interfaces/ui_api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// =======================================================================================
// ✅ GET /uiapi/node/list
//
// 🔍 查询所有 Node 资源信息（原始节点对象）
//
// 用于：集群节点列表页、节点信息展示页面
// =======================================================================================
func GetAllNodesHandler(c *gin.Context) {
	nodes, err := uiapi.GetAllNodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Node 列表失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, nodes)
}

// =======================================================================================
// ✅ GET /uiapi/node/metrics
//
// 🔍 获取所有节点的资源使用概要（如 CPU、Memory、Ready 状态等）
//
// 用于：UI 节点概览图表、资源使用汇总分析（非实时）
// =======================================================================================
func GetNodeMetricsSummaryHandler(c *gin.Context) {
	summary, err := uiapi.GetNodeMetricsSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Node Metrics 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// =======================================================================================
// ✅ GET /uiapi/node/overview
//
// 🔍 获取节点总览数据（包括统计卡片 + 表格简要信息）
//
// 用于：UI 概览页中节点模块的汇总展示
// =======================================================================================
func GetNodeOverviewHandler(c *gin.Context) {
	result, err := uiapi.GetNodeOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Node 总览失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// =======================================================================================
// ✅ GET /uiapi/node/get/:name
//
// 🔍 获取指定 Node 的完整详细信息（系统信息、资源、网络、镜像等）
//
// 用于：Node 详情页展示
// =======================================================================================
func GetNodeDetailHandler(c *gin.Context) {
	name := c.Param("name")
	node, err := uiapi.GetNodeDetail(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Node 详情失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, node)
}
