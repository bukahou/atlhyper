package handler

import (
	"AtlHyper/atlhyper_master/aiservice/builder"
	"AtlHyper/atlhyper_master/aiservice/model"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// 🤖 HandleFetchAIContext —— AI Service 拉取上下文接口
// ----------------------------------------------------------------------------
// 说明：
//   - 用于 AI Service 根据清单获取指定资源详情集合。
//   - 输入：model.AIFetchRequest（包含 clusterID + pods/deployments/... 清单）
//   - 输出：汇总后的结构化上下文（不做包装）。
//   - 特点：内部对接接口，无需 code/message 封装。
// ============================================================================
func HandleFetchAIContext(c *gin.Context) {
	var req model.AIFetchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"detail": err.Error(),
		})
		return
	}

	resp, err := builder.BuildAIContext(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to build AI context",
			"detail": err.Error(),
		})
		return
	}

	// ✅ 成功时直接返回聚合结构体
	c.JSON(http.StatusOK, resp)
}
