// =======================================================================================
// 📄 handler.go（external/uiapi/ingress）
//
// ✨ 文件说明：
//     连接 HTTP 路由与 interfaces/ui_api 中的 ingress_api.go，提供 Ingress 的 RESTful 接口处理逻辑：
//     - 查询全集群 Ingress
//     - 查询命名空间下 Ingress
//     - 获取指定 Ingress 对象详情
//     - 获取处于 Ready 状态（已分配 IP）的 Ingress 列表
//
// 📍 路由前缀：/uiapi/ingress/**
//
// 📦 接口来源：
//     - interfaces/ui_api/ingress_api.go
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package ingress

import (
	"NeuroController/external/uiapi/response"
	"NeuroController/sync/center/http/uiapi"
	"net/http"

	"github.com/gin-gonic/gin"
)

// =======================================================================================
// ✅ GET /uiapi/ingress/list/all
//
// 🔍 查询全集群中所有 Ingress 对象
//
// 用于：全局 Ingress 列表展示，集群总览页面
// =======================================================================================
func GetAllIngressesHandler(c *gin.Context) {
	list, err := uiapi.GetAllIngresses()
	if err != nil {
		response.ErrorCode(c, 50000, "获取 Ingress 列表失败: "+err.Error())
		return
	}
	response.Success(c, "获取 Ingress 列表成功", list)
}

// =======================================================================================
// ✅ GET /uiapi/ingress/list/by-namespace/:ns
//
// 🔍 查询指定命名空间下的所有 Ingress 对象
//
// 用于：命名空间详情页面展示其 Ingress 路由配置
// =======================================================================================
func GetIngressesByNamespaceHandler(c *gin.Context) {
	ns := c.Param("ns")

	list, err := uiapi.GetIngressesByNamespace(ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取命名空间 Ingress 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// =======================================================================================
// ✅ GET /uiapi/ingress/get/:ns/:name
//
// 🔍 获取指定命名空间和名称的 Ingress 对象详情
//
// 用于：Ingress 详情页、资源关联链路分析
// =======================================================================================
func GetIngressByNameHandler(c *gin.Context) {
	ns := c.Param("ns")
	name := c.Param("name")

	obj, err := uiapi.GetIngressByName(ns, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Ingress 对象失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, obj)
}

// =======================================================================================
// ✅ GET /uiapi/ingress/list/ready
//
// 🔍 获取已就绪的 Ingress（至少拥有一个 LoadBalancer IP）
//
// 用于：外部服务可访问性检查 / Dashboard 可视化展示
// =======================================================================================
func GetReadyIngressesHandler(c *gin.Context) {
	list, err := uiapi.GetReadyIngresses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Ready 状态 Ingress 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}
