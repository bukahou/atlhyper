package pod

import (
	"NeuroController/external/uiapi/response"
	"NeuroController/sync/center/http/uiapi"
	"log"

	"github.com/gin-gonic/gin"
)

// RegisterPodOpsRoutes 注册重大操作类路由（如重启）
// func RegisterPodOpsRoutes(router *gin.RouterGroup) {
// 	router.POST("/restart/:ns/:name", RestartPodHandler)
// }


func RestartPodHandler(c *gin.Context) {
	ns := c.Param("ns")
	name := c.Param("name")

	err := uiapi.RestartPod(ns, name)
	if err != nil {
		log.Printf("❌ 重启 Pod 失败：%v", err)
		response.Error(c, "重启 Pod 失败: "+err.Error()+"（可能是该 Pod 不存在，或权限不足）")
		return
	}

	response.Success(c, "Pod 已成功重启（控制器将自动拉起副本）", gin.H{
		"namespace": ns,
		"name":      name,
	})
}