// =======================================================================================
// 📄 configmap_handler.go（external/uiapi/configmap）
//
// ✨ 文件说明：
//     提供 ConfigMap 资源的 HTTP 路由处理逻辑，连接 interfaces 层逻辑与外部请求。
//     实现功能包括：
//       - 查询所有命名空间下的 ConfigMap
//       - 查询指定命名空间下的 ConfigMap
//       - 获取指定 ConfigMap 的详情
//
// 📍 路由前缀：/uiapi/configmap/**
//
// 📦 依赖模块：
//     - interfaces/ui_api/configmap_api.go
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package configmap

import (
	uiapi "NeuroController/interfaces/ui_api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// =======================================================================================
// ✅ GET /uiapi/configmap/list/by-namespace/:ns
//
// 🔍 查询指定命名空间下的 ConfigMap 列表
// =======================================================================================
func ListConfigMapsByNamespaceHandler(c *gin.Context) {
	ns := c.Param("ns")

	list, err := uiapi.GetConfigMapsByNamespace(c.Request.Context(), ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 ConfigMap 列表失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// =======================================================================================
// ✅ GET /uiapi/configmap/get/:ns/:name
//
// 🔍 获取指定命名空间和名称的 ConfigMap 详情
// =======================================================================================
func GetConfigMapDetailHandler(c *gin.Context) {
	ns := c.Param("ns")
	name := c.Param("name")

	cfg, err := uiapi.GetConfigMapDetail(c.Request.Context(), ns, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 ConfigMap 详情失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// =======================================================================================
// ✅ GET /uiapi/configmap/list
//
// 🔍 查询所有命名空间下的 ConfigMap 列表（用于全局视图）
// =======================================================================================
func ListAllConfigMapsHandler(c *gin.Context) {
	list, err := uiapi.GetAllConfigMaps(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取所有 ConfigMap 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}
