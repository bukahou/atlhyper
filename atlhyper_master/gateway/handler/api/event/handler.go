// gateway/handler/api/event/handler.go
package event

import (
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	"AtlHyper/atlhyper_master/service/datahub/event"

	"github.com/gin-gonic/gin"
)

// GetListHandler 获取事件列表
// POST /uiapi/event/list
func GetListHandler(c *gin.Context) {
	var req struct {
		ClusterID  string `json:"clusterID" binding:"required"`
		WithinDays int    `json:"withinDays" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取事件列表失败: 参数错误")
		return
	}

	dto, err := event.BuildEventOverview(c.Request.Context(), req.ClusterID, req.WithinDays)
	if err != nil {
		response.ErrorCode(c, 50000, "获取事件列表失败: "+err.Error())
		return
	}

	response.Success(c, "获取事件列表成功", dto)
}
