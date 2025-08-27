package uiapi

import (
	clusterapi "NeuroController/internal/interfaces/cluster_api"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /uiapi/node/overview
func HandleGetNodeOverview(c *gin.Context) {
	data, err := clusterapi.GetNodeOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取节点概览失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /uiapi/node/list
func HandleListAllNodes(c *gin.Context) {
	nodes, err := clusterapi.GetAllNodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取所有节点失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, nodes)
}

// GET /uiapi/node/get/:name
func HandleGetNodeDetail(c *gin.Context) {
	name := c.Param("name")
	node, err := clusterapi.GetNodeDetail(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取节点详情失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, node)
}

// GET /uiapi/node/metrics-summary
func HandleGetNodeMetricsSummary(c *gin.Context) {
	data, err := clusterapi.GetNodeMetricsSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取节点指标汇总失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// POST /uiapi/node/unschedulable
func HandleToggleNodeSchedulable(c *gin.Context) {
	type ToggleSchedulableRequest struct {
		Name          string `json:"name" binding:"required"` // 节点名
		Unschedulable bool   `json:"unschedulable"`           // true: 封锁；false: 解封
	}

	var req ToggleSchedulableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
		return
	}

	if err := clusterapi.ToggleNodeSchedulable(req.Name, req.Unschedulable); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "设置节点调度状态失败: " + err.Error()})
		return
	}

	// 🔽 根据参数选择反馈内容
	action := "已封锁"
	if !req.Unschedulable {
		action = "已解封"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("节点 %s %s", req.Name, action),
	})
}
