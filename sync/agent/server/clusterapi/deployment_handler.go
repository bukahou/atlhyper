package uiapi

import (
	clusterapi "NeuroController/interfaces/cluster_api"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ===============================
// 📌 GET /agent/uiapi/deployments/all
// ===============================
func HandleAllDeployments(c *gin.Context) {
	ctx := c.Request.Context()
	deployments, err := clusterapi.GetAllDeployments(ctx)
	if err != nil {
		log.Printf("❌ 获取所有 Deployment 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, deployments)
}

// ===============================
// 📌 GET /agent/uiapi/deployments/by-namespace/:ns
// ===============================
func HandleDeploymentsByNamespace(c *gin.Context) {
	ctx := c.Request.Context()
	ns := c.Param("ns")

	deployments, err := clusterapi.GetDeploymentsByNamespace(ctx, ns)
	if err != nil {
		log.Printf("❌ 获取命名空间 %s 的 Deployment 失败: %v", ns, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, deployments)
}

// ===============================
// 📌 GET /agent/uiapi/deployments/detail/:ns/:name
// ===============================
func HandleDeploymentDetail(c *gin.Context) {
	ctx := c.Request.Context()
	ns := c.Param("ns")
	name := c.Param("name")

	deployment, err := clusterapi.GetDeploymentByName(ctx, ns, name)
	if err != nil {
		log.Printf("❌ 获取 Deployment %s/%s 失败: %v", ns, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, deployment)
}

// ===============================
// 📌 GET /agent/uiapi/deployments/unavailable
// ===============================
func HandleUnavailableDeployments(c *gin.Context) {
	ctx := c.Request.Context()

	deployments, err := clusterapi.GetUnavailableDeployments(ctx)
	if err != nil {
		log.Printf("❌ 获取不可用 Deployment 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, deployments)
}

// ===============================
// 📌 GET /agent/uiapi/deployments/progressing
// ===============================
func HandleProgressingDeployments(c *gin.Context) {
	ctx := c.Request.Context()

	deployments, err := clusterapi.GetProgressingDeployments(ctx)
	if err != nil {
		log.Printf("❌ 获取 Progressing Deployment 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, deployments)
}

// ===============================
// 📌 POST /agent/uiapi/deployments/scale/:ns/:name?replicas=3
// ===============================
func HandleUpdateDeploymentReplicas(c *gin.Context) {

	type UpdateReplicasRequest struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
		Replicas  int32  `json:"replicas"`
	}

	var req UpdateReplicasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
		return
	}

	if req.Namespace == "" || req.Name == "" || req.Replicas < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数"})
		return
	}

	ctx := c.Request.Context()

	err := clusterapi.UpdateDeploymentReplicas(ctx, req.Namespace, req.Name, req.Replicas)
	if err != nil {
		log.Printf("❌ 更新 Deployment 副本数失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "副本数更新成功"})
}

// ===============================
// 📌 POST /agent/uiapi/deployments/image/:ns/:name?image=nginx:latest
// ===============================
func HandleUpdateDeploymentImage(c *gin.Context) {

	type UpdateImageRequest struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
		Image     string `json:"image"`
	}

	var req UpdateImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
		return
	}

	if req.Image == "" || req.Namespace == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数"})
		return
	}

	ctx := c.Request.Context()
	err := clusterapi.UpdateDeploymentImage(ctx, req.Namespace, req.Name, req.Image)
	if err != nil {
		log.Printf("❌ 更新 Deployment 镜像失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "镜像更新成功"})
}
