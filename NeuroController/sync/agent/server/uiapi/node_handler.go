package uiapi

import (
	uiapi "NeuroController/interfaces/ui_api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /uiapi/node/overview
func HandleGetNodeOverview(c *gin.Context) {
	data, err := uiapi.GetNodeOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取节点概览失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /uiapi/node/list
func HandleListAllNodes(c *gin.Context) {
	nodes, err := uiapi.GetAllNodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取所有节点失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, nodes)
}

// GET /uiapi/node/get/:name
func HandleGetNodeDetail(c *gin.Context) {
	name := c.Param("name")
	node, err := uiapi.GetNodeDetail(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取节点详情失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, node)
}

// GET /uiapi/node/metrics-summary
func HandleGetNodeMetricsSummary(c *gin.Context) {
	data, err := uiapi.GetNodeMetricsSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取节点指标汇总失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}