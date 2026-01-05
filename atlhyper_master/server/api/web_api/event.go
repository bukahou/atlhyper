package web_api

import (
	"AtlHyper/atlhyper_master/service/event"
	"AtlHyper/atlhyper_master/server/api/response"

	"github.com/gin-gonic/gin"
)

func GetEventLogsSinceHandler(c *gin.Context) {
	// 解析请求参数
	var req struct {
		ClusterID  string `json:"clusterID" binding:"required"`
		WithinDays int    `json:"withinDays" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	var clusterID = req.ClusterID
	var withinDays = req.WithinDays

	// 组装 overview（含 cards + rows）
	dto, err := event.BuildEventOverview(clusterID, withinDays)
	if err != nil {
		response.Error(c, "获取事件日志失败")
		return
	}

	response.Success(c, "获取事件日志成功", dto)
}