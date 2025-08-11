package uiapi

import (
	clusterapi "NeuroController/interfaces/cluster_api"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GET /uiapi/pod/list
func HandleListAllPods(c *gin.Context) {
	data, err := clusterapi.GetAllPods(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取所有 Pod 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /uiapi/pod/list/by-namespace/:ns
func HandleListPodsByNamespace(c *gin.Context) {
	ns := c.Param("ns")
	data, err := clusterapi.GetPodsByNamespace(c.Request.Context(), ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取命名空间 Pod 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /uiapi/pod/summary
func HandlePodStatusSummary(c *gin.Context) {
	data, err := clusterapi.GetPodStatusSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Pod 状态汇总失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /uiapi/pod/usage
func HandlePodUsage(c *gin.Context) {
	data, err := clusterapi.GetPodUsages(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Pod 资源使用情况失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /uiapi/pod/infos
func HandlePodInfos(c *gin.Context) {
	data, err := clusterapi.GetAllPodInfos(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Pod 精简信息失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /uiapi/pod/describe
func HandlePodDescribe(c *gin.Context) {
	ns := c.Query("namespace")
	name := c.Query("name")

	if ns == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数 namespace 或 name"})
		return
	}

	data, err := clusterapi.GetPodDescribe(c.Request.Context(), ns, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Pod 详情失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// POST /uiapi/pod/restart
func HandleRestartPod(c *gin.Context) {
	ns := c.PostForm("namespace")
	name := c.PostForm("name")

	if ns == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数 namespace 或 name"})
		return
	}

	err := clusterapi.RestartPod(c.Request.Context(), ns, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重启 Pod 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "✅ Pod 重启成功"})
}

// GET /uiapi/pod/logs
func HandleGetPodLogs(c *gin.Context) {
	ns := c.Query("namespace")
	name := c.Query("name")
	container := c.DefaultQuery("container", "")
	tailLinesStr := c.DefaultQuery("tailLines", "100")

	tailLines, err := strconv.ParseInt(tailLinesStr, 10, 64)
	if err != nil {
		tailLines = 100
	}

	logs, err := clusterapi.GetPodLogs(c.Request.Context(), ns, name, container, tailLines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Pod 日志失败: " + err.Error()})
		return
	}

	c.String(http.StatusOK, logs)
}