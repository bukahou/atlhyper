package testapi

import (
	"AtlHyper/atlhyper_master/interfaces/datasource"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 类型别名仅用于显式标注（不影响行为，可留可删）
type (
	Pod                 = datasource.Pod
	Node                = datasource.Node
	Service             = datasource.Service
	Namespace           = datasource.Namespace
	Ingress             = datasource.Ingress
	Deployment          = datasource.Deployment
	ConfigMap           = datasource.ConfigMap
	LogEvent            = datasource.LogEvent
	NodeMetricsSnapshot = datasource.NodeMetricsSnapshot
)

// 1) 事件：最近 N 条（[]LogEvent）
func GetRecentEventsHandler(c *gin.Context) {
	ctx := c.Request.Context()
	clusterID := c.DefaultQuery("cluster_id", "atlhyper")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	rows, err := datasource.GetK8sEventsRecent(ctx, clusterID, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, rows)
}

// 2) 指标：最新一次全量节点快照（[]NodeMetricsSnapshot）
func GetClusterMetricsLatestHandler(c *gin.Context) {
	ctx := c.Request.Context()
	clusterID := c.DefaultQuery("cluster_id", "atlhyper")

	rows, err := datasource.GetClusterMetricsLatest(ctx, clusterID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, rows)
}

// 3) 指标：时间区间（[]NodeMetricsSnapshot）
func GetClusterMetricsRangeHandler(c *gin.Context) {
	ctx := c.Request.Context()
	clusterID := c.DefaultQuery("cluster_id", "atlhyper")
	minutes, _ := strconv.Atoi(c.DefaultQuery("minutes", "15"))

	until := time.Now().UTC()
	since := until.Add(-time.Duration(minutes) * time.Minute)

	rows, err := datasource.GetClusterMetricsRange(ctx, clusterID, since, until)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, rows)
}

// 4) Pods（[]Pod）
func GetPodListLatestHandler(c *gin.Context) {
	ctx := c.Request.Context()
	clusterID := c.DefaultQuery("cluster_id", "atlhyper")

	rows, err := datasource.GetPodListLatest(ctx, clusterID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, rows)
}

// 5) Nodes（[]Node）
func GetNodeListLatestHandler(c *gin.Context) {
	ctx := c.Request.Context()
	clusterID := c.DefaultQuery("cluster_id", "atlhyper")

	rows, err := datasource.GetNodeListLatest(ctx, clusterID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, rows)
}

// 6) Services（[]Service）
func GetServiceListLatestHandler(c *gin.Context) {
	ctx := c.Request.Context()
	clusterID := c.DefaultQuery("cluster_id", "atlhyper")

	rows, err := datasource.GetServiceListLatest(ctx, clusterID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, rows)
}

// 7) Namespaces（[]Namespace）
func GetNamespaceListLatestHandler(c *gin.Context) {
	ctx := c.Request.Context()
	clusterID := c.DefaultQuery("cluster_id", "atlhyper")

	rows, err := datasource.GetNamespaceListLatest(ctx, clusterID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, rows)
}

// 8) Ingresses（[]Ingress）
func GetIngressListLatestHandler(c *gin.Context) {
	ctx := c.Request.Context()
	clusterID := c.DefaultQuery("cluster_id", "atlhyper")

	rows, err := datasource.GetIngressListLatest(ctx, clusterID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, rows)
}

// 9) Deployments（[]Deployment）
func GetDeploymentListLatestHandler(c *gin.Context) {
	ctx := c.Request.Context()
	clusterID := c.DefaultQuery("cluster_id", "atlhyper")

	rows, err := datasource.GetDeploymentListLatest(ctx, clusterID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, rows)
}

// 10) ConfigMaps（[]ConfigMap）
func GetConfigMapListLatestHandler(c *gin.Context) {
	ctx := c.Request.Context()
	clusterID := c.DefaultQuery("cluster_id", "atlhyper")

	rows, err := datasource.GetConfigMapListLatest(ctx, clusterID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, rows)
}
