package uiapi

import (
	clusterapi "NeuroController/internal/interfaces/cluster_api"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ===============================
// 📌 GET /agent/uiapi/configmaps/all
// ===============================

func HandleAllConfigMaps(c *gin.Context) {
	ctx := c.Request.Context()

	configMaps, err := clusterapi.GetAllConfigMaps(ctx)
	if err != nil {
		log.Printf("❌ 获取所有 ConfigMap 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, configMaps)
}

// ===============================
// 📌 GET /agent/uiapi/configmaps/by-namespace/:ns
// ===============================

func HandleConfigMapsByNamespace(c *gin.Context) {
	ctx := c.Request.Context()
	ns := c.Param("ns")

	configMaps, err := clusterapi.GetConfigMapsByNamespace(ctx, ns)
	if err != nil {
		log.Printf("❌ 获取命名空间 %s 的 ConfigMap 失败: %v", ns, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, configMaps)
}

// ===============================
// 📌 GET /agent/uiapi/configmaps/detail/:ns/:name
// ===============================

func HandleConfigMapDetail(c *gin.Context) {
	ctx := c.Request.Context()
	ns := c.Param("ns")
	name := c.Param("name")

	cm, err := clusterapi.GetConfigMapDetail(ctx, ns, name)
	if err != nil {
		log.Printf("❌ 获取 ConfigMap 详情失败（%s/%s）: %v", ns, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	if cm == nil {
		c.JSON(http.StatusNotFound, gin.H{"note": "系统保留 ConfigMap，已忽略"})
		return
	}

	c.JSON(http.StatusOK, cm)
}
