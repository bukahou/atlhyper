package web_api

import (
	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/service"
	"AtlHyper/atlhyper_master/server/api/response"

	"github.com/gin-gonic/gin"
)


func GetServiceOverviewHandler(c *gin.Context) {

	//获取上下文
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		ClusterID string `json:"clusterID"`
	}

	// 绑定请求参数
	err := c.ShouldBindJSON(&req)

	// 处理请求参数错误
	if err != nil {
		response.Error(c, "获取 Service 概览失败: 参数错误")
		return
	}

	// 校验参数
	if req.ClusterID == "" {

		response.Error(c, "获取 Service 概览失败: 参数错误")
		return
	}

	clusterID := req.ClusterID

	// 获取 Service 概览
	dto, err := service.BuildServiceOverview(ctx, clusterID)

	// 处理响应
	if err != nil {
		response.Error(c, "获取 Service 概览失败: "+err.Error())
		return
	}

	// 处理响应
	response.Success(c, "获取 Service 概览成功", dto)
}

//参数: clusterID
//url: /service/detail
func GetServiceDetailHandler(c *gin.Context) {

	//获取上下文
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
		Name      string `json:"name" binding:"required"`
	}

	// 绑定请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取 Service 详情失败: 参数错误")
		return
	}

	// 提取参数
	clusterID := req.ClusterID
	namespace := req.Namespace
	name := req.Name

	// 校验参数
	if clusterID == "" || namespace == "" || name == "" {

		response.Error(c, "获取 Service 详情失败: 参数错误")
		return
	}

	// 获取 Service 详情
	dto, err := service.GetServiceDetail(ctx, clusterID, namespace, name)

	// 处理响应
	if err != nil {
		response.Error(c, "获取 Service 详情失败: "+err.Error())
		return
	}
	response.Success(c, "获取 Service 详情成功", dto)
}


