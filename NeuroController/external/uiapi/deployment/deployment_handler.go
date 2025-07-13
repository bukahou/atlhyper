// =======================================================================================
// 📄 handler.go（external/uiapi/deployment）
//
// ✨ 文件说明：
//     实现 Deployment 相关查询的 HTTP handler，包括：
//     - 全部 Deployment
//     - 指定命名空间下的 Deployment
//     - 获取 Deployment 详情
//     - 获取不可用或进行中状态的 Deployment（用于 UI 告警中心）
//
// 📍 路由前缀：/uiapi/deployment/**
//
// 📦 调用接口：
//     - interfaces/ui_api/deployment_api.go
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package deployment

import (
	uiapi "NeuroController/interfaces/ui_api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// =======================================================================================
// ✅ GET /uiapi/deployment/list
//
// 🔍 获取所有命名空间下的 Deployment 列表
//
// 用于：前端全局视图 / 搜索 / 集群资源浏览
// =======================================================================================
func GetAllDeploymentsHandler(c *gin.Context) {
	ctx := c.Request.Context()

	list, err := uiapi.GetAllDeployments(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Deployment 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// =======================================================================================
// ✅ GET /uiapi/deployment/list/:namespace
//
// 🔍 获取指定命名空间下的 Deployment 列表
//
// 用于：命名空间资源详情页、资源分组展示
// =======================================================================================
func GetDeploymentsByNamespaceHandler(c *gin.Context) {
	ctx := c.Request.Context()
	ns := c.Param("ns")

	if ns == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少命名空间参数"})
		return
	}

	list, err := uiapi.GetDeploymentsByNamespace(ctx, ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Deployment 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// =======================================================================================
// ✅ GET /uiapi/deployment/get/:namespace/:name
//
// 🔍 获取指定命名空间和名称的 Deployment 对象详情
//
// 用于：Deployment 详情页 / 弹窗查看配置与状态
// =======================================================================================
func GetDeploymentByNameHandler(c *gin.Context) {
	ctx := c.Request.Context()
	ns := c.Param("ns")
	name := c.Param("name")

	if ns == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少命名空间或名称参数"})
		return
	}

	dep, err := uiapi.GetDeploymentByName(ctx, ns, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Deployment 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, dep)
}

// =======================================================================================
// ✅ GET /uiapi/deployment/unavailable
//
// 🔍 获取所有不可用状态的 Deployment（AvailableReplicas < DesiredReplicas）
//
// 用于：告警中心 / 概览卡片提醒 / 健康性检查
// =======================================================================================
func GetUnavailableDeploymentsHandler(c *gin.Context) {
	ctx := c.Request.Context()

	list, err := uiapi.GetUnavailableDeployments(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取不可用 Deployment 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// =======================================================================================
// ✅ GET /uiapi/deployment/progressing
//
// 🔍 获取处于更新中状态的 Deployment（Progressing 条件未满足）
//
// 用于：滚动更新进度监控 / 告警检测
// =======================================================================================
func GetProgressingDeploymentsHandler(c *gin.Context) {
	ctx := c.Request.Context()

	list, err := uiapi.GetProgressingDeployments(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 progressing 状态 Deployment 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// =======================================================================================
// ✅ POST /uiapi/deployment/scale
//
// 🔧 修改指定 Deployment 的副本数（扩/缩容）
//
// 用于：Deployment 详情页的副本数修改操作
// =======================================================================================

type ScaleDeploymentRequest struct {
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Replicas  int32  `json:"replicas" binding:"required"`
}

func ScaleDeploymentHandler(c *gin.Context) {
	var req ScaleDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
		return
	}

	ctx := c.Request.Context()
	err := uiapi.UpdateDeploymentReplicas(ctx, req.Namespace, req.Name, req.Replicas)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新副本数失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "副本数更新成功",
		"replicas": req.Replicas,
	})
}
