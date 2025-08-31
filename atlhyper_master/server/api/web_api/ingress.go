package web_api

import (
	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/ingress"
	"AtlHyper/atlhyper_master/server/api/response"

	"github.com/gin-gonic/gin"
)


func GetIngressOverviewHandler(c *gin.Context) {

	// 获取请求上下文
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		ClusterID string `json:"clusterID"`
	}

	// 绑定请求参数
	err := c.ShouldBindJSON(&req)

	// 检查参数是否有效
	if err != nil {
		response.Error(c, "获取 Ingress 概览失败: 参数错误")
		return
	}

	// 获取集群 ID
	clusterID := req.ClusterID

	// 构建 Ingress 概览
	dto, err := ingress.BuildIngressOverview(ctx, clusterID)

	// 检查构建结果是否有效
	if err != nil {
		response.Error(c, "获取 Ingress 概览失败: "+err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, "获取 Ingress 概览成功", dto)

}

func GetIngressDetailHandler(c *gin.Context) {

	// 获取请求上下文
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		ClusterID  string `json:"clusterID"`
		Namespace  string `json:"namespace"`
		Name       string `json:"name"`
	}

	// 绑定请求参数
	err := c.ShouldBindJSON(&req)

	// 检查参数是否有效
	if err != nil {
		response.Error(c, "获取 Namespace 详情失败: 参数错误")
		return
	}

	// 获取集群 ID 和 Namespace
	clusterID := req.ClusterID
	namespace := req.Namespace
	name := req.Name

	// 构建 Ingress 详情
	dto, err := ingress.BuildIngressDetail(ctx, clusterID, namespace, name)

	// 检查构建结果是否有效
	if err != nil {
		response.Error(c, "获取 Ingress 详情失败: "+err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, "获取 Ingress 详情成功", dto)
}