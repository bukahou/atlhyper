package web_api

import (
	"AtlHyper/atlhyper_master/service/namespace"
	"AtlHyper/atlhyper_master/server/api/response"

	"github.com/gin-gonic/gin"
)

func GetNamespaceOverviewHandler(c *gin.Context) {

	//获取请求上下文
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		ClusterID string `json:"ClusterID" binding:"required"`
	}

	// 绑定请求参数
	err := c.ShouldBindJSON(&req)

	// 参数校验
	if err != nil {
		response.Error(c, "获取命名空间概览失败: 参数错误")
		return
	}

	// 获取集群ID
	clusterID := req.ClusterID

	// 获取命名空间概览
	dto, err := namespace.BuildNamespaceOverview(ctx, clusterID)

	// 处理错误
	if err != nil {
		response.Error(c, "获取命名空间概览失败: "+err.Error())
		return
	}

	response.Success(c, "命名空间概览获取成功", dto)
	
}

func GetNamespaceDetailHandler(c *gin.Context) {
	// 获取请求上下文
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		ClusterID string `json:"ClusterID" binding:"required"`
		Namespace string `json:"Namespace" binding:"required"`
	}

	// 绑定请求参数
	err := c.ShouldBindJSON(&req)

	// 参数校验
	if err != nil {
		response.Error(c, "获取命名空间详情失败: 参数错误")
		return
	}

	// 获取集群ID和命名空间ID
	clusterID := req.ClusterID
	name := req.Namespace

	// 获取命名空间详情
	dto, err := namespace.BuildNamespaceDetail(ctx, clusterID, name)

	// 处理错误
	if err != nil {
		response.Error(c, "获取命名空间详情失败: "+err.Error())
		return
	}

	response.Success(c, "命名空间详情获取成功", dto)

}
