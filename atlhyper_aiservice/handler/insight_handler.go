package handler

import (
	"AtlHyper/atlhyper_aiservice/service/insight"
	"net/http"

	"github.com/gin-gonic/gin"
)

// InsightRequest —— 通用 AI 洞察请求体
type InsightRequest struct {
	Summary string `json:"summary" binding:"required"` // 必填：运维摘要 / 日志片段 / 告警内容
}

// InsightHandler —— 通用运维 AI 洞察接口
// ------------------------------------------------------------
// 输入自然语言的系统摘要，返回 AI 的结构化诊断结果。
func InsightHandler(c *gin.Context) {
	var req InsightRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid request: " + err.Error(),
			"example": gin.H{
				"summary": "节点 node-1 CPU 使用率持续 98%，Pod 重启频繁",
			},
		})
		return
	}

	// ⚙️ 调用通用 AI 洞察分析
	result, err := insight.RunInsightAnalysis(req.Summary)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "AI analysis failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
