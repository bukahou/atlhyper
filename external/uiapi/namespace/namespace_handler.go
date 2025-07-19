// =======================================================================================
// 📄 handler.go（external/uiapi/namespace）
//
// ✨ 文件说明：
//     将 REST 路由请求转发至 interfaces 层的 namespace_api.go 处理逻辑。
//     实现功能包括：
//       - 查询所有命名空间
//       - 查询指定命名空间
//       - 查询 Active / Terminating 状态命名空间
//       - 命名空间状态统计（概览）
//
// 📍 路由前缀：/uiapi/namespace/**
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package namespace

import (
	"NeuroController/sync/center/http/uiapi"
	"net/http"

	"github.com/gin-gonic/gin"
)

// =======================================================================================
// ✅ GET /uiapi/namespace/list
//
// 🔍 查询全集群所有命名空间对象
//
// 用于：命名空间总览、资源选择列表
// =======================================================================================
func ListAllNamespacesHandler(c *gin.Context) {
	namespaces, err := uiapi.GetAllNamespaces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Namespace 列表失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, namespaces)
}

// =======================================================================================
// ✅ GET /uiapi/namespace/get/:name
//
// 🔍 查询指定名称的命名空间对象
//
// 用于：命名空间详情页面或资源跳转定位
// =======================================================================================
// func GetNamespaceByNameHandler(c *gin.Context) {
// 	name := c.Param("name")
// 	ns, err := uiapi.GetNamespaceByName(c.Request.Context(), name)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Namespace 失败: " + err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, ns)
// }

// // =======================================================================================
// // ✅ GET /uiapi/namespace/list/active
// //
// // 🔍 查询所有处于 Active 状态的命名空间
// //
// // 用于：过滤正常使用中的 Namespace，供资源分组/管理使用
// // =======================================================================================
// func ListActiveNamespacesHandler(c *gin.Context) {
// 	active, err := uiapi.GetActiveNamespaces(c.Request.Context())
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Active 命名空间失败: " + err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, active)
// }

// // =======================================================================================
// // ✅ GET /uiapi/namespace/list/terminating
// //
// // 🔍 查询所有处于 Terminating 状态的命名空间
// //
// // 用于：发现删除卡顿命名空间，或告警提示用户关注清理异常
// // =======================================================================================
// func ListTerminatingNamespacesHandler(c *gin.Context) {
// 	terminating, err := uiapi.GetTerminatingNamespaces(c.Request.Context())
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Terminating 命名空间失败: " + err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, terminating)
// }

// // =======================================================================================
// // ✅ GET /uiapi/namespace/summary/status
// //
// // 🔍 获取命名空间状态统计信息（Active / Terminating 数量）
// //
// // 用于：集群状态概览页面，状态分布饼图 / 横条图
// // =======================================================================================
// func GetNamespaceStatusSummaryHandler(c *gin.Context) {
// 	active, terminating, err := uiapi.GetNamespaceStatusStats(c.Request.Context())
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取命名空间状态统计失败: " + err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{
// 		"active":      active,
// 		"terminating": terminating,
// 	})
// }
