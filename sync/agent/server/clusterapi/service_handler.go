package uiapi

import (
	clusterapi "NeuroController/interfaces/cluster_api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /uiapi/service/list
func HandleListAllServices(c *gin.Context) {
	data, err := clusterapi.GetAllServices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Service 列表失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /uiapi/service/list/by-namespace/:ns
func HandleListServicesByNamespace(c *gin.Context) {
	ns := c.Param("ns")
	data, err := clusterapi.GetServicesByNamespace(c.Request.Context(), ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取命名空间 Service 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /uiapi/service/describe
func HandleGetServiceByName(c *gin.Context) {
	ns := c.Query("namespace")
	name := c.Query("name")

	if ns == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数 namespace 或 name"})
		return
	}

	svc, err := clusterapi.GetServiceByName(c.Request.Context(), ns, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Service 详情失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, svc)
}

// GET /uiapi/service/list/external
func HandleListExternalServices(c *gin.Context) {
	data, err := clusterapi.GetExternalServices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取对外 Service 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /uiapi/service/list/headless
func HandleListHeadlessServices(c *gin.Context) {
	data, err := clusterapi.GetHeadlessServices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Headless Service 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}