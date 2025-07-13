// =======================================================================================
// 📄 handler.go（external/uiapi/event）
//
// ✨ 文件说明：
//     绑定 Gin 路由与 interfaces/ui_api/event_api.go，处理 Kubernetes Event 相关 RESTful 请求：
//     - 查询全集群事件
//     - 查询命名空间内事件
//     - 查询某资源对象相关事件
//     - 事件类型聚合统计（Normal / Warning）
//
// 📍 路由前缀：/uiapi/event/**
//
// 📦 调用接口：
//     - interfaces/ui_api/event_api.go
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package event

import (
	uiapi "NeuroController/interfaces/ui_api"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// =======================================================================================
// ✅ GET /uiapi/event/list/all
//
// 🔍 获取全集群范围内所有 Event 对象（含 Normal / Warning）
//
// 用于：事件中心首页、集群总览
// =======================================================================================
func GetAllEventsHandler(c *gin.Context) {
	events, err := uiapi.GetAllEvents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取全部事件失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

// =======================================================================================
// ✅ GET /uiapi/event/list/by-namespace/:ns
//
// 🔍 获取指定命名空间下的所有 Event（包括 Pod、Service 等对象相关）
//
// 用于：命名空间详情页、局部事件筛选
// =======================================================================================
func GetEventsByNamespaceHandler(c *gin.Context) {
	ns := c.Param("ns")

	events, err := uiapi.GetEventsByNamespace(c.Request.Context(), ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取命名空间事件失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

// =======================================================================================
// ✅ GET /uiapi/event/list/by-object/:ns/:kind/:name
//
// 🔍 获取某个资源对象（如 Pod、Deployment）关联的 Event 列表
//
// 用于：资源详情页查看状态历史 / 故障排查
//
// 🔸 示例：/uiapi/event/list/by-object/default/Pod/my-app-xx
// =======================================================================================
func GetEventsByInvolvedObjectHandler(c *gin.Context) {
	ns := c.Param("ns")
	kind := c.Param("kind")
	name := c.Param("name")

	events, err := uiapi.GetEventsByInvolvedObject(c.Request.Context(), ns, kind, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取关联事件失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

// =======================================================================================
// ✅ GET /uiapi/event/summary/type
//
// 🔍 返回 Event 类型分布统计（例如 Warning、Normal 的数量）
//
// 用于：仪表盘总览统计、趋势图分析
// =======================================================================================
func GetEventTypeStatsHandler(c *gin.Context) {
	stats, err := uiapi.GetEventTypeCounts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取事件类型统计失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// =======================================================================================
// ✅ GET /uiapi/eventlog/list/recent
//
// 🔍 获取最近 N 天的结构化告警日志（由 NeuroController 写入磁盘）
//   - 可选 query 参数：days=N（默认 1）
//
// 用于：UI 告警中心页面、节点/资源异常历史回溯
// =======================================================================================
func GetRecentLogEventsHandler(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "1")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		days = 1
	}

	logs, err := uiapi.GetPersistedEventLogs(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "日志读取失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
	})
}
