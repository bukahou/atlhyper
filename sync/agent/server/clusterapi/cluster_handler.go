// external/server/uiapi/handler.go

package uiapi

import (
	clusterapi "NeuroController/interfaces/cluster_api"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ===============================
// 📌 GET /agent/uiapi/cluster/overview
// ===============================


func HandleClusterOverview(c *gin.Context) {
	ctx := c.Request.Context()

	overview, err := clusterapi.GetClusterOverview(ctx)
	if err != nil {
		log.Printf("❌ 获取集群概要失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取集群概要失败",
		})
		return
	}

	c.JSON(http.StatusOK, overview)
}