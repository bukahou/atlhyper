// =======================================================================================
// 📄 handler.go（external/uiapi/service）
//
// ✨ 文件说明：
//     实现 Service 资源的 HTTP 路由处理逻辑，对接 interfaces 层 service_api.go 中封装的查询函数。
//     支持以下功能：
//       - 查询所有命名空间下的 Service
//       - 按命名空间过滤 Service 列表
//       - 获取指定 Service 对象详情
//       - 获取类型为 NodePort / LoadBalancer 的外部服务
//       - 获取 ClusterIP=None 的 Headless 服务（如用于 StatefulSet）
//
// 📍 路由前缀：/uiapi/service/**
//
// 📦 依赖模块：
//     - interfaces/ui_api/service_api.go
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package service

import (
	uiapi "NeuroController/interfaces/ui_api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// =======================================================================================
// ✅ GET /uiapi/service/list/all
//
// 🔍 查询所有命名空间下的 Service 列表
//
// 用于：全局 Service 列表视图展示
// =======================================================================================
func GetAllServicesHandler(c *gin.Context) {
	svcs, err := uiapi.GetAllServices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Service 列表失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, svcs)
}

// =======================================================================================
// ✅ GET /uiapi/service/list/by-namespace/:ns
//
// 🔍 查询指定命名空间下的 Service 列表
//
// 用于：命名空间详情视图、资源筛选等
// =======================================================================================
func GetServicesByNamespaceHandler(c *gin.Context) {
	ns := c.Param("ns")
	svcs, err := uiapi.GetServicesByNamespace(c.Request.Context(), ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取命名空间 Service 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, svcs)
}

// =======================================================================================
// ✅ GET /uiapi/service/get/:ns/:name
//
// 🔍 获取指定命名空间和名称的 Service 对象详情
//
// 用于：Service 详情页、目标对象资源关联跳转
// =======================================================================================
func GetServiceByNameHandler(c *gin.Context) {
	ns := c.Param("ns")
	name := c.Param("name")

	svc, err := uiapi.GetServiceByName(c.Request.Context(), ns, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Service 详情失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, svc)
}

// =======================================================================================
// ✅ GET /uiapi/service/list/external
//
// 🔍 获取所有暴露到集群外部的 Service（NodePort / LoadBalancer）
//
// 用于：负载暴露资源汇总、访问入口规划、安全审计
// =======================================================================================
func GetExternalServicesHandler(c *gin.Context) {
	svcs, err := uiapi.GetExternalServices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取外部 Service 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, svcs)
}

// =======================================================================================
// ✅ GET /uiapi/service/list/headless
//
// 🔍 获取所有 Headless 类型的 Service（ClusterIP=None）
//
// 用于：识别 StatefulSet 配置、服务发现设计等
// =======================================================================================
func GetHeadlessServicesHandler(c *gin.Context) {
	svcs, err := uiapi.GetHeadlessServices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Headless Service 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, svcs)
}
