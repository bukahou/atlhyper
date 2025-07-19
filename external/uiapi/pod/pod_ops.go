package pod

import (
	"NeuroController/sync/center/http/uiapi"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterPodOpsRoutes 注册重大操作类路由（如重启）
func RegisterPodOpsRoutes(router *gin.RouterGroup) {
	router.POST("/restart/:ns/:name", RestartPodHandler)
}


func RestartPodHandler(c *gin.Context) {
	ns := c.Param("ns")
	name := c.Param("name")

	err := uiapi.RestartPod(ns, name)
	if err != nil {
		// ✅ 打印详细错误信息
		log.Printf("❌ 重启 Pod 失败：%v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "重启 Pod 失败: " + err.Error(),
			"message": "可能是该 Pod 不存在，或权限不足",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pod 已成功重启（删除完成，控制器将自动拉起副本）",
		"pod": gin.H{
			"namespace": ns,
			"name":      name,
		},
	})
}
