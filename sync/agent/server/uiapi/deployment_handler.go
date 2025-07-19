package uiapi

import (
	uiapi "NeuroController/interfaces/ui_api"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ===============================
// 📌 GET /agent/uiapi/deployments/all
// ===============================
func HandleAllDeployments(c *gin.Context) {
	ctx := c.Request.Context()
	deployments, err := uiapi.GetAllDeployments(ctx)
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

	deployments, err := uiapi.GetDeploymentsByNamespace(ctx, ns)
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

	deployment, err := uiapi.GetDeploymentByName(ctx, ns, name)
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

	deployments, err := uiapi.GetUnavailableDeployments(ctx)
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

	deployments, err := uiapi.GetProgressingDeployments(ctx)
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
	ctx := c.Request.Context()
	namespace := c.Param("ns")
	name := c.Param("name")
	repStr := c.Param("replicas")

	replicas, err := strconv.Atoi(repStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的副本数"})
		return
	}

	err = uiapi.UpdateDeploymentReplicas(ctx, namespace, name, int32(replicas))
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
	ctx := c.Request.Context()
	namespace := c.Param("ns")
	name := c.Param("name")
	image := c.Param("image")

	if image == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 image 参数"})
		return
	}

	err := uiapi.UpdateDeploymentImage(ctx, namespace, name, image)
	if err != nil {
		log.Printf("❌ 更新 Deployment 镜像失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "镜像更新成功"})
}
