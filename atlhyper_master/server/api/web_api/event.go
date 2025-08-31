package web_api

import (
	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/event"
	"AtlHyper/atlhyper_master/server/api/response"

	"github.com/gin-gonic/gin"
)

func GetEventLogsSinceHandler(c *gin.Context) {

	// 解析请求参数
	var req struct {
		ClusterID  string `json:"clusterID" binding:"required"`
		WithinDays int    `json:"withinDays" binding:"required"`
	}

	// 绑定请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	// 提取参数
	clusterID := req.ClusterID
	withinDays := req.WithinDays

	// 获取最近的事件日志
	logs, err := event.GetRecentEventLogs(clusterID, withinDays)
	if err != nil {
		response.Error(c, "获取事件日志失败")
		return
	}

	// 返回成功响应
	response.Success(c, "获取事件日志成功", logs)
}
