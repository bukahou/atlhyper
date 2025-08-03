// =======================================================================================
// 📄 handler.go（external/uiapi/pod）
//
// ✨ 文件说明：
//     提供 Pod 资源的 HTTP 路由处理逻辑，连接 interfaces 层逻辑与外部请求。
//     实现功能包括：
//       - 查询全部 Pod
//       - 查询指定命名空间下的 Pod
//       - 获取 Pod 状态摘要（Running / Pending / Failed 等）
//       - 获取 Pod 的 CPU / 内存 使用量（非实时）
//
// 📍 路由前缀：/uiapi/pod/**
//
// 📦 依赖模块：
//     - interfaces/ui_api/pod_api.go
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package pod

import (
	"NeuroController/external/uiapi/response"
	"NeuroController/sync/center/http/uiapi"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// =======================================================================================
// ✅ GET /uiapi/pod/list
//
// 🔍 查询所有命名空间下的 Pod 列表（原始对象）
//
// 用于：Pod 全局视图、调试页面等
// =======================================================================================
func ListAllPodsHandler(c *gin.Context) {
	pods, err := uiapi.GetAllPods()
	if err != nil {
		response.Error(c, "获取 Pod 列表失败: "+err.Error())
		return
	}
	response.Success(c, "获取 Pod 列表成功", pods)
}

// =======================================================================================
// ✅ GET /uiapi/pod/list/by-namespace/:ns
//
// 🔍 查询指定命名空间下的 Pod 列表
//
// 用于：命名空间详情页 / 按资源过滤展示
// =======================================================================================
func ListPodsByNamespaceHandler(c *gin.Context) {
	ns := c.Param("ns")
	pods, err := uiapi.GetPodsByNamespace(ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取命名空间 Pod 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, pods)
}



// =======================================================================================
// ✅ GET /uiapi/pod/summary/status
//
// 🔍 获取所有 Pod 状态分布摘要（Running、Pending、Failed 等数量）
//
// 用于：集群 UI 总览图表、资源状态面板
// =======================================================================================

func PodStatusSummaryHandler(c *gin.Context) {
	fmt.Println("✅ PodStatusSummaryHandler 被调用了！")
	summary, err := uiapi.GetPodStatusSummary()
	if err != nil {
		response.Error(c, "获取 Pod 状态摘要失败: "+err.Error())
		return
	}
	response.Success(c, "获取 Pod 状态摘要成功", summary)
}



// =======================================================================================
// ✅ GET /uiapi/pod/metrics/usage
//
// 🔍 获取所有 Pod 的 CPU / Memory 使用信息（聚合视图）
//
// 用于：Pod 资源使用图表、趋势统计模块
// =======================================================================================
func PodMetricsUsageHandler(c *gin.Context) {
	usages, err := uiapi.GetPodUsages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Pod 使用量失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, usages)
}

// =======================================================================================
// ✅ GET /uiapi/pod/list/brief
//
// 🔍 获取所有 Pod 的简略信息（用于 UI 表格展示）
//
// 用于：Pod 列表页、命名空间面板简表、快速浏览
// =======================================================================================
func ListBriefPodsHandler(c *gin.Context) {
	infos, err := uiapi.GetAllPodInfos()
	if err != nil {
		response.Error(c, "获取简略 Pod 列表失败: "+err.Error())
		return
	}
	response.Success(c, "获取简略 Pod 列表成功", infos)
}

// =======================================================================================
// ✅ GET /uiapi/pod/describe/:ns/:name
//
// 🔍 获取指定 Pod 的详细信息（包含事件 Events）
//
// 用于：Pod 详情页、诊断页面跳转后展示
// =======================================================================================
func GetPodDescribeHandler(c *gin.Context) {
	ns := c.Param("ns")
	name := c.Param("name")

	info, err := uiapi.GetPodDescribe(ns, name)
	if err != nil {
		response.ErrorCode(c, 50000, "获取 Pod 详情失败: "+err.Error())
		return
	}

	response.Success(c, "获取成功", info)
}
// ============================================================================================================================================
// ============================================================================================================================================
// 操作函数
// ============================================================================================================================================
// ============================================================================================================================================

// =======================================================================================
// ✅ GET /uiapi/pod/logs/:ns/:name
//
// 🔍 获取指定 Pod 的容器日志（默认获取第一个容器）
//
//	支持通过 query 参数指定容器名和 tail 行数：
//	?container=xxx&tail=100
//
// 用于：Pod 详情页 / 日志弹窗展示
// =======================================================================================
func GetPodLogsHandler(c *gin.Context) {
	ns := c.Param("ns")
	name := c.Param("name")

	container := c.Query("container") // 可选
	tailLines := int64(100)           // 默认 tail 100 行

	if tailStr := c.Query("tail"); tailStr != "" {
		if parsed, err := strconv.ParseInt(tailStr, 10, 64); err == nil {
			tailLines = parsed
		}
	}

	logs, err := uiapi.GetPodLogs(ns, name, container, tailLines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取日志失败: " + err.Error(),
			"message": "请检查 Pod/容器是否存在，或日志权限配置",
		})
		return
	}

	c.String(http.StatusOK, logs)
}


// =======================================================================================
// ✅ POST /uiapi/pod/restart/:ns/:name
//
// 🔁 重启指定 Pod（通过删除实现，控制器自动重新创建）
//
// 用于：Pod 详情页「重启」按钮
// =======================================================================================
// func RestartPodHandler(c *gin.Context) {
// 	ns := c.Param("ns")
// 	name := c.Param("name")

// 	err := uiapi.RestartPod(ns, name)
// 	if err != nil {
// 		// ✅ 打印详细错误信息
// 		log.Printf("❌ 重启 Pod 失败：%v", err)

// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"error":   "重启 Pod 失败: " + err.Error(),
// 			"message": "可能是该 Pod 不存在，或权限不足",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Pod 已成功重启（删除完成，控制器将自动拉起副本）",
// 		"pod": gin.H{
// 			"namespace": ns,
// 			"name":      name,
// 		},
// 	})
// }