package web_api

import (
	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/pod"
	"AtlHyper/atlhyper_master/server/api/response"

	"github.com/gin-gonic/gin"
)

//参数: clusterID
//url: /pod/overview?clusterID=cluster1
func GetPodOverviewHandler(c *gin.Context) {
	//获取上下文
	ctx := c.Request.Context()

	//从查询参数中获取 clusterID
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
	}

	// 绑定请求参数
	err := c.ShouldBindJSON(&req)

	// 处理请求参数错误
	if err != nil {
		response.Error(c, "获取 Pod 概览失败: 参数错误")
		return
	}

	// 校验 clusterID 是否为空
	if req.ClusterID == "" {
		response.Error(c, "clusterID is required")
		return
	}

	clusterID := req.ClusterID

	//使用参数 clusterID 调用 BuildPodOverview 函数
	dto, err := pod.BuildPodOverview(ctx, clusterID)

	//如果获取失败，返回错误响应
	if err != nil {
		response.ErrorCode(c, 50000, "构建 Pod Overview 失败: "+err.Error())
		return
	}

	//统一成功返回
	response.Success(c, "获取 Pod 概览成功", dto)
}

//url参数: clusterID, namespace, podName
//url: /pod/detail
func GetPodDetailHandler(c *gin.Context) {
	//获取上下文
	ctx := c.Request.Context()

	//获取参数
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
		PodName   string `json:"podName" binding:"required"`
	}

	// 绑定请求参数
	err := c.ShouldBindJSON(&req)

	// 处理请求参数错误
	if err != nil {
		response.Error(c, "获取 Pod 详情失败: 参数错误")
		return
	}

	// 提取参数
	clusterID := req.ClusterID
	namespace := req.Namespace
	podName := req.PodName

	// 校验参数
	if clusterID == "" || namespace == "" || podName == "" {
		response.Error(c, "clusterID, namespace and podName are required")
		return
	}

	//使用参数调用 GetPodDetail 函数
	dto, err := pod.GetPodDetail(ctx, clusterID, namespace, podName)

	//校验返回结果
	if err != nil {
		response.ErrorCode(c, 50000, "获取 Pod 详情失败: "+err.Error())
		return
	}

	//统一成功返回
	response.Success(c, "获取 Pod 详情成功", dto)
}