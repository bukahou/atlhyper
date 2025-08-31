package testapi

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	ifaceevent "AtlHyper/atlhyper_master/interfaces/test_interfaces"

	"github.com/gin-gonic/gin"
)

// HandleGetRecentEvents
// GET /api/events/recent?cluster_id=xxx[&within=15m]
//
// within 可选，默认 15m，支持 time.ParseDuration 语法（如 10m、1h）。
func HandleGetStoreEvents(c *gin.Context) {
	clusterID := c.Query("cluster_id")
	if clusterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "缺少必填参数：cluster_id"})
		return
	}

	withinStr := c.DefaultQuery("within", "15m")
	within, err := time.ParseDuration(withinStr)
	if err != nil || within <= 0 {
		within = 15 * time.Minute
	}
	// 上限保护，避免一次查询过大窗口（可按需调整）
	if within > 15*time.Minute {
		within = 15 * time.Minute
	}

	// 避免阻塞
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	events, err := ifaceevent.GetRecentEventsByCluster(ctx, clusterID, within)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "获取近期事件失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clusterId": clusterID,
		"window":    within.String(),
		"count":     len(events),
		"events":    events,
	})
}


func HandleGetDbEvents(c *gin.Context) {
	clusterID := c.Query("cluster_id")
	if clusterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "缺少必填参数：cluster_id"})
		return
	}

	// 1) 优先读 days
	days := 1
	if ds := c.Query("days"); ds != "" {
		if v, err := strconv.Atoi(ds); err == nil && v > 0 {
			days = v
		}
	} else if ws := c.Query("within"); ws != "" {
		// 2) 兼容旧的 within（duration），向上取整为天数，至少 1
		if dur, err := time.ParseDuration(ws); err == nil && dur > 0 {
			ceilDays := int(math.Ceil(dur.Hours() / 24.0))
			if ceilDays < 1 {
				ceilDays = 1
			}
			days = ceilDays
		}
	}

	events, err := ifaceevent.GetRecentEventLogs(clusterID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "获取事件失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clusterId": clusterID,
		"days":      days,
		"count":     len(events),
		"events":    events,
	})
}