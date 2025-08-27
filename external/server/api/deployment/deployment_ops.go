package deployment

import (
	"NeuroController/external/server/api/response"
	"NeuroController/sync/center/http/uiapi"

	"github.com/gin-gonic/gin"
)

// RegisterDeploymentOpsRoutes 注册 Deployment 的操作类接口（如缩容/镜像更新）
// func RegisterDeploymentOpsRoutes(rg *gin.RouterGroup) {
// 	rg.POST("/scale", ScaleDeploymentHandler)
// }

type ScaleDeploymentRequest struct {
	Namespace string  `json:"namespace" binding:"required"`
	Name      string  `json:"name" binding:"required"`
	Replicas  *int32  `json:"replicas"` 
	Image     string  `json:"image"`    
}


// ScaleDeploymentHandler 处理 Deployment 的副本数和镜像更新
//
// 支持以下组合：
//   - 仅更新副本数
//   - 仅更新镜像
//   - 同时更新副本数与镜像
func ScaleDeploymentHandler(c *gin.Context) {
	var req ScaleDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效")
		return
	}

	hasImage := req.Image != ""
	hasReplicas := req.Replicas != nil

	var replicaUpdated, imageUpdated bool

	// ✅ 更新副本数
	if hasReplicas {
		if err := uiapi.UpdateDeploymentReplicas(req.Namespace, req.Name, *req.Replicas); err != nil {
			response.ErrorCode(c, 50000, "更新副本数失败: "+err.Error())
			return
		}
		replicaUpdated = true
	}

	// ✅ 更新镜像
	if hasImage {
		if err := uiapi.UpdateDeploymentImage(req.Namespace, req.Name, req.Image); err != nil {
			response.ErrorCode(c, 50000, "更新镜像失败: "+err.Error())
			return
		}
		imageUpdated = true
	}

	// ❌ 都没提供
	if !replicaUpdated && !imageUpdated {
		response.Error(c, "未提供需要更新的字段（replicas 或 image）")
		return
	}

	// ✅ 成功响应
	response.Success(c, "更新成功", gin.H{
		"replicasUpdated": replicaUpdated,
		"imageUpdated":    imageUpdated,
		"replicas":        req.Replicas,
		"image":           req.Image,
	})
}